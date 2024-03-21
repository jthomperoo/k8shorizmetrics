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
	"time"

	externalmetrics "github.com/jthomperoo/k8shorizmetrics/v2/metrics/external"
	objectmetrics "github.com/jthomperoo/k8shorizmetrics/v2/metrics/object"
	podsmetrics "github.com/jthomperoo/k8shorizmetrics/v2/metrics/pods"
	resourcemetrics "github.com/jthomperoo/k8shorizmetrics/v2/metrics/resource"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ExternalGatherer (fake) provides a way to insert functionality into a ExternalGatherer
type ExternalGatherer struct {
	GatherReactor       func(metricName, namespace string, metricSelector *metav1.LabelSelector, podSelector labels.Selector) (*externalmetrics.Metric, error)
	GatherPerPodReactor func(metricName, namespace string, metricSelector *metav1.LabelSelector) (*externalmetrics.Metric, error)
}

// Gather calls the fake ExternalGatherer function
func (f *ExternalGatherer) Gather(metricName, namespace string, metricSelector *metav1.LabelSelector,
	podSelector labels.Selector) (*externalmetrics.Metric, error) {
	return f.GatherReactor(metricName, namespace, metricSelector, podSelector)
}

// GatherPerPod calls the fake ExternalGatherer function
func (f *ExternalGatherer) GatherPerPod(metricName, namespace string,
	metricSelector *metav1.LabelSelector) (*externalmetrics.Metric, error) {
	return f.GatherPerPodReactor(metricName, namespace, metricSelector)
}

// ObjectGatherer (fake) provides a way to insert functionality into a ObjectGatherer
type ObjectGatherer struct {
	GatherReactor func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference,
		podSelector labels.Selector, metricSelector labels.Selector) (*objectmetrics.Metric, error)
	GatherPerPodReactor func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference,
		metricSelector labels.Selector) (*objectmetrics.Metric, error)
}

// Gather calls the fake ObjectGatherer function
func (f *ObjectGatherer) Gather(metricName string, namespace string,
	objectRef *autoscalingv2.CrossVersionObjectReference, podSelector labels.Selector,
	metricSelector labels.Selector) (*objectmetrics.Metric, error) {
	return f.GatherReactor(metricName, namespace, objectRef, podSelector, metricSelector)
}

// GatherPerPod calls the fake ObjectGatherer function
func (f *ObjectGatherer) GatherPerPod(metricName string, namespace string,
	objectRef *autoscalingv2.CrossVersionObjectReference,
	metricSelector labels.Selector) (*objectmetrics.Metric, error) {
	return f.GatherPerPodReactor(metricName, namespace, objectRef, metricSelector)
}

// PodsGatherer (fake) provides a way to insert functionality into a PodsGatherer
type PodsGatherer struct {
	GatherReactor func(metricName string, namespace string, podSelector labels.Selector,
		metricSelector labels.Selector) (*podsmetrics.Metric, error)
}

// Gather calls the fake PodsGatherer function
func (f *PodsGatherer) Gather(metricName string, namespace string, podSelector labels.Selector,
	metricSelector labels.Selector) (*podsmetrics.Metric, error) {
	return f.GatherReactor(metricName, namespace, podSelector, metricSelector)
}

// ResourceGatherer (fake) provides a way to insert functionality into a ResourceGatherer
type ResourceGatherer struct {
	GatherReactor func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
		cpuInitializationPeriod time.Duration,
		delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error)
	GatherRawReactor func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
		cpuInitializationPeriod time.Duration,
		delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error)
}

// Gather calls the fake ResourceGatherer function
func (f *ResourceGatherer) Gather(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
	cpuInitializationPeriod time.Duration,
	delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error) {
	return f.GatherReactor(resourceName, namespace, podSelector, cpuInitializationPeriod, delayOfInitialReadinessStatus)
}

// GatherRaw calls the fake ResourceGatherer function
func (f *ResourceGatherer) GatherRaw(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
	cpuInitializationPeriod time.Duration,
	delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error) {
	return f.GatherRawReactor(resourceName, namespace, podSelector, cpuInitializationPeriod, delayOfInitialReadinessStatus)
}
