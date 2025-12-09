package opts

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/kyverno/reports-server/pkg/api"
	generatedopenapi "github.com/kyverno/reports-server/pkg/api/generated/openapi"
	"github.com/kyverno/reports-server/pkg/storage/etcd"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/pkg/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
)

type Options struct {
	// genericoptions.RecomendedOptions - EtcdOptions
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Audit          *genericoptions.AuditOptions
	Features       *genericoptions.FeatureOptions
	Logging        *logs.Options

	ShowVersion   bool
	SkipMigration bool
	Etcd          bool
	Kubeconfig    string
	ClusterName   string

	// dbopts
	EtcdConfig    etcd.EtcdConfig
	EtcdDir       string
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBSSLRootCert string
	DBSSLKey      string
	DBSSLCert     string

	// apiservice install config
	ServiceName           string
	ServiceNamespace      string
	StoreReports          bool
	StoreEphemeralReports bool
	StoreOpenreports      bool

	// Only to be used to for testing
	DisableAuthForTesting bool
}

func (o *Options) Validate() []error {
	errors := o.validate()
	err := logsapi.ValidateAndApply(o.Logging, nil)
	if err != nil {
		errors = append(errors, err)
	}
	return errors
}

func (o *Options) validate() []error {
	errors := []error{}
	return errors
}

func (o *Options) Flags() (fs flag.NamedFlagSets) {
	msfs := fs.FlagSet("policy server")
	msfs.StringVar(&o.ClusterName, "clustername", "", "Optional name for cluster database records")
	msfs.BoolVar(&o.Etcd, "etcd", false, "Use embedded etcd database")
	msfs.StringVar(&o.EtcdConfig.Endpoints, "etcdEndpoints", "", "Enpoints used for connect to etcd server")
	msfs.BoolVar(&o.EtcdConfig.Insecure, "etcdSkipTLS", true, "Skip TLS verification when connecting to etcd")
	msfs.BoolVar(&o.ShowVersion, "version", false, "Show version")
	msfs.StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)")
	msfs.StringVar(&o.DBSSLMode, "dbsslmode", "disable", "SSL mode of the postgres database.")
	msfs.StringVar(&o.DBSSLRootCert, "dbsslrootcert", "", "Path to database root cert.")
	msfs.StringVar(&o.DBSSLKey, "dbsslkey", "", "Path to database ssl key.")
	msfs.StringVar(&o.DBSSLCert, "dbsslcert", "", "Path to database ssl cert.")
	msfs.StringVar(&o.ServiceName, "servicename", "", "Name of the service targeted by the APIService.")
	msfs.StringVar(&o.ServiceNamespace, "servicens", "", "Namespace of the service targeted by the APIService.")
	msfs.BoolVar(&o.StoreReports, "storereports", true, "Whether or not to store and manage Policy Reports.")
	msfs.BoolVar(&o.StoreOpenreports, "storeopenreports", true, "Whether or not to store and manage Open Reports.")
	msfs.BoolVar(&o.StoreEphemeralReports, "storeephemeralreports", true, "Whether or not to store and manage Ephemeral Reports.")
	msfs.BoolVar(&o.SkipMigration, "skipmigration", false, "Skip database migration on startup.")

	o.SecureServing.AddFlags(fs.FlagSet("apiserver secure serving"))
	o.Authentication.AddFlags(fs.FlagSet("apiserver authentication"))
	o.Authorization.AddFlags(fs.FlagSet("apiserver authorization"))
	o.Audit.AddFlags(fs.FlagSet("apiserver audit log"))
	o.Features.AddFlags(fs.FlagSet("features"))
	logsapi.AddFlags(o.Logging, fs.FlagSet("logging"))

	return fs
}

// NewOptions constructs a new set of default options for reports-server.
func NewOptions() *Options {
	return &Options{
		SecureServing:  genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		Features:       genericoptions.NewFeatureOptions(),
		Audit:          genericoptions.NewAuditOptions(),
		Logging:        logs.NewOptions(),
	}
}

func (o Options) ApiserverConfig() (*genericapiserver.Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if !o.DisableAuthForTesting {
		if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
			return nil, err
		}
		if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
			return nil, err
		}
	}

	if err := o.Audit.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	versionGet := version.Get()
	serverConfig.Version = &versionGet
	// enable OpenAPI schemas
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "reports-server"
	serverConfig.OpenAPIV3Config.Info.Title = "reports-server"
	serverConfig.OpenAPIConfig.Info.Version = strings.Split(serverConfig.Version.String(), "-")[0] // TODO(directxman12): remove this once autosetting this doesn't require security definitions
	serverConfig.OpenAPIV3Config.Info.Version = strings.Split(serverConfig.Version.String(), "-")[0]

	return serverConfig, nil
}

func (o Options) RestConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error
	if len(o.Kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: o.Kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		config, err = loader.ClientConfig()
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// config.ContentType = "application/json"
	// config.AcceptContentTypes = "application/json,application/vnd.kubernetes.protobuf"

	err = rest.SetKubernetesDefaults(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// dbConfig reads the database configuration directly from environment variables
// because these configurations contain sensitive data, this is not read directly from command line input,
// to enable usecases of env variable injection, such as using vault-env
func (o *Options) DBConfig() error {
	o.DBHost = os.Getenv("DB_HOST")
	o.DBName = os.Getenv("DB_DATABASE")
	o.DBUser = os.Getenv("DB_USER")
	o.DBPassword = os.Getenv("DB_PASSWORD")
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return err
	} else {
		o.DBPort = dbPort
	}
	return nil
}
