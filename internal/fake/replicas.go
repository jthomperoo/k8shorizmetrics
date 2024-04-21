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

package fake

import (
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/podmetrics"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Calculate (fake) provides a way to insert functionality into a Calculater
type Calculate struct {
	GetUsageRatioReplicaCountReactor  func(currentReplicas int32, usageRatio float64, readyPodCount int64) int32
	GetPlainMetricReplicaCountReactor func(metrics podmetrics.MetricsInfo,
		currentReplicas int32,
		targetUtilization int64,
		readyPodCount int64,
		missingPods,
		ignoredPods sets.String) int32
}

// GetUsageRatioReplicaCount calls the fake Calculater function
func (f *Calculate) GetUsageRatioReplicaCount(currentReplicas int32, usageRatio float64, readyPodCount int64) int32 {
	return f.GetUsageRatioReplicaCountReactor(currentReplicas, usageRatio, readyPodCount)
}

// GetPlainMetricReplicaCount calls the fake Calculater function
func (f *Calculate) GetPlainMetricReplicaCount(metrics podmetrics.MetricsInfo,
	currentReplicas int32,
	targetUtilization int64,
	readyPodCount int64,
	missingPods,
	ignoredPods sets.String) int32 {
	return f.GetPlainMetricReplicaCountReactor(metrics, currentReplicas, targetUtilization, readyPodCount, missingPods, ignoredPods)
}
