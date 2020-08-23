/*


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
	"flag"
	"fmt"
	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	"github.com/bakito/k8s-event-logger-operator/controllers/event"
	"github.com/bakito/k8s-event-logger-operator/controllers/eventlogger"
	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	gr "runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(eventloggerv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var configName string
	var enableLeaderElection bool
	var enableLoggerMode bool
	flag.StringVar(&metricsAddr, cnst.ArgMetricsAddr, cnst.DefaultMetricsAddr, "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, cnst.ArgEnableLeaderElection, false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableLoggerMode, cnst.ArgEnableLoggerMode, false,
		"Enable logger mode. Enabling this will only log events of the current namespace.")
	flag.StringVar(&configName, cnst.ArgConfigName, "",
		"The name of the eventlogger config to work with.")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	printVersion()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     !enableLoggerMode && enableLeaderElection,
		LeaderElectionID:   "9a62a63a.bakito.ch",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	watchNamespace := os.Getenv(cnst.EnvWatchNamespace)

	if enableLoggerMode {
		setupLog.WithValues("configName", configName).V(4).Info("Current configuration")
		if err = (&event.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Event"),
			Scheme: mgr.GetScheme(),
			Config: event.ConfigFor(watchNamespace, configName),
		}).SetupWithManager(mgr, watchNamespace); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Event")
			os.Exit(1)
		}
	} else {
		// Setup all Controllers
		if watchNamespace == "" {
			if err = (&eventlogger.Reconciler{
				Client: mgr.GetClient(),
				Log:    ctrl.Log.WithName("controllers").WithName("Pod"),
				Scheme: mgr.GetScheme(),
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "Pod")
				os.Exit(1)
			}
			setupLog.Info("Running in global mode.")

			if os.Getenv("ENABLE_WEBHOOKS") != "false" {
				if err = (&eventloggerv1.EventLogger{}).SetupWebhookWithManager(mgr); err != nil {
					setupLog.Error(err, "unable to create webhook", "webhook", "EventLogger")
					os.Exit(1)
				}
			}

		} else {
			if err = (&event.Reconciler{
				Client: mgr.GetClient(),
				Log:    ctrl.Log.WithName("controllers").WithName("Event"),
				Scheme: mgr.GetScheme(),
				Config: event.ConfigFor(watchNamespace, ""),
			}).SetupWithManager(mgr, watchNamespace); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "Event")
				os.Exit(1)
			}
			setupLog.WithValues("namespace", watchNamespace).Info("Running in single namespace mode.")
		}
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	setupLog.Info(fmt.Sprintf("Go Version: %s", gr.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", gr.GOOS, gr.GOARCH))
}
