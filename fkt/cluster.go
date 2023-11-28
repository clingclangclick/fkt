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
	Annotations  map[string]string    `yaml:"annotations"`
	Managed      *bool                `yaml:"managed"`
	Values       *Values              `yaml:"values,flow"`
	Resources    map[string]*Resource `yaml:"resources,flow"`
	AgePublicKey string               `yaml:"age_public_key"`
	path         *string
}

func (c *Cluster) config() Values {
	config := make(Values)

	config["path"] = *c.path
	config["annotations"] = c.Annotations
	config["managed"] = c.Managed

	return config
}

func (c *Cluster) load(path string) {
	c.path = &path

	if c.Annotations == nil {
		log.Trace("Cluster ", *c.path, " has no annotations")
		c.Annotations = make(map[string]string)
	}

	if _, ok := c.Annotations["name"]; !ok {
		_, name := filepath.Split(path)
		c.Annotations["name"] = name
	}

	if c.Managed == nil {
		log.Debug("Cluster managed unset, setting to `true`")
		c.Managed = new(bool)
		*c.Managed = true
	}
	log.Debug("Cluster managed: ", *c.Managed)

	if c.Values == nil {
		c.Values = new(Values)
	}
	log.Debug("Cluster Values: ", *c.Values)
}

func (c *Cluster) pathClusters(settings *Settings) string {
	return filepath.Join(settings.pathClusters(), *c.path)
}

func (c *Cluster) process(config *Config) error {
	log.Info("Processing cluster: ", *c.path)
	if c.Values == nil {
		log.Trace("Cluster ", *c.path, " has no values")
		c.Values = &Values{}
	}

	secrets := Secrets{
		ageKey: c.AgePublicKey,
	}
	if config.Secrets.SecretsFile != "" && secrets.ageKey != "" {
		err := secrets.read(filepath.Join(config.Settings.Directories.baseDirectory, config.Secrets.SecretsFile))
		if err != nil {
			return err
		}
	}

	err := utils.MkDir(c.pathClusters(config.Settings), config.Settings.DryRun)
	if err != nil {
		return err
	}

	processedResources := []string{}

	log.Info("Processing resources")
	for resourceName, resource := range c.Resources {
		log.Info("Attaching resource: ", resourceName, " to ", *c.path)
		if resource == nil {
			resource = &Resource{}
		}
		resource.load(resourceName)

		processedResources = append(processedResources, resourceName)

		if !*resource.Managed {
			log.Info("Skipping unmanaged resource: ", resourceName)
			continue
		}

		log.Info("Processing resource template: ", *resource.Template, ", into ", *c.path, "/", resourceName)

		values := make(Values)
		values["Cluster"] = c.config()
		values["Resource"] = resource.config()
		values["Values"] = ProcessValues(&config.Values, c.Values, &resource.Values)
		log.Trace("Values: ", values)

		err := resource.process(config.Settings, values, &secrets, *c.path)
		if err != nil {
			return fmt.Errorf("cannot process resource: %s; %w", resource.Name, err)
		}
	}

	resourceEntries, err := os.ReadDir(c.pathClusters(config.Settings))
	if err != nil {
		return fmt.Errorf("cannot get listing of resources in cluster path: %s; %w", c.pathClusters(config.Settings), err)
	}

	removableResourcePaths := []string{}

	for _, resourceEntry := range resourceEntries {
		resourceEntryName := resourceEntry.Name()
		resourcePath := filepath.Join(c.pathClusters(config.Settings), resourceEntryName)

		log.Debug("Checking resource ", resourceEntryName, ", path ", resourcePath)

		exists, err := utils.IsDir(resourcePath)
		if config.Settings.DryRun {
			if exists {
				return fmt.Errorf("%s exists when it should not", resourcePath)
			} else {
				return nil
			}
		}
		if !os.IsExist(err) {
			if !slices.Contains(processedResources, resourceEntryName) {
				removableResourcePaths = append(removableResourcePaths, resourcePath)
			}
		}
	}

	if *c.Managed {
		log.Debug("Removing unnecessary resource destination paths: ", removableResourcePaths)
		for _, removableResourcePath := range removableResourcePaths {
			err := os.RemoveAll(removableResourcePath)
			if err != nil {
				return fmt.Errorf("could not remove unnecessary resource destination path: %s; %w", removableResourcePath, err)
			}
		}

		log.Debug("Generating kustomization for cluster: ", *c.path)
		kustomization := &Kustomization{
			Cluster: c,
		}

		err := kustomization.generate(config.Settings, c.Annotations, config.Settings.DryRun)
		if err != nil {
			return fmt.Errorf("cannot generate kustomization: %w", err)
		}
	}

	return nil
}

func (c *Cluster) validate(config *Config) error {
	log.Info("Validating cluster: ", *c.path)

	for name, resource := range c.Resources {
		log.Debug("Validating resource: ", name)

		if resource == nil {
			resource = &Resource{}
		}
		resource.load(name)

		err := resource.validate(config.Settings, name)
		if err != nil {
			return err
		}
	}

	return nil
}
