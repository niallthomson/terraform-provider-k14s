package util

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
)

type DepsFactoryImpl struct {
	configFactory cmdcore.ConfigFactory
}

func NewDepsFactoryImpl(configFactory cmdcore.ConfigFactory) *DepsFactoryImpl {
	return &DepsFactoryImpl{configFactory}
}

func (f *DepsFactoryImpl) DynamicClient() (dynamic.Interface, error) {
	config, err := f.configFactory.RESTConfig()
	if err != nil {
		return nil, err
	}

	// TODO high QPS
	config.QPS = 1000
	config.Burst = 1000

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Building Dynamic clientset: %s", err)
	}

	//f.printTarget(config)

	return clientset, nil
}

func (f *DepsFactoryImpl) CoreClient() (kubernetes.Interface, error) {
	config, err := f.configFactory.RESTConfig()
	if err != nil {
		return nil, err
	}

	config.QPS = 1000
	config.Burst = 1000

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Building Core clientset: %s", err)
	}

	//f.printTarget(config)

	return clientset, nil
}

func Resolver(returnVal string) func() (string, error) {
	return func() (string, error) {
		return returnVal, nil
	}
}
