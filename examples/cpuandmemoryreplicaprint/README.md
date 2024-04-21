# CPU Replica Print

This example shows how the library can be used to both gather metrics based on metric specs, and then calculate the
replica count that the Horizontal Pod Autoscaler (HPA) would target based on those metrics.

This example targets two resource metrics, CPU and memory.

In this example a deployment called `php-apache` is created with 4 replicas that responds to simple HTTP requests
with an `OK!`. The example will query the CPU and memory metrics for the pods in this deployment, along with the number
of replicas the HPA would target based on those metrics and print them to stdout.

> Note this example uses out of cluster configuration of the Kubernetes client, if you want to run this inside the
> cluster you should use in cluster configuration.

## Usage

To follow the steps below and to see this example in action you need the following installed:

- [Go v1.22+](https://go.dev/doc/install)
- [K3D v5.6+](https://k3d.io/v5.6.0/#installation)

After you have installed the above you can provision a development Kubernetes cluster by running:

```bash
k3d cluster create
```

### Steps

Run `go get` to make sure you have all of the dependencies for running the application installed.

1. First create the deployment to monitor by applying the deployment YAML:

```bash
kubectl apply -f deploy.yaml
```

2. Run the example using:

```bash
go run main.go
```

3. If you see some errors like this:

```
2022/05/08 22:26:09 invalid metrics (1 invalid out of 1), first error is: failed to get resource metric: unable to get metrics for resource cpu: no metrics returned from resource metrics API
```

Leave it for a minute or two to let the deployment being targeted (`php-apache`) to generate some CPU metrics with
the metrics server.

Eventually it should provide output like this:

```
2024/03/17 23:30:03 Pod Metrics:
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-6rf4f, cpu usage: 1 (0.50% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-7ncxb, cpu usage: 1 (0.50% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-f27b4, cpu usage: 1 (0.50% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-tb8rn, cpu usage: 1 (0.50% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-tb8rn, memory usage: 33853440000 (50.45% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-6rf4f, memory usage: 32382976000 (48.25% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-7ncxb, memory usage: 32739328000 (48.79% of requested)
2024/03/17 23:30:03 Pod: php-apache-7cb7bd96b4-f27b4, memory usage: 32493568000 (48.42% of requested)
2024/03/17 23:30:03 The Horizontal Pod Autoscaler would stay at 4 replicas
```

4. Try increasing the CPU load:

```bash
kubectl run -it --rm load-generator --image=busybox -- /bin/sh
```

Once it has loaded, run this command to increase CPU load:

```bash
while true; do wget -q -O- http://php-apache.default.svc.cluster.local; done
```
