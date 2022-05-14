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

package object_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jthomperoo/k8shorizmetrics/internal/fake"
	"github.com/jthomperoo/k8shorizmetrics/internal/object"
	"github.com/jthomperoo/k8shorizmetrics/internal/podutil"
	"github.com/jthomperoo/k8shorizmetrics/internal/testutil"
	objectmetric "github.com/jthomperoo/k8shorizmetrics/metrics/object"
	"github.com/jthomperoo/k8shorizmetrics/metrics/value"
	metricsclient "github.com/jthomperoo/k8shorizmetrics/metricsclient"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGather(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description     string
		expected        *objectmetric.Metric
		expectedErr     error
		metricsclient   metricsclient.Client
		podReadyCounter podutil.PodReadyCounter
		metricName      string
		namespace       string
		objectRef       *autoscalingv2.CrossVersionObjectReference
		selector        labels.Selector
		metricSelector  labels.Selector
	}{
		{
			"Fail to get metric",
			nil,
			errors.New("unable to get metric test-metric:  on test-namespace : fail to get metric"),
			&fake.MetricsClient{
				GetObjectMetricReactor: func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
					return 0, time.Time{}, errors.New("fail to get metric")
				},
			},
			nil,
			"test-metric",
			"test-namespace",
			&autoscalingv2.CrossVersionObjectReference{},
			nil,
			nil,
		},
		{
			"Fail to get ready pods",
			nil,
			errors.New("unable to calculate ready pods: fail to get ready pods"),
			&fake.MetricsClient{
				GetObjectMetricReactor: func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
					return 0, time.Time{}, nil
				},
			},
			&fake.PodReadyCounter{
				GetReadyPodsCountReactor: func(namespace string, selector labels.Selector) (int64, error) {
					return 0, errors.New("fail to get ready pods")
				},
			},
			"test-metric",
			"test-namespace",
			&autoscalingv2.CrossVersionObjectReference{},
			nil,
			nil,
		},
		{
			"Success",
			&objectmetric.Metric{
				Current: value.MetricValue{
					Value: testutil.Int64Ptr(5),
				},
				ReadyPodCount: testutil.Int64Ptr(2),
			},
			nil,
			&fake.MetricsClient{
				GetObjectMetricReactor: func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
					return 5, time.Time{}, nil
				},
			},
			&fake.PodReadyCounter{
				GetReadyPodsCountReactor: func(namespace string, selector labels.Selector) (int64, error) {
					return 2, nil
				},
			},
			"test-metric",
			"test-namespace",
			&autoscalingv2.CrossVersionObjectReference{},
			nil,
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &object.Gather{
				MetricsClient:   test.metricsclient,
				PodReadyCounter: test.podReadyCounter,
			}
			metric, err := gatherer.Gather(test.metricName, test.namespace, test.objectRef, test.selector, test.metricSelector)
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

func TestGatherPerPod(t *testing.T) {
	equateErrorMessage := cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	var tests = []struct {
		description     string
		expected        *objectmetric.Metric
		expectedErr     error
		metricsclient   metricsclient.Client
		podReadyCounter podutil.PodReadyCounter
		metricName      string
		namespace       string
		objectRef       *autoscalingv2.CrossVersionObjectReference
		metricSelector  labels.Selector
	}{
		{
			"Fail to get metric",
			nil,
			errors.New("unable to get metric test-metric:  on test-namespace /fail to get metric"),
			&fake.MetricsClient{
				GetObjectMetricReactor: func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
					return 0, time.Time{}, errors.New("fail to get metric")
				},
			},
			nil,
			"test-metric",
			"test-namespace",
			&autoscalingv2.CrossVersionObjectReference{},
			nil,
		},
		{
			"Success",
			&objectmetric.Metric{
				Current: value.MetricValue{
					AverageValue: testutil.Int64Ptr(5),
				},
			},
			nil,
			&fake.MetricsClient{
				GetObjectMetricReactor: func(metricName string, namespace string, objectRef *autoscalingv2.CrossVersionObjectReference, metricSelector labels.Selector) (int64, time.Time, error) {
					return 5, time.Time{}, nil
				},
			},
			nil,
			"test-metric",
			"test-namespace",
			&autoscalingv2.CrossVersionObjectReference{},
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			gatherer := &object.Gather{
				MetricsClient:   test.metricsclient,
				PodReadyCounter: test.podReadyCounter,
			}
			metric, err := gatherer.GatherPerPod(test.metricName, test.namespace, test.objectRef, test.metricSelector)
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
