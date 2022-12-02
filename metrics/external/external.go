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

package external

import (
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/value"
)

// Metric (Resource) is a global metric that is not associated with any Kubernetes object. It allows autoscaling based
// on information coming from components running outside of cluster (for example length of queue in cloud messaging
// service, or QPS from loadbalancer running outside of cluster).
type Metric struct {
	Current       value.MetricValue `json:"current,omitempty"`
	ReadyPodCount *int64            `json:"ready_pod_count,omitempty"`
	Timestamp     time.Time         `json:"timestamp,omitempty"`
}
