/*
Copyright 2023 The K8sHorizMetrics Authors.

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

// Package k8shorizmetrics provides a simplified interface for gathering metrics and calculating replicas in the same
// way that the Horizontal Pod Autoscaler (HPA) does.
// This is split into two parts, gathering metrics, and evaluating metrics (calculating replicas).
// You can use these parts separately, or together to create a full evaluation process in the same way the HPA does.
package k8shorizmetrics
