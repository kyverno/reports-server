# Migration

This migration guide is for migrating an existing cluster using etcd to store policy reports to reports-server. 

You need to follow this if:
1. The cluster has kyverno already installed, or,
2. The cluster has policy reports crds already installed

Clusters with policy reports CRDs have existing API services for policy reports which need to be overwritten for reports-server to work. We do that by applying new api services with the label `kube-aggregator.kubernetes.io/automanaged: "false"`.

Follow the given methods to migrate to reports server on your existing cluster:

## Helm Chart

Helm cannot overwrite resources when they are not managed by helm. Thus we recommend installing the chart without the api services using the following command:
```bash
```bash
helm install reports-server --namespace reports-server --create-namespace --wait ./charts/reports-server \
        --set image.tag=latest \
        --set apiServices.enabled=false
```

Once the helm chart is installed, API services can be manually updated using `kubectl apply`. Update our [apiservices samples](./config/samples/apiservices.yaml) with the right reports-server name and namespace and apply that manifest.

# Cleaning up etcd

Switching to report-server does not delete existing reports in etcd. Reports-server will only recreate all the existing reports from etcd in its database. We recommend manually deleting keys from etcd using `etcdctl`. 

```bash
etcdctl del --prefix --key “/registry/reports.kyverno.io/”
```

```bash
etcdctl del --prefix --key “/registry/wgpolicyk8s.io/”
```