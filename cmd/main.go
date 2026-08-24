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

	"k8s.io/client-go/discovery"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	"gitlab.com/emeland/k8s-model/internal/crdcheck"
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
	var helmReleaseScanning bool
	var subscriberURLs string
	var rbacWhitelistPath string
	var crdChecklistRaw string
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
	flag.BoolVar(&helmReleaseScanning, "helm-release-scanning",
		envOrDefault("FEATURE_HELM_RELEASE_SCANNING", "true") == "true",
		"Enable Helm release scanning to create SystemInstances from installed releases.")
	flag.StringVar(&subscriberURLs, "subscriber-urls", envOrDefault("SUBSCRIBER_URLS", ""),
		"Comma-separated downstream modelsrv base API URLs to register as replication subscribers "+
			"(e.g. http://host:8080/api).")
	flag.StringVar(&rbacWhitelistPath, "rbac-whitelist", envOrDefault("RBAC_WHITELIST", ""),
		"Path to a YAML file containing name patterns for RBAC resources that should not generate findings.")
	flag.StringVar(&crdChecklistRaw, "crd-checklist", envOrDefault("CRD_CHECKLIST", ""),
		"Comma-separated list of CRDs to check (format: group/version/resource). "+
			"Overrides the built-in default checklist when set.")
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

	if err := registerSubscribers(b.GetEventManager(), parseCommaSeparatedList(subscriberURLs)); err != nil {
		setupLog.Error(err, "unable to register replication subscribers")
		os.Exit(1)
	}

	emModel := b.GetModel()
	nameIndex := controller.NewNameIndex()
	ruleRepo := controller.NewRuleRepo()
	evaluator := controller.NewEvaluator(emModel)

	rbacWhitelist := mustLoadRBACWhitelist(rbacWhitelistPath)

	sensorID, err := sensor.Register(emModel)
	if err != nil {
		setupLog.Error(err, "unable to register sensor identity")
		os.Exit(1)
	}

	// Check which expected CRDs are available in the cluster.
	crdChecklist := crdcheck.DefaultChecklist
	if crdChecklistRaw != "" {
		parsed, err := crdcheck.ParseChecklist(crdChecklistRaw)
		if err != nil {
			setupLog.Error(err, "unable to parse --crd-checklist")
			os.Exit(1)
		}
		crdChecklist = parsed
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create discovery client for CRD check")
	} else {
		crdResult := crdcheck.Check(context.Background(), discoveryClient, crdChecklist)
		crdcheck.LogAndReport(setupLog, emModel, crdResult)
	}

	c := mgr.GetClient()
	s := mgr.GetScheme()

	crdControllers := []struct {
		name string
		r    interface{ SetupWithManager(ctrl.Manager) error }
	}{
		{
			"System",
			&controller.SystemReconciler{
				Client:   c,
				Scheme:   s,
				Model:    emModel,
				Index:    nameIndex,
				RuleEval: controller.NewRuleEvaluation(ruleRepo, evaluator, "structure.emeland.io/systems"),
			},
		},
		{
			"API",
			&controller.APIReconciler{
				Client:   c,
				Scheme:   s,
				Model:    emModel,
				Index:    nameIndex,
				RuleEval: controller.NewRuleEvaluation(ruleRepo, evaluator, "structure.emeland.io/apis"),
			},
		},
		{
			"Component",
			&controller.ComponentReconciler{
				Client:   c,
				Scheme:   s,
				Model:    emModel,
				Index:    nameIndex,
				RuleEval: controller.NewRuleEvaluation(ruleRepo, evaluator, "structure.emeland.io/components"),
			},
		},
		{
			"SystemInstance",
			&controller.SystemInstanceReconciler{
				Client:   c,
				Scheme:   s,
				Model:    emModel,
				Index:    nameIndex,
				RuleEval: controller.NewRuleEvaluation(ruleRepo, evaluator, "structure.emeland.io/systeminstances"),
			},
		},
	}
	for _, cc := range crdControllers {
		if err = cc.r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", cc.name)
			os.Exit(1)
		}
	}

	workloads := []struct {
		kind         string
		resourceType string
		prototype    client.Object
		skipFunc     func(client.Object) bool
	}{
		{"Deployment", "apps/deployments", &appsv1.Deployment{}, nil},
		{"StatefulSet", "apps/statefulsets", &appsv1.StatefulSet{}, nil},
		{"DaemonSet", "apps/daemonsets", &appsv1.DaemonSet{}, nil},
		{"CronJob", "batch/cronjobs", &batchv1.CronJob{}, nil},
		{"Job", "batch/jobs", &batchv1.Job{}, controller.IsOwnedByCronJob},
	}
	for _, w := range workloads {
		r := controller.NewWorkloadReconciler(c, s, emModel, nameIndex, w.prototype, w.kind, w.skipFunc)
		r.RuleEval = controller.NewRuleEvaluation(ruleRepo, evaluator, w.resourceType)
		if err = r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", w.kind)
			os.Exit(1)
		}
	}

	apiResources := []struct {
		kind         string
		resourceType string
		prototype    client.Object
	}{
		{"Service", "/services", &corev1.Service{}},
		{"Ingress", "networking.k8s.io/ingresses", &networkingv1.Ingress{}},
	}
	for _, a := range apiResources {
		r := controller.NewAPIInstanceReconciler(c, s, emModel, nameIndex, a.prototype, a.kind)
		r.RuleEval = controller.NewRuleEvaluation(ruleRepo, evaluator, a.resourceType)
		if err = r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", a.kind)
			os.Exit(1)
		}
	}

	// RBAC controllers: map K8s Role/ClusterRole to EmELand Role,
	// and RoleBinding/ClusterRoleBinding to EmELand Binding.

	// Register well-known FindingTypes so they exist even with no active findings.
	if err = controller.RegisterRBACFindingTypes(emModel); err != nil {
		setupLog.Error(err, "unable to register RBAC finding types")
		os.Exit(1)
	}
	if err = controller.RegisterReferenceFindingTypes(emModel); err != nil {
		setupLog.Error(err, "unable to register reference finding types")
		os.Exit(1)
	}

	// Create binding reconcilers first so we can wire them into the role reconcilers.
	rbacBindings := []struct {
		kind      string
		prototype client.Object
	}{
		{"RoleBinding", &rbacv1.RoleBinding{}},
		{"ClusterRoleBinding", &rbacv1.ClusterRoleBinding{}},
	}
	var bindingReconcilers = make([]*controller.RoleBindingReconciler, 0, len(rbacBindings))
	for _, rb := range rbacBindings {
		rbc := controller.NewRoleBindingReconciler(c, s, emModel, nameIndex, rb.prototype, rb.kind, rbacWhitelist)
		if err = rbc.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", rb.kind)
			os.Exit(1)
		}
		bindingReconcilers = append(bindingReconcilers, rbc)
	}

	rbacRoles := []struct {
		kind      string
		prototype client.Object
	}{
		{"Role", &rbacv1.Role{}},
		{"ClusterRole", &rbacv1.ClusterRole{}},
	}
	for _, rr := range rbacRoles {
		rc := controller.NewRoleReconciler(c, s, emModel, nameIndex, rr.prototype, rr.kind, rbacWhitelist)
		// Wire binding reconcilers so late-arriving roles can re-enqueue
		// pending bindings. Routing picks the matching reconciler by scope.
		for _, rbc := range bindingReconcilers {
			rc.SetBindingReconciler(rbc)
		}
		if err = rc.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", rr.kind)
			os.Exit(1)
		}
	}

	if err = (&controller.NamespaceReconciler{
		Client:   c,
		Scheme:   s,
		Model:    emModel,
		Index:    nameIndex,
		RuleEval: controller.NewRuleEvaluation(ruleRepo, evaluator, "/namespaces"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}

	if err = controller.NewFindingRuleWatcher(mgr, ruleRepo); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FindingRule")
		os.Exit(1)
	}

	if helmReleaseScanning {
		setupLog.Info("Helm release scanning enabled")
		if err = (&controller.HelmReleaseReconciler{
			Client: c,
			Scheme: s,
			Model:  emModel,
			Index:  nameIndex,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "HelmRelease")
			os.Exit(1)
		}
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
	if err := sensorID.Close(); err != nil {
		setupLog.Error(err, "problem deregistering sensor node")
	}
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

func mustLoadRBACWhitelist(path string) *controller.RBACWhitelist {
	wl, err := controller.LoadRBACWhitelist(path)
	if err != nil {
		setupLog.Error(err, "unable to load RBAC whitelist", "path", path)
		os.Exit(1)
	}
	if path != "" {
		setupLog.Info("loaded RBAC whitelist", "path", path)
	}
	return wl
}
