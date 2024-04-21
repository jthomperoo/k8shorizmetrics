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

// Package metrics provides models for all of the metrics returned from the K8s APIs grouped into a single model.
package metrics

import (
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/external"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/object"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/pods"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics/resource"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// Metric is a metric that has been retrieved from the K8s metrics server
type Metric struct {
	Spec     autoscalingv2.MetricSpec `json:"spec"`
	Resource *resource.Metric         `json:"resource,omitempty"`
	Pods     *pods.Metric             `json:"pods,omitempty"`
	Object   *object.Metric           `json:"object,omitempty"`
	External *external.Metric         `json:"external,omitempty"`
}
