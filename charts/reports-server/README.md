# reports-server

![Version: 0.1.1](https://img.shields.io/badge/Version-0.1.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.1.1](https://img.shields.io/badge/AppVersion-v0.1.1-informational?style=flat-square)

TODO

## Installing the Chart

Add `reports-server` Helm repository:

```shell
helm repo add reports-server https://kyverno.github.io/reports-server/
```

Install `reports-server` Helm chart:

```shell
helm install reports-server --namespace reports-server --create-namespace reports-server/reports-server
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| postgresql.enabled | bool | `true` | Deploy postgresql dependency chart |
| postgresql.auth.postgresPassword | string | `"reports"` |  |
| postgresql.auth.database | string | `"reportsdb"` |  |
| nameOverride | string | `""` | Name override |
| fullnameOverride | string | `""` | Full name override |
| replicaCount | int | `1` | Number of pod replicas |
| image.registry | string | `"ghcr.io"` | Image registry |
| image.repository | string | `"kyverno/reports-server"` | Image repository |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.tag | string | `nil` | Image tag (will default to app version if not set) |
| imagePullSecrets | list | `[]` | Image pull secrets |
| priorityClassName | string | `"system-cluster-critical"` | Priority class name |
| serviceAccount.create | bool | `true` | Create service account |
| serviceAccount.annotations | object | `{}` | Service account annotations |
| serviceAccount.name | string | `""` | Service account name (required if `serviceAccount.create` is `false`) |
| podAnnotations | object | `{}` | Pod annotations |
| podSecurityContext | object | `{"fsGroup":2000}` | Pod security context |
| securityContext | object | See [values.yaml](values.yaml) | Container security context |
| livenessProbe | object | `{"failureThreshold":10,"httpGet":{"path":"/livez","port":"https","scheme":"HTTPS"},"initialDelaySeconds":20,"periodSeconds":10}` | Liveness probe |
| readinessProbe | object | `{"failureThreshold":10,"httpGet":{"path":"/readyz","port":"https","scheme":"HTTPS"},"initialDelaySeconds":30,"periodSeconds":10}` | Readiness probe |
| metrics.enabled | bool | `true` | Enable prometheus metrics |
| metrics.serviceMonitor.enabled | bool | `false` | Enable service monitor for scraping prometheus metrics |
| metrics.serviceMonitor.additionalLabels | object | `{}` | Service monitor additional labels |
| metrics.serviceMonitor.interval | string | `""` | Service monitor scrape interval |
| metrics.serviceMonitor.metricRelabelings | list | `[]` | Service monitor metric relabelings |
| metrics.serviceMonitor.relabelings | list | `[]` | Service monitor relabelings |
| metrics.serviceMonitor.scrapeTimeout | string | `""` | Service monitor scrape timeout |
| resources.limits | string | `nil` | Container resource limits |
| resources.requests | string | `nil` | Container resource requests |
| autoscaling.enabled | bool | `false` | Enable autoscaling |
| autoscaling.minReplicas | int | `1` | Min number of replicas |
| autoscaling.maxReplicas | int | `100` | Max number of replicas |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | Target CPU utilisation |
| autoscaling.targetMemoryUtilizationPercentage | string | `nil` | Target Memory utilisation |
| nodeSelector | object | `{}` | Node selector |
| tolerations | list | `[]` | Tolerations |
| affinity | object | `{}` | Affinity |
| service.type | string | `"ClusterIP"` | Service type |
| service.port | int | `443` | Service port |
| config.debug | bool | `false` | Enable debug (to use inmemorydatabase) |
| config.db.secretName | string | `""` | If set, database connection information will be read from the Secret with this name. Overrides `db.host`, `db.name`, `db.user`, and `db.password`. |
| config.db.host | string | `""` | Database host |
| config.db.hostSecretKeyName | string | `"host"` | The database host will be read from this `key` in the specified Secret, when `db.secretName` is set. |
| config.db.port | int | `5432` | Database port |
| config.db.portSecretKeyName | string | `"port"` | The database port will be read from this `key` in the specified Secret, when `db.secretName` is set. |
| config.db.name | string | `"reportsdb"` | Database name |
| config.db.dbNameSecretKeyName | string | `"dbname"` | The database name will be read from this `key` in the specified Secret, when `db.secretName` is set. |
| config.db.user | string | `"postgres"` | Database user |
| config.db.userSecretKeyName | string | `"username"` | The database username will be read from this `key` in the specified Secret, when `db.secretName` is set. |
| config.db.password | string | `"reports"` | Database password |
| config.db.passwordSecretKeyName | string | `"password"` | The database password will be read from this `key` in the specified Secret, when `db.secretName` is set. |
| config.db.sslmode | string | `"disable"` | Database SSL |
| config.db.sslrootcert | string | `""` | Database SSL root cert |
| config.db.sslkey | string | `""` | Database SSL key |
| config.db.sslcert | string | `""` | Database SSL cert |
| apiServicesManagement.enabled | bool | `true` | Create a helm hooks to install and delete api services |
| apiServicesManagement.installApiServices | object | `{"enabled":false,"installEphemeralReportsService":true}` | Install api services in manifest |
| apiServicesManagement.installApiServices.enabled | bool | `false` | Store reports in reports-server |
| apiServicesManagement.installApiServices.installEphemeralReportsService | bool | `true` | Store ephemeral reports in reports-server |
| apiServicesManagement.image.registry | string | `"docker.io"` | Image registry |
| apiServicesManagement.image.repository | string | `"bitnami/kubectl"` | Image repository |
| apiServicesManagement.image.tag | string | `"1.30.2"` | Image tag Defaults to `latest` if omitted |
| apiServicesManagement.image.pullPolicy | string | `nil` | Image pull policy Defaults to image.pullPolicy if omitted |
| apiServicesManagement.imagePullSecrets | list | `[]` | Image pull secrets |
| apiServicesManagement.podSecurityContext | object | `{}` | Security context for the pod |
| apiServicesManagement.nodeSelector | object | `{}` | Node labels for pod assignment |
| apiServicesManagement.tolerations | list | `[]` | List of node taints to tolerate |
| apiServicesManagement.podAntiAffinity | object | `{}` | Pod anti affinity constraints. |
| apiServicesManagement.podAffinity | object | `{}` | Pod affinity constraints. |
| apiServicesManagement.podLabels | object | `{}` | Pod labels. |
| apiServicesManagement.podAnnotations | object | `{}` | Pod annotations. |
| apiServicesManagement.nodeAffinity | object | `{}` | Node affinity constraints. |
| apiServicesManagement.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"privileged":false,"readOnlyRootFilesystem":true,"runAsGroup":65534,"runAsNonRoot":true,"runAsUser":65534,"seccompProfile":{"type":"RuntimeDefault"}}` | Security context for the hook containers |
| extraVolumes.volumeMounts | list | `[]` | Extra volume mounts for reports-server container |
| extraVolumes.volumes | list | `[]` | Extra volumes for reports-server deployment |

## Source Code

* <https://github.com/kyverno/reports-server>

## Requirements

Kubernetes: `>=1.16.0-0`

| Repository | Name | Version |
|------------|------|---------|
| oci://registry-1.docker.io/bitnamicharts | postgresql | 13.4.1 |

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Nirmata | <cncf-kyverno-maintainers@lists.cncf.io> | <https://kyverno.io/> |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.0](https://github.com/norwoodj/helm-docs/releases/v1.11.0)
