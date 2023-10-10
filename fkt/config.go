package fkt

import (
	"os"
	"fmt"

	"golang.org/x/sync/errgroup"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	utils "github.com/clingclangclick/fkt/utils"
)

type Config struct {
	Settings      *Settings `yaml:"settings"`
	Values        Values    `yaml:"values,flow"`
	Clusters      []Cluster `yaml:"clusters"`
}

func (config *Config) defaults() {
	if config.Settings == nil {
		config.Settings = &Settings{}
	}
}

func (config *Config) Validate() error {
	log.Info("Validating configuration...")

	var eg = new(errgroup.Group)
	for _, cluster := range config.Clusters {
		func(config *Config) {
			eg.Go(func() error {
				return cluster.Validate(config)
			})
		}(config)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}

func (config *Config) Process() error {
	log.Info("Processing configuration...")

	var eg = new(errgroup.Group)
	for _, cluster := range config.Clusters {
		func(config *Config) {
			eg.Go(func() error {
				return cluster.Process(config.Settings, config.Values)
			})
		}(config)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("confighema processing failed: %w", err)
	}
	return nil
}

func LoadConfig(configurationFile,
	baseDirectory string,
	dryRun bool,
	logLevel string,
	logFile string,
	logFormat string,
) (*Config, error) {
	config := Config{}
	config.defaults()

	configurationBytes, err := os.ReadFile(configurationFile)
	if err != nil {
		return &config, err
	}
	err = yaml.Unmarshal(configurationBytes, &config)
	if config.Settings == nil {
		config.Settings = &Settings{}
	}
	err = config.Settings.defaults(baseDirectory, dryRun, logLevel, logFile, logFormat)
	if err != nil {
		return &config, err
	}

	log.Info("Loaded configuration: ", utils.RelWD(configurationFile))
	return &config, err
}
