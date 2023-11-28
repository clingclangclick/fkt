package fkt

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Resource struct {
	Template  *string `yaml:"template"`
	Namespace *string `yaml:"namespace"`
	Values    Values  `yaml:"values,flow"`
	Managed   *bool   `yaml:"managed"`
	Name      string
}

func (r *Resource) config() Values {
	config := make(Values)

	config["name"] = r.Name
	config["template"] = *r.Template
	config["namespace"] = *r.Namespace

	return config
}

func (r *Resource) load(name string) {
	r.Name = name
	log.Debug("Resource name: ", r.Name)

	if r.Managed == nil {
		log.Debug("Resource managed unset, setting to `true`")
		r.Managed = new(bool)
		*r.Managed = true
	}
	log.Debug("Resource managed: ", *r.Managed)

	if r.Namespace == nil {
		log.Debug("Resource namespace unset, setting to ", name)
		r.Namespace = &name
	}
	log.Debug("Resource namespace: ", *r.Namespace)

	if r.Template == nil {
		log.Debug("Resource template path unset, setting to resource name")
		r.Template = &name
	}
	log.Debug("Resource template: ", *r.Template)

	if r.Values == nil {
		r.Values = make(Values)
	}
	log.Trace("Resource values: ", r.Values)
}

func (r *Resource) pathCluster(settings *Settings, clusterPath string) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Clusters, clusterPath, r.Name)
}

func (r *Resource) pathTemplates(settings *Settings) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Templates, *r.Template)
}

func (r *Resource) process(settings *Settings, values Values, secrets *Secrets, clusterPath string, subPaths ...string) error {
	subPath := ""

	if len(subPaths) > 0 {
		subPathSlice := []string{}
		subPathSlice = append(subPathSlice, subPaths...)
		subPath = filepath.Join(subPathSlice...)
	}

	templatePath := filepath.Join(r.pathTemplates(settings), subPath)
	log.Debug("Template path: ", utils.RelWD(templatePath))

	clusterResourcePath := filepath.Join(r.pathCluster(settings, clusterPath), subPath)
	log.Debug("Cluster resource path: ", utils.RelWD(clusterResourcePath))

	err := utils.MkCleanDir(clusterResourcePath, []string{}, settings.DryRun)
	if err != nil {
		return err
	}

	if !*r.Managed {
		log.Info("Unmanaged, skipping templating")
		return nil
	}

	de, err := utils.IsDir(templatePath)
	if err != nil {
		return fmt.Errorf("is not a directory: %s; %w", templatePath, err)
	}

	if !r.containsKustomization(settings) {
		return fmt.Errorf("kustomization file does not exist in: %s", templatePath)
	}

	if !de {
		return fmt.Errorf("template(%s) not a directory", templatePath)
	}
	de, _ = utils.IsDir(clusterResourcePath)
	if !de {
		err := utils.MkCleanDir(clusterResourcePath, []string{}, settings.DryRun)
		if err != nil {
			return fmt.Errorf("failed to create directory %s; %w", clusterResourcePath, err)
		}
	}

	sdh, err := os.Open(templatePath)
	if err != nil {
		return err
	}
	defer sdh.Close()

	entries, err := sdh.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		resourceEntryPath := filepath.Join(templatePath, entry)
		dt, err := utils.IsDir(resourceEntryPath)
		if err != nil {
			return err
		}
		if !dt {
			destinationEntryPath := filepath.Join(clusterResourcePath, entry)
			err := values.template(resourceEntryPath, destinationEntryPath, settings, secrets)
			if err != nil {
				return err
			}
		} else {
			err = r.process(settings, values, secrets, clusterPath, resourceEntryPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Resource) validate(settings *Settings, name string) error {
	if *r.Managed {
		path := filepath.Join(settings.pathTemplates(), *r.Template)
		_, err := utils.IsDir(path)
		if err != nil {
			return fmt.Errorf("resource template path validation failed for: %s; %w", name, err)
		}

		if !r.containsKustomization(settings) {
			return fmt.Errorf("kustomization file does not exist in: %s; %w", utils.RelWD(path), err)
		}
	}

	return nil
}

func (r *Resource) containsKustomization(settings *Settings) bool {
	path := r.pathTemplates(settings)
	log.Debug("Checking for kustomization.yaml at: ", path)
	kustomizations := []string{
		"Kustomization",
		"kustomization.yaml",
		"kustomization.yml",
	}

	for _, kustomization := range kustomizations {
		kustomizationFile := filepath.Join(path, kustomization)
		ft, err := utils.IsFile(kustomizationFile)
		if ft && err == nil {
			log.Debug("Found ", utils.RelWD(kustomizationFile))
			return true
		}
	}

	log.Trace("No kustomizations in ", path)
	return false
}
