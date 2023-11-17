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
	Annotations map[string]string  `yaml:"annotations"`
	Managed     *bool              `yaml:"managed"`
	Values      *Values            `yaml:"values,flow"`
	Sources     map[string]*Source `yaml:"sources"`
	path        string
}

func (c *Cluster) config() Values {
	config := make(Values)

	config["path"] = c.path
	config["annotations"] = c.Annotations
	config["managed"] = c.Managed

	return config
}

func (c *Cluster) defaults(path string) {
	c.path = path

	if c.Managed == nil {
		log.Debug("Cluster managed unset, setting to `true`")
		c.Managed = new(bool)
		*c.Managed = true
	}
	log.Debug("Cluster managed: ", *c.Managed)

	if c.Values == nil {
		log.Debug("Cluster values unset, setting to `true`")
		c.Values = new(Values)
	}
	log.Debug("Cluster managed: ", *c.Managed)
}

func (c *Cluster) pathOverlays(settings *Settings) string {
	return filepath.Join(settings.pathOverlays(), c.path)
}

func (c *Cluster) process(settings *Settings, path string, globalValues Values) error {
	if c.Values == nil {
		log.Trace("Cluster ", c.path, " has no values")
		c.Values = &Values{}
	}

	clusterGlobalValues := globalValues.processValues(*c.Values)
	log.Debug("clusterGlobalValues: ", clusterGlobalValues.dump())

	err := utils.MkDir(c.pathOverlays(settings), settings.DryRun)
	if err != nil {
		return err
	}

	processedSources := []string{}
	for sourceName, source := range c.Sources {
		if source == nil {
			source = &Source{}
		}
		source.defaults(sourceName)

		processedSources = append(processedSources, sourceName)

		if !*source.Managed {
			log.Info("Skipping unmanaged source: ", sourceName)
			continue
		}

		log.Info("Processing Source: ", *source.Origin, ", into ", c.path, "/", sourceName)

		values := make(Values)
		values["Cluster"] = c.config()
		values["Source"] = source.config()
		values["Values"] = clusterGlobalValues.processValues(source.Values)
		log.Trace("Values: ", values.dump())

		err := source.process(settings, values, c.path)
		if err != nil {
			return fmt.Errorf("cannot process source: %s; %w", source.Name, err)
		}
	}

	sourceEntries, err := os.ReadDir(c.pathOverlays(settings))
	if err != nil {
		return fmt.Errorf("cannot get listing of sources in cluster path: %s; %w", c.pathOverlays(settings), err)
	}

	log.Trace("Checking entries: ", sourceEntries)
	removableSourcePaths := []string{}
	for _, sourceEntry := range sourceEntries {
		sourceEntryName := sourceEntry.Name()
		sourcePath := filepath.Join(c.pathOverlays(settings), sourceEntryName)

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

	if *c.Managed {
		log.Debug("Removing unnecessary source destination paths: ", removableSourcePaths)
		for _, removableSourcePath := range removableSourcePaths {
			err := os.RemoveAll(removableSourcePath)
			if err != nil {
				return fmt.Errorf("could not remove unnecessary source destination path: %s; %w", removableSourcePath, err)
			}
		}

		log.Debug("Generating kustomization for cluster: ", c.path)
		kustomization := &Kustomization{
			Cluster: c,
		}

		err := kustomization.generate(settings, c.Annotations, settings.DryRun)
		if err != nil {
			return fmt.Errorf("cannot generate kustomization: %w", err)
		}
	}

	return nil
}

func (c *Cluster) validate(settings *Settings) error {
	log.Info("Validating cluster: ", c.path)

	for name, source := range c.Sources {
		log.Debug("Validating source: ", name)
		err := source.validate(settings, name)
		if err != nil {
			return err
		}
	}

	return nil
}
