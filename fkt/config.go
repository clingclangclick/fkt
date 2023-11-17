package fkt

import (
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	utils "github.com/clingclangclick/fkt/utils"
)

type Config struct {
	Settings *Settings           `yaml:"settings"`
	Values   Values              `yaml:"values,flow"`
	Clusters map[string]*Cluster `yaml:"clusters"`
}

func LoadConfig(configurationFile string) (*Config, error) {
	config := Config{}

	configurationBytes, err := os.ReadFile(configurationFile)
	if err != nil {
		return &config, err
	}
	err = yaml.Unmarshal(configurationBytes, &config)
	if err != nil {
		return &config, err
	}

	if config.Settings == nil {
		config.Settings = &Settings{}
	}

	log.Info("Loaded configuration: ", utils.RelWD(configurationFile))

	return &config, err
}

func (config *Config) Process() error {
	log.Info("Processing configuration...")

	invalid := []string{}
	errs := errors.New("error processing")
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}
		cluster.defaults(path)
		err := cluster.process(config.Settings, path, config.Values)
		if err != nil {
			invalid = append(invalid, cluster.path)
			errs = fmt.Errorf("%s\n%w\n%w", cluster.path, errs, err)

			log.Error("cannot process cluster: ", cluster.path)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("processing failed: %w", errs)
	}

	return nil
}

func (config *Config) Validate() error {
	log.Info("Validating configuration...")

	invalid := []string{}
	errs := errors.New("error processing")
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}
		cluster.defaults(path)
		err := cluster.validate(config.Settings)
		if err != nil {
			invalid = append(invalid, cluster.path)
			errs = fmt.Errorf("%s\n%w\n%w", cluster.path, errs, err)

			log.Error("cannot validate cluster: ", cluster.path)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("configuration validation failed: %w", errs)
	}

	return nil
}
