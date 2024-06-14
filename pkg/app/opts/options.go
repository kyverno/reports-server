package opts

import (
	"fmt"
	"net"
	"strings"

	"github.com/kyverno/reports-server/pkg/api"
	generatedopenapi "github.com/kyverno/reports-server/pkg/api/generated/openapi"
	"github.com/kyverno/reports-server/pkg/server"
	"github.com/kyverno/reports-server/pkg/storage/db"
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

	ShowVersion bool
	Debug       bool
	Kubeconfig  string

	// dbopts
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBSSLRootCert string
	DBSSLKey      string
	DBSSLCert     string

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
	msfs.BoolVar(&o.Debug, "debug", false, "Use inmemory database for debugging")
	msfs.BoolVar(&o.ShowVersion, "version", false, "Show version")
	msfs.StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)")
	msfs.StringVar(&o.DBHost, "dbhost", "reportsdb.kyverno", "Host url of postgres instance")
	msfs.IntVar(&o.DBPort, "dbport", 5432, "Port of the postgres instance")
	msfs.StringVar(&o.DBUser, "dbuser", "postgres", "Username to login into postgres")
	msfs.StringVar(&o.DBPassword, "dbpassword", "password", "Password to login into postgres")
	msfs.StringVar(&o.DBName, "dbname", "reportsdb", "Name of the database to store policy reports in")
	msfs.StringVar(&o.DBSSLMode, "dbsslmode", "disable", "SSL mode of the postgres database.")
	msfs.StringVar(&o.DBSSLRootCert, "dbsslrootcert", "", "Path to database root cert.")
	msfs.StringVar(&o.DBSSLKey, "dbsslkey", "", "Path to database ssl key.")
	msfs.StringVar(&o.DBSSLCert, "dbsslcert", "", "Path to database ssl cert.")

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

func (o Options) ServerConfig() (*server.Config, error) {
	apiserver, err := o.ApiserverConfig()
	if err != nil {
		return nil, err
	}
	restConfig, err := o.restConfig()
	if err != nil {
		return nil, err
	}

	dbconfig := &db.PostgresConfig{
		Host:        o.DBHost,
		Port:        o.DBPort,
		User:        o.DBUser,
		Password:    o.DBPassword,
		DBname:      o.DBName,
		SSLMode:     o.DBSSLMode,
		SSLRootCert: o.DBSSLRootCert,
		SSLKey:      o.DBSSLKey,
		SSLCert:     o.DBSSLCert,
	}

	return &server.Config{
		Apiserver: apiserver,
		Rest:      restConfig,
		Debug:     o.Debug,
		DBconfig:  dbconfig,
	}, nil
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

func (o Options) restConfig() (*rest.Config, error) {
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
