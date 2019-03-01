package config

import (
	"errors"
	"fmt"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/logging"
	"github.com/jenkins-x-apps/jx-app-jacoco/internal/util"
	"os"
	"runtime"
	"strings"
)

var (
	settings = map[string]Setting{}
)

func init() {
	// JX namespace
	settings["Namespace"] = Setting{"TEAM_NAMESPACE", "jx", []func(interface{}, string) error{util.IsNotEmpty}}

	// Logging
	settings["Level"] = Setting{"LOG_LEVEL", "info", []func(interface{}, string) error{util.IsNotEmpty}}
}

// Setting is an element in the proxy configuration. It contains the environment
// variable from which the setting is retrieved, its default value as well as a list
// of validations which the value of this setting needs to pass.
type Setting struct {
	key          string
	defaultValue string
	validations  []func(interface{}, string) error
}

// EnvConfig is a Configuration implementation which reads the configuration from the process environment.
type EnvConfig struct {
	clusters map[string]string
}

// NewConfiguration creates a configuration instance.
func NewConfiguration() (Configuration, error) {
	// Check if we have all we need.
	multiError := verifyEnv()
	if !multiError.Empty() {
		for _, err := range multiError.Errors {
			logging.AppLogger().Error(err)
		}
		return nil, errors.New("one or more required environment variables for this configuration are missing or invalid")
	}

	config := EnvConfig{}
	return &config, nil
}

// Namespace returns the JX namespace to watch.
func (c *EnvConfig) Namespace() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// Level returns the logging level.
func (c *EnvConfig) Level() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// String returns a string representation of the configuration.
func (c *EnvConfig) String() string {
	config := map[string]interface{}{}
	for key, setting := range settings {
		value := getConfigValueFromEnv(key)
		// don't echo passwords
		if strings.Contains(setting.key, "PASSWORD") && len(value) > 0 {
			value = "***"
		}
		config[key] = value

	}
	return fmt.Sprintf("%v", config)
}

// Verify checks whether all needed config options are set.
func verifyEnv() util.MultiError {
	var errors util.MultiError
	for key, setting := range settings {
		value := getConfigValueFromEnv(key)

		for _, validateFunc := range setting.validations {
			errors.Collect(validateFunc(value, setting.key))
		}
	}

	return errors
}

func getConfigValueFromEnv(funcName string) string {
	setting := settings[funcName]

	value, ok := os.LookupEnv(setting.key)
	if !ok {
		value = setting.defaultValue
	}
	return value
}
