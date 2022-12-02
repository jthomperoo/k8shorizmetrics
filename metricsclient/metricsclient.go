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

package metricsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/podmetrics"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	custommetricsv1 "k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/metrics/pkg/client/external_metrics"
)

const (
	metricServerDefaultMetricWindow = time.Minute
)

// Client allows for retrieval of Kubernetes metrics
type Client interface {
	GetResourceMetric(resource v1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error)
	GetRawMetric(metricName string, namespace string, selector labels.Selector, metricSelector labels.Selector) (podmetrics.MetricsInfo, time.Time, error)
	GetObjectMetric(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error)
	GetExternalMetric(metricName, namespace string, selector labels.Selector) ([]int64, time.Time, error)
}

func NewClient(clusterConfig *rest.Config, discovery discovery.DiscoveryInterface) *RESTClient {
	return &RESTClient{
		Client:                metricsv1beta1.NewForConfigOrDie(clusterConfig),
		ExternalMetricsClient: external_metrics.NewForConfigOrDie(clusterConfig),
		CustomMetricsClient: custom_metrics.NewForConfig(
			clusterConfig,
			restmapper.NewDeferredDiscoveryRESTMapper(cacheddiscovery.NewMemCacheClient(discovery)),
			custom_metrics.NewAvailableAPIsGetter(discovery),
		),
	}
}

// RESTClient retrieves Kubernetes metrics through the Kubernetes REST API
type RESTClient struct {
	Client                metricsv1beta1.MetricsV1beta1Interface
	ExternalMetricsClient external_metrics.ExternalMetricsClient
	CustomMetricsClient   custom_metrics.CustomMetricsClient
}

// GetResourceMetric gets the given resource metric (and an associated oldest timestamp)
// for all pods matching the specified selector in the given namespace
func (c *RESTClient) GetResourceMetric(resource v1.ResourceName, namespace string, selector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
	metrics, err := c.Client.PodMetricses(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from resource metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from resource metrics API")
	}

	res := make(podmetrics.MetricsInfo, len(metrics.Items))
	for _, m := range metrics.Items {
		podSum := int64(0)
		missing := len(m.Containers) == 0
		for _, c := range m.Containers {
			resValue, found := c.Usage[resource]
			if !found {
				missing = true
				break
			}
			podSum += resValue.MilliValue()
		}
		if !missing {
			res[m.Name] = podmetrics.Metric{
				Timestamp: m.Timestamp.Time,
				Window:    m.Window.Duration,
				Value:     podSum,
			}
		}
	}

	timestamp := metrics.Items[0].Timestamp.Time

	return res, timestamp, nil
}

// GetRawMetric gets the given metric (and an associated oldest timestamp)
// for all pods matching the specified selector in the given namespace
func (c *RESTClient) GetRawMetric(metricName string, namespace string, selector labels.Selector, metricSelector labels.Selector) (podmetrics.MetricsInfo, time.Time, error) {
	metrics, err := c.CustomMetricsClient.NamespacedMetrics(namespace).GetForObjects(schema.GroupKind{Kind: "Pod"}, selector, metricName, metricSelector)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from custom metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from custom metrics API")
	}

	res := make(podmetrics.MetricsInfo, len(metrics.Items))
	for _, m := range metrics.Items {
		window := metricServerDefaultMetricWindow
		if m.WindowSeconds != nil {
			window = time.Duration(*m.WindowSeconds) * time.Second
		}
		res[m.DescribedObject.Name] = podmetrics.Metric{
			Timestamp: m.Timestamp.Time,
			Window:    window,
			Value:     int64(m.Value.MilliValue()),
		}

		m.Value.MilliValue()
	}

	timestamp := metrics.Items[0].Timestamp.Time

	return res, timestamp, nil
}

// GetObjectMetric gets the given metric (and an associated timestamp) for the given
// object in the given namespace
func (c *RESTClient) GetObjectMetric(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
	gvk := schema.FromAPIVersionAndKind(objectRef.APIVersion, objectRef.Kind)
	var metricValue *custommetricsv1.MetricValue
	var err error
	if gvk.Kind == "Namespace" && gvk.Group == "" {
		// handle namespace separately
		// NB: we ignore namespace name here, since CrossVersionObjectReference isn't
		// supposed to allow you to escape your namespace
		metricValue, err = c.CustomMetricsClient.RootScopedMetrics().GetForObject(gvk.GroupKind(), namespace, metricName, metricSelector)
	} else {
		metricValue, err = c.CustomMetricsClient.NamespacedMetrics(namespace).GetForObject(gvk.GroupKind(), objectRef.Name, metricName, metricSelector)
	}

	if err != nil {
		return 0, time.Time{}, fmt.Errorf("unable to fetch metrics from custom metrics API: %v", err)
	}

	return metricValue.Value.MilliValue(), metricValue.Timestamp.Time, nil
}

// GetExternalMetric gets all the values of a given external metric
// that match the specified selector.
func (c *RESTClient) GetExternalMetric(metricName, namespace string, selector labels.Selector) ([]int64, time.Time, error) {
	metrics, err := c.ExternalMetricsClient.NamespacedMetrics(namespace).List(metricName, selector)
	if err != nil {
		return []int64{}, time.Time{}, fmt.Errorf("unable to fetch metrics from external metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from external metrics API")
	}

	res := make([]int64, 0)
	for _, m := range metrics.Items {
		res = append(res, m.Value.MilliValue())
	}
	timestamp := metrics.Items[0].Timestamp.Time
	return res, timestamp, nil
}

// GetResourceUtilizationRatio takes in a set of metrics, a set of matching requests,
// and a target utilization percentage, and calculates the ratio of
// desired to actual utilization (returning that, the actual utilization, and the raw average value)
func GetResourceUtilizationRatio(metrics podmetrics.MetricsInfo, requests map[string]int64, targetUtilization int32) (utilizationRatio float64, currentUtilization int32, rawAverageValue int64, err error) {
	metricsTotal := int64(0)
	requestsTotal := int64(0)
	numEntries := 0

	for podName, metric := range metrics {
		request, hasRequest := requests[podName]
		if !hasRequest {
			// we check for missing requests elsewhere, so assuming missing requests == extraneous metrics
			continue
		}

		metricsTotal += metric.Value
		requestsTotal += request
		numEntries++
	}

	// if the set of requests is completely disjoint from the set of metrics,
	// then we could have an issue where the requests total is zero
	if requestsTotal == 0 {
		return 0, 0, 0, fmt.Errorf("no metrics returned matched known pods")
	}

	currentUtilization = int32((metricsTotal * 100) / requestsTotal)

	return float64(currentUtilization) / float64(targetUtilization), currentUtilization, metricsTotal / int64(numEntries), nil
}

// GetMetricUtilizationRatio takes in a set of metrics and a target utilization value,
// and calculates the ratio of desired to actual utilization
// (returning that and the actual utilization)
func GetMetricUtilizationRatio(metrics podmetrics.MetricsInfo, targetUtilization int64) (utilizationRatio float64, currentUtilization int64) {
	metricsTotal := int64(0)
	for _, metric := range metrics {
		metricsTotal += metric.Value
	}

	currentUtilization = metricsTotal / int64(len(metrics))

	return float64(currentUtilization) / float64(targetUtilization), currentUtilization
}
