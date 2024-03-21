/*
Copyright 2024 The K8sHorizMetrics Authors.

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

package k8shorizmetrics_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/v3"
	"github.com/jthomperoo/k8shorizmetrics/v3/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/v3/metrics"
	v2 "k8s.io/api/autoscaling/v2"
)

func TestEvaluateSingleMetricWithOptions(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description     string
		expected        int32
		expectedErr     error
		external        k8shorizmetrics.ExternalEvaluater
		object          k8shorizmetrics.ObjectEvaluater
		pods            k8shorizmetrics.PodsEvaluater
		resource        k8shorizmetrics.ResourceEvaluater
		tolerance       float64
		gatheredMetric  *metrics.Metric
		currentReplicas int32
	}{
		{
			description: "Unknown Metric Source Type",
			expectedErr: errors.New(`unknown metric source type "unknown"`),
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: "unknown",
				},
			},
		},
		{
			description: "Object Metric: Failure",
			expectedErr: errors.New(`test error`),
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric,
					tolerance float64) (int32, error) {
					return 0, errors.New("test error")
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ObjectMetricSourceType,
				},
			},
		},
		{
			description: "Object Metric: Success",
			expected:    int32(3),
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric,
					tolerance float64) (int32, error) {
					return 3, nil
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ObjectMetricSourceType,
				},
			},
		},
		{
			description: "Pods Metric: Success",
			expected:    int32(3),
			pods: &fake.PodsEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric) int32 {
					return 3
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.PodsMetricSourceType,
				},
			},
		},
		{
			description: "Resource Metric: Failure",
			expectedErr: errors.New(`test error`),
			resource: &fake.ResourceEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 0, errors.New("test error")
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ResourceMetricSourceType,
				},
			},
		},
		{
			description: "Resource Metric: Success",
			expected:    int32(3),
			resource: &fake.ResourceEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 3, nil
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ResourceMetricSourceType,
				},
			},
		},
		{
			description: "External Metric: Failure",
			expectedErr: errors.New(`test error`),
			external: &fake.ExternalEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 0, errors.New("test error")
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ExternalMetricSourceType,
				},
			},
		},
		{
			description: "External Metric: Success",
			expected:    int32(3),
			external: &fake.ExternalEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 3, nil
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ExternalMetricSourceType,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := &k8shorizmetrics.Evaluator{
				External:  test.external,
				Object:    test.object,
				Pods:      test.pods,
				Resource:  test.resource,
				Tolerance: test.tolerance,
			}

			metric, err := evaluater.EvaluateSingleMetricWithOptions(test.gatheredMetric, test.currentReplicas,
				test.tolerance)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestEvaluateSingleMetrics(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description     string
		expected        int32
		expectedErr     error
		external        k8shorizmetrics.ExternalEvaluater
		object          k8shorizmetrics.ObjectEvaluater
		pods            k8shorizmetrics.PodsEvaluater
		resource        k8shorizmetrics.ResourceEvaluater
		tolerance       float64
		gatheredMetric  *metrics.Metric
		currentReplicas int32
	}{
		{
			description: "Failure",
			expectedErr: errors.New(`test error`),
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric,
					tolerance float64) (int32, error) {
					return 0, errors.New("test error")
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ObjectMetricSourceType,
				},
			},
		},
		{
			description: "Success",
			expected:    int32(3),
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric,
					tolerance float64) (int32, error) {
					return 3, nil
				},
			},
			gatheredMetric: &metrics.Metric{
				Spec: v2.MetricSpec{
					Type: v2.ObjectMetricSourceType,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := &k8shorizmetrics.Evaluator{
				External:  test.external,
				Object:    test.object,
				Pods:      test.pods,
				Resource:  test.resource,
				Tolerance: test.tolerance,
			}

			metric, err := evaluater.EvaluateSingleMetric(test.gatheredMetric, test.currentReplicas)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestEvaluateWithOptions(t *testing.T) {
	var tests = []struct {
		description     string
		expected        int32
		expectedErr     *k8shorizmetrics.EvaluatorMultiMetricError
		external        k8shorizmetrics.ExternalEvaluater
		object          k8shorizmetrics.ObjectEvaluater
		pods            k8shorizmetrics.PodsEvaluater
		resource        k8shorizmetrics.ResourceEvaluater
		tolerance       float64
		gatheredMetrics []*metrics.Metric
		currentReplicas int32
	}{
		{
			description: "Single metric fail to evaluate",
			expected:    0,
			expectedErr: &k8shorizmetrics.EvaluatorMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New("test error"),
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return -1, errors.New("test error")
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Two metrics fail to evaluate both",
			expected:    0,
			expectedErr: &k8shorizmetrics.EvaluatorMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New("test error"),
					errors.New("test error"),
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return -1, errors.New("test error")
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Two metrics fail to gather one, other successful",
			expected:    3,
			expectedErr: &k8shorizmetrics.EvaluatorMultiMetricError{
				Partial: true,
				Errors: []error{
					errors.New("test error"),
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					if gatheredMetric.Spec.Object.Metric.Name == "second" {
						return 3, nil
					}
					return -1, errors.New("test error")
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Object: &v2.ObjectMetricSource{
							Metric: v2.MetricIdentifier{
								Name: "first",
							},
						},
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Object: &v2.ObjectMetricSource{
							Metric: v2.MetricIdentifier{
								Name: "second",
							},
						},
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description:     "No metrics success",
			expected:        0,
			gatheredMetrics: []*metrics.Metric{},
		},
		{
			description: "Single metric success",
			expected:    4,
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 4, nil
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Two metric success, same evaluation",
			expected:    2,
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 2, nil
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Three metric success, pick highest",
			expected:    7,
			external: &fake.ExternalEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 5, nil
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 2, nil
				},
			},
			pods: &fake.PodsEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric) int32 {
					return 7
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ExternalMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Type: v2.PodsMetricSourceType,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := &k8shorizmetrics.Evaluator{
				External:  test.external,
				Object:    test.object,
				Pods:      test.pods,
				Resource:  test.resource,
				Tolerance: test.tolerance,
			}

			metric, err := evaluater.EvaluateWithOptions(test.gatheredMetrics, test.currentReplicas, test.tolerance)
			evaluateErr := &k8shorizmetrics.EvaluatorMultiMetricError{}

			if err == nil && test.expectedErr != nil {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), evaluateErr.Error()))
				return
			}

			if err != nil {
				if errors.As(err, &evaluateErr) {
					if !cmp.Equal(evaluateErr.Partial, test.expectedErr.Partial) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(&test.expectedErr.Partial, evaluateErr.Partial))
						return
					}

					if !cmp.Equal(evaluateErr.Error(), test.expectedErr.Error()) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), evaluateErr.Error()))
						return
					}
				} else {
					t.Error("unexpected error type returned, expected EvaluatorMultiMetricError")
					return
				}
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	var tests = []struct {
		description     string
		expected        int32
		expectedErr     *k8shorizmetrics.EvaluatorMultiMetricError
		external        k8shorizmetrics.ExternalEvaluater
		object          k8shorizmetrics.ObjectEvaluater
		pods            k8shorizmetrics.PodsEvaluater
		resource        k8shorizmetrics.ResourceEvaluater
		tolerance       float64
		gatheredMetrics []*metrics.Metric
		currentReplicas int32
	}{
		{
			description: "Full failure",
			expected:    0,
			expectedErr: &k8shorizmetrics.EvaluatorMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New("test error"),
					errors.New("test error"),
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return -1, errors.New("test error")
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Partial failure",
			expected:    3,
			expectedErr: &k8shorizmetrics.EvaluatorMultiMetricError{
				Partial: true,
				Errors: []error{
					errors.New("test error"),
				},
			},
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					if gatheredMetric.Spec.Object.Metric.Name == "second" {
						return 3, nil
					}
					return -1, errors.New("test error")
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Object: &v2.ObjectMetricSource{
							Metric: v2.MetricIdentifier{
								Name: "first",
							},
						},
						Type: v2.ObjectMetricSourceType,
					},
				},
				{
					Spec: v2.MetricSpec{
						Object: &v2.ObjectMetricSource{
							Metric: v2.MetricIdentifier{
								Name: "second",
							},
						},
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
		{
			description: "Success",
			expected:    4,
			object: &fake.ObjectEvaluater{
				EvaluateReactor: func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
					return 4, nil
				},
			},
			currentReplicas: 1,
			gatheredMetrics: []*metrics.Metric{
				{
					Spec: v2.MetricSpec{
						Type: v2.ObjectMetricSourceType,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := &k8shorizmetrics.Evaluator{
				External:  test.external,
				Object:    test.object,
				Pods:      test.pods,
				Resource:  test.resource,
				Tolerance: test.tolerance,
			}

			metric, err := evaluater.Evaluate(test.gatheredMetrics, test.currentReplicas)
			evaluateErr := &k8shorizmetrics.EvaluatorMultiMetricError{}

			if err == nil && test.expectedErr != nil {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), evaluateErr.Error()))
				return
			}

			if err != nil {
				if errors.As(err, &evaluateErr) {
					if !cmp.Equal(evaluateErr.Partial, test.expectedErr.Partial) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(&test.expectedErr.Partial, evaluateErr.Partial))
						return
					}

					if !cmp.Equal(evaluateErr.Error(), test.expectedErr.Error()) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), evaluateErr.Error()))
						return
					}
				} else {
					t.Error("unexpected error type returned, expected EvaluatorMultiMetricError")
					return
				}
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}
