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

package object_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/internal/object"
	"github.com/jthomperoo/k8shorizmetrics/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/internal/testutil"
	"github.com/jthomperoo/k8shorizmetrics/metrics"
	objectmetrics "github.com/jthomperoo/k8shorizmetrics/metrics/object"
	"github.com/jthomperoo/k8shorizmetrics/metrics/value"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
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
			errors.New("invalid object metric source: neither a value target nor an average value target was set"),
			nil,
			0,
			3,
			&metrics.Metric{
				Spec: v2beta2.MetricSpec{
					Object: &v2beta2.ObjectMetricSource{},
				},
			},
		},
		{
			"Success, average value, beyond tolerance",
			10,
			nil,
			nil,
			0,
			5,
			&metrics.Metric{
				Spec: v2beta2.MetricSpec{
					Object: &v2beta2.ObjectMetricSource{
						Target: v2beta2.MetricTarget{
							Type:         v2beta2.AverageValueMetricType,
							AverageValue: resource.NewMilliQuantity(50, resource.DecimalSI),
						},
					},
				},
				Object: &objectmetrics.Metric{
					Current: value.MetricValue{
						AverageValue: testutil.Int64Ptr(500),
					},
				},
			},
		},
		{
			"Success, average value, within tolerance",
			5,
			nil,
			nil,
			0,
			5,
			&metrics.Metric{
				Spec: v2beta2.MetricSpec{
					Object: &v2beta2.ObjectMetricSource{
						Target: v2beta2.MetricTarget{
							Type:         v2beta2.AverageValueMetricType,
							AverageValue: resource.NewMilliQuantity(50, resource.DecimalSI),
						},
					},
				},
				Object: &objectmetrics.Metric{
					Current: value.MetricValue{
						AverageValue: testutil.Int64Ptr(250),
					},
				},
			},
		},
		{
			"Success, value",
			3,
			nil,
			&fake.Calculate{
				GetUsageRatioReplicaCountReactor: func(currentReplicas int32, usageRatio float64, readyPodCount int64) int32 {
					return 3
				},
			},
			0,
			5,
			&metrics.Metric{
				Spec: v2beta2.MetricSpec{
					Object: &v2beta2.ObjectMetricSource{
						Target: v2beta2.MetricTarget{
							Type:  v2beta2.ValueMetricType,
							Value: resource.NewMilliQuantity(50, resource.DecimalSI),
						},
					},
				},
				Object: &objectmetrics.Metric{
					ReadyPodCount: testutil.Int64Ptr(2),
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(250),
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			evaluater := object.Evaluate{
				Calculater: test.calculater,
				Tolerance:  test.tolerance,
			}
			evaluation, err := evaluater.Evaluate(test.currentReplicas, test.gatheredMetric)
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
