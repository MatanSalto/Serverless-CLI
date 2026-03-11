package kube

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

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

// APIServerHost returns the host (IP or hostname) from the kubeconfig API server URL,
// so service URLs can use the same address used to reach the cluster (e.g. public IP).
func APIServerHost() (string, error) {
	config, err := NewRestConfig()
	if err != nil || config == nil || config.Host == "" {
		return "", err
	}
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return "", nil
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	h := u.Hostname()
	if h == "" {
		h = u.Host
	}
	return h, nil
}
