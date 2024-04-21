/*
Copyright 2022 The K8sHorizMetrics Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resource_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/resource"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/podmetrics"
	resourcemetric "github.com/jthomperoo/k8shorizmetrics/v4/metrics/resource"
	metricsclient "github.com/jthomperoo/k8shorizmetrics/v4/metricsclient"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func TestGather(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description                   string
		expected                      *resourcemetric.Metric
		expectedErr                   error
		metricsclient                 metricsclient.Client
		podLister                     corelisters.PodLister
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
		resourceName                  corev1.ResourceName
		namespace                     string
		selector                      labels.Selector
	}{
		{
			"Fail to get metric",
			nil,
			errors.New("unable to get metrics for resource test-metric: fail to get metric"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, errors.New("fail to get metric")
				},
			},
			nil,
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"Fail to get pods",
			nil,
			errors.New("unable to get pods while calculating replica count: fail to get pods"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return nil, errors.New("fail to get pods")
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"Fail no pods",
			nil,
			errors.New("no pods returned by selector while calculating replica count"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{}, nil
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"Fail calculating pod limits",
			nil,
			errors.New("missing request for test-metric"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "test-pod",
									},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "invalid-container",
											},
										},
									},
								},
							}, nil
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"3 ready, 2 missing pods success",
			&resourcemetric.Metric{
				TotalPods:     5,
				ReadyPodCount: 3,
				MissingPods: sets.String{
					"missing-pod-1": {},
					"missing-pod-2": {},
				},
				Requests: map[string]int64{
					"missing-pod-1": 0,
					"missing-pod-2": 0,
					"ready-pod-1":   5,
					"ready-pod-2":   0,
					"ready-pod-3":   0,
				},
				PodMetricsInfo: podmetrics.MetricsInfo{
					"ready-pod-1": podmetrics.Metric{
						Value: 1,
					},
					"ready-pod-2": podmetrics.Metric{
						Value: 2,
					},
					"ready-pod-3": podmetrics.Metric{
						Value: 3,
					},
				},
			},
			nil,
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return podmetrics.MetricsInfo{
						"ready-pod-1": podmetrics.Metric{
							Value: 1,
						},
						"ready-pod-2": podmetrics.Metric{
							Value: 2,
						},
						"ready-pod-3": podmetrics.Metric{
							Value: 3,
						},
					}, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-1",
									},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Requests: corev1.ResourceList{
														"test-metric": *k8sresource.NewMilliQuantity(5, k8sresource.DecimalSI),
													},
												},
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-2",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-3",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-2",
									},
								},
							}, nil
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"3 ready, 2 missing, 2 ignored pods success",
			&resourcemetric.Metric{
				TotalPods:     7,
				ReadyPodCount: 3,
				MissingPods: sets.String{
					"missing-pod-1": {},
					"missing-pod-2": {},
				},
				IgnoredPods: sets.String{
					"ignore-pod-1": {},
					"ignore-pod-2": {},
				},
				Requests: map[string]int64{
					"missing-pod-1": 0,
					"missing-pod-2": 0,
					"ready-pod-1":   5,
					"ready-pod-2":   0,
					"ready-pod-3":   0,
					"ignore-pod-1":  0,
					"ignore-pod-2":  0,
				},
				PodMetricsInfo: podmetrics.MetricsInfo{
					"ready-pod-1": podmetrics.Metric{
						Value: 1,
					},
					"ready-pod-2": podmetrics.Metric{
						Value: 2,
					},
					"ready-pod-3": podmetrics.Metric{
						Value: 3,
					},
				},
			},
			nil,
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return podmetrics.MetricsInfo{
						"ready-pod-1": podmetrics.Metric{
							Value: 1,
						},
						"ready-pod-2": podmetrics.Metric{
							Value: 2,
						},
						"ready-pod-3": podmetrics.Metric{
							Value: 3,
						},
						"ignore-pod-1": podmetrics.Metric{
							Value: 4,
						},
						"ignore-pod-2": podmetrics.Metric{
							Value: 5,
						},
					}, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-1",
									},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Requests: corev1.ResourceList{
														corev1.ResourceCPU: *k8sresource.NewMilliQuantity(5, k8sresource.DecimalSI),
													},
												},
											},
										},
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-2",
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-3",
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-2",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ignore-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ignore-pod-2",
									},
								},
							}, nil
						},
					}
				},
			},
			0,
			0,
			corev1.ResourceCPU,
			"test-namespace",
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &resource.Gather{
				MetricsClient: test.metricsclient,
				PodLister:     test.podLister,
			}
			metric, err := gatherer.Gather(test.resourceName, test.namespace, test.selector, test.cpuInitializationPeriod, test.delayOfInitialReadinessStatus)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}
			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestGatherRaw(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description                   string
		expected                      *resourcemetric.Metric
		expectedErr                   error
		metricsclient                 metricsclient.Client
		podLister                     corelisters.PodLister
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
		resourceName                  corev1.ResourceName
		namespace                     string
		selector                      labels.Selector
	}{
		{
			"Fail to get metric",
			nil,
			errors.New("unable to get metrics for resource test-metric: fail to get metric"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, errors.New("fail to get metric")
				},
			},
			nil,
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"Fail to get pods",
			nil,
			errors.New("unable to get pods while calculating replica count: fail to get pods"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return nil, errors.New("fail to get pods")
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"Fail no pods",
			nil,
			errors.New("no pods returned by selector while calculating replica count"),
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return nil, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{}, nil
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"3 ready, 2 missing pods success",
			&resourcemetric.Metric{
				TotalPods:     5,
				ReadyPodCount: 3,
				MissingPods: sets.String{
					"missing-pod-1": {},
					"missing-pod-2": {},
				},
				PodMetricsInfo: podmetrics.MetricsInfo{
					"ready-pod-1": podmetrics.Metric{
						Value: 1,
					},
					"ready-pod-2": podmetrics.Metric{
						Value: 2,
					},
					"ready-pod-3": podmetrics.Metric{
						Value: 3,
					},
				},
			},
			nil,
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return podmetrics.MetricsInfo{
						"ready-pod-1": podmetrics.Metric{
							Value: 1,
						},
						"ready-pod-2": podmetrics.Metric{
							Value: 2,
						},
						"ready-pod-3": podmetrics.Metric{
							Value: 3,
						},
					}, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-1",
									},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Requests: corev1.ResourceList{
														"test-metric": *k8sresource.NewMilliQuantity(5, k8sresource.DecimalSI),
													},
												},
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-2",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-3",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-2",
									},
								},
							}, nil
						},
					}
				},
			},
			0,
			0,
			"test-metric",
			"test-namespace",
			nil,
		},
		{
			"3 ready, 2 missing, 2 ignored pods success",
			&resourcemetric.Metric{
				TotalPods:     7,
				ReadyPodCount: 3,
				MissingPods: sets.String{
					"missing-pod-1": {},
					"missing-pod-2": {},
				},
				IgnoredPods: sets.String{
					"ignore-pod-1": {},
					"ignore-pod-2": {},
				},
				PodMetricsInfo: podmetrics.MetricsInfo{
					"ready-pod-1": podmetrics.Metric{
						Value: 1,
					},
					"ready-pod-2": podmetrics.Metric{
						Value: 2,
					},
					"ready-pod-3": podmetrics.Metric{
						Value: 3,
					},
				},
			},
			nil,
			&fake.MetricsClient{
				GetResourceMetricReactor: func(resource corev1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
					return podmetrics.MetricsInfo{
						"ready-pod-1": podmetrics.Metric{
							Value: 1,
						},
						"ready-pod-2": podmetrics.Metric{
							Value: 2,
						},
						"ready-pod-3": podmetrics.Metric{
							Value: 3,
						},
						"ignore-pod-1": podmetrics.Metric{
							Value: 4,
						},
						"ignore-pod-2": podmetrics.Metric{
							Value: 5,
						},
					}, time.Time{}, nil
				},
			},
			&fake.PodLister{
				PodsReactor: func(namespace string) corelisters.PodNamespaceLister {
					return &fake.PodNamespaceLister{
						ListReactor: func(selector labels.Selector) (ret []*corev1.Pod, err error) {
							return []*corev1.Pod{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-1",
									},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Requests: corev1.ResourceList{
														corev1.ResourceCPU: *k8sresource.NewMilliQuantity(5, k8sresource.DecimalSI),
													},
												},
											},
										},
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-2",
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ready-pod-3",
									},
									Status: corev1.PodStatus{
										StartTime: &metav1.Time{},
										Conditions: []corev1.PodCondition{
											{
												Type: corev1.PodReady,
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "missing-pod-2",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ignore-pod-1",
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "ignore-pod-2",
									},
								},
							}, nil
						},
					}
				},
			},
			0,
			0,
			corev1.ResourceCPU,
			"test-namespace",
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &resource.Gather{
				MetricsClient: test.metricsclient,
				PodLister:     test.podLister,
			}
			metric, err := gatherer.GatherRaw(test.resourceName, test.namespace, test.selector, test.cpuInitializationPeriod, test.delayOfInitialReadinessStatus)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}
			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}
