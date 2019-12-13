package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/pkg/apis/eventlogger/v1"
	c "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/bakito/k8s-event-logger-operator/pkg/controller/event"
	"github.com/bakito/k8s-event-logger-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

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

	configFilePath, ok := os.LookupEnv(c.EnvConfigFilePath)
	if !ok {
		log.Error(fmt.Errorf("config path env variable '%s' not set", c.EnvConfigFilePath), "")
		os.Exit(1)
	}

	if _, err := os.Stat(configFilePath); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	config := &eventloggerv1.EventLoggerConf{}
	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if _, ok := os.LookupEnv("DEBUG_CONFIG"); ok {
		log.WithValues("file", configFilePath, "config", config).Info("Current configuration")
	}

	log.Info("Registering Components.")

	// Setup all Controllers
	if err := event.Add(mgr, config); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
