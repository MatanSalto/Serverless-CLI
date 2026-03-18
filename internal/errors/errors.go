package errors

import (
	"fmt"
	"strings"
)

// ValidationError represents an invalid or missing input parameter.
type ValidationError struct {
	Field  string
	Reason string
	Value  any
}

func (e ValidationError) Error() string {
	field := strings.TrimSpace(e.Field)
	reason := strings.TrimSpace(e.Reason)

	if field == "" {
		field = "value"
	}
	if reason == "" {
		reason = "is invalid"
	}
	return fmt.Sprintf("%s %s", field, reason)
}

// SizeLimitError indicates a size constraint violation (e.g., ConfigMap limit).
type SizeLimitError struct {
	Resource    string
	LimitBytes  int64
	ActualBytes int64
}

func (e SizeLimitError) Error() string {
	res := strings.TrimSpace(e.Resource)
	if res == "" {
		res = "resource"
	}
	return fmt.Sprintf("%s size %d bytes exceeds limit (%d bytes)", res, e.ActualBytes, e.LimitBytes)
}

// NotFoundError is used when a higher-level resource resolution fails
type NotFoundError struct {
	Resource  string
	Name      string
	Namespace string
}

func (e NotFoundError) Error() string {
	res := strings.TrimSpace(e.Resource)
	if res == "" {
		res = "resource"
	}
	if e.Namespace != "" {
		return fmt.Sprintf("%s %q not found in namespace %q", res, e.Name, e.Namespace)
	}
	return fmt.Sprintf("%s %q not found", res, e.Name)
}

// StateError is used when a resource exists but is not in a usable state yet.
type StateError struct {
	Resource  string
	Name      string
	Namespace string
	Reason    string
}

func (e StateError) Error() string {
	res := strings.TrimSpace(e.Resource)
	if res == "" {
		res = "resource"
	}
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		reason = "is not ready"
	}
	if e.Namespace != "" {
		return fmt.Sprintf("%s %q in namespace %q %s", res, e.Name, e.Namespace, reason)
	}
	return fmt.Sprintf("%s %q %s", res, e.Name, reason)
}

// KubeOpError wraps a Kubernetes API-related failure with structured context.
// It preserves the cause via Unwrap function.
type KubeOpError struct {
	Op        string
	Resource  string
	Name      string
	Namespace string
	Err       error
}

func (e KubeOpError) Error() string {
	op := strings.TrimSpace(e.Op)
	res := strings.TrimSpace(e.Resource)
	if op == "" {
		op = "kube-op"
	}
	if res == "" {
		res = "resource"
	}

	var target string
	if e.Name != "" && e.Namespace != "" {
		target = fmt.Sprintf("%s %q in namespace %q", res, e.Name, e.Namespace)
	} else if e.Name != "" {
		target = fmt.Sprintf("%s %q", res, e.Name)
	} else if e.Namespace != "" {
		target = fmt.Sprintf("%s in namespace %q", res, e.Namespace)
	} else {
		target = res
	}

	if e.Err != nil {
		return fmt.Sprintf("%s %s: %v", op, target, e.Err)
	}
	return fmt.Sprintf("%s %s", op, target)
}

func (e KubeOpError) Unwrap() error {
	return e.Err
}
