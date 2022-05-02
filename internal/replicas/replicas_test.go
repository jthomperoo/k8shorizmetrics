/*
Copyright 2019 The Custom Pod Autoscaler Authors.

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

package replicas_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/metrics/podmetrics"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestReplicaCalculate_GetUsageRatioReplicaCount(t *testing.T) {
	var tests = []struct {
		description     string
		expected        int32
		tolerance       float64
		currentReplicas int32
		usageRatio      float64
		readyPodCount   int64
	}{
		{
			"No current replicas, scale to zero",
			0,
			0.1,
			0,
			0,
			0,
		},
		{
			"No current replicas, scale to 2",
			2,
			0.1,
			0,
			2,
			0,
		},
		{
			"3 current replicas, within tolerance, no scale",
			3,
			0.1,
			3,
			0.95,
			3,
		},
		{
			"3 current replicas, beyond tolerance, scale up",
			5,
			0.1,
			3,
			1.4,
			3,
		},
		{
			"3 current replicas, beyond tolerance, scale down",
			1,
			0.1,
			3,
			0.3,
			3,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			calc := replicas.ReplicaCalculator{
				Tolerance: test.tolerance,
			}
			result := calc.GetUsageRatioReplicaCount(test.currentReplicas, test.usageRatio, test.readyPodCount)
			if !cmp.Equal(test.expected, result) {
				t.Errorf("replica mismatch (-want +got):\n%s", cmp.Diff(test.expected, result))
			}
		})
	}
}

func TestReplicaCalculate_GetPlainMetricReplicaCount(t *testing.T) {
	var tests = []struct {
		description       string
		expected          int32
		tolerance         float64
		metrics           podmetrics.MetricsInfo
		currentReplicas   int32
		targetUtilization int64
		readyPodCount     int64
		missingPods       sets.String
		ignoredPods       sets.String
	}{
		{
			"No ignored pods, no missing pods, within tolerance, no scale change",
			2,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 50,
				},
				"pod-2": podmetrics.Metric{
					Value: 50,
				},
			},
			2,
			50,
			2,
			sets.String{},
			sets.String{},
		},
		{
			"No ignored pods, no missing pods, beyond tolerance, scale up",
			4,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 100,
				},
				"pod-2": podmetrics.Metric{
					Value: 100,
				},
			},
			2,
			50,
			2,
			sets.String{},
			sets.String{},
		},
		{
			"No ignored pods, no missing pods, beyond tolerance, scale down",
			1,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 25,
				},
				"pod-2": podmetrics.Metric{
					Value: 25,
				},
			},
			2,
			50,
			2,
			sets.String{},
			sets.String{},
		},
		{
			"No ignored pods, 2 missing pods, beyond tolerance, scale up",
			8,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 200,
				},
				"pod-2": podmetrics.Metric{
					Value: 200,
				},
			},
			4,
			50,
			2,
			sets.String{
				"missing-1": {},
				"missing-2": {},
			},
			sets.String{},
		},
		{
			"No ignored pods, 2 missing pods, beyond tolerance, scale down",
			3,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 25,
				},
				"pod-2": podmetrics.Metric{
					Value: 25,
				},
			},
			4,
			50,
			2,
			sets.String{
				"missing-1": {},
				"missing-2": {},
			},
			sets.String{},
		},
		{
			"2 ignored pods, 2 missing pods, beyond tolerance, scale up",
			16,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 400,
				},
				"pod-2": podmetrics.Metric{
					Value: 400,
				},
			},
			6,
			50,
			2,
			sets.String{
				"missing-1": {},
				"missing-2": {},
			},
			sets.String{
				"ignored-1": {},
				"ignored-2": {},
			},
		},
		{
			"2 ignored pods, 2 missing pods, within tolerance, no scale change",
			6,
			0.1,
			podmetrics.MetricsInfo{
				"pod-1": podmetrics.Metric{
					Value: 150,
				},
				"pod-2": podmetrics.Metric{
					Value: 150,
				},
			},
			6,
			50,
			2,
			sets.String{
				"missing-1": {},
				"missing-2": {},
			},
			sets.String{
				"ignored-1": {},
				"ignored-2": {},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			calc := replicas.ReplicaCalculator{
				Tolerance: test.tolerance,
			}
			result := calc.GetPlainMetricReplicaCount(test.metrics, test.currentReplicas, test.targetUtilization, test.readyPodCount, test.missingPods, test.ignoredPods)
			if !cmp.Equal(test.expected, result) {
				t.Errorf("replica mismatch (-want +got):\n%s", cmp.Diff(test.expected, result))
			}
		})
	}
}
