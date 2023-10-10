package fkt

import (
	"os"
	"fmt"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type LogConfig struct {
	Level  LogLevel  `yaml:"level"`
	File   string    `yaml:"file"`
	Format LogFormat `yaml:"format"`
}

type LogLevel string
const (
	TraceLevel LogLevel = "trace"
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "none"
)

type LogFormat string
const (
	ConsoleFormat LogFormat = "console"
	JsonFormat    LogFormat = "json"
)

func (l *LogLevel) UnmarshalYAML(value *yaml.Node) error {
	var levelStr string
	if err := value.Decode(&levelStr); err != nil {
		return err
	}

	switch LogLevel(levelStr) {
	case TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, PanicLevel:
		*l = LogLevel(levelStr)
	default:
		*l = LogLevel("panic")
	}

	return nil
}

func (l *LogFormat) UnmarshalYAML(value *yaml.Node) error {
	var formatStr string
	if err := value.Decode(&formatStr); err != nil {
		return err
	}

	switch LogFormat(formatStr) {
	case ConsoleFormat, JsonFormat:
		*l = LogFormat(formatStr)
	default:
		*l = LogFormat("console")
	}

	return nil
}

func (logConfig *LogConfig) Settings(logLevel string, logFile string, logFormat string) error {	
	if logLevel != "default" {
		logConfig.Level = LogLevel(logLevel)
	}

	switch logConfig.Level {
	case TraceLevel:
		log.SetLevel(log.TraceLevel)
	case DebugLevel:
		log.SetLevel(log.DebugLevel)
	case InfoLevel:
		log.SetLevel(log.InfoLevel)
	case WarnLevel:
		log.SetLevel(log.WarnLevel)
	case ErrorLevel:
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.PanicLevel)
	}

	if logFile != "" {
		logConfig.File = logFile
	}

	if logConfig.File != "" {
		file, err := os.OpenFile(logConfig.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			return fmt.Errorf("failed to open log file: %w", err)
		}
	}

	if logFormat != "default" {
		logConfig.Format = LogFormat(logFormat)
	}

	switch logConfig.Format {
	case ConsoleFormat:
		log.SetFormatter(&log.TextFormatter{})
	case JsonFormat:
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.Info("default to console log format")
		log.SetFormatter(&log.TextFormatter{})	
	}

	log.Info("Logging:",
		"\n\tLevel: ", log.GetLevel(),
		"\n\tFormat: ", logConfig.Format,
	)
	if logConfig != nil {
		if logConfig.File != "" {
			log.Debug("\n\tFile: ", logConfig.File)
		}
	}

	return nil
}