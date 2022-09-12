package kubeinventory

import (
	"context"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"

	corev1 "k8s.io/api/core/v1"
)

func collectPersistentVolumes(ctx context.Context, acc cua.Accumulator, ki *KubernetesInventory) {
	list, err := ki.client.getPersistentVolumes(ctx)
	if err != nil {
		acc.AddError(err)
		return
	}
	for _, pv := range list.Items {
		ki.gatherPersistentVolume(pv, acc)
	}
}

func (ki *KubernetesInventory) gatherPersistentVolume(pv corev1.PersistentVolume, acc cua.Accumulator) {
	phaseType := 5
	switch strings.ToLower(string(pv.Status.Phase)) {
	case "bound":
		phaseType = 0
	case "failed":
		phaseType = 1
	case "pending":
		phaseType = 2
	case "released":
		phaseType = 3
	case "available":
		phaseType = 4
	}
	fields := map[string]interface{}{
		"phase_type": phaseType,
	}
	tags := map[string]string{
		"pv_name":      pv.Name,
		"phase":        string(pv.Status.Phase),
		"storageclass": pv.Spec.StorageClassName,
	}

	acc.AddFields(persistentVolumeMeasurement, fields, tags)
}
