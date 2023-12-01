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
	Secrets  struct {
		SecretsFile string `yaml:"file"`
		secrets     Secrets
	} `yaml:"secrets"`
}

func LoadConfig(configurationFile string) (*Config, error) {
	config := Config{}

	configurationFileInfo, err := os.Stat(configurationFile)
	if err != nil {
		return &config, err
	}

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
	config.Settings.configFileModifiedTime = configurationFileInfo.ModTime()

	log.Info("Loaded configuration: ", utils.RelWD(configurationFile))

	return &config, err
}

func (config *Config) Process() error {
	log.Info("Processing configuration...")

	var eg = new(errgroup.Group)
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}

		c := cluster
		c.load(path)

		func(c *Cluster) {
			eg.Go(func() error {
				return c.process(config)
			})
		}(c)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	return nil
}

func (config *Config) Validate() error {
	log.Info("Validating configuration...")

	var eg = new(errgroup.Group)
	for path, cluster := range config.Clusters {
		if cluster == nil {
			cluster = &Cluster{}
		}

		c := cluster
		c.load(path)

		func(c *Cluster) {
			eg.Go(func() error {
				return c.validate(config)
			})
		}(c)
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
