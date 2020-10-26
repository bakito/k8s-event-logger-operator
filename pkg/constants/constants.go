package constants

const (
	// ArgEnableLoggerMode enable logger mode
	ArgEnableLoggerMode = "enable-logger-mode"

	// ArgMetricsAddr metrics address
	ArgMetricsAddr = "metrics-addr"

	// ArgConfigName name of the config
	ArgConfigName = "config-name"

	// DefaultMetricsAddr default metrics address
	DefaultMetricsAddr = ":8080"

	// ArgEnableLeaderElection enable leader election
	ArgEnableLeaderElection = "enable-leader-election"

	// ArgEnableProfiling enable profiling
	ArgEnableProfiling = "enable-profiling"

	// EnvWatchNamespace watch namespace env variable
	EnvWatchNamespace = "WATCH_NAMESPACE"

	// EnvEventLoggerImage env variable name for the image if the event logger
	EnvEventLoggerImage = "EVENT_LOGGER_IMAGE"

	// EnvLoggerPodReqCPU set the logger pod request cpu
	EnvLoggerPodReqCPU = "LOGGER_POD_REQUEST_CPU"

	// EnvLoggerPodReqMem set the logger pod request memory
	EnvLoggerPodReqMem = "LOGGER_POD_REQUEST_MEM"

	// EnvLoggerPodMaxCPU set the logger pod max cpu
	EnvLoggerPodMaxCPU = "LOGGER_POD_LIMIT_CPU"

	// EnvLoggerPodMaxMem set the logger pod max memory
	EnvLoggerPodMaxMem = "LOGGER_POD_LIMIT_MEM"

	// EnvEnableWebhook enable webhooks
	EnvEnableWebhook = "LOGGER_POD_LIMIT_MEM"
)
