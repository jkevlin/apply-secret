# apply-secret

This code builds a minimal binary of the lightweight kubernetes
client and exposes it for manual testing.
The intention is that the binary can be built and dropped into
a Kube environment like this:

https://kubernetes.io/docs/tasks/debug-application-cluster/get-shell-running-container/

Then, commands can be run to test its API calls.
The above commands are intended to be run inside an instance of
minikube that has been started.
After building this binary, place it in the container like this:

```bash
kubectl cp kubeclient /shell-demo:/
```

At first you may get 403's, which can be resolved using this:

https://github.com/fabric8io/fabric8/issues/6840#issuecomment-307560275

Example calls:

```bash
./apply-secret -namespace='default' -secret-name='shell-demo' -file=secret.yaml
```
