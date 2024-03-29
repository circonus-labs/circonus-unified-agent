package kubeinventory

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/testutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod(t *testing.T) {
	cli := &client{}
	selectInclude := []string{}
	selectExclude := []string{}
	now := time.Now()
	started := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 1, 36, 0, now.Location())
	created := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-2, 1, 36, 0, now.Location())
	cond1 := time.Date(now.Year(), 7, 5, 7, 53, 29, 0, now.Location())
	cond2 := time.Date(now.Year(), 7, 5, 7, 53, 31, 0, now.Location())

	tests := []struct {
		handler  *mockHandler
		output   *testutil.Accumulator
		name     string
		hasError bool
	}{
		{
			name: "no pods",
			handler: &mockHandler{
				responseMap: map[string]interface{}{
					"/pods/": &corev1.PodList{},
				},
			},
			hasError: false,
		},
		{
			name: "collect pods",
			handler: &mockHandler{
				responseMap: map[string]interface{}{
					"/pods/": &corev1.PodList{
						Items: []corev1.Pod{
							{
								Spec: corev1.PodSpec{
									NodeName: "node1",
									Containers: []corev1.Container{
										{
											Name:  "running",
											Image: "image1",
											Ports: []corev1.ContainerPort{
												{
													ContainerPort: 8080,
													Protocol:      "TCP",
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
												Requests: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
											},
										},
										{
											Name:  "completed",
											Image: "image1",
											Ports: []corev1.ContainerPort{
												{
													ContainerPort: 8080,
													Protocol:      "TCP",
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
												Requests: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
											},
										},
										{
											Name:  "waiting",
											Image: "image1",
											Ports: []corev1.ContainerPort{
												{
													ContainerPort: 8080,
													Protocol:      "TCP",
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
												Requests: corev1.ResourceList{
													"cpu": resource.Quantity{Format: "100m"},
												},
											},
										},
									},
									Volumes: []corev1.Volume{
										{
											Name: "vol1",
											VolumeSource: corev1.VolumeSource{
												PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
													ClaimName: "pc1",
													ReadOnly:  true,
												},
											},
										},
										{
											Name: "vol2",
										},
									},
									NodeSelector: map[string]string{
										"select1": "s1",
										"select2": "s2",
									},
								},
								Status: corev1.PodStatus{
									Phase:     "Running",
									HostIP:    "180.12.10.18",
									PodIP:     "10.244.2.15",
									StartTime: &metav1.Time{Time: started},
									Conditions: []corev1.PodCondition{
										{
											Type:               "Initialized",
											Status:             "True",
											LastTransitionTime: metav1.Time{Time: cond1},
										},
										{
											Type:               "Ready",
											Status:             "True",
											LastTransitionTime: metav1.Time{Time: cond2},
										},
										{
											Type:               "Scheduled",
											Status:             "True",
											LastTransitionTime: metav1.Time{Time: cond1},
										},
									},
									ContainerStatuses: []corev1.ContainerStatus{
										{
											Name: "running",
											State: corev1.ContainerState{
												Running: &corev1.ContainerStateRunning{
													StartedAt: metav1.Time{Time: started},
												},
											},
											Ready:        true,
											RestartCount: 3,
											Image:        "image1",
											ImageID:      "image_id1",
											ContainerID:  "docker://54abe32d0094479d3d",
										},
										{
											Name: "completed",
											State: corev1.ContainerState{
												Terminated: &corev1.ContainerStateTerminated{
													StartedAt: metav1.Time{Time: now},
													ExitCode:  0,
													Reason:    "Completed",
												},
											},
											Ready:        false,
											RestartCount: 3,
											Image:        "image1",
											ImageID:      "image_id1",
											ContainerID:  "docker://54abe32d0094479d3d",
										},
										{
											Name: "waiting",
											State: corev1.ContainerState{
												Waiting: &corev1.ContainerStateWaiting{
													Reason: "PodUninitialized",
												},
											},
											Ready:        false,
											RestartCount: 3,
											Image:        "image1",
											ImageID:      "image_id1",
											ContainerID:  "docker://54abe32d0094479d3d",
										},
									},
								},
								ObjectMeta: metav1.ObjectMeta{
									OwnerReferences: []metav1.OwnerReference{
										{
											APIVersion: "apps/v1",
											Kind:       "DaemonSet",
											Name:       "forwarder",
											Controller: toBoolPtr(true),
										},
									},
									Generation: 11232,
									Namespace:  "ns1",
									Name:       "pod1",
									Labels: map[string]string{
										"lab1": "v1",
										"lab2": "v2",
									},
									CreationTimestamp: metav1.Time{Time: created},
								},
							},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Measurement: podContainerMeasurement,
						Fields: map[string]interface{}{
							"restarts_total":                   int32(3),
							"state_code":                       0,
							"resource_requests_millicpu_units": int64(100),
							"resource_limits_millicpu_units":   int64(100),
						},
						Tags: map[string]string{
							"namespace":             "ns1",
							"container_name":        "running",
							"node_name":             "node1",
							"pod_name":              "pod1",
							"state":                 "running",
							"readiness":             "ready",
							"node_selector_select1": "s1",
							"node_selector_select2": "s2",
						},
					},
					{
						Measurement: podContainerMeasurement,
						Fields: map[string]interface{}{
							"restarts_total":                   int32(3),
							"state_code":                       1,
							"state_reason":                     "Completed",
							"resource_requests_millicpu_units": int64(100),
							"resource_limits_millicpu_units":   int64(100),
						},
						Tags: map[string]string{
							"namespace":      "ns1",
							"container_name": "completed",
							"node_name":      "node1",
							"pod_name":       "pod1",
							"state":          "terminated",
							"readiness":      "unready",
						},
					},
					{
						Measurement: podContainerMeasurement,
						Fields: map[string]interface{}{
							"restarts_total":                   int32(3),
							"state_code":                       2,
							"state_reason":                     "PodUninitialized",
							"resource_requests_millicpu_units": int64(100),
							"resource_limits_millicpu_units":   int64(100),
						},
						Tags: map[string]string{
							"namespace":      "ns1",
							"container_name": "waiting",
							"node_name":      "node1",
							"pod_name":       "pod1",
							"state":          "waiting",
							"readiness":      "unready",
						},
					},
				},
			},
			hasError: false,
		},
	}
	for _, v := range tests {
		ks := &KubernetesInventory{
			client:          cli,
			SelectorInclude: selectInclude,
			SelectorExclude: selectExclude,
		}
		_ = ks.createSelectorFilters()
		acc := new(testutil.Accumulator)
		for _, pod := range ((v.handler.responseMap["/pods/"]).(corev1.PodList)).Items {
			ks.gatherPod(pod, acc)
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
						t.Fatalf("%s: tag %s metrics unmatch Expected %s, got %s, i %d\n", v.name, k, m, acc.Metrics[i].Tags[k], i)
					}
				}
				for k, m := range v.output.Metrics[i].Fields {
					if acc.Metrics[i].Fields[k] != m {
						t.Fatalf("%s: field %s metrics unmatch Expected %v(%T), got %v(%T), i %d\n", v.name, k, m, m, acc.Metrics[i].Fields[k], acc.Metrics[i].Fields[k], i)
					}
				}
			}
		}
	}
}

func TestPodSelectorFilter(t *testing.T) {
	cli := &client{}
	now := time.Now()
	started := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 1, 36, 0, now.Location())
	created := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-2, 1, 36, 0, now.Location())
	cond1 := time.Date(now.Year(), 7, 5, 7, 53, 29, 0, now.Location())
	cond2 := time.Date(now.Year(), 7, 5, 7, 53, 31, 0, now.Location())

	responseMap := map[string]interface{}{
		"/pods/": &corev1.PodList{
			Items: []corev1.Pod{
				{
					Spec: corev1.PodSpec{
						NodeName: "node1",
						Containers: []corev1.Container{
							{
								Name:  "forwarder",
								Image: "image1",
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8080,
										Protocol:      "TCP",
									},
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										"cpu": resource.Quantity{Format: "100m"},
									},
									Requests: corev1.ResourceList{
										"cpu": resource.Quantity{Format: "100m"},
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "vol1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pc1",
										ReadOnly:  true,
									},
								},
							},
							{
								Name: "vol2",
							},
						},
						NodeSelector: map[string]string{
							"select1": "s1",
							"select2": "s2",
						},
					},
					Status: corev1.PodStatus{
						Phase:     "Running",
						HostIP:    "180.12.10.18",
						PodIP:     "10.244.2.15",
						StartTime: &metav1.Time{Time: started},
						Conditions: []corev1.PodCondition{
							{
								Type:               "Initialized",
								Status:             "True",
								LastTransitionTime: metav1.Time{Time: cond1},
							},
							{
								Type:               "Ready",
								Status:             "True",
								LastTransitionTime: metav1.Time{Time: cond2},
							},
							{
								Type:               "Scheduled",
								Status:             "True",
								LastTransitionTime: metav1.Time{Time: cond1},
							},
						},
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name: "forwarder",
								State: corev1.ContainerState{
									Running: &corev1.ContainerStateRunning{
										StartedAt: metav1.Time{Time: cond2},
									},
								},
								Ready:        true,
								RestartCount: 3,
								Image:        "image1",
								ImageID:      "image_id1",
								ContainerID:  "docker://54abe32d0094479d3d",
							},
						},
					},
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "apps/v1",
								Kind:       "DaemonSet",
								Name:       "forwarder",
								Controller: toBoolPtr(true),
							},
						},
						Generation: 11232,
						Namespace:  "ns1",
						Name:       "pod1",
						Labels: map[string]string{
							"lab1": "v1",
							"lab2": "v2",
						},
						CreationTimestamp: metav1.Time{Time: created},
					},
				},
			},
		},
	}

	tests := []struct {
		handler  *mockHandler
		expected map[string]string
		name     string
		include  []string
		exclude  []string
		hasError bool
	}{
		{
			name: "nil filters equals all selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  nil,
			exclude:  nil,
			expected: map[string]string{
				"node_selector_select1": "s1",
				"node_selector_select2": "s2",
			},
		},
		{
			name: "empty filters equals all selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{},
			exclude:  []string{},
			expected: map[string]string{
				"node_selector_select1": "s1",
				"node_selector_select2": "s2",
			},
		},
		{
			name: "include filter equals only include-matched selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{"select1"},
			exclude:  []string{},
			expected: map[string]string{
				"node_selector_select1": "s1",
			},
		},
		{
			name: "exclude filter equals only non-excluded selectors (overrides include filter)",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{},
			exclude:  []string{"select2"},
			expected: map[string]string{
				"node_selector_select1": "s1",
			},
		},
		{
			name: "include glob filter equals only include-matched selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{"*1"},
			exclude:  []string{},
			expected: map[string]string{
				"node_selector_select1": "s1",
			},
		},
		{
			name: "exclude glob filter equals only non-excluded selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{},
			exclude:  []string{"*2"},
			expected: map[string]string{
				"node_selector_select1": "s1",
			},
		},
		{
			name: "exclude glob filter equals only non-excluded selectors",
			handler: &mockHandler{
				responseMap: responseMap,
			},
			hasError: false,
			include:  []string{},
			exclude:  []string{"*2"},
			expected: map[string]string{
				"node_selector_select1": "s1",
			},
		},
	}
	for _, v := range tests {
		ks := &KubernetesInventory{
			client: cli,
		}
		ks.SelectorInclude = v.include
		ks.SelectorExclude = v.exclude
		_ = ks.createSelectorFilters()
		acc := new(testutil.Accumulator)
		for _, pod := range ((v.handler.responseMap["/pods/"]).(corev1.PodList)).Items {
			ks.gatherPod(pod, acc)
		}

		// Grab selector tags
		actual := map[string]string{}
		for _, metric := range acc.Metrics {
			for key, val := range metric.Tags {
				if strings.Contains(key, "node_selector_") {
					actual[key] = val
				}
			}
		}

		if !reflect.DeepEqual(v.expected, actual) {
			t.Fatalf("actual selector tags (%v) do not match expected selector tags (%v)", actual, v.expected)
		}
	}
}
