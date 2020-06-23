package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bakito/k8s-event-logger-operator/pkg/apis"
	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/event"
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/pod"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logf.Log.WithName("cmd")

func main() {
	InitLogging()
	PrintVersion(log)

	loggerMode := false
	if mode, ok := os.LookupEnv(cnst.EnvEventLoggerMode); ok {
		loggerMode = cnst.ModeLogger == mode
	}

	wNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	opNamespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if err == k8sutil.ErrRunLocal {
			opNamespace = os.Getenv(cnst.EnvDevOperatorNamespace)
		} else {
			log.Error(err, "Failed to get operator namespace")
			os.Exit(1)
		}
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if !loggerMode {
		ctx := context.TODO()
		// Become the leader before proceeding
		err = leader.Become(ctx, "event-logger-operator-lock")
		if err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          wNamespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", cnst.MetricsHost, cnst.MetricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.V(4).Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.V(4).Info("Registering Components.")
	if loggerMode {
		configName := os.Getenv(cnst.EnvConfigName)
		if _, ok := os.LookupEnv(cnst.EnvDebugConfig); ok {
			log.WithValues("configName", configName).Info("Current configuration")
		}
		// Setup all Controllers
		if err := event.Add(mgr, opNamespace, configName); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	} else {
		// Setup all Controllers
		if wNamespace == "" {
			if err := pod.Add(mgr); err != nil {
				log.Error(err, "")
				os.Exit(1)
			}
			log.Info("Running in global mode.")
		} else {
			if err := event.Add(mgr, wNamespace, ""); err != nil {
				log.Error(err, "")
				os.Exit(1)
			}
			log.WithValues("namespace", wNamespace).Info("Running in single namespace mode.")
		}
	}
	log.V(4).Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
