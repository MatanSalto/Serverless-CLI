package kube

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClientSet returns a clientSet for the cluster.
func NewClientSet() (kubernetes.Interface, error) {
	config, err := restConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// NewRestConfig returns the rest.Config for the cluster (e.g. for port-forwarding).
func NewRestConfig() (*rest.Config, error) {
	return restConfig()
}

func restConfig() (*rest.Config, error) {
	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
