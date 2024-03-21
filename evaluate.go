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

Modifications Copyright 2024 The K8sHorizMetrics Authors.

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

	"github.com/jthomperoo/k8shorizmetrics/v3/internal/external"
	"github.com/jthomperoo/k8shorizmetrics/v3/internal/object"
	"github.com/jthomperoo/k8shorizmetrics/v3/internal/pods"
	"github.com/jthomperoo/k8shorizmetrics/v3/internal/replicas"
	"github.com/jthomperoo/k8shorizmetrics/v3/internal/resource"
	"github.com/jthomperoo/k8shorizmetrics/v3/metrics"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// EvaluatorMultiMetricError occurs when evaluating multiple metrics, if any metric fails to be evaluated this error
// will be returned which contains all of the individual errors in the 'Errors' slice, if some metrics
// were evaluated successfully the error will have the 'Partial' property set to true.
type EvaluatorMultiMetricError struct {
	Partial bool
	Errors  []error
}

func (e *EvaluatorMultiMetricError) Error() string {
	return fmt.Sprintf("evaluator multi metric error: %d errors, first error is %s", len(e.Errors), e.Errors[0])
}

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
// If an error occurs evaluating any metric this will return a EvaluatorMultiMetricError. If a partial error occurs,
// meaning some metrics were evaluated successfully and others failed, the 'Partial' property of this error will be
// set to true.
func (e *Evaluator) Evaluate(gatheredMetrics []*metrics.Metric, currentReplicas int32) (int32, error) {
	return e.EvaluateWithOptions(gatheredMetrics, currentReplicas, e.Tolerance)
}

// EvaluateWithOptions returns the target replica count for an array of multiple metrics with provided options
// If an error occurs evaluating any metric this will return a EvaluatorMultiMetricError. If a partial error occurs,
// meaning some metrics were evaluated successfully and others failed, the 'Partial' property of this error will be
// set to true.
func (e *Evaluator) EvaluateWithOptions(gatheredMetrics []*metrics.Metric, currentReplicas int32,
	tolerance float64) (int32, error) {
	var evaluation int32
	var evaluationErrors []error

	for i, gatheredMetric := range gatheredMetrics {
		proposedEvaluation, err := e.EvaluateSingleMetricWithOptions(gatheredMetric, currentReplicas, tolerance)
		if err != nil {
			evaluationErrors = append(evaluationErrors, err)
			continue
		}

		if i == 0 {
			evaluation = proposedEvaluation
		}

		// Multiple evaluations, take the highest replica count
		if proposedEvaluation > evaluation {
			evaluation = proposedEvaluation
		}
	}

	if len(evaluationErrors) > 0 {
		partial := len(evaluationErrors) < len(gatheredMetrics)
		if partial {
			return evaluation, &EvaluatorMultiMetricError{
				Partial: partial,
				Errors:  evaluationErrors,
			}
		}

		return 0, &EvaluatorMultiMetricError{
			Partial: partial,
			Errors:  evaluationErrors,
		}
	}

	return evaluation, nil
}

// EvaluateSingleMetric returns the target replica count for a single metrics
func (e *Evaluator) EvaluateSingleMetric(gatheredMetric *metrics.Metric, currentReplicas int32) (int32, error) {
	return e.EvaluateSingleMetricWithOptions(gatheredMetric, currentReplicas, e.Tolerance)
}

// EvaluateSingleMetricWithOptions returns the target replica count for a single metrics with provided options
func (e *Evaluator) EvaluateSingleMetricWithOptions(gatheredMetric *metrics.Metric, currentReplicas int32,
	tolerance float64) (int32, error) {
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
