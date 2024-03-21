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

package k8shorizmetrics_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/v2"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/v2/internal/testutil"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/external"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/object"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/podmetrics"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/pods"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/resource"
	"github.com/jthomperoo/k8shorizmetrics/v2/metrics/value"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sscale "k8s.io/client-go/scale"
)

func TestGatherSingleMetricWithOptions(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description                   string
		expected                      *metrics.Metric
		expectedErr                   error
		resource                      k8shorizmetrics.ResourceGatherer
		pods                          k8shorizmetrics.PodsGatherer
		object                        k8shorizmetrics.ObjectGatherer
		external                      k8shorizmetrics.ExternalGatherer
		scaleClient                   k8sscale.ScalesGetter
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
		spec                          autoscalingv2.MetricSpec
		namespace                     string
		podSelector                   labels.Selector
	}{
		{
			description:                   "Unknown metric type",
			expectedErr:                   errors.New(`unknown metric source type "unknown"`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.MetricSourceType("unknown"),
			},
			namespace: "test",
		},
		{
			description:                   "Object Metric: Fail convert metric selector",
			expectedErr:                   errors.New(`failed to get object metric: "invalid" is not a valid label selector operator`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Operator: "invalid",
								},
							},
						},
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "Object Metric: No target",
			expectedErr:                   errors.New(`invalid object metric source: must be either value or average value`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "Object Metric: Target not value or average value",
			expectedErr:                   errors.New(`invalid object metric source: must be either value or average value`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.UtilizationMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Object Metric: Fail to get value",
			expectedErr: errors.New(`failed to get object metric: test error`),
			object: &fake.ObjectGatherer{
				GatherReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, podSelector, metricSelector labels.Selector) (*object.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Object Metric: Fail to get average",
			expectedErr: errors.New(`failed to get object metric: test error`),
			object: &fake.ObjectGatherer{
				GatherPerPodReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (*object.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Object Metric: Value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Selector: metav1.SetAsLabelSelector(labels.Set{}),
						},
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.ValueMetricType,
						},
					},
				},
				Object: &object.Metric{
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(1),
					},
				},
			},
			object: &fake.ObjectGatherer{
				GatherReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, podSelector, metricSelector labels.Selector) (*object.Metric, error) {
					return &object.Metric{
						Current: value.MetricValue{
							Value: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Object Metric: Average value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Selector: metav1.SetAsLabelSelector(labels.Set{}),
						},
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				Object: &object.Metric{
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(1),
					},
				},
			},
			object: &fake.ObjectGatherer{
				GatherPerPodReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (*object.Metric, error) {
					return &object.Metric{
						Current: value.MetricValue{
							Value: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Object Metric: Average Value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Selector: metav1.SetAsLabelSelector(labels.Set{}),
						},
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				Object: &object.Metric{
					Current: value.MetricValue{
						AverageValue: testutil.Int64Ptr(1),
					},
				},
			},
			object: &fake.ObjectGatherer{
				GatherPerPodReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (*object.Metric, error) {
					return &object.Metric{
						Current: value.MetricValue{
							AverageValue: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "Pod Metric: Fail convert metric selector",
			expectedErr:                   errors.New(`failed to get pods metric: "invalid" is not a valid label selector operator`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.PodsMetricSourceType,
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Operator: "invalid",
								},
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "Pods Metric: Target not average value",
			expectedErr:                   errors.New(`invalid pods metric source: must be average value`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.PodsMetricSourceType,
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Pods Metric: Fail to get",
			expectedErr: errors.New(`failed to get pods metric: test error`),
			pods: &fake.PodsGatherer{
				GatherReactor: func(metricName, namespace string, podSelector, metricSelector labels.Selector) (*pods.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.PodsMetricSourceType,
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Pods Metric: Success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.PodsMetricSourceType,
					Pods: &autoscalingv2.PodsMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Selector: metav1.SetAsLabelSelector(labels.Set{}),
						},
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				Pods: &pods.Metric{
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
					TotalPods:     2,
					Timestamp:     time.Time{},
					PodMetricsInfo: podmetrics.MetricsInfo{
						"test": podmetrics.Metric{
							Value:     10,
							Timestamp: time.Time{},
						},
					},
				},
			},
			pods: &fake.PodsGatherer{
				GatherReactor: func(metricName, namespace string, podSelector, metricSelector labels.Selector) (*pods.Metric, error) {
					return &pods.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.PodsMetricSourceType,
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "Resource Metric: No target",
			expectedErr:                   errors.New(`invalid resource metric source: must be either average value or average utilization`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type:     autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{},
			},
			namespace: "test",
		},
		{
			description:                   "Resource Metric: Target not average value or average utilization",
			expectedErr:                   errors.New(`invalid resource metric source: must be either average value or average utilization`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Resource Metric: Fail to get average value",
			expectedErr: errors.New(`failed to get resource metric: test error`),
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Resource Metric: Fail to get average utilization",
			expectedErr: errors.New(`failed to get resource metric: test error`),
			resource: &fake.ResourceGatherer{
				GatherReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.UtilizationMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Resource Metric: Average value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				Resource: &resource.Metric{
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
					TotalPods:     2,
					Timestamp:     time.Time{},
					PodMetricsInfo: podmetrics.MetricsInfo{
						"test": podmetrics.Metric{
							Value:     10,
							Timestamp: time.Time{},
						},
					},
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Resource Metric: Average utilization success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.UtilizationMetricType,
						},
					},
				},
				Resource: &resource.Metric{
					ReadyPodCount: 2,
					IgnoredPods:   sets.String{},
					MissingPods:   sets.String{},
					TotalPods:     2,
					Timestamp:     time.Time{},
					PodMetricsInfo: podmetrics.MetricsInfo{
						"test": podmetrics.Metric{
							Value:     10,
							Timestamp: time.Time{},
						},
					},
				},
			},
			resource: &fake.ResourceGatherer{
				GatherReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.UtilizationMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description:                   "External Metric: No target",
			expectedErr:                   errors.New(`invalid external metric source: must be either value or average value`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type:     autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{},
			},
			namespace: "test",
		},
		{
			description:                   "External Metric: Target not value or average value",
			expectedErr:                   errors.New(`invalid external metric source: must be either value or average value`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.UtilizationMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "External Metric: Fail to get value",
			expectedErr: errors.New(`failed to get external metric: test error`),
			external: &fake.ExternalGatherer{
				GatherReactor: func(metricName, namespace string, metricSelector *metav1.LabelSelector, podSelector labels.Selector) (*external.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			scaleClient:                   nil,
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "External Metric: Fail to get average",
			expectedErr: errors.New(`failed to get external metric: test error`),
			external: &fake.ExternalGatherer{
				GatherPerPodReactor: func(metricName, namespace string, metricSelector *metav1.LabelSelector) (*external.Metric, error) {
					return nil, errors.New("test error")
				},
			},
			scaleClient:                   nil,
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "External Metric: Value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.ValueMetricType,
						},
					},
				},
				External: &external.Metric{
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(1),
					},
				},
			},
			external: &fake.ExternalGatherer{
				GatherReactor: func(metricName, namespace string, metricSelector *metav1.LabelSelector, podSelector labels.Selector) (*external.Metric, error) {
					return &external.Metric{
						Current: value.MetricValue{
							Value: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			scaleClient:                   nil,
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
		{
			description: "External Metric: Average value success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				External: &external.Metric{
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(1),
					},
				},
			},
			external: &fake.ExternalGatherer{
				GatherPerPodReactor: func(metricName, namespace string, metricSelector *metav1.LabelSelector) (*external.Metric, error) {
					return &external.Metric{
						Current: value.MetricValue{
							Value: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			scaleClient:                   nil,
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				},
			},
			namespace: "test",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &k8shorizmetrics.Gatherer{
				External:                      test.external,
				Object:                        test.object,
				Pods:                          test.pods,
				Resource:                      test.resource,
				ScaleClient:                   test.scaleClient,
				CPUInitializationPeriod:       test.cpuInitializationPeriod,
				DelayOfInitialReadinessStatus: test.delayOfInitialReadinessStatus,
			}
			metric, err := gatherer.GatherSingleMetricWithOptions(test.spec, test.namespace, test.podSelector, test.cpuInitializationPeriod, test.delayOfInitialReadinessStatus)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}
			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestGatherSingleMetric(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description                   string
		expected                      *metrics.Metric
		expectedErr                   error
		resource                      k8shorizmetrics.ResourceGatherer
		pods                          k8shorizmetrics.PodsGatherer
		object                        k8shorizmetrics.ObjectGatherer
		external                      k8shorizmetrics.ExternalGatherer
		scaleClient                   k8sscale.ScalesGetter
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
		spec                          autoscalingv2.MetricSpec
		namespace                     string
		podSelector                   labels.Selector
	}{
		{
			description:                   "Failure",
			expectedErr:                   errors.New(`failed to get object metric: "invalid" is not a valid label selector operator`),
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Operator: "invalid",
								},
							},
						},
					},
				},
			},
			namespace: "test",
		},
		{
			description: "Success",
			expected: &metrics.Metric{
				Spec: autoscalingv2.MetricSpec{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Selector: metav1.SetAsLabelSelector(labels.Set{}),
						},
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.ValueMetricType,
						},
					},
				},
				Object: &object.Metric{
					Current: value.MetricValue{
						Value: testutil.Int64Ptr(1),
					},
				},
			},
			object: &fake.ObjectGatherer{
				GatherReactor: func(metricName, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, podSelector, metricSelector labels.Selector) (*object.Metric, error) {
					return &object.Metric{
						Current: value.MetricValue{
							Value: testutil.Int64Ptr(1),
						},
					}, nil
				},
			},
			cpuInitializationPeriod:       0,
			delayOfInitialReadinessStatus: 0,
			spec: autoscalingv2.MetricSpec{
				Type: autoscalingv2.ObjectMetricSourceType,
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Selector: metav1.SetAsLabelSelector(labels.Set{}),
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.ValueMetricType,
					},
				},
			},
			namespace: "test",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &k8shorizmetrics.Gatherer{
				External:                      test.external,
				Object:                        test.object,
				Pods:                          test.pods,
				Resource:                      test.resource,
				ScaleClient:                   test.scaleClient,
				CPUInitializationPeriod:       test.cpuInitializationPeriod,
				DelayOfInitialReadinessStatus: test.delayOfInitialReadinessStatus,
			}
			metric, err := gatherer.GatherSingleMetric(test.spec, test.namespace, test.podSelector)
			if !cmp.Equal(&err, &test.expectedErr, equateErrorMessage) {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr, err, equateErrorMessage))
				return
			}
			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestGatherWithOptions(t *testing.T) {
	var tests = []struct {
		description                   string
		expected                      []*metrics.Metric
		expectedErr                   *k8shorizmetrics.GathererMultiMetricError
		resource                      k8shorizmetrics.ResourceGatherer
		pods                          k8shorizmetrics.PodsGatherer
		object                        k8shorizmetrics.ObjectGatherer
		external                      k8shorizmetrics.ExternalGatherer
		scaleClient                   k8sscale.ScalesGetter
		specs                         []autoscalingv2.MetricSpec
		namespace                     string
		podSelector                   labels.Selector
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
	}{
		{
			description: "Single spec fail to gather",
			expectedErr: &k8shorizmetrics.GathererMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New(`failed to get resource metric: test error`),
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return nil, errors.New(`test error`)
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "Two specs fail to gather both",
			expectedErr: &k8shorizmetrics.GathererMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New(`failed to get resource metric: test error`),
					errors.New(`failed to get resource metric: test error`),
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return nil, errors.New(`test error`)
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "Two specs fail to gather one, other successful",
			expected: []*metrics.Metric{
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "second",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
			},
			expectedErr: &k8shorizmetrics.GathererMultiMetricError{
				Partial: true,
				Errors: []error{
					errors.New(`failed to get resource metric: test error`),
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					if resourceName == "first" {
						return nil, errors.New(`test error`)
					}
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "first",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "second",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "No specs success",
			expected:    []*metrics.Metric{},
			specs:       []autoscalingv2.MetricSpec{},
		},
		{
			description: "One spec success",
			expected: []*metrics.Metric{
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "first",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "first",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "Two spec success",
			expected: []*metrics.Metric{
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "first",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"first": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "second",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"second": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							resourceName.String(): podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "first",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "second",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &k8shorizmetrics.Gatherer{
				External:                      test.external,
				Object:                        test.object,
				Pods:                          test.pods,
				Resource:                      test.resource,
				ScaleClient:                   test.scaleClient,
				CPUInitializationPeriod:       test.cpuInitializationPeriod,
				DelayOfInitialReadinessStatus: test.delayOfInitialReadinessStatus,
			}
			metric, err := gatherer.GatherWithOptions(test.specs, test.namespace, test.podSelector, test.cpuInitializationPeriod, test.delayOfInitialReadinessStatus)
			gatherErr := &k8shorizmetrics.GathererMultiMetricError{}

			if err == nil && test.expectedErr != nil {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), gatherErr.Error()))
				return
			}

			if err != nil {
				if errors.As(err, &gatherErr) {
					if !cmp.Equal(gatherErr.Partial, test.expectedErr.Partial) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(&test.expectedErr.Partial, gatherErr.Partial))
						return
					}

					if !cmp.Equal(gatherErr.Error(), test.expectedErr.Error()) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), gatherErr.Error()))
						return
					}
				} else {
					t.Error("unexpected error type returned, expected GathererMutliMetricError")
					return
				}
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}

func TestGather(t *testing.T) {
	var tests = []struct {
		description                   string
		expected                      []*metrics.Metric
		expectedErr                   *k8shorizmetrics.GathererMultiMetricError
		resource                      k8shorizmetrics.ResourceGatherer
		pods                          k8shorizmetrics.PodsGatherer
		object                        k8shorizmetrics.ObjectGatherer
		external                      k8shorizmetrics.ExternalGatherer
		scaleClient                   k8sscale.ScalesGetter
		specs                         []autoscalingv2.MetricSpec
		namespace                     string
		podSelector                   labels.Selector
		cpuInitializationPeriod       time.Duration
		delayOfInitialReadinessStatus time.Duration
	}{
		{
			description: "Full failure",
			expectedErr: &k8shorizmetrics.GathererMultiMetricError{
				Partial: false,
				Errors: []error{
					errors.New(`failed to get resource metric: test error`),
					errors.New(`failed to get resource metric: test error`),
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return nil, errors.New(`test error`)
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "Partial failure",
			expected: []*metrics.Metric{
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "second",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
			},
			expectedErr: &k8shorizmetrics.GathererMultiMetricError{
				Partial: true,
				Errors: []error{
					errors.New(`failed to get resource metric: test error`),
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					if resourceName == "first" {
						return nil, errors.New(`test error`)
					}
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "first",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "second",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
		{
			description: "Success",
			expected: []*metrics.Metric{
				{
					Spec: autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "first",
							Target: autoscalingv2.MetricTarget{
								Type: autoscalingv2.AverageValueMetricType,
							},
						},
					},
					Resource: &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					},
				},
			},
			resource: &fake.ResourceGatherer{
				GatherRawReactor: func(resourceName corev1.ResourceName, namespace string, podSelector labels.Selector, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (*resource.Metric, error) {
					return &resource.Metric{
						ReadyPodCount: 2,
						IgnoredPods:   sets.String{},
						MissingPods:   sets.String{},
						TotalPods:     2,
						Timestamp:     time.Time{},
						PodMetricsInfo: podmetrics.MetricsInfo{
							"test": podmetrics.Metric{
								Value:     10,
								Timestamp: time.Time{},
							},
						},
					}, nil
				},
			},
			specs: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "first",
						Target: autoscalingv2.MetricTarget{
							Type: autoscalingv2.AverageValueMetricType,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &k8shorizmetrics.Gatherer{
				External:                      test.external,
				Object:                        test.object,
				Pods:                          test.pods,
				Resource:                      test.resource,
				ScaleClient:                   test.scaleClient,
				CPUInitializationPeriod:       test.cpuInitializationPeriod,
				DelayOfInitialReadinessStatus: test.delayOfInitialReadinessStatus,
			}
			metric, err := gatherer.Gather(test.specs, test.namespace, test.podSelector)
			gatherErr := &k8shorizmetrics.GathererMultiMetricError{}

			if err == nil && test.expectedErr != nil {
				t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), gatherErr.Error()))
				return
			}

			if err != nil {
				if errors.As(err, &gatherErr) {
					if !cmp.Equal(gatherErr.Partial, test.expectedErr.Partial) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(&test.expectedErr.Partial, gatherErr.Partial))
						return
					}

					if !cmp.Equal(gatherErr.Error(), test.expectedErr.Error()) {
						t.Errorf("error mismatch (-want +got):\n%s", cmp.Diff(test.expectedErr.Error(), gatherErr.Error()))
						return
					}
				} else {
					t.Error("unexpected error type returned, expected GathererMutliMetricError")
					return
				}
			}

			if !cmp.Equal(test.expected, metric) {
				t.Errorf("metrics mismatch (-want +got):\n%s", cmp.Diff(test.expected, metric))
			}
		})
	}
}
