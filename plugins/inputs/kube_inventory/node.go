package kubeinventory

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"

	corev1 "k8s.io/api/core/v1"
)

const (
	resourceCPU    = "cpu"
	resourceMemory = "memory"
	resourcePods   = "pods"
)

func collectNodes(ctx context.Context, acc cua.Accumulator, ki *KubernetesInventory) {
	list, err := ki.client.getNodes(ctx)
	if err != nil {
		acc.AddError(err)
		return
	}
	for _, n := range list.Items {
		ki.gatherNode(n, acc)
	}
}

func (ki *KubernetesInventory) gatherNode(n corev1.Node, acc cua.Accumulator) {
	fields := map[string]interface{}{}
	tags := map[string]string{
		"node_name": n.Name,
	}

	for resourceName, val := range n.Status.Capacity {
		switch resourceName {
		case resourceCPU:
			fields["capacity_cpu_cores"] = convertQuantity(string(val.Format), 1)
			fields["capicity_millicpu_cores"] = convertQuantity(string(val.Format), 1000)
		case resourceMemory:
			fields["capacity_memory_bytes"] = convertQuantity(string(val.Format), 1)
		case resourcePods:
			fields["capacity_pods"] = atoi(string(val.Format))
		}
	}

	for resourceName, val := range n.Status.Allocatable {
		switch resourceName {
		case resourceCPU:
			fields["allocatable_cpu_cores"] = convertQuantity(string(val.Format), 1)
			fields["allocatable_millicpu_cores"] = convertQuantity(string(val.Format), 1000)
		case resourceMemory:
			fields["allocatable_memory_bytes"] = convertQuantity(string(val.Format), 1)
		case resourcePods:
			fields["allocatable_pods"] = atoi(string(val.Format))
		}
	}

	acc.AddFields(nodeMeasurement, fields, tags)
}
