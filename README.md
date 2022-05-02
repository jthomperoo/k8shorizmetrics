[![Build](https://github.com/jthomperoo/k8shorizmetrics/workflows/main/badge.svg)](https://github.com/jthomperoo/k8shorizmetrics/actions)
[![go.dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/jthomperoo/k8shorizmetrics)
[![Go Report
Card](https://goreportcard.com/badge/github.com/jthomperoo/k8shorizmetrics)](https://goreportcard.com/report/github.com/jthomperoo/k8shorizmetrics)
[![Documentation
Status](https://readthedocs.org/projects/k8shorizmetrics/badge/?version=latest)](https://k8shorizmetrics.readthedocs.io/en/latest)
[![License](https://img.shields.io/:license-apache-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

# k8shorizmetrics

`k8shorizmetrics` is a library that provides the internal workings of the Kubernetes Horizontal Pod Autoscaler (HPA)
wrapped up in a simple API. The project allows querying metrics just as the HPA does, and also running the calculations
the HPA does

## Install

```bash
go get -u github.com/jthomperoo/k8shorizmetrics
```

## Features

- Simple API, based directly on the code from the HPA, but detangled for ease of use.
- Dependent only on versioned and public Kubernetes Golang modules, allows easy install without replace directives.
- Splits the HPA into two parts, metric gathering and evaluation, only use what you need.
- Allows insights into how the HPA makes decisions.

## Examples

See the [examples directory](./examples/) for some examples, [cpuprint](./examples/cpuprint/) is a good start.

## Developing and Contributing

See the [contribution guidelines](CONTRIBUTING.md) and [code of conduct](CODE_OF_CONDUCT.md).
