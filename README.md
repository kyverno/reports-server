# Reports Server

Reports server provides a scalable solution for storing policy reports and cluster policy reports. It moves reports out of etcd and stores them in a PostgreSQL database, etcd, or in-memory storage.

## Why Reports Server?

### Problem: etcd Limitations

- **Size limits**: etcd has an 8GB maximum size. Reports are large objects that quickly fill this capacity
- **Performance degradation**: Heavy report activity causes API server to buffer large amounts of data, leading to cluster unavailability
- **Wrong tool**: Reports are analytical data, not transactional. etcd is designed for configuration data
- **CAP guarantees unnecessary**: Reports are ephemeral and can be regenerated

### Solution: Dedicated Reports Server

- **Scalability**: PostgreSQL/dedicated storage handles much larger datasets
- **Performance**: Offloads etcd and Kubernetes API server
- **Efficient queries**: Direct database queries for analytical workloads
- **Flexibility**: Multiple storage backends (postgres, etcd, memory)

## Architecture

Reports server is a **Kubernetes aggregated API server** that implements three API groups:

- `wgpolicyk8s.io/v1alpha2` - PolicyReport, ClusterPolicyReport
- `reports.kyverno.io/v1` - EphemeralReport, ClusterEphemeralReport  
- `openreports.io/v1alpha1` - Report, ClusterReport

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│ 1. kubectl get policyreports                                │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Kubernetes API Server                                    │
│    - Receives request                                       │
│    - Looks up APIService for wgpolicyk8s.io/v1alpha2       │
│    - Routes to reports-server service                       │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Reports Server (Our Aggregated API Server)              │
│    ┌──────────────────────────────────────────────────┐    │
│    │ pkg/v2/cmd - CLI Layer                          │    │
│    │ • Parses flags & env vars                       │    │
│    │ • Creates server config                         │    │
│    └──────────────────┬───────────────────────────────┘    │
│                       ↓                                     │
│    ┌──────────────────────────────────────────────────┐    │
│    │ pkg/v2/server - Server Layer                    │    │
│    │ • Installs API groups                           │    │
│    │ • Manages APIServices                           │    │
│    │ • Health checks                                 │    │
│    └──────────────────┬───────────────────────────────┘    │
│                       ↓                                     │
│    ┌──────────────────────────────────────────────────┐    │
│    │ pkg/v2/rest - REST Handler Layer                │    │
│    │ • Generic CRUD handlers                         │    │
│    │ • Processes GET /apis/.../policyreports         │    │
│    │ • Filters, validates, transforms                │    │
│    └──────────────────┬───────────────────────────────┘    │
│                       ↓                                     │
│    ┌──────────────────────────────────────────────────┐    │
│    │ pkg/v2/storage - Storage Layer                  │    │
│    │ • Repository pattern (postgres/etcd/memory)     │    │
│    │ • Executes database queries                     │    │
│    └──────────────────┬───────────────────────────────┘    │
└────────────────────────┼────────────────────────────────────┘
                         ↓
         ┌───────────────────────────────┐
         │ 4. Storage Backend            │
         │ • PostgreSQL                  │
         │ • etcd                        │
         │ • In-Memory                   │
         └───────────────┬───────────────┘
                         │
         Results flow back up the chain
                         │
         ┌───────────────↓───────────────┐
         │ 5. Response to kubectl        │
         │ • Formatted as table          │
         │ • JSON/YAML supported         │
         └───────────────────────────────┘
```

### Request Flow (Detailed)

#### 1. **Client Request**
```bash
kubectl get policyreports -n default
```

#### 2. **Kubernetes API Server**
- Receives request for `/apis/wgpolicyk8s.io/v1alpha2/namespaces/default/policyreports`
- Looks up APIService `v1alpha2.wgpolicyk8s.io`
- Routes to our aggregated API server

#### 3. **pkg/v2/cmd** (Entry Point)
```go
cmd/command.go:Run()
  ├─> Parses flags (--storage-backend, --db-host, etc.)
  ├─> Loads environment variables (DB_PASSWORD, etc.)
  ├─> Validates configuration
  └─> Creates server.Config
```

#### 4. **pkg/v2/server** (Server Setup)
```go
server/config.go:Complete()
  ├─> Creates storage repositories (postgres/etcd/memory)
  ├─> Installs APIServices in Kubernetes
  └─> Returns configured Server

server/server.go:Run()
  ├─> InstallAPIGroups() - Registers all API endpoints
  │    ├─> Creates REST handlers using factory pattern
  │    ├─> Registers with GenericAPIServer
  │    └─> Makes APIs available to kubectl
  ├─> InstallHealthChecks() - /readyz and /healthz
  └─> Starts server (listens on port 443)
```

#### 5. **pkg/v2/rest** (Request Processing)
```go
rest/retrieve.go:List()
  ├─> Extracts namespace from context
  ├─> Calls storage.List() to fetch items
  ├─> Filters by label selector (in-memory for now)
  ├─> Filters by resource version
  ├─> Builds Kubernetes list response
  └─> Returns PolicyReportList to client
```

#### 6. **pkg/v2/storage** (Data Access)
```go
storage/postgres/retrieve.go:List()
  ├─> Builds SQL query: SELECT * FROM policyreports WHERE namespace=...
  ├─> Executes query on database
  ├─> Unmarshals JSON data to PolicyReport structs
  └─> Returns []PolicyReport
```

#### 7. **Response Path**
```
Storage → REST handler → Server → Kubernetes API → kubectl
```

### Watch Flow

```
kubectl get policyreports --watch
  ↓
pkg/v2/rest/watch.go:Watch()
  ├─> Creates watch connection via broadcaster
  ├─> Returns initial state (current resources)
  └─> Streams subsequent changes

When a resource changes:
  Create/Update/Delete operation
    ↓
  broadcaster.Action(watch.Added/Modified/Deleted, resource)
    ↓
  All active watchers receive the event
    ↓
  kubectl displays: "ADDED report-1" or "MODIFIED report-1"
```

### Kyverno Integration: How Reports Flow Into the Server

When Kyverno creates a PolicyReport, here's how it flows through the system:

```
┌──────────────────────────────────────────────────────────┐
│ 1. Kyverno Policy Engine                                 │
│    • Evaluates policies against resources                │
│    • Generates PolicyReport                              │
│    • Calls: k8sClient.Create(policyReport)               │
└───────────────────┬──────────────────────────────────────┘
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 2. Kubernetes API Server                                 │
│    • Receives: POST /apis/wgpolicyk8s.io/v1alpha2/       │
│                namespaces/default/policyreports          │
│    • Looks up APIService for wgpolicyk8s.io              │
│    • Routes to reports-server aggregated API             │
└───────────────────┬──────────────────────────────────────┘
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 3. Reports Server - REST Layer                           │
│    pkg/v2/rest/create.go:Create()                        │
│    ├─> Validates the PolicyReport                        │
│    ├─> Sets namespace (from context)                     │
│    ├─> Generates name (if generateName used)             │
│    ├─> Sets metadata (UID, resourceVersion, timestamps)  │
│    ├─> Adds annotation:                                  │
│    │   "reports.kyverno.io/served-by-reports-server=true"│
│    └─> Metrics: Track API request, validation            │
└───────────────────┬──────────────────────────────────────┘
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 4. Storage Layer                                         │
│    pkg/v2/storage/postgres/create.go:Create()            │
│    ├─> Marshals PolicyReport to JSON                     │
│    ├─> Executes: INSERT INTO policyreports (...)         │
│    │   VALUES (name, namespace, cluster_id, data, ...)   │
│    ├─> Returns success or AlreadyExists error            │
│    └─> Metrics: Track storage operation timing           │
└───────────────────┬──────────────────────────────────────┘
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 5. Database (PostgreSQL/etcd/memory)                     │
│    • Stores the report persistently                      │
│    • Returns success                                     │
└───────────────────┬──────────────────────────────────────┘
                    │
         Success flows back up
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 6. Watch Event Broadcasting                              │
│    pkg/v2/rest/create.go                                 │
│    ├─> broadcaster.Action(watch.Added, policyReport)     │
│    └─> All active watchers receive the event             │
│         • kubectl get policyreports --watch               │
│         • Kyverno controllers watching for changes       │
│         • External monitoring tools                      │
│    └─> Metrics: Track watch events sent/dropped          │
└───────────────────┬──────────────────────────────────────┘
                    │
                    ↓
┌──────────────────────────────────────────────────────────┐
│ 7. Response to Kyverno                                   │
│    • HTTP 201 Created                                    │
│    • Returns created PolicyReport with:                  │
│      - resourceVersion (for optimistic concurrency)      │
│      - UID (unique identifier)                           │
│      - timestamps (creationTimestamp)                    │
└──────────────────────────────────────────────────────────┘
```

### Detailed Code Flow: Kyverno → Reports Server

#### Step 1: Kyverno Creates Report
```go
// In Kyverno controller
policyReport := &v1alpha2.PolicyReport{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "pod-policy-report",
        Namespace: "default",
    },
    Results: []v1alpha2.PolicyReportResult{...},
}

// Kyverno calls Kubernetes API
client.PolicyReports("default").Create(ctx, policyReport)
```

#### Step 2: Request Arrives at Reports Server
```go
// pkg/v2/rest/create.go:Create()

// 1. Extract namespace from request context
namespace := genericapirequest.NamespaceValue(ctx)

// 2. Validate the object
err := createValidation(ctx, policyReport)
// Metrics: Record validation errors if any

// 3. Set metadata
resource.SetUID(uuid.NewUUID())
resource.SetResourceVersion(versioning.UseResourceVersion())
resource.SetCreationTimestamp(metav1.Now())
resource.SetGeneration(1)

// 4. Add our annotation
resource.SetAnnotations(map[string]string{
    "reports.kyverno.io/served-by-reports-server": "true",
})
```

#### Step 3: Persist to Storage
```go
// pkg/v2/rest/create.go calls storage
err = h.repo.Create(ctx, resource)

// ↓ Goes to pkg/v2/storage/postgres/create.go

// 1. Check if already exists (strict semantics)
existing, _ := p.repo.Get(ctx, filter)
if existing != nil {
    return storage.ErrAlreadyExists
}

// 2. Marshal to JSON
data, _ := json.Marshal(resource)

// 3. Execute INSERT
INSERT INTO policyreports (
    name, namespace, cluster_id, 
    uid, resource_version, data, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)

// Metrics: Record storage operation timing
```

#### Step 4: Broadcast to Watchers
```go
// Back in pkg/v2/rest/create.go

// Notify all active watchers
broadcaster.Action(watch.Added, resource)

// Any kubectl watching gets notified:
$ kubectl get policyreports --watch
NAME                 AGE
pod-policy-report    1s   ← Just appeared!

// Metrics: Track watch event sent/dropped
```

#### Step 5: Return to Kyverno
```go
// Response: HTTP 201 Created
{
    "apiVersion": "wgpolicyk8s.io/v1alpha2",
    "kind": "PolicyReport",
    "metadata": {
        "name": "pod-policy-report",
        "namespace": "default",
        "uid": "abc-123",
        "resourceVersion": "12345",
        "creationTimestamp": "2024-01-01T00:00:00Z",
        "annotations": {
            "reports.kyverno.io/served-by-reports-server": "true"
        }
    },
    "results": [...]
}

// Metrics: Record API request completed (201, duration)
```

### Kyverno Update Flow

When Kyverno updates a report:

```
Kyverno Updates Report
  ↓
GET /apis/.../policyreports/report-name
  ├─> pkg/v2/rest/retrieve.go:Get()
  ├─> Fetches current version from storage
  └─> Returns to Kyverno

Kyverno Modifies Report
  ↓
PUT /apis/.../policyreports/report-name
  ├─> pkg/v2/rest/update.go:Update()
  ├─> Validates update
  ├─> Checks resourceVersion (optimistic concurrency)
  ├─> Stores updated version
  ├─> Broadcasts watch.Modified event
  └─> Returns updated report

All watchers notified:
  kubectl get policyreports --watch
  → MODIFIED pod-policy-report
```

### Kyverno Delete Flow

When Kyverno deletes a report:

```
DELETE /apis/.../policyreports/report-name
  ↓
pkg/v2/rest/delete.go:Delete()
  ├─> Gets report from storage (for return value)
  ├─> Validates deletion
  ├─> Deletes from storage
  ├─> Broadcasts watch.Deleted event
  └─> Returns deleted report

All watchers notified:
  kubectl get policyreports --watch
  → DELETED pod-policy-report

Storage cleaned up:
  PostgreSQL: DELETE FROM policyreports WHERE name=...
```

### Supported Storage Backends

#### PostgreSQL (Production)
```bash
--storage-backend=postgres
--db-host=localhost --db-port=5432 --db-name=reports
# Or use environment variables:
export DB_HOST=localhost
export DB_PASSWORD=secret
```

#### etcd (Production)
```bash
--storage-backend=etcd
--etcd-endpoints=localhost:2379,localhost:2380
```

#### In-Memory (Development/Testing)
```bash
--storage-backend=memory
# No additional configuration needed
```

## V2 Architecture

The codebase has both V1 (legacy) and V2 (modern) implementations:

### V1 (Legacy - pkg/api, pkg/server, pkg/app)
- Per-resource implementations (~350 lines each)
- Will be deprecated after V2 migration

### V2 (Modern - pkg/v2/*)
```
pkg/v2/
├── cmd/           - CLI interface (flags, validation, config)
├── metrics/       - Prometheus metrics (storage, API, server, watch)
├── versioning/    - Resource version management
├── storage/       - Generic storage repositories
├── rest/          - Generic REST handlers (one impl for all resources)
└── server/        - Server setup with resource registry pattern
```

**Benefits of V2:**
- Generic implementation (no code duplication)
- Resource registry pattern (easy to add new resources)
- Independent of V1 (can remove V1 safely)
- Better organized (single responsibility per file)
- Comprehensive metrics and logging
- Production ready

## Installation

See [docs/INSTALL.md](docs/INSTALL.md)

## Adding New Resources

See [docs/ADDING_RESOURCES.md](docs/ADDING_RESOURCES.md)

## Metrics

Reports server exposes Prometheus metrics at `/metrics`:

- **Storage**: Operations, duration, efficiency, connection pools
- **API**: Requests, latency, in-flight, validation errors
- **Watch**: Active connections, events, drops
- **Server**: Uptime, health checks, API groups

See `pkg/v2/metrics/` for complete metric definitions.

## Development

### Running Locally
```bash
# With in-memory storage (quick start)
go run main.go --storage-backend=memory

# With PostgreSQL
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=secret
export DB_DATABASE=reports

go run main.go --storage-backend=postgres
```

### Running Tests
```bash
go test ./pkg/v2/...
```

## Contributing

When adding new features to V2:
1. Follow the resource registry pattern (see docs/ADDING_RESOURCES.md)
2. Add appropriate metrics (see pkg/v2/metrics/)
3. Add structured logging with klog
4. Update documentation
5. Add tests

## License

See [LICENSE](LICENSE)
