/*
Copyright 2024 The K8sHorizMetrics Authors.

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
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics"
)

// ExternalEvaluater (fake) provides a way to insert functionality into a ExternalEvaluater
type ExternalEvaluater struct {
	EvaluateReactor func(currentReplicas int32, gatheredMetric *metrics.Metric,
		tolerance float64) (int32, error)
}

// Evaluate calls the fake ExternalEvaluater function
func (f *ExternalEvaluater) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric,
	tolerance float64) (int32, error) {
	return f.EvaluateReactor(currentReplicas, gatheredMetric, tolerance)
}

// ObjectEvaluater (fake) provides a way to insert functionality into a ObjectEvaluater
type ObjectEvaluater struct {
	EvaluateReactor func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error)
}

// Evaluate calls the fake ObjectEvaluater function
func (f *ObjectEvaluater) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric,
	tolerance float64) (int32, error) {
	return f.EvaluateReactor(currentReplicas, gatheredMetric, tolerance)
}

// PodsEvaluater (fake) provides a way to insert functionality into a PodsEvaluater
type PodsEvaluater struct {
	EvaluateReactor func(currentReplicas int32, gatheredMetric *metrics.Metric) int32
}

// Evaluate calls the fake PodsEvaluater function
func (f *PodsEvaluater) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric) int32 {
	return f.EvaluateReactor(currentReplicas, gatheredMetric)
}

// ResourceEvaluater (fake) provides a way to insert functionality into a ResourceEvaluater
type ResourceEvaluater struct {
	EvaluateReactor func(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error)
}

// Evaluate calls the fake ResourceEvaluater function
func (f *ResourceEvaluater) Evaluate(currentReplicas int32, gatheredMetric *metrics.Metric, tolerance float64) (int32, error) {
	return f.EvaluateReactor(currentReplicas, gatheredMetric, tolerance)
}
