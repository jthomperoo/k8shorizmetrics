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

package pods

import (
	"fmt"

	"github.com/jthomperoo/k8shorizmetrics/v2/internal/podutil"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/pods"
	metricsclient "github.com/jthomperoo/k8shorizmetrics/v2/metricsclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// Gather (Pods) provides functionality for retrieving metrics for pods metric specs.
type Gather struct {
	MetricsClient metricsclient.Client
	PodLister     corelisters.PodLister
}

// Gather retrieves a pods metric
func (c *Gather) Gather(metricName string, namespace string, podSelector labels.Selector, metricSelector labels.Selector) (*pods.Metric, error) {
	// Get metrics
	metrics, timestamp, err := c.MetricsClient.GetRawMetric(metricName, namespace, podSelector, metricSelector)
	if err != nil {
		return nil, fmt.Errorf("unable to get metric %s: %w", metricName, err)
	}

	// Get pods
	podList, err := c.PodLister.Pods(namespace).List(podSelector)
	if err != nil {
		return nil, fmt.Errorf("unable to get pods while calculating replica count: %w", err)
	}

	totalPods := len(podList)
	if totalPods == 0 {
		return &pods.Metric{
			ReadyPodCount: 0,
			TotalPods:     0,
			Timestamp:     timestamp,
		}, nil
	}

	// Remove missing pod metrics
	readyPodCount, _, missingPods := podutil.GroupPods(podList, metrics, corev1.ResourceName(""), 0, 0)

	return &pods.Metric{
		PodMetricsInfo: metrics,
		ReadyPodCount:  int64(readyPodCount),
		IgnoredPods:    nil, // Pods metric cannot be CPU based, so Pods cannot be ignored
		MissingPods:    missingPods,
		TotalPods:      totalPods,
		Timestamp:      timestamp,
	}, nil
}
