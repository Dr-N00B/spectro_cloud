# Spectro Cloud

## Deploy
```
make deploy
```

This will deploy the pod to the cluster. It will create a standalone pod and pull the image from dockerhub.

## Testing
Run  
``` 
make tests 
```
 to create test deployments.

The pod should delete only deployments and standalone pods, statefulset and replicaset should not be removed.

## delete deployment
Run 
```
make delete 
```

## Build image from code
Run
```
make build
```

## Run code
Currently the code will delete deployments every 10 seconds. This interval can be overridden by passing **--poll num** argument in deploy.yaml file.

## Skip deployment to delete
Update **assignment-config** configmap from deploy.yaml, set *skip-deployments* to desired deployment name. If multiple deployments are to be skipped, pass comma seprated values