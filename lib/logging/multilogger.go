package logging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"github.com/sirupsen/logrus"
)

type LogInfo map[string]interface{}

/*
Application wide logging package
*/

func init() {
	ConfigureLogging(false, "info")
}

type multiLogger struct {
	loggers []*logrus.Logger
}

func newMultiLogger() *multiLogger {
	return &multiLogger{}
}

func (multi *multiLogger) addLogger(logger *logrus.Logger) {
	multi.loggers = append(multi.loggers, logger)
}

var multi *multiLogger



type enumVerbosity int

const (
	V_ enumVerbosity = -1
	V0 enumVerbosity = 0
	V1 enumVerbosity = 1
	V2 enumVerbosity = 2
)

var _verbosity enumVerbosity

func InitApiTestLogging(v int) error {
	switch v {
	case 0:
		_verbosity = V0
	case 1:
		_verbosity = V1
	case 2:
		_verbosity = V2
	case -1:
		_verbosity = V_
	default:
		return fmt.Errorf("verbosity must be -1, 0, 1 or 2")
	}
	if v >= 0 {
		if err := SetLevel("debug"); err != nil {
			return err
		}
	}
	return nil
}

func DebugWithVerbosity(verbosity enumVerbosity, msg string) {
	if verbosity == _verbosity {
		Debug(msg)
	}
}

func DebugWithVerbosityf(verbosity enumVerbosity, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	DebugWithVerbosity(verbosity, msg)
}


//Initializes Logging with global config
func ConfigureLogging(
	consoleEnabled bool,
	consoleLevel string) (err error) {

	multi = newMultiLogger()

	if consoleEnabled {
		consoleLogger, err := newConsoleLogger(consoleLevel)
		if err != nil {
			return fmt.Errorf("error adding consoleLogger: %s", err)
		}
		multi.addLogger(consoleLogger)
		log.Printf("Console: { enabled: %t; level: %s }\n", consoleEnabled, consoleLevel)
	}

	return nil
}

//Set Loglevel for active loggers
func SetLevel(lvl string) error {
	logLevel, err := logrus.ParseLevel(lvl)
	if err != nil {
		return err
	}
	for _, logger := range multi.loggers {
		logger.SetLevel(logLevel)
	}
	return nil
}

//Set Output for active loggers; needed to capture logging in Unittests
func SetOutput(buf *bytes.Buffer) {
	for _, logger := range multi.loggers {
		logger.SetOutput(buf)
	}
}

func newConsoleLogger(lvl string) (*logrus.Logger, error) {
	logLevel, err := logrus.ParseLevel(lvl)
	if err != nil {
		return nil, fmt.Errorf("error parsing loglevel: %s", err)
	}
	logger := logrus.New()
	logger.Out = os.Stderr
	logger.SetLevel(logLevel)
	return logger, nil
}

//Method to allow subpackages to intercept logging statements
func AddHook(hook logrus.Hook) {
	logger := logrus.New()
	logger.Out = ioutil.Discard
	logger.AddHook(hook)
	multi.addLogger(logger)
}

//LOGRUS Package Methods
// includes Debug, Info, Warn, Error and the *f variants; NOT the *ln variants

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Debug(args...)
	}
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Info(args...)
	}
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Warn(args...)
	}
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Error(args...)
	}
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Debugf(format, args...)
	}
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Infof(format, args...)
	}
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Warnf(format, args...)
	}
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	for _, logger := range multi.loggers {
		logger.Errorf(format, args...)
	}
}

func InfoWithFieldsF(fields map[string]interface{}, msg string, args ...interface{}) {
	for _, logger := range multi.loggers {
		entry := logrus.NewEntry(logger)
		for key, val := range fields {
			entry = entry.WithFields(logrus.Fields{key: val})
		}
		entry.Infof(msg, args...)
	}
}

func DebugWithFieldsF(fields map[string]interface{}, msg string, args ...interface{}) {
	for _, logger := range multi.loggers {
		entry := logrus.NewEntry(logger)
		for key, val := range fields {
			entry = entry.WithFields(logrus.Fields{key: val})
		}
		entry.Debugf(msg, args...)
	}
}

func ErrorWithFieldsF(fields map[string]interface{}, msg string, args ...interface{}) {
	for _, logger := range multi.loggers {
		entry := logrus.NewEntry(logger)
		for key, val := range fields {
			entry = entry.WithFields(logrus.Fields{key: val})
		}
		entry.Errorf(msg, args...)
	}
}

func WarnWithFieldsF(fields map[string]interface{}, msg string, args ...interface{}) {
	for _, logger := range multi.loggers {
		entry := logrus.NewEntry(logger)
		for key, val := range fields {
			entry = entry.WithFields(logrus.Fields{key: val})
		}
		entry.Warnf(msg, args...)
	}
}
