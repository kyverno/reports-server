# Installation

Reports-server has multiple methods for installation: YAML manifest and Helm Chart.YAML manifest is the recommended method to install reports server when kyverno or policy reports CRDs are already installed in the cluster. Helm chart is the most flexible method as it offers a wide range of configurations.

If kyverno is already installed in the cluster, follow the [migration guide](#migration).

There are three configuration to install reports server:
1. Reports server with managed postgres: Use a centralised postgres database outside of the cluster
2. Reports server with incluster postgres: Create a postgres instance in the cluster
3. Reports server with inmemory reports store: Store reports in the memory of reports server pod

## With Managed Postgres

Reports server can be configured to work with any postgres instance in an out of the cluster. You can install reports server with a postgres instance outside of the cluster with helm as follows.

<!-- In order to install reports-server with Helm, first add the Reports-server Helm repository: -->
<!-- ```bash -->
<!-- helm repo add reports-server https://kyverno.github.io/reports-server -->
<!-- ``` -->
<!---->
<!-- Scan the new repository for charts: -->
<!-- ```bash -->
<!-- helm repo update -->
<!-- ``` -->
<!---->
<!-- Optionally, show all available chart versions for reports-server. -->
<!---->
<!-- ```bash -->
<!-- helm search repo reports-server --l -->
<!-- ``` -->
Get the values for hostname, dbname, postgres username and postgres password from managed postgres and fill the values in helm values

Install the reports-server chart:

```bash
helm install reports-server -n reports-server --create-namespace --wait ./charts/reports-server/ \
        --set image.tag=latest \
        --set postgresql.enabled=false \
        --set config.db.host=<HOST_NAME> \
        --set config.db.name=<DB_NAME> \
        --set config.db.user=<POSTGRES_USERNAME> \
        --set config.db.password=<POSTGRES_PASSWORD> \
        --set config.db.sslmode=<SSL_MODE>
```

## With Incluster database

Reports server default install creates a postgres instance by default, but for production, it is recommended to use an postgres operator such as [CloudNativePG](https://cloudnative-pg.io/). Reports-server can be installed along side CloudNativePG as follows:

Create a namespace for reports-server:
```bash
kubectl create ns reports-server
```

Install CloudNativePG using one of their recommended installation methods:
```bash
kubectl apply -f \
  https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.18/releases/cnpg-1.18.5.yaml
```

Wait for cloud native pg controller to start:

```bash
kubectl wait pod --all --for=condition=Ready --namespace=cnpg-system
```

Create a CloudNativePG postgres cluster:
```bash
kubectl create -f config/samples/cnpg-cluster.yaml
```

<!-- In order to install reports-server with Helm, first add the Reports-server Helm repository: -->
<!-- ```bash -->
<!-- helm repo add reports-server https://kyverno.github.io/reports-server -->
<!-- ``` -->
<!---->
<!-- Scan the new repository for charts: -->
<!-- ```bash -->
<!-- helm repo update -->
<!-- ``` -->
<!---->
<!-- Optionally, show all available chart versions for reports-server. -->
<!---->
<!-- ```bash -->
<!-- helm search repo reports-server --l -->
<!-- ``` -->
Install the reports-server chart:

```bash
helm install reports-server -n reports-server --create-namespace --wait ./charts/reports-server \
        --set image.tag=latest \
        --set postgresql.enabled=false \
        --set config.db.host=reports-server-cluster-rw.reports-server \
        --set config.db.name=reportsdb \
        --set config.db.user=$(kubectl get secret -n reports-server reports-server-cluster-app --template={{.data.username}} | base64 -d) \
        --set config.db.password=$(kubectl get secret -n reports-server reports-server-cluster-app --template={{.data.password}} | base64 -d)
```

## With inmemory storage
Reports server can be installed without any database as well. In this case, reports will be stored in the memory of reports-server pod. You can install reports-server with inmemory configuration as follows:

<!-- In order to install reports-server with Helm, first add the Reports-server Helm repository: -->
<!-- ```bash -->
<!-- helm repo add reports-server https://kyverno.github.io/reports-server -->
<!-- ``` -->
<!---->
<!-- Scan the new repository for charts: -->
<!-- ```bash -->
<!-- helm repo update -->
<!-- ``` -->
<!---->
<!-- Optionally, show all available chart versions for reports-server. -->
<!---->
<!-- ```bash -->
<!-- helm search repo reports-server --l -->
<!-- ``` -->

Install the reports-server chart:

```bash
helm install reports-server --namespace reports-server --create-namespace --wait ./charts/reports-server \
        --set image.tag=latest \
        --set config.debug=true \
        --set postgresql.enabled=false
```
### Testing

To install Reports-server on a kind cluster for testing, run the following commands:

Create a local kind cluster
```bash
make kind-create
```

Build docker images, load images in kind cluster, and deploy helm chart
```bash
make kind-install
```

## Migration 

See [MIGRATION.md](./MIGRATION.md)


## Database Configuration

See [DBCONFIG.md](./DBCONFIG.md)
