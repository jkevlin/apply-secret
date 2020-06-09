package main

// This code builds a minimal binary of the lightweight kubernetes
// client and exposes it for manual testing.
// The intention is that the binary can be built and dropped into
// a Kube environment like this:
// https://kubernetes.io/docs/tasks/debug-application-cluster/get-shell-running-container/
// Then, commands can be run to test its API calls.
// The above commands are intended to be run inside an instance of
// minikube that has been started.
// After building this binary, place it in the container like this:
// $ kubectl cp kubeclient /shell-demo:/
// At first you may get 403's, which can be resolved using this:
// https://github.com/fabric8io/fabric8/issues/6840#issuecomment-307560275
//
// Example calls:
// 	./kubeclient -call='get-secret' -namespace='default' -secret-name='shell-demo'
// 	./kubeclient -call='patch-secret' -namespace='default' -secret-name='shell-demo' -patches='/metadata/labels/fizz:buzz,/metadata/labels/foo:bar'

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/jkevlin/apply-secret/client"
)

var callToMake string
var patchesToAdd string
var namespace string
var secretName string
var yamlFilename string

func init() {
	flag.StringVar(&callToMake, "call", "", `the call to make: 'get-secret' or 'apply-secret'`)
	flag.StringVar(&patchesToAdd, "patches", "", `if call is "patch-secret", the patches to do like so: "/metadata/labels/fizz:buzz,/metadata/labels/foo:bar"`)
	flag.StringVar(&namespace, "namespace", "", "the namespace to use")
	flag.StringVar(&secretName, "secret-name", "", "the secret name to use")
	flag.StringVar(&yamlFilename, "file", "", "the secret file name to apply")
}

func main() {
	flag.Parse()

	c, err := client.New(hclog.Default())
	if err != nil {
		panic(err)
	}

	reqCh := make(chan struct{})
	shutdownCh := makeShutdownCh()

	go func() {
		defer close(reqCh)

		switch callToMake {
		case "get-secret":
			secret, err := c.GetSecret(namespace, secretName)
			if err != nil {
				panic(err)
			}
			b, _ := json.Marshal(secret)
			fmt.Printf("secret: %s\n", b)
			return
		case "apply-secret":
			_, err = c.ApplySecret(namespace, secretName, yamlFilename)
			if err != nil {
				panic(err)
			}
			return
		case "patch-secret":
			patchPairs := strings.Split(patchesToAdd, ",")
			var patches []*client.Patch
			for _, patchPair := range patchPairs {
				fields := strings.Split(patchPair, ":")
				if len(fields) != 2 {
					panic(fmt.Errorf("unable to split %s from selectors provided of %s", fields, patchesToAdd))
				}
				patches = append(patches, &client.Patch{
					Operation: client.Replace,
					Path:      fields[0],
					Value:     fields[1],
				})
			}
			if err := c.PatchSecret(namespace, secretName, patches...); err != nil {
				panic(err)
			}
			return
		default:
			panic(fmt.Errorf(`unsupported call provided: %q`, callToMake))
		}
	}()

	select {
	case <-shutdownCh:
		fmt.Println("Interrupt received, exiting...")
	case <-reqCh:
	}
}

func makeShutdownCh() chan struct{} {
	resultCh := make(chan struct{})

	shutdownCh := make(chan os.Signal, 4)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownCh
		close(resultCh)
	}()
	return resultCh
}
