package fkt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Cluster struct {
	Platform    string             `yaml:"platform"`
	Name        string             `yaml:"name"`
	Region      string             `yaml:"region"`
	Environment string             `yaml:"environment"`
	Managed     bool               `yaml:"managed"`
	Values      Values             `yaml:"values,flow"`
	Sources     map[string]*Source `yaml:"sources"`
}

func (c *Cluster) clusterPath() string {
	return filepath.Join(c.Platform, c.Environment, c.Region, c.Name)
}

func (c *Cluster) overlayPath(settings *Settings) string {
	return filepath.Join(settings.overlaysPath(), c.clusterPath())
}

func (c *Cluster) Config() map[string]string {
	config := make(map[string]string)

	config["platform"] = c.Platform
	config["region"] = c.Region
	config["environment"] = c.Environment
	config["name"] = c.Name

	return config
}

func (c *Cluster) Process(settings *Settings, globalValues Values) error {
	if c.Values == nil {
		log.Trace("Cluster ", c.Name, " has no values")
		c.Values = Values{}
	}

	clusterGlobalValues := globalValues.ProcessValues(c.Values)
	log.Debug("clusterGlobalValues: ", clusterGlobalValues.Dump())

	err := utils.MkDir(c.overlayPath(settings), settings.DryRun)
	if err != nil {
		return err
	}

	processedSources := []string{}
	for sourceName, source := range c.Sources {
		if source == nil {
			source = &Source{}
		}
		source.Defaults(sourceName)

		processedSources = append(processedSources, sourceName)

		if !*source.Managed {
			log.Info("Skipping unmanaged source: ", sourceName)
			continue
		}

		log.Info("Processing Source: ", *source.Origin, ", into ", c.Name, "/", sourceName)
		values := make(Values)

		values["Cluster"] = c.Config()
		values["Source"] = source.Config()
		values["Values"] = clusterGlobalValues.ProcessValues(source.Values)
		log.Trace("Values: ", values.Dump())

		err := source.Process(settings, values, c.clusterPath())
		if err != nil {
			return fmt.Errorf("cannot process source: %s; %w", source.Name, err)
		}
	}

	sourceEntries, err := os.ReadDir(c.overlayPath(settings))
	if err != nil {
		return fmt.Errorf("cannot get listing of sources in cluster path: %s; %w", c.overlayPath(settings), err)
	}

	log.Trace("Checking entries: ", sourceEntries)
	removableSourcePaths := []string{}
	for _, sourceEntry := range sourceEntries {
		sourceEntryName := sourceEntry.Name()
		sourcePath := filepath.Join(c.overlayPath(settings), sourceEntryName)
		exists, err := utils.IsDir(sourcePath)
		if settings.DryRun {
			if exists {
				return fmt.Errorf("%s exists when it should not", sourcePath)
			} else {
				return nil
			}
		}
		if !os.IsExist(err) {
			if !slices.Contains(processedSources, sourceEntryName) {
				removableSourcePaths = append(removableSourcePaths, sourcePath)
			}
		}
	}

	if c.Managed {
		log.Debug("Removing unnecessary source destination paths: ", removableSourcePaths)
		for _, removableSourcePath := range removableSourcePaths {
			err := os.RemoveAll(removableSourcePath)
			if err != nil {
				return fmt.Errorf("could not remove unnecessary source destination path: %s; %w", removableSourcePath, err)
			}
		}

		log.Debug("Generating kustomization for cluster: ", c.Name)
		kustomization := &Kustomization{
			Cluster: c,
		}

		err := kustomization.generate(settings, c.Config(), settings.DryRun)
		if err != nil {
			return fmt.Errorf("cannot generate kustomization: %w", err)
		}
	}

	return nil
}

func (c *Cluster) Validate(settings *Settings) error {
	log.Info("Validating cluster: ", c.Name)

	for name, source := range c.Sources {
		log.Debug("Validating source: ", name)
		err := source.Validate(settings, name)
		if err != nil {
			return err
		}
	}

	return nil
}
