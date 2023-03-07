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

package k8shorizmetrics

import (
	"fmt"

	"github.com/jthomperoo/k8shorizmetrics/v2/internal/external"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/object"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/pods"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/resource"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// ExternalEvaluater produces a replica count based on an external metric provided
type ExternalEvaluater interface {
	Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error)
}

// ObjectEvaluater produces a replica count based on an object metric provided
type ObjectEvaluater interface {
	Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error)
}

// PodsEvaluater produces a replica count based on a pods metric provided
type PodsEvaluater interface {
	Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric) int32
}

// ResourceEvaluater produces an evaluation based on a resource metric provided
type ResourceEvaluater interface {
	Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error)
}

// Evaluator provides functionality for deciding how many replicas a resource should have based on provided metrics.
type Evaluator struct {
	External  ExternalEvaluater
	Object    ObjectEvaluater
	Pods      PodsEvaluater
	Resource  ResourceEvaluater
	Tolerance float64
}

// NewEvaluator sets up an evaluate that can process external, object, pod and resource metrics
func NewEvaluator(tolerance float64) *Evaluator {
	calculate := &replicas.ReplicaCalculator{
		Tolerance: tolerance,
	}
	return &Evaluator{
		External: &external.Evaluate{
			Calculater: calculate,
		},
		Object: &object.Evaluate{
			Calculater: calculate,
		},
		Pods: &pods.Evaluate{
			Calculater: calculate,
		},
		Resource: &resource.Evaluate{
			Calculater: calculate,
		},
	}
}

// Evaluate returns the target replica count for an array of multiple metrics
func (e *Evaluator) Evaluate(gatheredMetrics []*metrics.Metric, currentReplicas int32) (int32, error) {
	return e.EvaluateWithOptions(gatheredMetrics, currentReplicas, e.Tolerance)
}

// EvaluateWithOptions returns the target replica count for an array of multiple metrics with provided options
func (e *Evaluator) EvaluateWithOptions(gatheredMetrics []*metrics.Metric, currentReplicas int32, tolerance float64) (int32, error) {
	var evaluation int32
	var invalidEvaluationError error
	invalidEvaluationsCount := 0

	for i, gatheredMetric := range gatheredMetrics {
		proposedEvaluation, err := e.EvaluateSingleMetricWithOptions(gatheredMetric, currentReplicas, tolerance)
		if err != nil {
			if invalidEvaluationsCount <= 0 {
				invalidEvaluationError = err
			}
			invalidEvaluationsCount++
			continue
		}
		if i == 0 {
			evaluation = proposedEvaluation
		}
		// Mutliple calculations, take the highest replica count
		if proposedEvaluation > evaluation {
			evaluation = proposedEvaluation
		}
	}

	// If all evaluations are invalid return error and return first evaluation error.
	if invalidEvaluationsCount >= len(gatheredMetrics) {
		return 0, fmt.Errorf("invalid calculations (%v invalid out of %v), first error is: %v", invalidEvaluationsCount, len(gatheredMetrics), invalidEvaluationError)
	}
	return evaluation, nil
}

// EvaluateSingleMetric returns the target replica count for a single metrics
func (e *Evaluator) EvaluateSingleMetric(gatheredMetric *metrics.Metric, currentReplicas int32) (int32, error) {
	return e.EvaluateSingleMetricWithOptions(gatheredMetric, currentReplicas, e.Tolerance)
}

// EvaluateSingleMetricWithOptions returns the target replica count for a single metrics with provided options
func (e *Evaluator) EvaluateSingleMetricWithOptions(gatheredMetric *metrics.Metric, currentReplicas int32, tolerance float64) (int32, error) {
	switch gatheredMetric.Spec.Type {
	case autoscalingv2.ObjectMetricSourceType:
		return e.Object.Evaluate(currentReplicas, gatheredMetric, tolerance)
	case autoscalingv2.PodsMetricSourceType:
		return e.Pods.Evaluate(currentReplicas, gatheredMetric), nil
	case autoscalingv2.ResourceMetricSourceType:
		return e.Resource.Evaluate(currentReplicas, gatheredMetric, tolerance)
	case autoscalingv2.ExternalMetricSourceType:
		return e.External.Evaluate(currentReplicas, gatheredMetric, tolerance)
	default:
		return 0, fmt.Errorf("unknown metric source type %q", string(gatheredMetric.Spec.Type))
	}
}
