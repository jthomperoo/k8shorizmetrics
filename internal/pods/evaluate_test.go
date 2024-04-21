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

package pods_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/pods"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/podmetrics"
	metricspods "github.com/jthomperoo/k8shorizmetrics/v4/metrics/pods"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestEvaluate(t *testing.T) {
	var tests = []struct {
		description     string
		expected        int32
		calculater      replicas.Calculator
		currentReplicas int32
		gatheredMetric  *metrics.Metric
	}{
		{
			"Calculate 5 replicas, 2 ready pods, 1 ignored and 1 missing",
			5,
			&fake.Calculate{
				GetPlainMetricReplicaCountReactor: func(metrics podmetrics.MetricsInfo, currentReplicas int32, targetUtilization, readyPodCount int64, missingPods, ignoredPods sets.String) int32 {
					return 5
				},
			},
			4,
			&metrics.Metric{
				Spec: v2.MetricSpec{
					Pods: &v2.PodsMetricSource{
						Target: v2.MetricTarget{
							AverageValue: resource.NewMilliQuantity(50, resource.DecimalSI),
						},
					},
				},
				Pods: &metricspods.Metric{
					PodMetricsInfo: podmetrics.MetricsInfo{},
					ReadyPodCount:  2,
					IgnoredPods:    sets.String{"ignored": {}},
					MissingPods:    sets.String{"missing": {}},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			eval := pods.Evaluate{
				Calculater: test.calculater,
			}
			result := eval.Evaluate(test.currentReplicas, test.gatheredMetric)
			if !cmp.Equal(test.expected, result) {
				t.Errorf("evaluation mismatch (-want +got):\n%s", cmp.Diff(test.expected, result))
			}
		})
	}
}
