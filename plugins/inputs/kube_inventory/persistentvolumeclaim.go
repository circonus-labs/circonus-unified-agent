package kubeinventory

import (
	"context"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	v1 "github.com/ericchiang/k8s/apis/core/v1"
)

func collectPersistentVolumeClaims(ctx context.Context, acc cua.Accumulator, ki *KubernetesInventory) {
	list, err := ki.client.getPersistentVolumeClaims(ctx)
	if err != nil {
		acc.AddError(err)
		return
	}
	for _, pvc := range list.Items {
		ki.gatherPersistentVolumeClaim(*pvc, acc)
		// if err = ki.gatherPersistentVolumeClaim(*pvc, acc); err != nil {
		// 	acc.AddError(err)
		// 	return
		// }
	}
}

func (ki *KubernetesInventory) gatherPersistentVolumeClaim(pvc v1.PersistentVolumeClaim, acc cua.Accumulator) {
	phaseType := 3
	switch strings.ToLower(pvc.Status.GetPhase()) {
	case "bound":
		phaseType = 0
	case "lost":
		phaseType = 1
	case "pending":
		phaseType = 2
	}
	fields := map[string]interface{}{
		"phase_type": phaseType,
	}
	tags := map[string]string{
		"pvc_name":     pvc.Metadata.GetName(),
		"namespace":    pvc.Metadata.GetNamespace(),
		"phase":        pvc.Status.GetPhase(),
		"storageclass": pvc.Spec.GetStorageClassName(),
	}
	for key, val := range pvc.GetSpec().GetSelector().GetMatchLabels() {
		if ki.selectorFilter.Match(key) {
			tags["selector_"+key] = val
		}
	}

	acc.AddFields(persistentVolumeClaimMeasurement, fields, tags)
}
