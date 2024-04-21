/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 The K8sHorizMetrics Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

Modified to split up evaluations and metric gathering to work with the
Custom Pod Autoscaler framework.
Original source:
https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/horizontal.go
https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/replica_calculator.go
*/

// Package replicas provides utilities for getting replica counts from the K8s APIs.
package replicas

import (
	"math"

	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/podmetrics"
	"github.com/jthomperoo/k8shorizmetrics/v4/metricsclient"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Calculator is used to calculate replica counts
type Calculator interface {
	GetUsageRatioReplicaCount(currentReplicas int32, usageRatio float64, readyPodCount int64) int32
	GetPlainMetricReplicaCount(metrics podmetrics.MetricsInfo,
		currentReplicas int32,
		targetUtilization int64,
		readyPodCount int64,
		missingPods,
		ignoredPods sets.String) int32
}

// ReplicaCalculator uses a tolerance provided to calculate replica counts for scaling up/down/remaining the same
type ReplicaCalculator struct {
	Tolerance float64
}

// GetUsageRatioReplicaCount calculates the replica count based on the number of replicas, number of ready pods and the
// usage ratio of the metric - providing a different value if beyond the tolerance
func (r *ReplicaCalculator) GetUsageRatioReplicaCount(currentReplicas int32, usageRatio float64, readyPodCount int64) int32 {
	var replicaCount int32
	if currentReplicas != 0 {
		if math.Abs(1.0-usageRatio) <= r.Tolerance {
			// return the current replicas if the change would be too small
			return currentReplicas
		}
		replicaCount = int32(math.Ceil(usageRatio * float64(readyPodCount)))
	} else {
		// Scale to zero or n pods depending on usageRatio
		replicaCount = int32(math.Ceil(usageRatio))
	}

	return replicaCount
}

// GetPlainMetricReplicaCount calculates the replica count based on the metrics of each pod and a target utilization, providing
// a different replica count if the calculated usage ratio is beyond the tolerance
func (r *ReplicaCalculator) GetPlainMetricReplicaCount(metrics podmetrics.MetricsInfo,
	currentReplicas int32,
	targetUtilization int64,
	readyPodCount int64,
	missingPods,
	ignoredPods sets.String) int32 {

	usageRatio, _ := metricsclient.GetMetricUtilizationRatio(metrics, targetUtilization)

	// usageRatio = SUM(pod metrics) / number of pods / targetUtilization
	// usageRatio = averageUtilization / targetUtilization
	// usageRatio ~ 1.0 == no scale
	// usageRatio > 1.0 == scale up
	// usageRatio < 1.0 == scale down

	rebalanceIgnored := len(ignoredPods) > 0 && usageRatio > 1.0

	if !rebalanceIgnored && len(missingPods) == 0 {
		if math.Abs(1.0-usageRatio) <= r.Tolerance {
			// return the current replicas if the change would be too small
			return currentReplicas
		}

		// if we don't have any unready or missing pods, we can calculate the new replica count now
		return int32(math.Ceil(usageRatio * float64(readyPodCount)))
	}

	if len(missingPods) > 0 {
		if usageRatio < 1.0 {
			// on a scale-down, treat missing pods as using 100% of the resource request
			for podName := range missingPods {
				metrics[podName] = podmetrics.Metric{Value: targetUtilization}
			}
		} else {
			// on a scale-up, treat missing pods as using 0% of the resource request
			for podName := range missingPods {
				metrics[podName] = podmetrics.Metric{Value: 0}
			}
		}
	}

	if rebalanceIgnored {
		// on a scale-up, treat unready pods as using 0% of the resource request
		for podName := range ignoredPods {
			metrics[podName] = podmetrics.Metric{Value: 0}
		}
	}

	// re-run the utilization calculation with our new numbers
	newUsageRatio, _ := metricsclient.GetMetricUtilizationRatio(metrics, targetUtilization)

	if math.Abs(1.0-newUsageRatio) <= r.Tolerance || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		// return the current replicas if the change would be too small,
		// or if the new usage ratio would cause a change in scale direction
		return currentReplicas
	}

	// return the result, where the number of replicas considered is
	// however many replicas factored into our calculation
	return int32(math.Ceil(newUsageRatio * float64(len(metrics))))
}
