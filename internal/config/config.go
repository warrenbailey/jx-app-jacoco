package config

// Configuration declares the configuration properties of this app.
type Configuration interface {
	JXConfig
	LogConfig

	// String returns a string representation of the configuration.
	String() string
}

// JXConfig defines JX specific configuration.
type JXConfig interface {
	// Namespace returns the JX namespace to watch.
	Namespace() string
}

// LogConfig defines the logging configuration.
type LogConfig interface {
	// Level returns the logging level.
	Level() string
}
