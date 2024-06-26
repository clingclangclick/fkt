package fkt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/exp/maps"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Cluster struct {
	Kustomization *Kustomization       `yaml:"kustomization,flow"`
	Managed       *bool                `yaml:"managed"`
	Values        *Values              `yaml:"values,flow"`
	Resources     map[string]*Resource `yaml:"resources,flow"`
	AgePublicKey  string               `yaml:"age_public_key"`
	path          *string
}

func (c *Cluster) config() Values {
	config := make(Values)

	config["path"] = *c.path
	config["commonAnnotations"] = c.Kustomization.CommonAnnotations
	config["managed"] = *c.Managed

	return config
}

func (c *Cluster) load(path string) {
	c.path = &path

	if c.Kustomization == nil {
		log.Trace("Cluster ", *c.path, " has no kustomization settings")
		c.Kustomization = &Kustomization{}
	}

	if c.Kustomization.CommonAnnotations == nil {
		_, name := filepath.Split(path)
		c.Kustomization.CommonAnnotations = map[string]string{
			"name": name,
		}
	}

	if c.Managed == nil {
		log.Trace("Cluster managed unset, setting to `true`")
		c.Managed = new(bool)
		*c.Managed = true
	}
	log.Debug("Cluster managed: ", *c.Managed)

	if c.Values == nil {
		c.Values = new(Values)
	}
}

func (c *Cluster) pathTargets(settings *Settings) string {
	return filepath.Join(settings.pathTargets(), *c.path)
}

func (c *Cluster) process(config *Config) error {
	log.Info("Processing cluster: ", *c.path)
	if c.Values == nil {
		log.Trace("Cluster ", *c.path, " has no values")
		c.Values = &Values{}
	}

	secrets := Secrets{
		ageKey:       c.AgePublicKey,
		lastModified: nil,
	}
	if config.Secrets.SecretsFile != "" && secrets.ageKey != "" {
		err := secrets.read(filepath.Join(config.Settings.Directories.baseDirectory, config.Secrets.SecretsFile))
		if err != nil {
			return err
		}
	}

	err := utils.MkDir(c.pathTargets(config.Settings), config.Settings.DryRun)
	if err != nil {
		return err
	}

	var processedResources []string

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

		log.Info("Processing ", resource.Name)
		err := resource.process(config.Settings, values, &secrets, *c.path)
		if err != nil {
			return fmt.Errorf("cannot process resource: %s; %w", resource.Name, err)
		}
	}

	if *c.Managed {
		var removableResourcePaths []string

		resourceEntries, err := os.ReadDir(c.pathTargets(config.Settings))
		if err != nil {
			return fmt.Errorf("cannot get listing of resources in cluster path: %s; %w", c.pathTargets(config.Settings), err)
		}

		for _, resourceEntry := range resourceEntries {
			if !resourceEntry.IsDir() {
				continue
			}
			resourceEntryName := resourceEntry.Name()
			resourcePath := filepath.Join(c.pathTargets(config.Settings), resourceEntryName)

			log.Debug("Checking resource ", resourceEntryName, ", path ", utils.RelWD(resourcePath))

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
					log.Info("Adding ", resourcePath)
					removableResourcePaths = append(removableResourcePaths, resourcePath)
				}
			}
		}

		log.Debug("Removing unnecessary resource target paths, ", removableResourcePaths)
		for _, removableResourcePath := range removableResourcePaths {
			log.Trace("Removing path: ", utils.RelWD(removableResourcePath))
			if config.Settings.DryRun {
				if utils.IsExist(removableResourcePath) {
					return fmt.Errorf("dry-run, %s should not exist", removableResourcePath)
				}
			}
			err := os.RemoveAll(removableResourcePath)
			if err != nil {
				return fmt.Errorf("could not remove unnecessary resource target path: %s; %w", removableResourcePath, err)
			}
		}

		log.Debug("Generating kustomization for cluster: ", *c.path)
		kustomization := &Kustomization{
			Kind:              "Kustomization",
			APIVersion:        "kustomize.config.k8s.io/v1beta1",
			CommonAnnotations: c.Kustomization.CommonAnnotations,
			Patches:           c.Kustomization.Patches,
		}

		err = kustomization.generate(c.pathTargets(config.Settings), maps.Keys(c.Resources), config.Settings.DryRun)
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
