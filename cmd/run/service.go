package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	serr "serverless-cli/internal/errors"
	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/runner"
)

var (
	serviceEntrypoint string
	serviceName       string
	servicePort       int
)

var ServiceCmd = &cobra.Command{
	Use:   "service <source-path> [args...]",
	Short: "Run a Python program as a long-running service",
	Long:  `Run a Python program as a Deployment with a NodePort Service. The entrypoint should bind to the port (e.g. Flask on 0.0.0.0:PORT). The Service will be exposed via a NodePort on your cluster nodes.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runService,
}

func init() {
	ServiceCmd.Flags().StringVar(&serviceEntrypoint, "entrypoint", "", "Script to run under /opt/code (default: main.py for dirs, or the filename for a single file)")
	ServiceCmd.Flags().StringVar(&serviceName, "name", "", "Deployment/Service name (default: generated from source path)")
	ServiceCmd.Flags().IntVar(&servicePort, "port", 8080, "Container port the app listens on")
}

func runService(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sourcePath := args[0]
	var restArgs []string
	if len(args) > 1 {
		restArgs = args[1:]
	}

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-workloads"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return serr.KubeOpError{
			Op:       "create",
			Resource: "kubernetes client",
			Err:      err,
		}
	}

	entrypoint := serviceEntrypoint
	if entrypoint == "" {
		abs, _ := filepath.Abs(sourcePath)
		if abs != "" {
			sourcePath = abs
		}
		info, err := os.Stat(sourcePath)
		if err == nil && info != nil && !info.IsDir() {
			entrypoint = filepath.Base(sourcePath)
		} else {
			entrypoint = "main.py"
		}
	}

	name := serviceName
	if name == "" {
		base := filepath.Base(sourcePath)
		if base == "." || base == "/" {
			base = "run"
		}
		suffix, _ := randomHex(6)
		name = "slp-" + base + "-" + suffix
	}

	port := int32(servicePort)
	if port <= 0 {
		port = 8080
	}
	svc, err := runner.RunServiceSource(ctx, client, runner.RunServiceSourceParams{
		SourcePath:  sourcePath,
		Namespace:   namespace,
		ServiceName: name,
		RunnerImage: "matansalto/serverless-python:1.0.0",
		Entrypoint:  entrypoint,
		Port:        port,
		Args:        restArgs,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Deployment %q and Service %q created in namespace %q.\n", name, name, namespace)

	nodePort, urlStr := kube.NodePortServiceURL(ctx, client, svc)
	if nodePort > 0 {
		fmt.Printf("Service is exposed on NodePort %d.\n", nodePort)
		if urlStr != "" {
			fmt.Printf("You can access it at: %s\n", urlStr)
		} else {
			fmt.Printf("Use any cluster node IP with NodePort %d (e.g. http://<node-ip>:%d).\n", nodePort, nodePort)
		}
	} else {
		fmt.Printf("Warning: Service does not have a NodePort assigned. Check the Service spec.\n")
	}

	return nil
}
