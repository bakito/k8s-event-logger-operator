package constants

const (

	// EnvLeaderElectionResourceLock leader election release lock mode
	EnvLeaderElectionResourceLock = "LEADER_ELECTION_RESOURCE_LOCK"

	// ArgEnableLoggerMode enable logger mode
	ArgEnableLoggerMode = "enable-logger-mode"

	// ArgConfigName name of the config
	ArgConfigName = "config-name"

	// ArgMetricsAddr metrics address
	ArgMetricsAddr = "metrics-addr"

	// DefaultMetricsAddr default metrics address
	DefaultMetricsAddr = ":8080"

	// ArgHealthAddr health address
	ArgHealthAddr = "health-addr"

	// DefaultHealthAddr default health address
	DefaultHealthAddr = ":8081"

	// ArgProfilingAddr profiling address
	ArgProfilingAddr = "profiling-addr"

	// DefaultProfilingAddr default profiling address
	DefaultProfilingAddr = ":8082"

	// ArgEnableLeaderElection enable leader election
	ArgEnableLeaderElection = "enable-leader-election"

	// ArgEnableProfiling enable profiling
	ArgEnableProfiling = "enable-profiling"

	// EnvWatchNamespace watch namespace env variable
	EnvWatchNamespace = "WATCH_NAMESPACE"

	// EnvEventLoggerImage env variable name for the image if the event logger
	EnvEventLoggerImage = "EVENT_LOGGER_IMAGE"

	// EnvEnableWebhook enable webhooks
	EnvEnableWebhook = "ENABLE_WEBHOOKS"

	// EnvPodName the name the pod
	EnvPodName = "POD_NAME"

	// EnvPodNamespace the namespace the pod
	EnvPodNamespace = "POD_NAMESPACE"

	// EnvConfigMapName the name of the configmap
	EnvConfigMapName = "CONFIG_MAP_NAME"

	// EnvConfigReload watch the configmap for changes
	EnvConfigReload = "CONFIG_RELOAD"

	// ConfigKeyContainerTemplate pod template config key
	ConfigKeyContainerTemplate = "container_template.yaml"
)
