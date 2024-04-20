# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- **BREAKING CHANGE** Types now use JSON tags which match Kubernetes convention, with naming using camel case rather
than snake case. For example the Resource Metric field `PodMetricsInfo` is now serialized as `podMetricsInfo` rather
than `pod_metrics_info`.

## [v3.0.0] - 2024-03-21
### Changed
- **BREAKING CHANGE** Changed module path from `github.com/jthomperoo/k8shorizmetrics/v2` to
`github.com/jthomperoo/k8shorizmetrics/v3`.
- **BREAKING CHANGE** Gather now returns the `GathererMultiMetricError` error type if any of the metrics fail to
gather. This error is returned for partial errors, meaning some metrics gathered successfully and others did not.
If this partial error occurs the `GathererMultiMetricError` error will have the `Partial` property set to `true`. This
can be checked for using `errors.As`.
- **BREAKING CHANGE** Evaluate now returns the `EvaluatorMultiMetricError` error type if any of the metrics fail to
evaluate. This error is returned for partial errors, meaning some metrics evaluted successfully and others did not.
If this partial error occurs the `EvaluatorMultiMetricError` error will have the `Partial` property set to `true`. This
can be checked for using `errors.As`.

## [v2.0.2] - 2023-12-23
### Changed
- Upgraded to Go `v1.21`.
- Upgraded package dependencies.

## [v2.0.1] - 2023-03-07
### Changed
- Upgraded to Go `v1.20`.
- Upgraded package dependencies.

## [v2.0.0] - 2022-12-02
### Changed
- **BREAKING CHANGE** Upgraded from Kubernetes `autoscaling/v2beta2` to the graduated `autoscaling/v2`. This drops
support for Kubernetes versions `v1.22` and below.
- **BREAKING CHANGE** Changed module path from `github.com/jthomperoo/k8shorizmetrics` to
`github.com/jthomperoo/k8shorizmetrics/v2`.
- Upgraded to Kubernetes client libaries `v0.25.4` to support Kubernetes `v1.23+`.
- Upgraded to Go `v1.19`.


## [v1.1.0] - 2022-12-02
### Added
- New `GatherWithOptions` and `GatherSingleMetricWithOptions` methods which allow you to provide the CPU initialization
and delay of initial readiness status at call time as parameters.
- New `EvaluateWithOptions` and `EvaluateSingleMetricWithOptions` methods which allow you to provide the tolerance at
call time as a parameter.
- `CPUInitializationPeriod` and `DelayOfInitialReadinessStatus` now exposed as member variables of the `Gatherer`.
- `Tolerance` now exposed as a member variable of the `Evaluator`.

## [v1.0.0] - 2022-05-14
### Added
- Simple API, based directly on the code from the HPA, but detangled for ease of use.
- Dependent only on versioned and public Kubernetes Golang modules, allows easy install without replace directives.
- Splits the HPA into two parts, metric gathering and evaluation, only use what you need.
- Allows insights into how the HPA makes decisions.
- Supports scaling to and from 0.

[Unreleased]: https://github.com/jthomperoo/k8shorizmetrics/compare/v3.0.0...HEAD
[v3.0.0]: https://github.com/jthomperoo/k8shorizmetrics/compare/v2.0.2...v3.0.0
[v2.0.2]: https://github.com/jthomperoo/k8shorizmetrics/compare/v2.0.1...v2.0.2
[v2.0.1]: https://github.com/jthomperoo/k8shorizmetrics/compare/v2.0.0...v2.0.1
[v2.0.0]: https://github.com/jthomperoo/k8shorizmetrics/compare/v1.1.0...v2.0.0
[v1.1.0]: https://github.com/jthomperoo/k8shorizmetrics/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/jthomperoo/k8shorizmetrics/releases/tag/v1.0.0
