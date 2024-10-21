package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	kyverno "github.com/kyverno/kyverno/pkg/clients/kyverno"
	"github.com/kyverno/reports-server/pkg/api"
	"github.com/kyverno/reports-server/pkg/storage"
	"github.com/kyverno/reports-server/pkg/storage/db"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // for client-go metrics registration
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/generated/v1alpha2/clientset/versioned"
)

type Config struct {
	Apiserver *genericapiserver.Config
	Rest      *rest.Config
	Embedded  bool
	DBconfig  *db.PostgresConfig
}

func (c Config) Complete() (*server, error) {
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

	store, err := storage.New(c.Embedded, c.DBconfig)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// Embedded runs in a stateful set in high availability deployment
	// TODO: Add leader election to add embedded
	if !c.Embedded {
		klog.Info("performing migration...")
		if err := c.migration(store); err != nil {
			klog.Error(err)
			return nil, err
		}
	}

	if err := api.Install(store, genericServer); err != nil {
		klog.Error(err)
		return nil, err
	}

	s := NewServer(
		genericServer,
		store,
	)
	err = s.RegisterProbes()
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return s, nil
}

func (c Config) metricsHandler() (http.HandlerFunc, error) {
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

func (c Config) migration(store storage.Interface) error {
	kyvernoClient, err := kyverno.NewForConfig(c.Rest)
	if err != nil {
		return err
	}

	policyClient, err := versioned.NewForConfig(c.Rest)
	if err != nil {
		return err
	}

	cpolrs, err := policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	for _, c := range cpolrs.Items {
		if c.Annotations != nil {
			if _, ok := c.Annotations[api.ServedByReportsServerAnnotation]; ok {
				continue
			}
		} else {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
		err := store.ClusterPolicyReports().Create(context.TODO(), &c)
		if err != nil {
			return err
		}
	}
	err = store.SetResourceVersion(cpolrs.ResourceVersion)
	if err != nil {
		return err
	}

	cpolrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return policyClient.Wgpolicyk8sV1alpha2().ClusterPolicyReports().Watch(context.TODO(), options)
		},
	}
	cpolrWatchInterface, err := watchtools.NewRetryWatcher(cpolrs.GetResourceVersion(), cpolrWatcher)
	if err != nil {
		return err
	}
	go func() {
		for event := range cpolrWatchInterface.ResultChan() {
			switch event.Type {
			case watch.Added:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				if cpolr.Annotations != nil {
					if _, ok := cpolr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cpolr.Annotations = make(map[string]string)
				}
				cpolr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterPolicyReports().Create(context.TODO(), cpolr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				if cpolr.Annotations != nil {
					if _, ok := cpolr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cpolr.Annotations = make(map[string]string)
				}
				cpolr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterPolicyReports().Update(context.TODO(), cpolr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				cpolr := event.Object.(*v1alpha2.ClusterPolicyReport)
				if cpolr.Annotations != nil {
					if _, ok := cpolr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cpolr.Annotations = make(map[string]string)
				}
				cpolr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterPolicyReports().Delete(context.TODO(), cpolr.Name)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}()

	polrs, err := policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	for _, c := range polrs.Items {
		if c.Annotations != nil {
			if _, ok := c.Annotations[api.ServedByReportsServerAnnotation]; ok {
				continue
			}
		} else {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
		err := store.PolicyReports().Create(context.TODO(), &c)
		if err != nil {
			return err
		}
	}
	err = store.SetResourceVersion(cpolrs.ResourceVersion)
	if err != nil {
		return err
	}

	polrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return policyClient.Wgpolicyk8sV1alpha2().PolicyReports("").Watch(context.TODO(), options)
		},
	}
	polrWatchInterface, err := watchtools.NewRetryWatcher(polrs.GetResourceVersion(), polrWatcher)
	if err != nil {
		return err
	}
	go func() {
		for event := range polrWatchInterface.ResultChan() {
			switch event.Type {
			case watch.Added:
				polr := event.Object.(*v1alpha2.PolicyReport)
				if polr.Annotations != nil {
					if _, ok := polr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					polr.Annotations = make(map[string]string)
				}
				polr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.PolicyReports().Create(context.TODO(), polr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				polr := event.Object.(*v1alpha2.PolicyReport)
				if polr.Annotations != nil {
					if _, ok := polr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					polr.Annotations = make(map[string]string)
				}
				polr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.PolicyReports().Update(context.TODO(), polr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				polr := event.Object.(*v1alpha2.PolicyReport)
				if polr.Annotations != nil {
					if _, ok := polr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					polr.Annotations = make(map[string]string)
				}
				polr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.PolicyReports().Delete(context.TODO(), polr.Name, polr.Namespace)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}()

	cephrs, err := kyvernoClient.ReportsV1().ClusterEphemeralReports().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	for _, c := range cephrs.Items {
		if c.Annotations != nil {
			if _, ok := c.Annotations[api.ServedByReportsServerAnnotation]; ok {
				continue
			}
		} else {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
		err := store.ClusterEphemeralReports().Create(context.TODO(), &c)
		if err != nil {
			return err
		}
	}
	err = store.SetResourceVersion(cephrs.ResourceVersion)
	if err != nil {
		return err
	}
	cephrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kyvernoClient.ReportsV1().ClusterEphemeralReports().Watch(context.TODO(), options)
		},
	}
	cephrWatchInterface, err := watchtools.NewRetryWatcher(cephrs.GetResourceVersion(), cephrWatcher)
	if err != nil {
		return err
	}
	go func() {
		for event := range cephrWatchInterface.ResultChan() {
			switch event.Type {
			case watch.Added:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				if cephr.Annotations != nil {
					if _, ok := cephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cephr.Annotations = make(map[string]string)
				}
				cephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterEphemeralReports().Create(context.TODO(), cephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				if cephr.Annotations != nil {
					if _, ok := cephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cephr.Annotations = make(map[string]string)
				}
				cephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterEphemeralReports().Update(context.TODO(), cephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				cephr := event.Object.(*reportsv1.ClusterEphemeralReport)
				if cephr.Annotations != nil {
					if _, ok := cephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					cephr.Annotations = make(map[string]string)
				}
				cephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.ClusterEphemeralReports().Delete(context.TODO(), cephr.Name)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}()
	ephrs, err := kyvernoClient.ReportsV1().EphemeralReports("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	for _, c := range ephrs.Items {
		if c.Annotations != nil {
			if _, ok := c.Annotations[api.ServedByReportsServerAnnotation]; ok {
				continue
			}
		} else {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
		err := store.EphemeralReports().Create(context.TODO(), &c)
		if err != nil {
			return err
		}
	}
	err = store.SetResourceVersion(ephrs.ResourceVersion)
	if err != nil {
		return err
	}
	ephrWatcher := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kyvernoClient.ReportsV1().EphemeralReports("").Watch(context.TODO(), options)
		},
	}
	ephrWatchInterface, err := watchtools.NewRetryWatcher(ephrs.GetResourceVersion(), ephrWatcher)
	if err != nil {
		return err
	}
	go func() {
		for event := range ephrWatchInterface.ResultChan() {
			switch event.Type {
			case watch.Added:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				if ephr.Annotations != nil {
					if _, ok := ephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					ephr.Annotations = make(map[string]string)
				}
				ephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.EphemeralReports().Create(context.TODO(), ephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Modified:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				if ephr.Annotations != nil {
					if _, ok := ephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					ephr.Annotations = make(map[string]string)
				}
				ephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.EphemeralReports().Update(context.TODO(), ephr)
				if err != nil {
					klog.Error(err)
				}
			case watch.Deleted:
				ephr := event.Object.(*reportsv1.EphemeralReport)
				if ephr.Annotations != nil {
					if _, ok := ephr.Annotations[api.ServedByReportsServerAnnotation]; ok {
						return
					}
				} else {
					ephr.Annotations = make(map[string]string)
				}
				ephr.Annotations[api.ServedByReportsServerAnnotation] = api.ServedByReportsServerValue
				err := store.EphemeralReports().Delete(context.TODO(), ephr.Name, ephr.Namespace)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}()
	rv, err := strconv.ParseUint(store.UseResourceVersion(), 10, 64)
	if err != nil {
		return err
	}
	// use leave some versions for resources added using watchers
	return store.SetResourceVersion(fmt.Sprint((rv + 9999)))
}
