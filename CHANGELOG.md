# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
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

[Unreleased]: https://github.com/jthomperoo/k8shorizmetrics/compare/v1.0.0...HEAD
[v1.0.0]: https://github.com/jthomperoo/k8shorizmetrics/releases/tag/v1.0.0
