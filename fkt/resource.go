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

	if r.Managed == nil {
		log.Debug("Resource managed unset, setting to `true`")
		r.Managed = new(bool)
		*r.Managed = true
	}
	log.Debug("Resource ", r.Name, " managed: ", *r.Managed)

	if r.Namespace == nil {
		log.Debug("Resource namespace unset, setting to ", name)
		r.Namespace = &name
	}

	if r.Template == nil {
		log.Debug("Resource template path unset, setting to resource name")
		r.Template = &name
	}

	if r.Values == nil {
		r.Values = make(Values)
	}
}

func (r *Resource) pathCluster(settings *Settings, clusterPath string) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Target, clusterPath, r.Name)
}

func (r *Resource) pathTemplates(settings *Settings) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Templates, *r.Template)
}

func (r *Resource) process(settings *Settings, values Values, secrets *Secrets, clusterPath string, subPaths ...string) error {
	if !*r.Managed {
		log.Info("Unmanaged, skipping templates for resource: ", r.Name)
		return nil
	}

	var subPath string
	if len(subPaths) > 0 {
		var subPathSlice []string
		subPathSlice = append(subPathSlice, subPaths...)
		subPath = filepath.Join(subPathSlice...)
	}

	templatePath := filepath.Join(r.pathTemplates(settings), subPath)
	log.Debug("Template path: ", utils.RelWD(templatePath))

	templatePathExists, err := utils.IsDir(templatePath)
	if err != nil {
		return fmt.Errorf("is not a directory: %s; %w", templatePath, err)
	}
	if !templatePathExists {
		return fmt.Errorf("template(%s) not a directory", templatePath)
	}

	if !r.containsKustomization(settings) {
		log.Warn("kustomization file does not exist in: ", templatePath)
		return nil
	}

	clusterResourcePath := filepath.Join(r.pathCluster(settings, clusterPath), subPath)
	log.Debug("Cluster resource path: ", utils.RelWD(clusterResourcePath))

	clusterResourcePathExists, _ := utils.IsDir(clusterResourcePath)
	if clusterResourcePathExists {
		err := utils.RemoveExtraFilesAndDirectories(clusterResourcePath, templatePath, settings.DryRun)
		if err != nil {
			return nil
		}
	}

	templatePathDir, err := os.Open(templatePath)
	if err != nil {
		return err
	}
	defer templatePathDir.Close()

	entries, err := templatePathDir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		resourceEntryPath := filepath.Join(templatePath, entry)
		dt, err := utils.IsDir(resourceEntryPath)
		if err != nil {
			return err
		}
		targetEntryPath := filepath.Join(clusterResourcePath, entry)
		err = utils.MkDir(filepath.Dir(targetEntryPath), settings.DryRun)
		if err != nil && !os.IsExist(err) {
			return err
		}
		if !dt {
			err := values.template(resourceEntryPath, targetEntryPath, settings, secrets)
			if err != nil {
				return err
			}
		} else {
			err = r.process(settings, values, secrets, clusterPath, entry)
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
	log.Debug("Checking for kustomization.yaml at: ", utils.RelWD(path))
	kustomizations := []string{
		"Kustomization",
		"kustomization.yaml",
		"kustomization.yml",
	}

	for _, kustomization := range kustomizations {
		kustomizationFile := filepath.Join(path, kustomization)
		ft, err := utils.IsFile(kustomizationFile)
		if ft && err == nil {
			return true
		}
	}

	log.Warn("No kustomizations in ", path)
	return false
}
