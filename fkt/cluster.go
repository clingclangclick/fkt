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

		err := c.generateKustomization(settings, c.overlayPath(settings), c.Config(), settings.DryRun)
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

func (c *Cluster) kustomizationResources(settings *Settings, destinationPath string) ([]string, error) {
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

func (c *Cluster) generateKustomization(settings *Settings, destinationPath string, commonAnnotations map[string]string, dryRun bool) error {
	resources, err := c.kustomizationResources(settings, destinationPath)
	if err != nil {
		return fmt.Errorf("cannot generate kustomization resources: %w", err)
	}

	if len(resources) == 0 {
		log.Warn("No kustomization resources for cluster: ", c.clusterPath())
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
