package constants

const (
	// EnvConfigFilePath env variable name for config path
	EnvConfigFilePath = "CONFIG_PATH"

	// EnvEventLoggerImage env variable name for the image if the event logger
	EnvEventLoggerImage = "EVENT_LOGGER_IMAGE"

	// MetricsHost host for the metrics
	MetricsHost = "0.0.0.0"

	// MetricsPort port for the metrics
	MetricsPort int32 = 8383

	// OperatorMetricsPort port for the operator metrics
	OperatorMetricsPort int32 = 8686
)
