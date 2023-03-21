package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const TypeReplicaSet = "ReplicaSet"
const TypeDeployment = "Deployment"

var myPodName string
var myPodNameSpace string

func contains(In []string, find string) bool {
	for _, v := range In {
		if strings.EqualFold(v, find) {
			return true
		}
	}
	return false
}

func isPartOfDeployment(ctx context.Context, clientset *kubernetes.Clientset, pod *v1.Pod, ownerRefs []metav1.OwnerReference) bool {
	if len(ownerRefs) == 0 {
		return false
	}

	ownerRef := ownerRefs[0]

	switch ownerRef.Kind {
	case TypeReplicaSet:

		replica, repErr := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
		if repErr != nil {
			log.Printf("Error getting replica info : [%v]\n", repErr.Error())
			return false
		}

		repOwnRef := replica.OwnerReferences

		if len(repOwnRef) > 0 && repOwnRef[0].Kind == TypeDeployment {
			return true
		}
	}
	return false
}

func delPod(ctx context.Context, clientset *kubernetes.Clientset) {
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting pod info. Err : [%v]\n", err.Error())
	}

	deps, ok := os.LookupEnv("SKIP_DEPLOYMENTS")

	var depsArr []string
	if ok {
		depsArr = strings.Split(deps, ",")
		log.Println("Deployments to skip are:", deps)
	}

	for _, pod := range pods.Items {

		if contains(depsArr, pod.Namespace) {
			log.Printf("Skipped: Pod [%v] belong to skipped deployment : [%v]\n", pod.Name, pod.Namespace)
			continue
		}

		// do not delete the pods which belongs to k8 related deployments
		if pod.Namespace == "kube-system" {
			continue
		}

		if isPartOfDeployment(ctx, clientset, &pod, pod.OwnerReferences) {

			if strings.EqualFold(myPodName, pod.Name) && strings.EqualFold(myPodNameSpace, pod.Namespace) {
				log.Printf("Skipped: Pod [%v] cannot delete itself\n", pod.Name)
				continue
			}

			// delete the pod
			err := clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				log.Printf("Error deleting pod [%v]. Err: [%v]", pod.Name, err.Error())
				continue
			}
			log.Printf("Deleted: pod name : [%v], namespace: [%v]\n", pod.Name, pod.Namespace)
		}
	}
}

func main() {
	ctx, stopF := context.WithCancel(context.Background())
	var config *rest.Config
	var err error
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	pollDuration := flag.Int("poll", 10, "poll duration in seconds. Default 10s.")
	flag.Parse()

	if *kubeconfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	log.Printf("kubeconfig : [%v]\n", *kubeconfig)
	log.Printf("pollDuration : [%v]s\n", *pollDuration)

	if err != nil {
		log.Fatalf("Error gettting configs : Err : [%v]\n", err.Error())
	}

	var ok bool
	myPodName, ok = os.LookupEnv("POD_NAME")
	if !ok {
		log.Fatalf("POD_NAME env not set")
	}

	myPodNameSpace, ok = os.LookupEnv("POD_NAMESPACE")
	if !ok {
		log.Fatalf("POD_NAMESPACE env not set")
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset : Err : [%v]\n", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	delPod(ctx, clientset)
	for {
		select {
		case <-c:
			stopF()
			return
		case <-time.After(time.Duration(*pollDuration) * time.Second):
			delPod(ctx, clientset)
		}
	}
}
