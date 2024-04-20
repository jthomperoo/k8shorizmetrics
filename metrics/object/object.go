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

// Package object contains models for object metrics as returned by the K8s metrics APIs.
package object

import (
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v3/metrics/value"
)

// Metric (Object) is a metric describing a kubernetes object (for example, hits-per-second on an Ingress object).
type Metric struct {
	Current       value.MetricValue `json:"current,omitempty"`
	ReadyPodCount *int64            `json:"readyPodCount,omitempty"`
	Timestamp     time.Time         `json:"timestamp,omitempty"`
}
