/*
Copyright 2025 Lutz Behnke <lutz.behnke@emeland.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/endpoint"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/controller"
	"gitlab.com/emeland/k8s-model/internal/sensor"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(structurev1alpha1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var apiAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var allowInboundPush bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&apiAddr, "api-bind-address", envOrDefault("API_ADDR", ":8080"),
		"The address the modelsrv REST API binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.BoolVar(&allowInboundPush, "allow-inbound-push", false,
		"If set, allow POST /api/events/push (inbound replication). Default false: sensor is replication source only.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "64445a55.emeland.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	b, err := backend.New()
	if err != nil {
		setupLog.Error(err, "unable to create modelsrv backend")
		os.Exit(1)
	}

	emModel := b.GetModel()
	nameIndex := controller.NewNameIndex()

	c := mgr.GetClient()
	s := mgr.GetScheme()

	crdControllers := []struct {
		name string
		r    interface{ SetupWithManager(ctrl.Manager) error }
	}{
		{"System", &controller.SystemReconciler{Client: c, Scheme: s, Model: emModel, Index: nameIndex}},
		{"API", &controller.APIReconciler{Client: c, Scheme: s, Model: emModel, Index: nameIndex}},
		{"Component", &controller.ComponentReconciler{Client: c, Scheme: s, Model: emModel, Index: nameIndex}},
		{"SystemInstance", &controller.SystemInstanceReconciler{Client: c, Scheme: s, Model: emModel, Index: nameIndex}},
	}
	for _, cc := range crdControllers {
		if err = cc.r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", cc.name)
			os.Exit(1)
		}
	}

	workloads := []struct {
		kind      string
		prototype client.Object
		skipFunc  func(client.Object) bool
	}{
		{"Deployment", &appsv1.Deployment{}, nil},
		{"StatefulSet", &appsv1.StatefulSet{}, nil},
		{"DaemonSet", &appsv1.DaemonSet{}, nil},
		{"CronJob", &batchv1.CronJob{}, nil},
		{"Job", &batchv1.Job{}, controller.IsOwnedByCronJob},
	}
	for _, w := range workloads {
		r := controller.NewWorkloadReconciler(c, s, emModel, nameIndex, w.prototype, w.kind, w.skipFunc)
		if err = r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", w.kind)
			os.Exit(1)
		}
	}

	apiResources := []struct {
		kind      string
		prototype client.Object
	}{
		{"Service", &corev1.Service{}},
		{"Ingress", &networkingv1.Ingress{}},
	}
	for _, a := range apiResources {
		r := controller.NewAPIInstanceReconciler(c, s, emModel, nameIndex, a.prototype, a.kind)
		if err = r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", a.kind)
			os.Exit(1)
		}
	}

	if err = (&controller.NamespaceReconciler{
		Client: c,
		Scheme: s,
		Model:  emModel,
		Index:  nameIndex,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	apiServer, apiListenAddr, err := startAPIServer(b, apiAddr, allowInboundPush)
	if err != nil {
		setupLog.Error(err, "unable to start modelsrv API")
		os.Exit(1)
	}
	setupLog.Info("modelsrv API listening", "address", apiListenAddr)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		setupLog.Error(err, "problem shutting down modelsrv API")
	}
}

func startAPIServer(b backend.Backend, addr string, allowInboundPush bool) (*http.Server, string, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", fmt.Errorf("listen %s: %w", addr, err)
	}

	baseURL := fmt.Sprintf("http://%s/api", ln.Addr().String())
	handler := endpoint.NewHandler(b.GetModel(), b.GetEventManager(), baseURL, endpoint.WebListenerOptions{})
	wrapped := sensor.ReplicationGuard{
		Handler:          handler,
		AllowInboundPush: allowInboundPush,
	}

	srv := &http.Server{Handler: wrapped}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			setupLog.Error(err, "modelsrv API server error")
			os.Exit(1)
		}
	}()

	return srv, ln.Addr().String(), nil
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
