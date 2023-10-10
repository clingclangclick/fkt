package fkt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"

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

func (c *Cluster) path() string {
	return filepath.Join(c.Platform, c.Environment, c.Region, c.Name)
}

func (c *Cluster) destination(settings *Settings) string {
	return filepath.Join(settings.overlaysPath(), c.path())
}

func (c *Cluster) Defaults(settings *Settings) {
	if c.Values == nil {
		log.Debug("Cluster ", c.Name, " has no values")
		c.Values = make(Values)
	}
}

func (c *Cluster) Config() map[string]string {
	config := make(map[string]string)

	config["platform"] = c.Platform
	config["region"] = c.Region
	config["environment"] = c.Environment
	config["name"] = c.Name

	return config
}

func (c *Cluster) Validate(config *Config) error {
	log.Info("Validating cluster: ", c.path())

	for name, source := range c.Sources {
		log.Debug("Validating source: ", name)
		err := source.Validate(config.Settings, name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) Process(settings *Settings, globalValues Values) error {
	log.Info("Processing cluster path: ", c.path())

	c.Defaults(settings)
	if c.Managed {
		log.Info("Managed cluster")
	}

	clusterGlobalValues := globalValues.ProcessValues(c.Values)
	log.Debug("clusterGlobalValues: ", clusterGlobalValues.Dump())

	if c.Managed {
		err := c.mkCleanDir(settings)
		if err != nil {
			return fmt.Errorf("cannot create and clean directory: %w", err)
		}
	}

	for sourceName, source := range c.Sources {
		if source == nil {
			source = &Source{}
		}
		source.Defaults(sourceName)

		if *source.Managed {
			log.Info("Processing Source: ", *source.Origin, ", into ", c.path(), "/", sourceName)
			values := make(Values)

			values["Cluster"] = c.Config()
			values["Source"] = source.Config()
			values["Values"] = clusterGlobalValues.ProcessValues(source.Values)
			log.Trace("Values: ", values.Dump())
		
			err := source.Process(settings, values, c.path())
			if err != nil {
				return err
			}
		} else {
			log.Info("Skipping unmanaged source: ", sourceName)
		}
	}

	if c.Managed {
		err := c.generateKustomization(c.destination(settings), c.Config(), settings.DryRun)
		if err != nil {
			return fmt.Errorf("cannot generate kustomization: %w", err)
		}
	}

	return nil
}

type kustomization struct {
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	CommonAnnotations map[string]string `yaml:"commonAnnotations"`
	Resources         []string          `yaml:"resources"`
}

func (c *Cluster) kustomizationResources(destinationPath string) ([]string, error) {
	resources := []string{}

	for sourceName, source := range c.Sources {
		if source == nil {
			source = &Source{}
		}
		source.Defaults(sourceName)
		if !*source.Managed {
			continue
		}
		resources = append(resources, sourceName)
	}

	d, err := os.Open(destinationPath)
	if err != nil {
		return []string{}, err
	}
	defer d.Close()

	entries, err := d.Readdirnames(-1)
	if err != nil {
		return []string{}, err
	}
	for _, entry := range entries {
		found := slices.Contains(resources, entry)
		if found {
			continue
		}
		de, err := utils.IsDir(filepath.Join(destinationPath, entry))

		if err != nil {
			return []string{}, err
		}
		if de {
			if utils.ContainsKustomization(filepath.Join(destinationPath, entry)) {
				resources = append(resources, entry)
			}
		}
	}

	sort.Strings(resources)

	return resources, nil
}

func (c *Cluster) generateKustomization(destinationPath string, commonAnnotations map[string]string, dryRun bool) error {
	resources, err := c.kustomizationResources(destinationPath)
	if err != nil {
		return fmt.Errorf("cannot generate kustomization resources: %w", err)
	}

	if len(resources) == 0 {
		return nil
	}

	log.Info("Generating kustomization for: ", resources)
	isDir, err := utils.IsDir(destinationPath)
	if err != nil {
		return fmt.Errorf("error determinig directory: %w", err)
	}
	if !isDir {
		return fmt.Errorf("path is not directory: %s", destinationPath)
	}
	kustomizationYAML, err := yaml.Marshal(&kustomization{
		Kind:              "Kustomization",
		APIVersion:        "kustomize.config.k8s.io/v1beta1",
		CommonAnnotations: commonAnnotations,
		Resources:         resources,
	})
	if err != nil {
		return fmt.Errorf("cannot marshal kustomization: %w", err)
	}
	kustomizationYAML = []byte(fmt.Sprintf("---\n%s", kustomizationYAML))
	kustomizationFile := filepath.Join(destinationPath, "kustomization.yaml")

	if !dryRun {
		err = utils.WriteFile(kustomizationFile, kustomizationYAML, uint32(0666), dryRun)
		if err != nil {
			return fmt.Errorf("cannot write kustomization: %w", err)
		}
	}

	return nil
}

func (c *Cluster) mkCleanDir(settings *Settings) error {
	unmanaged := []string{}
	for sourceName, source := range c.Sources {
		if source != nil {
			if !*source.Managed {
				unmanaged = append(unmanaged, sourceName)
			}
		}
	}

	log.Debug("Unmanaged sources: ", unmanaged)
	err := utils.MkCleanDir(c.destination(settings), unmanaged, settings.DryRun)
	if err != nil {
		return err
	}

	return nil
}
