package server

import "errors"

// Configuration errors
var (
	ErrMissingGenericConfig = errors.New("generic API server config is required")
	ErrMissingRESTConfig    = errors.New("REST config is required")
	ErrMissingVersioning    = errors.New("versioning is required")
)

// Storage configuration errors
var (
	ErrInvalidStorageBackend   = errors.New("invalid storage backend (must be: postgres, etcd, or memory)")
	ErrMissingPostgresConfig   = errors.New("postgres config is required when backend=postgres")
	ErrMissingEtcdConfig       = errors.New("etcd config is required when backend=etcd")
	ErrMissingPostgresHost     = errors.New("postgres host is required")
	ErrMissingPostgresDatabase = errors.New("postgres database is required")
	ErrMissingPostgresUser     = errors.New("postgres user is required")
	ErrMissingEtcdEndpoints    = errors.New("etcd endpoints are required")
)

// Implementation errors
var (
	ErrNotImplemented = errors.New("not yet implemented")
)

// Server errors
var (
	ErrServerNotRunning     = errors.New("server is not running")
	ErrServerAlreadyRunning = errors.New("server is already running")
)
