package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodePortServiceURL returns the NodePort and a full URL (http://host:nodePort) for a NodePort Service.
// Host is taken from the kubeconfig API server (public IP) when available, otherwise from a node's ExternalIP or InternalIP.
// Returns (0, "") if the service is nil, has no ports, or has no NodePort.
func NodePortServiceURL(ctx context.Context, client kubernetes.Interface, svc *corev1.Service) (int32, string) {
	if svc == nil || len(svc.Spec.Ports) == 0 {
		return 0, ""
	}
	p := svc.Spec.Ports[0]
	if p.NodePort == 0 {
		return 0, ""
	}
	host, _ := APIServerHost()
	if host == "" {
		host = nodeHost(ctx, client)
	}
	if host == "" {
		return p.NodePort, ""
	}
	return p.NodePort, fmt.Sprintf("http://%s:%d", host, p.NodePort)
}

// nodeHost returns the first node's ExternalIP or InternalIP for building service URLs.
func nodeHost(ctx context.Context, client kubernetes.Interface) string {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		return ""
	}
	var internalIP string
	for _, addr := range nodes.Items[0].Status.Addresses {
		if addr.Address == "" {
			continue
		}
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address
		}
		if addr.Type == corev1.NodeInternalIP && internalIP == "" {
			internalIP = addr.Address
		}
	}
	return internalIP
}
