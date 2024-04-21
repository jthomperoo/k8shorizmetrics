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
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v4/internal/external"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/object"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/pods"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/podutil"
	"github.com/jthomperoo/k8shorizmetrics/v4/internal/resource"
	"github.com/jthomperoo/k8shorizmetrics/v4/metrics"
	externalmetrics "github.com/jthomperoo/k8shorizmetrics/v4/metrics/external"
	objectmetrics "github.com/jthomperoo/k8shorizmetrics/v4/metrics/object"
	podsmetrics "github.com/jthomperoo/k8shorizmetrics/v4/metrics/pods"
	resourcemetrics "github.com/jthomperoo/k8shorizmetrics/v4/metrics/resource"
	metricsclient "github.com/jthomperoo/k8shorizmetrics/v4/metricsclient"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	k8sscale "k8s.io/client-go/scale"
)

// GathererMultiMetricError occurs when gathering multiple metrics, if any metric fails to be gathered this error will
// be returned which contains all of the individual errors in the 'Errors' slice, if some metrics were gathered
// successfully the error will have the 'Partial' property set to true.
type GathererMultiMetricError struct {
	Partial bool
	Errors  []error
}

func (e *GathererMultiMetricError) Error() string {
	return fmt.Sprintf("gatherer multi metric error: %d errors, first error is %s", len(e.Errors), e.Errors[0])
}

// ExternalGatherer allows retrieval of external metrics.
type ExternalGatherer interface {
	Gather(metricName, namespace string, metricSelector *metav1.LabelSelector, podSelector labels.Selector) (*externalmetrics.Metric, error)
	GatherPerPod(metricName, namespace string, metricSelector *metav1.LabelSelector) (*externalmetrics.Metric, error)
}

// ObjectGatherer allows retrieval of object metrics.
type ObjectGatherer interface {
	Gather(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, podSelector labels.Selector, metricSelector labels.Selector) (*objectmetrics.Metric, error)
	GatherPerPod(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (*objectmetrics.Metric, error)
}

// PodsGatherer allows retrieval of pods metrics.
type PodsGatherer interface {
	Gather(metricName string, namespace string, podSelector labels.Selector, metricSelector labels.Selector) (*podsmetrics.Metric, error)
}

// ResourceGatherer allows retrieval of resource metrics.
type ResourceGatherer interface {
	Gather(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
		cpuInitializationPeriod time.Duration, delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error)
	GatherRaw(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector,
		cpuInitializationPeriod time.Duration, delayOfInitialReadinessStatus time.Duration) (*resourcemetrics.Metric, error)
}

// Gatherer provides functionality for retrieving metrics on supplied metric specs.
type Gatherer struct {
	Resource                      ResourceGatherer
	Pods                          PodsGatherer
	Object                        ObjectGatherer
	External                      ExternalGatherer
	ScaleClient                   k8sscale.ScalesGetter
	CPUInitializationPeriod       time.Duration
	DelayOfInitialReadinessStatus time.Duration
}

// NewGatherer sets up a new Metric Gatherer
func NewGatherer(
	metricsclient metricsclient.Client,
	podlister corelisters.PodLister,
	cpuInitializationPeriod time.Duration,
	delayOfInitialReadinessStatus time.Duration) *Gatherer {

	// Set up pod ready counter
	podReadyCounter := &podutil.PodReadyCount{
		PodLister: podlister,
	}

	return &Gatherer{
		Resource: &resource.Gather{
			MetricsClient: metricsclient,
			PodLister:     podlister,
		},
		Pods: &pods.Gather{
			MetricsClient: metricsclient,
			PodLister:     podlister,
		},
		Object: &object.Gather{
			MetricsClient:   metricsclient,
			PodReadyCounter: podReadyCounter,
		},
		External: &external.Gather{
			MetricsClient:   metricsclient,
			PodReadyCounter: podReadyCounter,
		},
		CPUInitializationPeriod:       cpuInitializationPeriod,
		DelayOfInitialReadinessStatus: delayOfInitialReadinessStatus,
	}
}

// Gather returns all of the metrics gathered based on the metric specs provided.
// If an error occurs gathering any metric this will return a GatherMultiMetricError. If a partial error occurs,
// meaning some metrics were gathered successfully and others failed, the 'Partial' property of this error will be
// set to true.
func (c *Gatherer) Gather(specs []autoscalingv2.MetricSpec, namespace string, podSelector labels.Selector) ([]*metrics.Metric, error) {
	return c.GatherWithOptions(specs, namespace, podSelector, c.CPUInitializationPeriod, c.DelayOfInitialReadinessStatus)
}

// GatherWithOptions returns all of the metrics gathered based on the metric specs provided with options.
// If an error occurs gathering any metric this will return a GatherMultiMetricError. If a partial error occurs,
// meaning some metrics were gathered successfully and others failed, the 'Partial' property of this error will be
// set to true.
func (c *Gatherer) GatherWithOptions(specs []autoscalingv2.MetricSpec, namespace string, podSelector labels.Selector,
	cpuInitializationPeriod time.Duration, delayOfInitialReadinessStatus time.Duration) ([]*metrics.Metric, error) {
	combinedMetrics := []*metrics.Metric{}
	gatherErrors := []error{}
	for _, spec := range specs {
		metric, err := c.GatherSingleMetricWithOptions(spec, namespace, podSelector, cpuInitializationPeriod, delayOfInitialReadinessStatus)
		if err != nil {
			gatherErrors = append(gatherErrors, err)
			continue
		}
		combinedMetrics = append(combinedMetrics, metric)
	}

	if len(gatherErrors) > 0 {
		partial := len(gatherErrors) < len(specs)
		if partial {
			return combinedMetrics, &GathererMultiMetricError{
				Partial: partial,
				Errors:  gatherErrors,
			}
		}

		return nil, &GathererMultiMetricError{
			Partial: partial,
			Errors:  gatherErrors,
		}
	}

	return combinedMetrics, nil
}

// GatherSingleMetric returns the metric gathered based on a single metric spec.
func (c *Gatherer) GatherSingleMetric(spec autoscalingv2.MetricSpec, namespace string, podSelector labels.Selector) (*metrics.Metric, error) {
	return c.GatherSingleMetricWithOptions(spec, namespace, podSelector, c.CPUInitializationPeriod, c.DelayOfInitialReadinessStatus)
}

// GatherSingleMetricWithOptions returns the metric gathered based on a single metric spec with options.
func (c *Gatherer) GatherSingleMetricWithOptions(spec autoscalingv2.MetricSpec, namespace string, podSelector labels.Selector,
	cpuInitializationPeriod time.Duration, delayOfInitialReadinessStatus time.Duration) (*metrics.Metric, error) {
	switch spec.Type {
	case autoscalingv2.ObjectMetricSourceType:
		metricSelector, err := metav1.LabelSelectorAsSelector(spec.Object.Metric.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to get object metric: %w", err)
		}

		switch spec.Object.Target.Type {
		case autoscalingv2.ValueMetricType:
			objectMetric, err := c.Object.Gather(spec.Object.Metric.Name, namespace, &spec.Object.DescribedObject, podSelector, metricSelector)
			if err != nil {
				return nil, fmt.Errorf("failed to get object metric: %w", err)
			}
			return &metrics.Metric{
				Spec:   spec,
				Object: objectMetric,
			}, nil
		case autoscalingv2.AverageValueMetricType:
			objectMetric, err := c.Object.GatherPerPod(spec.Object.Metric.Name, namespace, &spec.Object.DescribedObject, metricSelector)
			if err != nil {
				return nil, fmt.Errorf("failed to get object metric: %w", err)
			}
			return &metrics.Metric{
				Spec:   spec,
				Object: objectMetric,
			}, nil
		default:
			return nil, fmt.Errorf("invalid object metric source: must be either value or average value")
		}
	case autoscalingv2.PodsMetricSourceType:
		metricSelector, err := metav1.LabelSelectorAsSelector(spec.Pods.Metric.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to get pods metric: %w", err)
		}

		if spec.Pods.Target.Type != autoscalingv2.AverageValueMetricType {
			return nil, fmt.Errorf("invalid pods metric source: must be average value")
		}

		podsMetric, err := c.Pods.Gather(spec.Pods.Metric.Name, namespace, podSelector, metricSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to get pods metric: %w", err)
		}
		return &metrics.Metric{
			Spec: spec,
			Pods: podsMetric,
		}, nil
	case autoscalingv2.ResourceMetricSourceType:
		switch spec.Resource.Target.Type {
		case autoscalingv2.AverageValueMetricType:
			resourceMetric, err := c.Resource.GatherRaw(spec.Resource.Name, namespace, podSelector, cpuInitializationPeriod, delayOfInitialReadinessStatus)
			if err != nil {
				return nil, fmt.Errorf("failed to get resource metric: %w", err)
			}
			return &metrics.Metric{
				Spec:     spec,
				Resource: resourceMetric,
			}, nil
		case autoscalingv2.UtilizationMetricType:
			resourceMetric, err := c.Resource.Gather(spec.Resource.Name, namespace, podSelector, cpuInitializationPeriod, delayOfInitialReadinessStatus)
			if err != nil {
				return nil, fmt.Errorf("failed to get resource metric: %w", err)
			}
			return &metrics.Metric{
				Spec:     spec,
				Resource: resourceMetric,
			}, nil
		default:
			return nil, fmt.Errorf("invalid resource metric source: must be either average value or average utilization")
		}

	case autoscalingv2.ExternalMetricSourceType:
		switch spec.External.Target.Type {
		case autoscalingv2.ValueMetricType:
			externalMetric, err := c.External.Gather(spec.External.Metric.Name, namespace, spec.External.Metric.Selector, podSelector)
			if err != nil {
				return nil, fmt.Errorf("failed to get external metric: %w", err)
			}
			return &metrics.Metric{
				Spec:     spec,
				External: externalMetric,
			}, nil
		case autoscalingv2.AverageValueMetricType:
			externalMetric, err := c.External.GatherPerPod(spec.External.Metric.Name, namespace, spec.External.Metric.Selector)
			if err != nil {
				return nil, fmt.Errorf("failed to get external metric: %w", err)
			}
			return &metrics.Metric{
				Spec:     spec,
				External: externalMetric,
			}, nil
		default:
			return nil, fmt.Errorf("invalid external metric source: must be either value or average value")
		}

	default:
		return nil, fmt.Errorf("unknown metric source type %q", string(spec.Type))
	}
}
