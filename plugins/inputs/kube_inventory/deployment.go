package kubeinventory

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"

	appsv1 "k8s.io/api/apps/v1"
)

func collectDeployments(ctx context.Context, acc cua.Accumulator, ki *KubernetesInventory) {
	list, err := ki.client.getDeployments(ctx)
	if err != nil {
		acc.AddError(err)
		return
	}
	for _, d := range list.Items {
		ki.gatherDeployment(d, acc)
	}
}

func (ki *KubernetesInventory) gatherDeployment(d appsv1.Deployment, acc cua.Accumulator) {
	fields := map[string]interface{}{
		"replicas_available":   d.Status.AvailableReplicas,
		"replicas_unavailable": d.Status.UnavailableReplicas,
		"created":              d.GetCreationTimestamp().UnixNano(),
	}
	tags := map[string]string{
		"deployment_name": d.Name,
		"namespace":       d.Namespace,
	}
	for key, val := range d.Spec.Selector.MatchLabels {
		if ki.selectorFilter.Match(key) {
			tags["selector_"+key] = val
		}
	}

	acc.AddFields(deploymentMeasurement, fields, tags)
}
