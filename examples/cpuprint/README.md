# CPU Print

This example shows how the library can be used to both gather metrics based on metric specs.

In this example a deployment called `php-apache` is created with 4 replicas that responds to simple HTTP requests
with an `OK!`. A separate single pod deployment is set up called `cpuprint` running a Docker image that will
report into the logs the CPU metrics retrieved.

## Usage

To follow the steps below and to see this example in action you need the following installed:

- [Docker](https://docs.docker.com/get-docker/)
- [Go v1.17+](https://go.dev/doc/install)
- [K3D v5.4+](https://k3d.io/v5.4.1/#installation)

After you have installed the above you can provision a development Kubernetes cluster by running:

```bash
k3d cluster create
```

### Steps

1. First build the binary and bundle it into the Docker image:

```bash
CGO_ENABLED=0 GOOS=linux go build -o dist/main && docker build -t cpuprint .
```

2. Next import the Docker image into the k3d cluster:

```bash
k3d image import cpuprint:latest
```

3. Then deploy the entire example by applying the deployment YAML:

```bash
kubectl apply -f deploy.yaml
```

4. Finally you can see the log output of the example container by running:

```bash
kubectl logs -l run=cpuprint -f
```

5. If you see some errors like this:

```
2022/05/08 22:26:09 invalid metrics (1 invalid out of 1), first error is: failed to get resource metric: unable to get metrics for resource cpu: no metrics returned from resource metrics API
```

Leave it for a minute or two to let the deployment being targeted (`php-apache`) to generate some CPU metrics with
the metrics server.

Eventually it should provide output like this:

```
2022/05/08 22:27:39 CPU statistics:
2022/05/08 22:27:39 Pod: php-apache-d4cf67d68-s9w2g, CPU usage: 1m (0.50% of requested)
2022/05/08 22:27:39 Pod: php-apache-d4cf67d68-v9fc2, CPU usage: 1m (0.50% of requested)
2022/05/08 22:27:39 Pod: php-apache-d4cf67d68-h4z4k, CPU usage: 1m (0.50% of requested)
2022/05/08 22:27:39 Pod: php-apache-d4cf67d68-jrskj, CPU usage: 1m (0.50% of requested)
2022/05/08 22:27:39 ----------
```

6. Try increasing the CPU load:

```bash
kubectl run -it --rm load-generator --image=busybox -- /bin/sh
```

Once it has loaded, run this command to increase CPU load:

```bash
while true; do wget -q -O- http://php-apache.default.svc.cluster.local; done
```
