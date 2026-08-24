// Command tenant-controller runs the Tenant operator against the current kubeconfig.
package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	platformv1alpha1 "github.com/mohammed/tenant-controller/api/v1alpha1"
	"github.com/mohammed/tenant-controller/controllers"
)

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "Address for the metrics endpoint; 0 disables it.")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "unable to add core types to scheme")
		os.Exit(1)
	}
	if err := platformv1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "unable to add Tenant types to scheme")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: metricsAddr},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	r := &controllers.TenantReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
	if err := r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
