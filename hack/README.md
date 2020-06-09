# Shell Demo

```bash
kubectl apply -f shell-demo.yaml

kubectl get pod shell-demo
kubectl exec -it shell-demo -- /bin/bash

```

## in your shell

```bash

apt-get update
apt-get install curl
curl localhost
```

# in a local window

```bash
env GOOS=linux GOARCH=arm64 go build ../client/cmd/kubeclient
kubectl cp kubeclient /shell-demo:/
kubectl exec -it shell-demo -- /bin/bash
```

```bash

./kubeclient -call='get-secret' -namespace='default' -secret-name='shell-demo'
```