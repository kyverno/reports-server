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
