package kubeinventory

import (
	"context"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	v1 "github.com/ericchiang/k8s/apis/apps/v1"
)

func collectDeployments(ctx context.Context, acc cua.Accumulator, ki *KubernetesInventory) {
	list, err := ki.client.getDeployments(ctx)
	if err != nil {
		acc.AddError(err)
		return
	}
	for _, d := range list.Items {
		ki.gatherDeployment(*d, acc)
		// if err = ki.gatherDeployment(*d, acc); err != nil {
		// 	acc.AddError(err)
		// 	return
		// }
	}
}

func (ki *KubernetesInventory) gatherDeployment(d v1.Deployment, acc cua.Accumulator) {
	fields := map[string]interface{}{
		"replicas_available":   d.Status.GetAvailableReplicas(),
		"replicas_unavailable": d.Status.GetUnavailableReplicas(),
		"created":              time.Unix(d.Metadata.CreationTimestamp.GetSeconds(), int64(d.Metadata.CreationTimestamp.GetNanos())).UnixNano(),
	}
	tags := map[string]string{
		"deployment_name": d.Metadata.GetName(),
		"namespace":       d.Metadata.GetNamespace(),
	}
	for key, val := range d.GetSpec().GetSelector().GetMatchLabels() {
		if ki.selectorFilter.Match(key) {
			tags["selector_"+key] = val
		}
	}

	acc.AddFields(deploymentMeasurement, fields, tags)
}
