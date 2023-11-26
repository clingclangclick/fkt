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
	Annotations map[string]string    `yaml:"annotations"`
	Managed     *bool                `yaml:"managed"`
	Values      *Values              `yaml:"values,flow"`
	Resources   map[string]*Resource `yaml:"resources,flow"`
	path        *string
}

func (c *Cluster) config() Values {
	config := make(Values)

	config["path"] = c.path
	config["annotations"] = c.Annotations
	config["managed"] = c.Managed

	return config
}

func (c *Cluster) load(path string) {
	c.path = &path

	if c.Annotations == nil {
		log.Trace("Cluster ", c.path, " has no annotations")
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

func (c *Cluster) process(settings *Settings, globalValues *Values) error {
	if c.Values == nil {
		log.Trace("Cluster ", c.path, " has no values")
		c.Values = &Values{}
	}

	err := utils.MkDir(c.pathClusters(settings), settings.DryRun)
	if err != nil {
		return err
	}

	processedResources := []string{}

	for resourceName, resource := range c.Resources {
		if resource == nil {
			resource = &Resource{}
		}
		resource.load(resourceName)

		processedResources = append(processedResources, resourceName)

		if !*resource.Managed {
			log.Info("Skipping unmanaged resource: ", resourceName)
			continue
		}

		log.Info("Processing resource template: ", *resource.Template, ", into ", c.path, "/", resourceName)

		values := make(Values)
		values["Cluster"] = c.config()
		values["Resource"] = resource.config()
		values["Values"] = ProcessValues(globalValues, c.Values, &resource.Values)
		log.Trace("Values: ", values)

		err := resource.process(settings, values, *c.path)
		if err != nil {
			return fmt.Errorf("cannot process resource: %s; %w", resource.Name, err)
		}
	}

	resourceEntries, err := os.ReadDir(c.pathClusters(settings))
	if err != nil {
		return fmt.Errorf("cannot get listing of resources in cluster path: %s; %w", c.pathClusters(settings), err)
	}

	removableResourcePaths := []string{}

	for _, resourceEntry := range resourceEntries {
		resourceEntryName := resourceEntry.Name()
		resourcePath := filepath.Join(c.pathClusters(settings), resourceEntryName)

		log.Debug("Checking resource ", resourceEntryName, ", path ", resourcePath)

		exists, err := utils.IsDir(resourcePath)
		if settings.DryRun {
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

	for name, resource := range c.Resources {
		log.Debug("Validating resource: ", name)

		if resource == nil {
			resource = &Resource{}
		}
		resource.load(name)

		err := resource.validate(settings, name)
		if err != nil {
			return err
		}
	}

	return nil
}
