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

package external

import (
	"fmt"
	"math"

	"github.com/jthomperoo/k8shorizmetrics/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/metrics"
)

// Evaluate (external) calculates a replica count evaluation, using the tolerance and calculater provided
type Evaluate struct {
	Calculater replicas.Calculator
}

// Evaluate calculates an evaluation based on the metric provided and the current number of replicas
func (e *Evaluate) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
	if gatheredMetric.Spec.External.Target.AverageValue != nil {
		utilization := float64(*gatheredMetric.External.Current.AverageValue)
		targetUtilizationPerPod := gatheredMetric.Spec.External.Target.AverageValue.MilliValue()
		replicaCount := currentReplicas
		usageRatio := float64(utilization) / (float64(targetUtilizationPerPod) * float64(replicaCount))
		if math.Abs(1.0-usageRatio) > tolerance {
			// update number of replicas if the change is large enough
			replicaCount = int32(math.Ceil(float64(utilization) / float64(targetUtilizationPerPod)))
		}
		return replicaCount, nil
	}

	if gatheredMetric.Spec.External.Target.Value != nil {
		utilization := float64(*gatheredMetric.External.Current.Value)
		replicaCount := currentReplicas
		targetUtilization := gatheredMetric.Spec.External.Target.Value.MilliValue()
		readyPodCount := gatheredMetric.External.ReadyPodCount

		usageRatio := float64(utilization) / float64(targetUtilization)
		replicaCount = e.Calculater.GetUsageRatioReplicaCount(currentReplicas, usageRatio, *readyPodCount)
		return replicaCount, nil
	}
	return 0, fmt.Errorf("invalid external metric source: neither a value target nor an average value target was set")
}
