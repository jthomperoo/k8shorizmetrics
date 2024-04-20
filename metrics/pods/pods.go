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

// Package pods contains models for metrics relating to a set of pods as returned by the K8s metrics APIs.
package pods

import (
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v3/metrics/podmetrics"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Metric (Pods) is a metric describing each pod in the current scale target (for example,
// transactions-processed-per-second).  The values will be averaged together before being compared to the target value.
type Metric struct {
	PodMetricsInfo podmetrics.MetricsInfo `json:"podMetricsInfo"`
	ReadyPodCount  int64                  `json:"readyPodCount"`
	IgnoredPods    sets.String            `json:"ignoredPods"`
	MissingPods    sets.String            `json:"missingPods"`
	TotalPods      int                    `json:"totalPods"`
	Timestamp      time.Time              `json:"timestamp,omitempty"`
}
