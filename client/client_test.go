package client

import (
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	kubetest "github.com/jkevlin/apply-secret/testing"
)

func TestClient(t *testing.T) {
	testState, testConf, closeFunc := kubetest.Server(t)
	defer closeFunc()

	Scheme = testConf.ClientScheme
	TokenFile = testConf.PathToTokenFile
	RootCAFile = testConf.PathToRootCAFile
	if err := os.Setenv(EnvVarKubernetesServiceHost, testConf.ServiceHost); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(EnvVarKubernetesServicePort, testConf.ServicePort); err != nil {
		t.Fatal(err)
	}

	client, err := New(hclog.Default())
	if err != nil {
		t.Fatal(err)
	}
	e := &env{
		client:    client,
		testState: testState,
	}
	e.TestApplySecret(t)
}

type env struct {
	client    *Client
	testState *kubetest.State
}

func (e *env) TestApplySecret(t *testing.T) {
	err := e.client.ApplySecret(kubetest.ExpectedNamespace, kubetest.ExpectedSecretName, "../testing/"+kubetest.ExpectedYAMLFile)
	if err != nil {
		t.Fatal(err)
	}
}

func (e *env) TestGetSecretNotFound(t *testing.T) {
	_, err := e.client.GetSecret(kubetest.ExpectedNamespace, "no-exist")
	if err == nil {
		t.Fatal("expected error because secret is unfound")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Fatalf("expected *ErrNotFound but received %T", err)
	}
}
