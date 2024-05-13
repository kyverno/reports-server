# Installation

Reports-server has multiple methods for installation: YAML manifest and Helm Chart.YAML manifest is the recommended method to install reports server when kyverno or policy reports CRDs are already installed in the cluster. Helm chart is the most flexible method as it offers a wide range of configurations.

If kyverno is already installed in the cluster, follow the [migration guide](#migration).

Reports-server comes with a postgreSQL database. It is recommended to bring-your-own postgres database to have finer control of database configurations ([see database configuration guide](#database-configuration)).

### YAML Manifest
It is recommended to install Reports-server using `kubectl apply`, especially when policy reports CRDs are already installed in the cluster ([see migration guide](#migration)). To install reports server using YAML manifest, cretae a `reports-server` namespace and run the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/kyverno/reports-server/main/config/install.yaml
```

### Helm Chart

Reports-server can be deployed via a Helm chart for a production install–which is accessible either through the reports-server repository.

In order to install reports-server with Helm, first add the Reports-server Helm repository:
```bash
helm repo add reports-server https://kyverno.github.io/reports-server
```

Scan the new repository for charts:
```bash
helm repo update
```

Optionally, show all available chart versions for reports-server.

```bash
helm search repo reports-server --l
```

Create a namespace and install the reports-server chart:

```bash
helm install reports-server -n reports-server reports-server/reports-server --create-namespace
```

To install pre-releases, add the --devel switch to Helm:

```bash
helm install reports-server -n reports-server reports-server/reports-server --create-namespace --devel
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
