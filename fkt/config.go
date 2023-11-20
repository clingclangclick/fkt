package fkt

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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

func (config *Config) Process(settings *Settings) error {
	log.Info("Processing configuration...")

	var eg = new(errgroup.Group)
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}

		c := cluster
		c.load(path)

		func(settings *Settings, c *Cluster) {
			eg.Go(func() error {
				return c.process(settings, &config.Values)
			})
		}(settings, c)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	return nil
}

func (config *Config) Validate(settings *Settings) error {
	log.Info("Validating configuration...")

	var eg = new(errgroup.Group)
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}

		c := cluster
		c.load(path)

		func(settings *Settings, c *Cluster) {
			eg.Go(func() error {
				return c.validate(settings)
			})
		}(settings, c)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
