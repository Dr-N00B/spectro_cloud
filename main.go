package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func contains(In []string, find string) bool {
	for _, v := range In {
		if strings.EqualFold(v, find) {
			return true
		}
	}
	return false
}

func delDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting deployments : [%v]\n", err)
		return
	}

	deps, ok := os.LookupEnv("SKIP_DEPLOYMENTS")

	var depsArr []string
	if ok {
		depsArr = strings.Split(deps, ",")
	}

	for _, d := range deployments.Items {

		if contains(depsArr, d.Name) {
			log.Printf("deployment [%v] set to skip", d.Name)
			continue
		}

		if d.Namespace == "kube-system" {
			log.Printf("deployment [%v] is part of kube-system. Skipping", d.Name)
			continue
		}

		deletePolicy := metav1.DeletePropagationForeground
		err := clientset.AppsV1().Deployments(d.Namespace).Delete(ctx, d.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})

		if err != nil {
			log.Printf("Error deleting deployment : [%v]. Err : [%v]\n", d.Name, err)
			continue
		}

		fmt.Printf("Deleted deployment : [%v].\n", d.Name)
	}

}

func main() {
	ctx := context.Background()

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

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset : Err : [%v]\n", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	delDeployments(ctx, clientset, "")

	for {
		select {
		case <-c:
			return
		case <-time.After(time.Duration(*pollDuration) * time.Second):
			delDeployments(ctx, clientset, "")
		}
	}

}
