package server

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-multierror"
	"github.com/kyverno/reports-server/pkg/api"
	"github.com/kyverno/reports-server/pkg/app/opts"
	"github.com/kyverno/reports-server/pkg/storage"
	"github.com/kyverno/reports-server/pkg/storage/db"
	"github.com/kyverno/reports-server/pkg/storage/etcd"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // for client-go metrics registration
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1client "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

const numWorkers = 50

type Config struct {
	Apiserver   *genericapiserver.Config
	Rest        *rest.Config
	Embedded    bool
	EtcdConfig  *etcd.EtcdConfig
	DBconfig    *db.PostgresConfig
	ClusterName string
	APIServices APIServices
	Store       storage.Interface
}

func NewServerConfig(o opts.Options) (*Config, error) {
	apiserver, err := o.ApiserverConfig()
	if err != nil {
		return nil, err
	}
	restConfig, err := o.RestConfig()
	if err != nil {
		return nil, err
	}
	err = o.DBConfig()
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

	apiservices := BuildApiServices(o.ServiceName, o.ServiceNamespace)
	apiservices.StoreReports = o.StoreReports
	apiservices.StoreEphemeralReports = o.StoreEphemeralReports
	apiservices.StoreOpenreports = o.StoreOpenreports

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	kubeSystem, err := client.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	store, err := storage.New(o.Etcd, dbconfig, &o.EtcdConfig, string(kubeSystem.GetUID()), o.ClusterName)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Apiserver:   apiserver,
		Rest:        restConfig,
		Embedded:    o.Etcd,
		EtcdConfig:  &o.EtcdConfig,
		DBconfig:    dbconfig,
		ClusterName: o.ClusterName,
		APIServices: apiservices,
		Store:       store,
	}

	return config, nil
}

func (c *Config) Complete() (*server, error) {
	// Disable default metrics handler and create custom one
	c.Apiserver.EnableMetrics = false
	metricsHandler, err := c.metricsHandler()
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	genericServer, err := c.Apiserver.Complete(nil).New("reports-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	genericServer.Handler.NonGoRestfulMux.HandleFunc("/metrics", metricsHandler)

	klog.Info("performing migration...")
	if err := c.migration(context.TODO()); err != nil {
		klog.Error(err)
		return nil, err
	}

	klog.Info("checking APIServices...")
	if err := c.installApiServices(); err != nil {
		klog.Error(err)
		return nil, err
	}

	if err := api.Install(c.Store, genericServer); err != nil {
		klog.Error(err)
		return nil, err
	}

	s := NewServer(
		genericServer,
		c.Store,
	)
	err = s.RegisterProbes()
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return s, nil
}

func (c *Config) CleanupApiServices() error {
	apiRegClient, err := apiregistrationv1client.NewForConfig(c.Rest)
	if err != nil {
		return err
	}
	var finalErr error

	if c.APIServices.StoreReports {
		if err := c.toLocalApiService(c.APIServices.wgpolicyApiService.GetName(), *apiRegClient); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}
	if c.APIServices.StoreEphemeralReports {
		if err := c.toLocalApiService(c.APIServices.v1ReportsApiService.GetName(), *apiRegClient); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}
	if c.APIServices.StoreOpenreports {
		if err := c.toLocalApiService(c.APIServices.openreportsApiService.GetName(), *apiRegClient); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}

	return finalErr
}

func (c *Config) metricsHandler() (http.HandlerFunc, error) {
	// Create registry for Policy Server metrics
	registry := metrics.NewKubeRegistry()
	err := RegisterMetrics(registry)
	if err != nil {
		return nil, err
	}
	// Register apiserver metrics in legacy registry
	apimetrics.Register()

	// Return handler that serves metrics from both legacy and Metrics Server registry
	return func(w http.ResponseWriter, req *http.Request) {
		legacyregistry.Handler().ServeHTTP(w, req)
		metrics.HandlerFor(registry, metrics.HandlerOpts{}).ServeHTTP(w, req)
	}, nil
}

func applyReportsServerAnnotation(o metav1.Object) {
	a := o.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
	o.SetAnnotations(a)
}

func (c *Config) installApiServices() error {
	apiRegClient, err := apiregistrationv1client.NewForConfig(c.Rest)
	if err != nil {
		return err
	}
	if err := c.createOrDeleteApiservice(c.APIServices.wgpolicyApiService, *apiRegClient, c.APIServices.StoreReports); err != nil {
		return err
	}
	if err := c.createOrDeleteApiservice(c.APIServices.v1ReportsApiService, *apiRegClient, c.APIServices.StoreEphemeralReports); err != nil {
		return err
	}
	if err := c.createOrDeleteApiservice(c.APIServices.openreportsApiService, *apiRegClient, c.APIServices.StoreOpenreports); err != nil {
		return err
	}

	return nil
}

func (c *Config) toLocalApiService(apiSvcName string, client apiregistrationv1client.ApiregistrationV1Client) error {
	_, err := client.APIServices().Get(context.TODO(), apiSvcName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	patch := []byte(`{
		"spec": {
		  "service": null,
		  "insecureSkipTLSVerify": false
		}
	  }`)

	_, err = client.APIServices().Patch(context.TODO(), apiSvcName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) createOrDeleteApiservice(apiservice apiregistrationv1.APIService, client apiregistrationv1client.ApiregistrationV1Client, enabled bool) error {
	inClusterApiService, err := client.APIServices().Get(context.TODO(), apiservice.GetName(), metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if enabled {
		if errors.IsNotFound(err) {
			_, err = client.APIServices().Create(context.TODO(), &apiservice, metav1.CreateOptions{})
			if err != nil {
				if !errors.IsAlreadyExists(err) {
					klog.Errorf("Error creating APIService %s: %v", apiservice.GetName(), err)
					return err
				}
			} else {
				klog.Infof("APIService %s created successfully", apiservice.GetName())
			}
		} else {
			// APIService already exists, update it
			if inClusterApiService.Spec.Service == nil || inClusterApiService.Spec.Service.Name != apiservice.Spec.Service.Name || inClusterApiService.Spec.Service.Namespace != apiservice.Spec.Service.Namespace {
				apiservice.SetResourceVersion(inClusterApiService.GetResourceVersion())
				_, err = client.APIServices().Update(context.TODO(), &apiservice, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
				klog.Infof("Updated existing APIService %s", apiservice.GetName())
			}
		}
	} else if err == nil {
		klog.Infof("APIService for %s is installed, but storing reports has been disabled via configuration, deleting APIService...", apiservice.GetName())
		// Delete the APIService since we're no longer managing this object, APIServer will automatically create a local automanaged APIService if CRD is installed
		err = client.APIServices().Delete(context.TODO(), apiservice.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
