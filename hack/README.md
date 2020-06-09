# Demo

## In a terminal window

```bash
minikube start --driver=docker

kubectl apply -f shell-demo.yaml
# wait for pod to be ready
kubectl get pod shell-demo

# If you want to build a new version
env GOOS=linux GOARCH=arm64 go build -o apply-secret ../client/cmd/kubeclient
# Copy artifacts to minikube
kubectl cp apply-secret /shell-demo:/
kubectl cp create-secret.yaml /shell-demo:/
kubectl cp update-secret.yaml /shell-demo:/
kubectl exec -it shell-demo -- /bin/bash
```

## In container terminal

```bash
# create the secret
./apply-secret -namespace='default' -file='create-secret.yaml'
# update the secret
./apply-secret -namespace='default' -file='update-secret.yaml'
exit
```

## Back in the terminal window

```bash
kubectl get secret some-secret -o yaml
```
