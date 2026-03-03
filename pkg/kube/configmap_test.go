package kube

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateConfigMap(t *testing.T) {
	t.Run("configmap is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ConfigMapParams{
			Namespace: "default",
			Name:      "test-config",
			Data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}
		cm, err := CreateConfigMap(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateConfigMap: %v", err)
		}

		if cm == nil {
			t.Fatal("CreateConfigMap returned nil configmap")
		}
		if cm.Name != params.Name {
			t.Errorf("cm.Name = %q, want %q", cm.Name, params.Name)
		}
		if cm.Namespace != params.Namespace {
			t.Errorf("cm.Namespace = %q, want %q", cm.Namespace, params.Namespace)
		}
		if len(cm.Data) != 2 {
			t.Errorf("len(cm.Data) = %d, want 2", len(cm.Data))
		}
		if cm.Data["key1"] != "value1" || cm.Data["key2"] != "value2" {
			t.Errorf("cm.Data = %v, want map[key1:value1 key2:value2]", cm.Data)
		}

		got, err := client.CoreV1().ConfigMaps(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("ConfigMaps().Get: %v", err)
		}
		if got.UID != cm.UID {
			t.Error("retrieved configmap UID does not match created configmap")
		}
	})

	t.Run("configmap with nil data is created with empty data", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ConfigMapParams{
			Namespace: "default",
			Name:      "empty-config",
			Data:      nil,
		}
		cm, err := CreateConfigMap(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateConfigMap: %v", err)
		}
		if cm.Data == nil {
			t.Error("expected non-nil empty Data map")
		}
		if len(cm.Data) != 0 {
			t.Errorf("len(cm.Data) = %d, want 0", len(cm.Data))
		}
	})

	t.Run("configmap with empty namespace returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateConfigMap(ctx, client, ConfigMapParams{Namespace: "", Name: "test", Data: map[string]string{}})
		if err == nil {
			t.Error("CreateConfigMap with empty namespace: want error, got nil")
		}
	})

	t.Run("configmap with empty name returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateConfigMap(ctx, client, ConfigMapParams{Namespace: "default", Name: "", Data: map[string]string{}})
		if err == nil {
			t.Error("CreateConfigMap with empty name: want error, got nil")
		}
	})
}
