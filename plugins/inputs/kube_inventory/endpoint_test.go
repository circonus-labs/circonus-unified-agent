package kubeinventory

import (
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/testutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEndpoint(t *testing.T) {
	cli := &client{}

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 1, 36, 0, now.Location())

	tests := []struct {
		handler  *mockHandler
		output   *testutil.Accumulator
		name     string
		hasError bool
	}{
		{
			name: "no endpoints",
			handler: &mockHandler{
				responseMap: map[string]interface{}{
					"/endpoints/": &corev1.EndpointsList{},
				},
			},
			hasError: false,
		},
		{
			name: "collect ready endpoints",
			handler: &mockHandler{
				responseMap: map[string]interface{}{
					"/endpoints/": &corev1.EndpointsList{
						Items: []corev1.Endpoints{
							{
								Subsets: []corev1.EndpointSubset{
									{
										Addresses: []corev1.EndpointAddress{
											{
												Hostname: "storage-6",
												NodeName: toStrPtr("b.storage.internal"),
												TargetRef: &corev1.ObjectReference{
													Kind: "pod",
													Name: "storage-6",
												},
											},
										},
										Ports: []corev1.EndpointPort{
											{
												Name:     "server",
												Protocol: "TCP",
												Port:     8080,
											},
										},
									},
								},
								ObjectMeta: metav1.ObjectMeta{
									Generation:        12,
									Namespace:         "ns1",
									Name:              "storage",
									CreationTimestamp: metav1.Time{Time: now},
								},
							},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Fields: map[string]interface{}{
							"ready":      true,
							"port":       int32(8080),
							"generation": int64(12),
							"created":    now.UnixNano(),
						},
						Tags: map[string]string{
							"endpoint_name": "storage",
							"namespace":     "ns1",
							"hostname":      "storage-6",
							"node_name":     "b.storage.internal",
							"port_name":     "server",
							"port_protocol": "TCP",
							"pod":           "storage-6",
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "collect notready endpoints",
			handler: &mockHandler{
				responseMap: map[string]interface{}{
					"/endpoints/": &corev1.EndpointsList{
						Items: []corev1.Endpoints{
							{
								Subsets: []corev1.EndpointSubset{
									{
										NotReadyAddresses: []corev1.EndpointAddress{
											{
												Hostname: "storage-6",
												NodeName: toStrPtr("b.storage.internal"),
												TargetRef: &corev1.ObjectReference{
													Kind: "pod",
													Name: "storage-6",
												},
											},
										},
										Ports: []corev1.EndpointPort{
											{
												Name:     "server",
												Protocol: "TCP",
												Port:     8080,
											},
										},
									},
								},
								ObjectMeta: metav1.ObjectMeta{
									Generation:        12,
									Namespace:         "ns1",
									Name:              "storage",
									CreationTimestamp: metav1.Time{Time: now},
								},
							},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Fields: map[string]interface{}{
							"ready":      false,
							"port":       int32(8080),
							"generation": int64(12),
							"created":    now.UnixNano(),
						},
						Tags: map[string]string{
							"endpoint_name": "storage",
							"namespace":     "ns1",
							"hostname":      "storage-6",
							"node_name":     "b.storage.internal",
							"port_name":     "server",
							"port_protocol": "TCP",
							"pod":           "storage-6",
						},
					},
				},
			},
			hasError: false,
		},
	}

	for _, v := range tests {
		ks := &KubernetesInventory{
			client: cli,
		}
		acc := new(testutil.Accumulator)
		for _, endpoint := range ((v.handler.responseMap["/endpoints/"]).(*corev1.EndpointsList)).Items {
			ks.gatherEndpoint(endpoint, acc)
		}

		err := acc.FirstError()
		if err == nil && v.hasError {
			t.Fatalf("%s failed, should have error", v.name)
		} else if err != nil && !v.hasError {
			t.Fatalf("%s failed, err: %v", v.name, err)
		}
		if v.output == nil && len(acc.Metrics) > 0 {
			t.Fatalf("%s: collected extra data", v.name)
		} else if v.output != nil && len(v.output.Metrics) > 0 {
			for i := range v.output.Metrics {
				for k, m := range v.output.Metrics[i].Tags {
					if acc.Metrics[i].Tags[k] != m {
						t.Fatalf("%s: tag %s metrics unmatch Expected %s, got '%v'\n", v.name, k, m, acc.Metrics[i].Tags[k])
					}
				}
				for k, m := range v.output.Metrics[i].Fields {
					if acc.Metrics[i].Fields[k] != m {
						t.Fatalf("%s: field %s metrics unmatch Expected %v(%T), got %v(%T)\n", v.name, k, m, m, acc.Metrics[i].Fields[k], acc.Metrics[i].Fields[k])
					}
				}
			}
		}
	}
}
