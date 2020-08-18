package controllers

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	scheme "k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	TestingNamespace string = "nvmesh"
)

func init() {
	// change dir to repo root
	os.Chdir("../")
	newDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error changing directory")
	}
	fmt.Printf("Current Working Direcotry: %s\n", newDir)
}

type MyTestEnv struct {
	Config *rest.Config
	Scheme *runtime.Scheme
	Client client.Client
}

func NewTestEnv() (*MyTestEnv, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	myScheme := runtime.NewScheme()
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	testEnv := &MyTestEnv{
		Config: cfg,
		Scheme: myScheme,
		Client: c,
	}
	return testEnv, nil
}
