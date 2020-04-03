package util

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DepsFactoryImpl struct {
	config *rest.Config

	client *kubernetes.Clientset
}

func NewDepsFactoryImpl(config *rest.Config) *DepsFactoryImpl {
	return &DepsFactoryImpl{config: config}
}

func (f *DepsFactoryImpl) DynamicClient() (dynamic.Interface, error) {
	f.config.QPS = 1000
	f.config.Burst = 1000

	clientset, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return nil, fmt.Errorf("Building Dynamic clientset: %s", err)
	}

	return clientset, nil
}

func (f *DepsFactoryImpl) CoreClient() (kubernetes.Interface, error) {
	if f.client != nil {
		return f.client, nil
	}

	f.config.QPS = 1000
	f.config.Burst = 1000

	clientset, err := kubernetes.NewForConfig(f.config)
	if err != nil {
		return nil, fmt.Errorf("Building Core clientset: %s", err)
	}

	return clientset, nil
}

func Resolver(returnVal string) func() (string, error) {
	return func() (string, error) {
		return returnVal, nil
	}
}
