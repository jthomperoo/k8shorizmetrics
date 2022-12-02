[![Build](https://github.com/jthomperoo/k8shorizmetrics/workflows/main/badge.svg)](https://github.com/jthomperoo/k8shorizmetrics/actions)
[![go.dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/jthomperoo/k8shorizmetrics)
[![Go Report
Card](https://goreportcard.com/badge/github.com/jthomperoo/k8shorizmetrics)](https://goreportcard.com/report/github.com/jthomperoo/k8shorizmetrics)
[![License](https://img.shields.io/:license-apache-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

# k8shorizmetrics

`k8shorizmetrics` is a library that provides the internal workings of the Kubernetes Horizontal Pod Autoscaler (HPA)
wrapped up in a simple API. The project allows querying metrics just as the HPA does, and also running the calculations
to work out the target replica count that the HPA does.

## Install

```bash
go get -u github.com/jthomperoo/k8shorizmetrics
```

## Features

- Simple API, based directly on the code from the HPA, but detangled for ease of use.
- Dependent only on versioned and public Kubernetes Golang modules, allows easy install without replace directives.
- Splits the HPA into two parts, metric gathering and evaluation, only use what you need.
- Allows insights into how the HPA makes decisions.
- Supports scaling to and from 0.

## Quick Start

The following is a simple program that can run inside a Kubernetes cluster that gets the CPU resource metrics for
pods with the label `run: php-apache`.

```go
package main

import (
	"log"
	"time"

	"github.com/jthomperoo/k8shorizmetrics"
	"github.com/jthomperoo/k8shorizmetrics/metricsclient"
	"github.com/jthomperoo/k8shorizmetrics/podsclient"
	"k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// Kubernetes API setup
	clusterConfig, _ := rest.InClusterConfig()
	clientset, _ := kubernetes.NewForConfig(clusterConfig)
	// Metrics and pods clients setup
	metricsclient := metricsclient.NewClient(clusterConfig, clientset.Discovery())
	podsclient := &podsclient.OnDemandPodLister{Clientset: clientset}
	// HPA configuration options
	cpuInitializationPeriod := time.Duration(300) * time.Second
	initialReadinessDelay := time.Duration(30) * time.Second

	// Setup gatherer
	gather := k8shorizmetrics.NewGatherer(metricsclient, podsclient, cpuInitializationPeriod, initialReadinessDelay)

	// Target resource values
	namespace := "default"
	podSelector := labels.SelectorFromSet(labels.Set{
		"run": "php-apache",
	})

	// Metric spec to gather, CPU resource utilization
	spec := v2beta2.MetricSpec{
		Type: v2beta2.ResourceMetricSourceType,
		Resource: &v2beta2.ResourceMetricSource{
			Name: corev1.ResourceCPU,
			Target: v2beta2.MetricTarget{
				Type: v2beta2.UtilizationMetricType,
			},
		},
	}

	metric, _ := gather.GatherSingleMetric(spec, namespace, podSelector)

	for pod, podmetric := range metric.Resource.PodMetricsInfo {
		actualCPU := podmetric.Value
		requestedCPU := metric.Resource.Requests[pod]
		log.Printf("Pod: %s, CPU usage: %dm (%0.2f%% of requested)\n", pod, actualCPU, float64(actualCPU)/float64(requestedCPU)*100.0)
	}
}
```

## Documentation

See the [Go doc](https://pkg.go.dev/github.com/jthomperoo/k8shorizmetrics).

## Migration from v1 to v2

There are two changes you need to make to migrate from `v1` to `v2`:

1. Switch from using `k8s.io/api/autoscaling/v2beta2` to `k8s.io/api/autoscaling/v2`.
2. Switch from using `github.com/jthomperoo/k8shorizmetrics` to `github.com/jthomperoo/k8shorizmetrics/v2`.

## Examples

See the [examples directory](./examples/) for some examples, [cpuprint](./examples/cpuprint/) is a good start.

## Developing and Contributing

See the [contribution guidelines](CONTRIBUTING.md) and [code of conduct](CODE_OF_CONDUCT.md).
