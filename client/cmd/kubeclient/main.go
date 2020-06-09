package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/jkevlin/apply-secret/client"
)

var namespace string
var secretName string
var yamlFilename string

func init() {
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

		err = c.ApplySecret(namespace, yamlFilename)
		if err != nil {
			panic(err)
		}
		return
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
