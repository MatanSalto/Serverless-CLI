package kube

// Labels applied to all workload resources created by the CLI.
// Use these when creating Jobs, CronJobs, Deployments, etc., and when
// listing or filtering workloads (e.g. list command).
const (
	// LabelManagedKey is set on every resource created by the CLI.
	// Value should be LabelManagedValue. Use ManagedBySelector to list all such resources.
	LabelManagedKey   = "serverless-cli.dev/managed"
	LabelManagedValue = "true"

	// LabelWorkloadTypeKey indicates the kind of workload (one-off, async, cron, website).
	// Values: WorkloadTypeOneOff, WorkloadTypeAsync, WorkloadTypeCron, WorkloadTypeWebsite.
	LabelWorkloadTypeKey = "serverless-cli.dev/workload-type"
	WorkloadTypeOneOff   = "one-off"
	WorkloadTypeAsync    = "async"
	WorkloadTypeCron     = "cron"
	WorkloadTypeWebsite  = "website"
	WorkloadTypeService  = "service"
)

// ManagedBySelector is a label selector that matches any resource created by this CLI.
// Use it when listing Jobs, CronJobs, Deployments, etc. to show only CLI-managed workloads.
const ManagedBySelector = LabelManagedKey + "=" + LabelManagedValue
