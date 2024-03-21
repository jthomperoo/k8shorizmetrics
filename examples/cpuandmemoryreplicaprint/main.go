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

package main

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/jthomperoo/k8shorizmetrics/v2"
	"github.com/jthomperoo/k8shorizmetrics/v2/metricsclient"
	"github.com/jthomperoo/k8shorizmetrics/v2/podsclient"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	cpuInitializationPeriodSeconds = 300
	initialReadinessDelaySeconds   = 30
	tolerance                      = 0.1
	namespace                      = "default"
	deploymentName                 = "php-apache"
)

var targetAverageUtilization int32 = 50

var podMatchSelector = labels.SelectorFromSet(labels.Set{
	"run": "php-apache",
})

func main() {
	clusterConfig, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		log.Fatalf("Fail to create out-of-cluster Kubernetes config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalf("Fail to set up Kubernetes clientset: %s", err)
	}

	metricsclient := metricsclient.NewClient(clusterConfig, clientset.Discovery())
	podsclient := &podsclient.OnDemandPodLister{
		Clientset: clientset,
	}
	cpuInitializationPeriod := time.Duration(cpuInitializationPeriodSeconds) * time.Second
	initialReadinessDelay := time.Duration(initialReadinessDelaySeconds) * time.Second

	// Set up the metric gatherer, needs to be able to query metrics and pods with the clients provided, along with
	// config options
	gather := k8shorizmetrics.NewGatherer(metricsclient, podsclient, cpuInitializationPeriod, initialReadinessDelay)
	// Set up the evaluator, only needs to know the tolerance configuration value for determining replica counts
	evaluator := k8shorizmetrics.NewEvaluator(tolerance)

	// This is the CPU metric spec, this targets the CPU resource metric, gathering utilization values and targeting
	// an average utilization of 50%
	// Equivalent to the following YAML:
	// metrics:
	// - type: Resource
	//   resource:
	// 	   name: cpu
	// 	   target:
	// 	     type: Utilization
	// 	     averageUtilization: 50
	cpuSpec := v2.MetricSpec{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name: corev1.ResourceCPU,
			Target: v2.MetricTarget{
				Type:               v2.UtilizationMetricType,
				AverageUtilization: &targetAverageUtilization,
			},
		},
	}

	// This is the memory metric spec, this targets the memory resource metric, gathering utilization values and
	// targeting an average utilization of 50%
	// Equivalent to the following YAML:
	// metrics:
	// - type: Resource
	//   resource:
	// 	   name: memory
	// 	   target:
	// 	     type: Utilization
	// 	     averageUtilization: 50
	memorySpec := v2.MetricSpec{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name: corev1.ResourceMemory,
			Target: v2.MetricTarget{
				Type:               v2.UtilizationMetricType,
				AverageUtilization: &targetAverageUtilization,
			},
		},
	}

	// Loop infinitely, wait 5 seconds between each loop
	for {
		time.Sleep(5 * time.Second)

		specs := []v2.MetricSpec{cpuSpec, memorySpec}

		// Gather the metrics using the specs, targeting the namespace and pod selector defined above
		metrics, err := gather.Gather(specs, namespace, podMatchSelector)
		if err != nil {
			var gatherError *k8shorizmetrics.GathererMultiMetricError
			if errors.As(err, &gatherError) {
				for i, partialErr := range gatherError.Errors {
					log.Printf("partial error %d/%d: %s", i+1, len(gatherError.Errors), partialErr)
				}

				if !gatherError.Partial {
					continue
				}

				log.Printf("%d/%d metrics were gathered successfully, continuing to evaluate with available metrics", len(metrics), len(specs))
			} else {
				log.Println(err)
				continue
			}
		}

		log.Println("Pod Metrics:")

		for _, metric := range metrics {
			for pod, podmetric := range metric.Resource.PodMetricsInfo {
				actual := podmetric.Value
				requested := metric.Resource.Requests[pod]
				log.Printf("Pod: %s, %s usage: %d (%0.2f%% of requested)\n", pod, &metric.Spec.Resource.Name, actual, float64(actual)/float64(requested)*100.0)
			}
		}

		// To find out the current replica count we can use the Kubernetes client-go client to get the scale sub
		// resource of the deployment which contains the current replica count
		scale, err := clientset.AppsV1().Deployments(namespace).GetScale(context.Background(), deploymentName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Failed to get scale resource for deployment '%s', err: %v", deploymentName, err)
			continue
		}

		currentReplicaCount := scale.Spec.Replicas

		// Calculate the target number of replicas that the HPA would scale to based on the metrics provided and current
		// replicas
		targetReplicaCount, err := evaluator.Evaluate(metrics, scale.Spec.Replicas)
		if err != nil {
			var evaluatorError *k8shorizmetrics.EvaluatorMultiMetricError
			if errors.As(err, &evaluatorError) {
				for i, partialErr := range evaluatorError.Errors {
					log.Printf("partial error %d/%d: %s", i+1, len(evaluatorError.Errors), partialErr)
				}

				if !evaluatorError.Partial {
					continue
				}

				log.Printf("%d/%d metrics were evaluated successfully, printing the expected evaluation using the available metrics", len(metrics), len(specs))
			} else {
				log.Println(err)
				continue
			}
		}

		if targetReplicaCount == currentReplicaCount {
			log.Printf("The Horizontal Pod Autoscaler would stay at %d replicas", targetReplicaCount)
		} else {
			log.Printf("The Horizontal Pod Autoscaler would scale from %d to %d replicas", currentReplicaCount, targetReplicaCount)
		}

		log.Println("----------")
	}
}
