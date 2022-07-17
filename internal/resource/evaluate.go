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

package resource

import (
	"fmt"
	"math"

	"github.com/jthomperoo/k8shorizmetrics/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/metrics"
	"github.com/jthomperoo/k8shorizmetrics/metrics/podmetrics"
	"github.com/jthomperoo/k8shorizmetrics/metricsclient"
)

// Evaluate (resource) calculates a replica count evaluation, using the tolerance and calculater provided
type Evaluate struct {
	Calculater replicas.Calculator
}

// Evaluate calculates an evaluation based on the metric provided and the current number of replicas
func (e *Evaluate) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
	if gatheredMetric.Spec.Resource.Target.AverageValue != nil {
		replicaCount := e.Calculater.GetPlainMetricReplicaCount(
			gatheredMetric.Resource.PodMetricsInfo,
			currentReplicas,
			gatheredMetric.Spec.Resource.Target.AverageValue.MilliValue(),
			gatheredMetric.Resource.ReadyPodCount,
			gatheredMetric.Resource.MissingPods,
			gatheredMetric.Resource.IgnoredPods,
		)
		return replicaCount, nil
	}

	if gatheredMetric.Spec.Resource.Target.AverageUtilization != nil {
		metrics := gatheredMetric.Resource.PodMetricsInfo
		requests := gatheredMetric.Resource.Requests
		targetUtilization := *gatheredMetric.Spec.Resource.Target.AverageUtilization
		ignoredPods := gatheredMetric.Resource.IgnoredPods
		missingPods := gatheredMetric.Resource.MissingPods
		readyPodCount := gatheredMetric.Resource.ReadyPodCount

		usageRatio, _, _, err := metricsclient.GetResourceUtilizationRatio(metrics, requests, targetUtilization)
		if err != nil {
			return 0, err
		}

		// usageRatio = SUM(pod metrics) / SUM(pod requests) / targetUtilization
		// usageRatio = averageUtilization / targetUtilization
		// usageRatio ~ 1.0 == no scale
		// usageRatio > 1.0 == scale up
		// usageRatio < 1.0 == scale down

		rebalanceIgnored := len(ignoredPods) > 0 && usageRatio > 1.0
		if !rebalanceIgnored && len(missingPods) == 0 {
			if math.Abs(1.0-usageRatio) <= tolerance {
				// return the current replicas if the change would be too small
				return currentReplicas, nil
			}
			targetReplicas := int32(math.Ceil(usageRatio * float64(readyPodCount)))
			// if we don't have any unready or missing pods, we can calculate the new replica count now
			return targetReplicas, nil
		}

		if len(missingPods) > 0 {
			if usageRatio < 1.0 {
				// on a scale-down, treat missing pods as using 100% of the resource request
				for podName := range missingPods {
					metrics[podName] = podmetrics.Metric{Value: requests[podName]}
				}
			} else if usageRatio > 1.0 {
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
		newUsageRatio, _, _, err := metricsclient.GetResourceUtilizationRatio(metrics, requests, targetUtilization)
		if err != nil {
			// NOTE - Unsure if this can be triggered.
			return 0, err
		}

		if math.Abs(1.0-newUsageRatio) <= tolerance || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
			// return the current replicas if the change would be too small,
			// or if the new usage ratio would cause a change in scale direction
			return currentReplicas, nil
		}

		// return the result, where the number of replicas considered is
		// however many replicas factored into our calculation
		targetReplicas := int32(math.Ceil(newUsageRatio * float64(len(metrics))))
		return targetReplicas, nil
	}

	return 0, fmt.Errorf("invalid resource metric source: neither a utilization target nor a value target was set")
}
