package testing

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync"
	"testing"

	"go.uber.org/atomic"
)

const (
	ExpectedNamespace  = "default"
	ExpectedSecretName = "shell-demo"
	ExpectedYAMLFile   = "secret.yaml"

	// File names of samples pulled from real life.
	caCrtFile        = "ca.crt"
	respGetSecret    = "resp-get-secret.json"
	respNotFound     = "resp-not-found.json"
	respUpdateSecret = "resp-update-secret.json"
	tokenFile        = "token"
	yamlFile         = "secret.yaml"
)

var (
	// ReturnGatewayTimeouts toggles whether the test server should return,
	// well, gateway timeouts...
	ReturnGatewayTimeouts = atomic.NewBool(false)

	pathToFiles = func() string {
		wd, _ := os.Getwd()
		resecretName := "vault-enterprise"
		if !strings.Contains(wd, resecretName) {
			resecretName = "vault"
		}
		pathParts := strings.Split(wd, resecretName)
		return pathParts[0] + "/../testing/"
	}()
)

// Conf returns the info needed to configure the client to secretint at
// the test server. This must be done by the caller to avoid an imsecretrt
// cycle between the client and the testserver. Example usage:
//
//		client.Scheme = testConf.ClientScheme
//		client.TokenFile = testConf.PathToTokenFile
//		client.RootCAFile = testConf.PathToRootCAFile
//		if err := os.Setenv(client.EnvVarKubernetesServiceHost, testConf.ServiceHost); err != nil {
//			t.Fatal(err)
//		}
//		if err := os.Setenv(client.EnvVarKubernetesServicePort, testConf.ServicePort); err != nil {
//			t.Fatal(err)
//		}
type Conf struct {
	ClientScheme, PathToTokenFile, PathToRootCAFile, ServiceHost, ServicePort string
}

// Server returns an http test server that can be used to test
// Kubernetes client code. It also retains the current state,
// and a func to close the server and to clean up any temsecretrary
// files.
func Server(t *testing.T) (testState *State, testConf *Conf, closeFunc func()) {
	testState = &State{m: &sync.Map{}}
	testConf = &Conf{
		ClientScheme: "http://",
	}

	// We're going to have multiple close funcs to call.
	var closers []func()
	closeFunc = func() {
		for _, closer := range closers {
			closer()
		}
	}

	// Read in our sample files.
	token, err := readFile(tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	caCrt, err := readFile(caCrtFile)
	if err != nil {
		t.Fatal(err)
	}

	// Plant our token in a place where it can be read for the config.
	tmpToken, err := ioutil.TempFile("", "token")
	if err != nil {
		t.Fatal(err)
	}
	closers = append(closers, func() {
		os.Remove(tmpToken.Name())
	})
	if _, err = tmpToken.WriteString(token); err != nil {
		closeFunc()
		t.Fatal(err)
	}
	if err := tmpToken.Close(); err != nil {
		closeFunc()
		t.Fatal(err)
	}
	testConf.PathToTokenFile = tmpToken.Name()

	tmpCACrt, err := ioutil.TempFile("", "ca.crt")
	if err != nil {
		closeFunc()
		t.Fatal(err)
	}
	closers = append(closers, func() {
		os.Remove(tmpCACrt.Name())
	})
	if _, err = tmpCACrt.WriteString(caCrt); err != nil {
		closeFunc()
		t.Fatal(err)
	}
	if err := tmpCACrt.Close(); err != nil {
		closeFunc()
		t.Fatal(err)
	}
	testConf.PathToRootCAFile = tmpCACrt.Name()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ReturnGatewayTimeouts.Load() {
			w.WriteHeader(504)
			return
		}
		namespace, secretName, err := parsePath(r.URL.Path)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("unable to parse %s: %s", r.URL.Path, err.Error())))
			return
		}

		switch {
		case namespace != ExpectedNamespace, secretName != ExpectedSecretName:
			w.WriteHeader(404)
			w.Write([]byte("ok"))
			return
		case r.Method == http.MethodPost:
			w.WriteHeader(200)
			w.Write([]byte("ok"))
			return
		case r.Method == http.MethodPut:
			w.WriteHeader(200)
			//w.Write([]byte(getSecretResponse))
			return
		default:
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("unexpected request method: %s", r.Method)))
		}
	}))
	closers = append(closers, ts.Close)

	// ts.URL example: http://127.0.0.1:35681
	urlFields := strings.Split(ts.URL, "://")
	if len(urlFields) != 2 {
		closeFunc()
		t.Fatal("received unexpected test url: " + ts.URL)
	}
	urlFields = strings.Split(urlFields[1], ":")
	if len(urlFields) != 2 {
		closeFunc()
		t.Fatal("received unexpected test url: " + ts.URL)
	}
	testConf.ServiceHost = urlFields[0]
	testConf.ServicePort = urlFields[1]
	return testState, testConf, closeFunc
}

type State struct {
	m *sync.Map
}

func (s *State) NumPatches() int {
	l := 0
	f := func(key, value interface{}) bool {
		l++
		return true
	}
	s.m.Range(f)
	return l
}

func (s *State) Get(key string) map[string]interface{} {
	v, ok := s.m.Load(key)
	if !ok {
		return nil
	}
	patch, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return patch
}

func (s *State) store(k string, p map[string]interface{}) {
	s.m.Store(k, p)
}

// The path should be formatted like this:
// fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, secretName)
func parsePath(urlPath string) (namespace, secretName string, err error) {
	original := urlPath
	secretName = path.Base(urlPath)
	urlPath = strings.TrimSuffix(urlPath, "/secrets/"+secretName)
	namespace = path.Base(urlPath)
	if original != fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, secretName) {
		return "", "", fmt.Errorf("received unexpected path: %s", original)
	}
	return namespace, secretName, nil
}

func readFile(fileName string) (string, error) {
	b, err := ioutil.ReadFile(pathToFiles + fileName)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
