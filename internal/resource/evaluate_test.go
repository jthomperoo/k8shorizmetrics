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

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/resource"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/testutil"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/podmetrics"
	resourcemetrics "github.com/jthomperoo/k8shorizmetrics/v4/metrics/resource"
	v2 "k8s.io/api/autoscaling/v2"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestEvaluate(t *testing.T) {
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
		calculater      replicas.Calculator
		tolerance       float64
		currentReplicas int32
		gatheredMetric  *metrics.Metric
	}{
		{
			"Invalid metric source",
			0,
			errors.New("invalid resource metric source: neither a utilization target nor a value target was set"),
			nil,
			0,
			3,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{},
				},
			},
		},
		{
			"Success, average value",
			6,
			nil,
			&fake.Calculate{
				GetPlainMetricReplicaCountReactor: func(metrics podmetrics.MetricsInfo, currentReplicas int32, targetUtilization, readyPodCount int64, missingPods, ignoredPods sets.String) int32 {
					return 6
				},
			},
			0,
			5,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageValue: k8sresource.NewMilliQuantity(50, k8sresource.DecimalSI),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{},
					ReadyPodCount:  3,
					IgnoredPods:    sets.String{"ignored": {}},
					MissingPods:    sets.String{"missing": {}},
				},
			},
		},
		{
			"Fail, average utilization, no metrics for pods",
			0,
			errors.New(`no metrics returned matched known pods`),
			nil,
			0,
			3,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(15),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{},
					Requests:       map[string]int64{},
					ReadyPodCount:  3,
					IgnoredPods:    sets.String{"ignored": {}},
					MissingPods:    sets.String{"missing": {}},
				},
			},
		},
		{
			"Success, average utilization, no ignored pods, no missing pods, within tolerance, no scale change",
			2,
			nil,
			nil,
			0,
			2,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 5,
						},
						"pod-2": podmetrics.Metric{
							Value: 5,
						},
					},
					Requests: map[string]int64{
						"pod-1": 10,
						"pod-2": 10,
					},
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
				},
			},
		},
		{
			"Success, average utilization, no ignored pods, no missing pods, beyond tolerance, scale up",
			8,
			nil,
			nil,
			0,
			2,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 20,
						},
						"pod-2": podmetrics.Metric{
							Value: 20,
						},
					},
					Requests: map[string]int64{
						"pod-1": 10,
						"pod-2": 10,
					},
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
				},
			},
		},
		{
			"Success, average utilization, no ignored pods, no missing pods, beyond tolerance, scale down",
			1,
			nil,
			nil,
			0,
			2,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 2,
						},
						"pod-2": podmetrics.Metric{
							Value: 2,
						},
					},
					Requests: map[string]int64{
						"pod-1": 10,
						"pod-2": 10,
					},
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
				},
			},
		},
		{
			"Success, average utilization, no ignored pods, 2 missing pods, beyond tolerance, scale up",
			8,
			nil,
			nil,
			0,
			4,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 20,
						},
						"pod-2": podmetrics.Metric{
							Value: 20,
						},
					},
					Requests: map[string]int64{
						"pod-1":     10,
						"pod-2":     10,
						"missing-1": 10,
						"missing-2": 10,
					},
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods: sets.String{
						"missing-1": {},
						"missing-2": {},
					},
				},
			},
		},
		{
			"Success, average utilization, no ignored pods, 2 missing pods, beyond tolerance, scale down",
			2,
			nil,
			nil,
			0,
			4,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 1,
						},
						"pod-2": podmetrics.Metric{
							Value: 1,
						},
					},
					Requests: map[string]int64{
						"pod-1":     20,
						"pod-2":     20,
						"missing-1": 3,
						"missing-2": 3,
					},
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods: sets.String{
						"missing-1": {},
						"missing-2": {},
					},
				},
			},
		},
		{
			"Success, average utilization, 2 ignored pods, 2 missing pods, beyond tolerance, scale up",
			12,
			nil,
			nil,
			0,
			4,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 20,
						},
						"pod-2": podmetrics.Metric{
							Value: 20,
						},
					},
					Requests: map[string]int64{
						"pod-1":     10,
						"pod-2":     10,
						"missing-1": 5,
						"missing-2": 5,
						"ignored-1": 5,
						"ignored-2": 5,
					},
					ReadyPodCount: 2,
					IgnoredPods: sets.String{
						"ignored-1": {},
						"ignored-2": {},
					},
					MissingPods: sets.String{
						"missing-1": {},
						"missing-2": {},
					},
				},
			},
		},
		{
			"Success, average utilization, 2 ignored pods, 2 missing pods, within tolerance, no scale change",
			4,
			nil,
			nil,
			0.5,
			4,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Resource: &v2.ResourceMetricSource{
						Target: v2.MetricTarget{
							AverageUtilization: testutil.Int32Ptr(50),
						},
					},
				},
				Resource: &resourcemetrics.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{
						"pod-1": podmetrics.Metric{
							Value: 20,
						},
						"pod-2": podmetrics.Metric{
							Value: 20,
						},
					},
					Requests: map[string]int64{
						"pod-1":     10,
						"pod-2":     10,
						"missing-1": 10,
						"missing-2": 10,
						"ignored-1": 10,
						"ignored-2": 10,
					},
					ReadyPodCount: 2,
					IgnoredPods: sets.String{
						"ignored-1": {},
						"ignored-2": {},
					},
					MissingPods: sets.String{
						"missing-1": {},
						"missing-2": {},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := resource.Evaluate{
				Calculater: test.calculater,
			}
			evaluation, err := evaluater.Evaluate(test.currentReplicas, test.gatheredMetric, test.tolerance)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}
			if !cmp.Equal(test.expected, evaluation) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, evaluation))
			}
		})
	}
}
