# 📖 How to Add a New Resource Type

This guide shows EXACTLY what you need to do to add a new Kubernetes resource type to the v2 server.

## 🎯 Complete Checklist (7 Steps)

When you want to add a new resource (e.g., `MyResource` in `mygroup.io/v1alpha1`):

---

### ✅ Step 1: Register Type in Scheme (`scheme.go`)

**This is CRITICAL!** Kubernetes needs to know about your type.

#### If adding to an existing API group:

Find the appropriate install function and add your types:

```go
// Example: Adding to wgpolicyk8s.io
func installWgPolicyTypesInternal(scheme *runtime.Scheme) error {
	schemeGroupVersion := schema.GroupVersion{
		Group:   "wgpolicyk8s.io",
		Version: runtime.APIVersionInternal,
	}

	addKnownTypes := func(s *runtime.Scheme) error {
		s.AddKnownTypes(
			schemeGroupVersion,
			&v1alpha2.ClusterPolicyReport{},
			&v1alpha2.PolicyReport{},
			&v1alpha2.ClusterPolicyReportList{},
			&v1alpha2.PolicyReportList{},
			&v1alpha2.MyResource{},          // ← ADD THIS
			&v1alpha2.MyResourceList{},      // ← ADD THIS
		)
		return nil
	}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	return schemeBuilder.AddToScheme(scheme)
}
```

#### If creating a NEW API group:

1. **Import the package:**
```go
import mygroup "mygroup.io/apis/mygroup.io/v1alpha1"
```

2. **Add to `addKnownTypes()`:**
```go
func addKnownTypes(scheme *runtime.Scheme) error {
	// ... existing groups
	
	// mygroup.io types (NEW)
	utilruntime.Must(installMyGroupTypesInternal(scheme))
	utilruntime.Must(mygroup.AddToScheme(scheme))
	
	return nil
}
```

3. **Create install function:**
```go
func installMyGroupTypesInternal(scheme *runtime.Scheme) error {
	schemeGroupVersion := schema.GroupVersion{
		Group:   "mygroup.io",
		Version: runtime.APIVersionInternal,
	}

	addKnownTypes := func(s *runtime.Scheme) error {
		s.AddKnownTypes(
			schemeGroupVersion,
			&mygroup.MyResource{},
			&mygroup.MyResourceList{},
		)
		return nil
	}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	return schemeBuilder.AddToScheme(scheme)
}
```

4. **Set version priority:**
```go
func setPriorities(scheme *runtime.Scheme) error {
	// ... existing priorities
	utilruntime.Must(scheme.SetVersionPriority(mygroup.SchemeGroupVersion))  // ← ADD
	return nil
}
```

---

### ✅ Step 2: Add Helper Functions (`helpers.go`)

Add 2 simple functions for list manipulation:

```go
// MyResource helpers
func extractMyResourceItems(list runtime.Object) []runtime.Object {
	myList := list.(*v1alpha2.MyResourceList)
	items := make([]runtime.Object, len(myList.Items))
	for i := range myList.Items {
		items[i] = &myList.Items[i]
	}
	return items
}

func setMyResourceItems(list runtime.Object, items []runtime.Object) {
	myList := list.(*v1alpha2.MyResourceList)
	myList.Items = make([]v1alpha2.MyResource, len(items))
	for i, item := range items {
		myList.Items[i] = *item.(*v1alpha2.MyResource)
	}
}
```

**Lines: ~15**

---

### ✅ Step 3: Register in Registry (`registry.go`)

Add one field to `ResourceRegistry` struct:

```go
type ResourceRegistry struct {
	PolicyReport            ResourceDefinition
	ClusterPolicyReport     ResourceDefinition
	EphemeralReport         ResourceDefinition
	ClusterEphemeralReport  ResourceDefinition
	Report                  ResourceDefinition
	ClusterReport           ResourceDefinition
	MyResource              ResourceDefinition  // ← ADD THIS
}
```

Then add one entry in `NewResourceRegistry()`:

```go
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		// ... existing entries
		
		MyResource: NewResourceDefinition(
			"MyResource",           // Kind
			"myresource",           // Singular name
			[]string{"myr"},        // Short names
			true,                   // Namespaced? (true/false)
			"mygroup.io",           // API group
			"v1alpha1",             // Version
			"myresources",          // Resource name (plural)
			func() runtime.Object { return &v1alpha2.MyResource{} },
			func() runtime.Object { return &v1alpha2.MyResourceList{} },
			extractMyResourceItems,
			setMyResourceItems,
		),
	}
}
```

Update `GetByAPIGroup()` if new API group:

```go
func (r *ResourceRegistry) GetByAPIGroup() map[string][]ResourceDefinition {
	return map[string][]ResourceDefinition{
		// ... existing groups
		"mygroup.io": {      // ← ADD IF NEW GROUP
			r.MyResource,
		},
	}
}
```

**Lines: ~15-20**

---

### ✅ Step 4: Add Factory Method (`factory.go`)

Add ONE simple method:

```go
// CreateMyResourceHandler creates a REST handler for MyResource
func (f *HandlerFactory) CreateMyResourceHandler(
	repo storage.IRepository[*v1alpha2.MyResource],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*v1alpha2.MyResource](
		repo,
		f.versioning,
		toRestMetadata(f.registry.MyResource),
	)
}
```

**Lines: ~6**

---

### ✅ Step 5: Add Repository Field (`config.go`)

Add field to `Repositories` struct:

```go
type Repositories struct {
	PolicyReports           storage.IRepository[*v1alpha2.PolicyReport]
	ClusterPolicyReports    storage.IRepository[*v1alpha2.ClusterPolicyReport]
	EphemeralReports        storage.IRepository[*reportsv1.EphemeralReport]
	ClusterEphemeralReports storage.IRepository[*reportsv1.ClusterEphemeralReport]
	Reports                 storage.IRepository[*openreportsv1alpha1.Report]
	ClusterReports          storage.IRepository[*openreportsv1alpha1.ClusterReport]
	MyResource              storage.IRepository[*v1alpha2.MyResource]  // ← ADD THIS
}
```

**Lines: ~1**

---

### ✅ Step 6: Add Repository Creation (`config.go`)

In each `create*Repositories()` function, add one entry:

#### Postgres:
```go
func (c *Config) createPostgresRepositories() (*Repositories, error) {
	// ... existing code
	
	return &Repositories{
		// ... existing repos
		
		MyResource: postgres.NewPostgresRepository[*v1alpha2.MyResource](
			router,
			c.Storage.ClusterID,
			"myresources",    // table name
			"MyResource",     // for logging
			true,             // namespaced
			schema.GroupResource{Group: "mygroup.io", Resource: "myresources"},
		),
	}, nil
}
```

#### Etcd:
```go
func (c *Config) createEtcdRepositories() (*Repositories, error) {
	// ... existing code
	
	return &Repositories{
		// ... existing repos
		
		MyResource: etcd.NewEtcdRepository[*v1alpha2.MyResource](
			client,
			schema.GroupVersionKind{Group: "mygroup.io", Version: "v1alpha1", Kind: "MyResource"},
			schema.GroupResource{Group: "mygroup.io", Resource: "myresources"},
			"MyResource",
			true,  // namespaced
		),
	}, nil
}
```

#### In-Memory:
```go
func (c *Config) createInMemoryRepositories() (*Repositories, error) {
	return &Repositories{
		// ... existing repos
		
		MyResource: inmemory.NewInMemoryRepository[*v1alpha2.MyResource](
			"MyResource",
			true,  // namespaced
			schema.GroupResource{Group: "mygroup.io", Resource: "myresources"},
		),
	}, nil
}
```

**Lines: ~8 × 3 = ~24**

---

### ✅ Step 7: Wire Up in Server (`server.go`)

#### Option A: Add to existing API group

```go
func (s *Server) installMyGroupAPI() error {
	factory := NewHandlerFactory(s.config.Versioning)
	
	// Add your resource to existing resources map
	myResourceHandler := factory.CreateMyResourceHandler(s.repositories.MyResource)
	
	resources := map[string]rest.Storage{
		// ... existing resources
		"myresources": myResourceHandler,  // ← ADD THIS
	}
	
	apiGroupInfo := BuildAPIGroupInfo("mygroup.io", "v1alpha1", resources, GetScheme(), GetCodecs())
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}
```

#### Option B: Create new API installation function (if new group)

```go
// Add to Server struct or just in InstallAPIGroups():

func (s *Server) installMyGroupAPI() error {
	factory := NewHandlerFactory(s.config.Versioning)
	
	myResourceHandler := factory.CreateMyResourceHandler(s.repositories.MyResource)
	
	resources := map[string]rest.Storage{
		"myresources": myResourceHandler,
	}
	
	apiGroupInfo := BuildAPIGroupInfo(
		"mygroup.io",
		"v1alpha1",
		resources,
		GetScheme(),
		GetCodecs(),
	)
	
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}

// Then call it in InstallAPIGroups():
func (s *Server) InstallAPIGroups() error {
	// ... existing installations
	
	if s.config.Server.EnableMyGroup {  // Add flag to ServerConfig if needed
		if err := s.installMyGroupAPI(); err != nil {
			return fmt.Errorf("failed to install mygroup API: %w", err)
		}
		klog.Info("Installed mygroup.io/v1alpha1 API group")
	}
	
	return nil
}
```

**Lines: ~15-30**

---

## 📊 Complete Summary

| Step | File | Lines | What to Add |
|------|------|-------|-------------|
| **1. Scheme registration** | `scheme.go` | ~5-30 | Import, register types, set priority |
| **2. Helper functions** | `helpers.go` | ~15 | Extract & set functions |
| **3. Registry entry** | `registry.go` | ~15 | Resource definition |
| **4. Factory method** | `factory.go` | ~6 | Handler creation |
| **5. Repo field** | `config.go` | ~1 | Repository field |
| **6. Repo creation** | `config.go` | ~24 | Postgres, etcd, in-memory |
| **7. API wiring** | `server.go` | ~15-30 | API installation |

**Total: ~81-111 lines** (depending on new vs existing API group)

---

## 🎯 Files to Modify (in order)

```
1. pkg/v2/server/scheme.go    ← Register in Kubernetes scheme (CRITICAL!)
2. pkg/v2/server/helpers.go   ← Add extract/set functions
3. pkg/v2/server/registry.go  ← Define resource
4. pkg/v2/server/factory.go   ← Add factory method  
5. pkg/v2/server/config.go    ← Add repos (field + creation)
6. pkg/v2/server/server.go    ← Install API group
```

---

## ✨ Complete Example: Adding "ValidationReport"

### 1. **scheme.go** - Register type
```go
// Import (at top of file)
import validationv1 "mygroup.io/apis/mygroup.io/v1alpha1"

// Add to addKnownTypes():
func addKnownTypes(scheme *runtime.Scheme) error {
	// ... existing
	
	// mygroup.io types (NEW)
	utilruntime.Must(installMyGroupTypesInternal(scheme))
	utilruntime.Must(validationv1.AddToScheme(scheme))
	
	return nil
}

// Add install function:
func installMyGroupTypesInternal(scheme *runtime.Scheme) error {
	schemeGroupVersion := schema.GroupVersion{
		Group:   "mygroup.io",
		Version: runtime.APIVersionInternal,
	}

	addKnownTypes := func(s *runtime.Scheme) error {
		s.AddKnownTypes(
			schemeGroupVersion,
			&validationv1.ValidationReport{},
			&validationv1.ValidationReportList{},
		)
		return nil
	}

	return runtime.NewSchemeBuilder(addKnownTypes).AddToScheme(scheme)
}

// Set priority:
func setPriorities(scheme *runtime.Scheme) error {
	// ... existing
	utilruntime.Must(scheme.SetVersionPriority(validationv1.SchemeGroupVersion))
	return nil
}
```

### 2. **helpers.go** - Add helpers
```go
func extractValidationReportItems(list runtime.Object) []runtime.Object {
	valList := list.(*validationv1.ValidationReportList)
	items := make([]runtime.Object, len(valList.Items))
	for i := range valList.Items {
		items[i] = &valList.Items[i]
	}
	return items
}

func setValidationReportItems(list runtime.Object, items []runtime.Object) {
	valList := list.(*validationv1.ValidationReportList)
	valList.Items = make([]validationv1.ValidationReport, len(items))
	for i, item := range items {
		valList.Items[i] = *item.(*validationv1.ValidationReport)
	}
}
```

### 3. **registry.go** - Define resource
```go
type ResourceRegistry struct {
	// ... existing
	ValidationReport ResourceDefinition  // ← ADD
}

func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		// ... existing
		
		ValidationReport: NewResourceDefinition(
			"ValidationReport",
			"validationreport",
			[]string{"valr"},
			true,  // namespaced
			"mygroup.io",
			"v1alpha1",
			"validationreports",
			func() runtime.Object { return &validationv1.ValidationReport{} },
			func() runtime.Object { return &validationv1.ValidationReportList{} },
			extractValidationReportItems,
			setValidationReportItems,
		),
	}
}

// Update GetByAPIGroup():
func (r *ResourceRegistry) GetByAPIGroup() map[string][]ResourceDefinition {
	return map[string][]ResourceDefinition{
		// ... existing
		"mygroup.io": {r.ValidationReport},  // ← ADD IF NEW GROUP
	}
}
```

### 4. **factory.go** - Add factory method
```go
func (f *HandlerFactory) CreateValidationReportHandler(
	repo storage.IRepository[*validationv1.ValidationReport],
) restAPI.Storage {
	return rest.NewGenericRESTHandler[*validationv1.ValidationReport](
		repo,
		f.versioning,
		toRestMetadata(f.registry.ValidationReport),
	)
}
```

### 5. **config.go** - Add repository field
```go
type Repositories struct {
	// ... existing
	ValidationReport storage.IRepository[*validationv1.ValidationReport]
}
```

### 6. **config.go** - Create repositories
```go
// Postgres:
func (c *Config) createPostgresRepositories() (*Repositories, error) {
	// ... existing
	return &Repositories{
		// ... existing
		ValidationReport: postgres.NewPostgresRepository[*validationv1.ValidationReport](
			router, c.Storage.ClusterID,
			"validationreports", "ValidationReport", true,
			schema.GroupResource{Group: "mygroup.io", Resource: "validationreports"},
		),
	}, nil
}

// Etcd:
func (c *Config) createEtcdRepositories() (*Repositories, error) {
	// ... existing
	return &Repositories{
		// ... existing
		ValidationReport: etcd.NewEtcdRepository[*validationv1.ValidationReport](
			client,
			schema.GroupVersionKind{Group: "mygroup.io", Version: "v1alpha1", Kind: "ValidationReport"},
			schema.GroupResource{Group: "mygroup.io", Resource: "validationreports"},
			"ValidationReport", true,
		),
	}, nil
}

// In-Memory:
func (c *Config) createInMemoryRepositories() (*Repositories, error) {
	return &Repositories{
		// ... existing
		ValidationReport: inmemory.NewInMemoryRepository[*validationv1.ValidationReport](
			"ValidationReport", true,
			schema.GroupResource{Group: "mygroup.io", Resource: "validationreports"},
		),
	}, nil
}
```

### 7. **server.go** - Install API
```go
// Add to InstallAPIGroups():
if s.config.Server.EnableValidationReports {  // Add flag if needed
	if err := s.installValidationReportsAPI(); err != nil {
		return fmt.Errorf("failed to install validation reports API: %w", err)
	}
	klog.Info("Installed mygroup.io/v1alpha1 API group")
}

// Create installation function:
func (s *Server) installValidationReportsAPI() error {
	factory := NewHandlerFactory(s.config.Versioning)
	
	valHandler := factory.CreateValidationReportHandler(s.repositories.ValidationReport)
	
	resources := map[string]rest.Storage{
		"validationreports": valHandler,
	}
	
	apiGroupInfo := BuildAPIGroupInfo(
		"mygroup.io",
		"v1alpha1",
		resources,
		GetScheme(),
		GetCodecs(),
	)
	
	return s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
}
```

---

## ⚠️ Common Mistakes to Avoid

### ❌ Mistake 1: Forgetting Scheme Registration
```
Error: "no kind is registered for the type..."
Solution: Add types to scheme.go!
```

### ❌ Mistake 2: Wrong API Group in Scheme
```go
// Wrong:
scheme.AddKnownTypes(wrongGroupVersion, &MyResource{})

// Right:
schemeGroupVersion := schema.GroupVersion{
    Group:   "mygroup.io",          // Must match ResourceDefinition!
    Version: runtime.APIVersionInternal,
}
```

### ❌ Mistake 3: Forgetting List Type
```go
// Wrong:
s.AddKnownTypes(gv, &MyResource{})  // Missing list!

// Right:
s.AddKnownTypes(gv, 
    &MyResource{},      // Single resource
    &MyResourceList{},  // List type (REQUIRED!)
)
```

### ❌ Mistake 4: Not Setting Version Priority
```go
// For new API groups, you MUST set priority:
utilruntime.Must(scheme.SetVersionPriority(mygroup.SchemeGroupVersion))
```

---

## 🎯 Checklist When Adding Resource

- [ ] Step 1: Import package in `scheme.go`
- [ ] Step 1: Add to `addKnownTypes()` or create install function
- [ ] Step 1: Register both resource AND list type
- [ ] Step 1: Set version priority (if new group)
- [ ] Step 2: Add helpers in `helpers.go`
- [ ] Step 3: Add to `ResourceRegistry` in `registry.go`
- [ ] Step 3: Update `GetByAPIGroup()` if new group
- [ ] Step 4: Add factory method in `factory.go`
- [ ] Step 5: Add repository field in `config.go`
- [ ] Step 6: Add repo creation in all 3 backends in `config.go`
- [ ] Step 7: Wire up in `server.go`
- [ ] Test: Create, Get, List, Update, Delete operations
- [ ] Test: Watch functionality
- [ ] Test: `kubectl get myresources` (table output)

---

## 🔍 Why Scheme Registration is Critical

### What the Scheme Does:
1. **Type Registration**: Tells Kubernetes what types exist
2. **Serialization**: Converts between JSON/YAML and Go structs
3. **Versioning**: Handles API version conversions
4. **Discovery**: Enables `kubectl api-resources`

### Without Scheme Registration:
```bash
$ kubectl get myresources
Error: the server doesn't have a resource type "myresources"
```

### With Scheme Registration:
```bash
$ kubectl get myresources
NAME          AGE
my-report-1   5m
my-report-2   3m
```

---

## 📊 Lines Required

### For Existing API Group (e.g., adding to wgpolicyk8s.io):
```
Step 1 (scheme):    ~5 lines  (just add 2 types to existing function)
Steps 2-7:          ~76 lines (same as before)
Total:              ~81 lines
```

### For NEW API Group (e.g., mygroup.io):
```
Step 1 (scheme):    ~30 lines (import, install function, priority)
Steps 2-7:          ~76 lines (same as before)
Total:              ~106 lines
```

---

## 🎉 What You Get

After these 7 steps, your new resource automatically has:

- ✅ Full CRUD operations
- ✅ Watch support (real-time updates)
- ✅ kubectl support (`kubectl get myresources`)
- ✅ kubectl table output
- ✅ Dry-run support
- ✅ Field validation modes
- ✅ Label selector filtering
- ✅ Resource version filtering
- ✅ All 3 storage backends supported

**All powered by pkg/v2/rest generic implementation!**

---

## 📝 Summary

### Checklist: 7 Steps
1. ✅ **scheme.go** - Register in Kubernetes scheme (**CRITICAL - you caught this!**)
2. ✅ **helpers.go** - Add list manipulation helpers
3. ✅ **registry.go** - Define resource metadata
4. ✅ **factory.go** - Add factory method
5. ✅ **config.go** - Add repository field
6. ✅ **config.go** - Create repositories (3 backends)
7. ✅ **server.go** - Wire up API installation

**Total**: ~81-106 lines, ~20 minutes

---

**Thank you for catching the scheme registration step! It's absolutely critical.** 🙏

