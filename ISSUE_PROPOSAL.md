# Proposal: Generic REST Handler for Kubernetes Resources (pkg/v2/rest)

## 📋 Overview

This proposal introduces a **generic, reusable REST handler** for Kubernetes resources in the reports-server, eliminating code duplication and providing a consistent API implementation across all resource types.

## 🎯 Problem Statement

**Current Situation** (`pkg/api`):
- Each resource type (PolicyReport, ClusterPolicyReport, EphemeralReport, ClusterEphemeralReport) has its own REST handler implementation
- ~350 lines of duplicated code per resource type
- **Total code duplication**: ~1,400 lines across 4 resource types
- Inconsistent error handling and validation across resource types
- In-memory filtering of resources (performance bottleneck)
- Hard to maintain and extend

**Example of Duplication:**
```go
// pkg/api/polr.go - 359 lines
// pkg/api/cpolr.go - 359 lines  
// pkg/api/ephr.go - 380 lines
// pkg/api/cephr.go - Similar duplication
```

## ✨ Proposed Solution

Introduce a **generic REST handler** (`pkg/v2/rest`) using Go generics that:

1. ✅ **Eliminates code duplication** - One implementation for all resource types
2. ✅ **Provides type safety** - Compile-time type checking with generics
3. ✅ **Improves maintainability** - Fix once, applies to all resources
4. ✅ **Consistent behavior** - All resources behave identically
5. ✅ **Better architecture** - Clean separation of concerns

## 🏗️ Architecture

### Package Structure

```
pkg/v2/rest/
├── handler.go          # Core generic handler
├── create.go           # Create operations
├── retrieve.go         # Get & List operations
├── update.go           # Update operations
├── delete.go           # Delete & DeleteCollection operations
├── watch.go            # Watch support
├── table.go            # kubectl table output
├── resourceMetadata.go # Resource configuration
├── object.go           # Type constraints
├── helpers.go          # Utility functions
└── IRest.go            # Interface definition
```

### Key Components

#### 1. **Generic Handler**
```go
type GenericRESTHandler[T Object] struct {
    repo        v2storage.IRepository[T]  // Storage backend
    versioning  api.Versioning            // Resource versioning
    broadcaster *watch.Broadcaster        // Watch events
    metadata    ResourceMetadata          // Resource config
}
```

#### 2. **Resource Metadata**
```go
type ResourceMetadata struct {
    Kind             string
    SingularName     string
    ShortNames       []string
    Namespaced       bool
    Group            string
    Resource         string
    NewFunc          func() runtime.Object
    NewListFunc      func() runtime.Object
    ListItemsFunc    func(list runtime.Object) []runtime.Object
    SetListItemsFunc func(list runtime.Object, items []runtime.Object)
    TableConverter   func(table *metav1beta1.Table, objects ...runtime.Object)
}
```

## 📊 Benefits

### Code Reduction
| Metric | Before (pkg/api) | After (pkg/v2/rest) | Savings |
|--------|------------------|---------------------|---------|
| **Lines per resource** | ~350 lines | ~50 lines (metadata only) | **86% reduction** |
| **Total code for 4 resources** | ~1,400 lines | ~800 lines (shared) | **600 lines saved** |
| **Duplicate code** | 100% | 0% | **Eliminated** |

### Performance
- 🚀 **Consistent behavior** across all resource types
- 🚀 **Better error handling** with detailed context
- 🚀 **Ready for storage-level filtering** (future optimization)

### Maintainability
- ✅ Bug fixes apply to all resources automatically
- ✅ New features (pagination, field selectors) add once, available everywhere
- ✅ Easier to test - test once, confidence in all resources
- ✅ New resource types - just provide metadata, zero code

## 🎬 Usage Example

### Adding a New Resource Type

**Before (pkg/api)** - ~350 lines of code:
```go
// Create polrStore struct
// Implement New(), Destroy(), Kind(), NewList()
// Implement Get() - ~25 lines
// Implement List() - ~45 lines
// Implement Create() - ~50 lines
// Implement Update() - ~70 lines
// Implement Delete() - ~30 lines
// Implement DeleteCollection() - ~40 lines
// Implement Watch() - ~30 lines
// Implement ConvertToTable() - ~40 lines
// ... 350+ lines total
```

**After (pkg/v2/rest)** - ~50 lines of metadata:
```go
handler := rest.NewGenericRESTHandler[*v1alpha2.PolicyReport](
    postgresRepo,
    versioning,
    rest.ResourceMetadata{
        Kind:         "PolicyReport",
        SingularName: "policyreport",
        ShortNames:   []string{"polr"},
        Namespaced:   true,
        Group:        "wgpolicyk8s.io",
        Resource:     "policyreports",
        
        // Factory functions
        NewFunc: func() runtime.Object {
            return &v1alpha2.PolicyReport{}
        },
        NewListFunc: func() runtime.Object {
            return &v1alpha2.PolicyReportList{}
        },
        
        // List manipulation
        ListItemsFunc: func(list runtime.Object) []runtime.Object {
            polrList := list.(*v1alpha2.PolicyReportList)
            items := make([]runtime.Object, len(polrList.Items))
            for i := range polrList.Items {
                items[i] = &polrList.Items[i]
            }
            return items
        },
        SetListItemsFunc: func(list runtime.Object, items []runtime.Object) {
            polrList := list.(*v1alpha2.PolicyReportList)
            polrList.Items = make([]v1alpha2.PolicyReport, len(items))
            for i, item := range items {
                polrList.Items[i] = *item.(*v1alpha2.PolicyReport)
            }
        },
        
        // Optional: Custom table output
        TableConverter: addPolicyReportToTable, // Optional
    },
)

// That's it! All CRUD operations, watch, and table output work automatically.
```

## 🔧 Implementation Details

### Implemented Interfaces
- ✅ `rest.Storage` (New, Destroy)
- ✅ `rest.Scoper` (NamespaceScoped)
- ✅ `rest.KindProvider` (Kind)
- ✅ `rest.SingularNameProvider` (GetSingularName)
- ✅ `rest.ShortNamesProvider` (ShortNames)
- ✅ `rest.Creater` (Create)
- ✅ `rest.Getter` (Get)
- ✅ `rest.Lister` (List, NewList)
- ✅ `rest.Updater` (Update)
- ✅ `rest.GracefulDeleter` (Delete)
- ✅ `rest.CollectionDeleter` (DeleteCollection)
- ✅ `rest.Watcher` (Watch)
- ✅ `rest.TableConvertor` (ConvertToTable)

### Key Features

#### 1. **Create Operations**
- Validates objects before creation
- Sets namespace from context if not provided
- Generates name from generateName if needed
- Sets metadata (UID, resourceVersion, generation, creationTimestamp)
- Dry-run support
- Broadcasts watch events

#### 2. **Retrieve Operations**
- Single resource Get by name
- List with filtering (labels, resource version)
- Clear separation of concerns
- Efficient filtering logic

#### 3. **Update Operations**
- Validates updates
- Handles optimistic concurrency
- Dry-run support
- Field validation modes (Strict, Warn, Ignore)

#### 4. **Delete Operations**
- Single resource deletion
- Collection deletion
- Validation before deletion
- Dry-run support
- Efficient (no redundant queries in DeleteCollection)

#### 5. **Watch Support**
- Real-time resource updates
- Bookmark watches with initial state
- Broadcaster-based implementation

#### 6. **Table Output**
- Default kubectl table format (Name, Namespace, Age)
- Custom converters supported
- Single and list support

## 🐛 Known Limitations & Future Work

### Current Limitations
1. **In-memory filtering** - Label selector and resource version filtering happens in the REST layer
   - **Impact**: Fetches all items, filters in memory
   - **Future**: Push filtering to storage layer (next PR)

2. **Double-fetch pattern** - Get operations fetch objects to validate before CRUD
   - **Impact**: 2x database queries for single operations
   - **Assessment**: Acceptable for single operations, can optimize later if needed

3. **Storage Filter limitations** - Only supports `Name` and `Namespace`
   - **Future**: Enhance `Filter` struct with labels, resourceVersion, pagination

### Proposed Future Enhancements

#### Phase 2: Storage-Level Filtering
```go
type Filter struct {
    Name              string
    Namespace         string
    Labels            map[string]string        // NEW
    LabelSelector     labels.Selector          // NEW
    ResourceVersion   string                   // NEW
    ResourceVersionOp string                   // NEW: "gte", "lte", "eq"
    Limit             int64                    // NEW: Pagination
    Continue          string                   // NEW: Pagination
}
```

**Benefits**:
- 🚀 Postgres can use JSONB indexes for label filtering
- 🚀 Filter at database level instead of in-memory
- 🚀 Support for pagination and efficient large-scale operations

#### Phase 3: Optional Optimizations
- Consider `UpdateAndReturn`/`DeleteAndReturn` patterns to eliminate double-fetch
- Field selector support
- Advanced watch filtering

## 📈 Migration Path

### Gradual Migration Strategy
1. ✅ **Phase 1** (This PR): Introduce `pkg/v2/rest` alongside existing `pkg/api`
2. **Phase 2**: Add storage-level filtering support
3. **Phase 3**: Migrate one resource type to v2 (e.g., PolicyReport)
4. **Phase 4**: Migrate remaining resource types
5. **Phase 5**: Remove `pkg/api` once all migrated

### Backward Compatibility
- New code in `pkg/v2` - no changes to existing `pkg/api`
- Existing API endpoints continue working
- Can be rolled out resource-by-resource
- Feature parity with existing implementation

## ✅ Testing Strategy

### Unit Tests
- Test each CRUD operation
- Test filtering logic
- Test error handling
- Test dry-run behavior

### Integration Tests
- Test with actual storage backends (Postgres, etcd, in-memory)
- Test watch functionality
- Test table conversion

### Compatibility Tests
- Ensure behavior matches existing `pkg/api` implementation
- Verify Kubernetes API compliance

## 📚 Documentation

### Required Documentation
- [ ] API documentation (GoDoc)
- [ ] Usage examples for each resource type
- [ ] Migration guide from pkg/api to pkg/v2
- [ ] Architecture decision records

## 🎯 Success Criteria

- [ ] All CRUD operations working correctly
- [ ] Watch support functional
- [ ] Table output working
- [ ] No code duplication
- [ ] Type-safe implementation
- [ ] Comprehensive tests
- [ ] Documentation complete

## 🤔 Open Questions

1. **Should we add storage-level filtering in this PR or separate PR?**
   - Recommendation: Separate PR for focused review

2. **Migration timeline for existing resources?**
   - Recommendation: One resource per release for safe rollout

3. **Backward compatibility requirements?**
   - Keep both implementations until full migration

## 👥 Reviewers

Please review:
- Architecture and design patterns
- Error handling approach
- Performance considerations
- API compatibility with Kubernetes standards

---

**This proposal aims to modernize the reports-server codebase, reduce technical debt, and provide a solid foundation for future enhancements.**

