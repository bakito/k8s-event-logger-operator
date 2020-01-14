package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bakito/k8s-event-logger-operator/cmd/cli"
	"github.com/bakito/k8s-event-logger-operator/pkg/apis"
	c "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/event"
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/pod"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logf.Log.WithName("cmd")

func main() {

	cli.InitLogging()

	cli.PrintVersion(log)

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "event-logger-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", c.MetricsHost, c.MetricsPort),
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

	// Setup all Controllers
	if namespace == "" {
		if err := pod.Add(mgr); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		log.Info("Running in global mode.")
	} else {
		if err := event.Add(mgr, ""); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		log.WithValues("namespace", namespace).Info("Running in single namespace mode.")
	}

	log.V(4).Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
