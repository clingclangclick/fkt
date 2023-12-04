package fkt

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Kustomization struct {
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	Resources         []string          `yaml:"resources"`
	CommonAnnotations map[string]string `yaml:"commonAnnotations"`
	Patches           []interface{}     `yaml:"patches"`
}

func (k *Kustomization) generate(path string, resources []string, dryRun bool) error {
	for _, resourceName := range resources {
		resourcePath := filepath.Join(path, resourceName)
		if utils.ContainsKustomization(resourcePath) {
			k.Resources = append(k.Resources, resourceName)
		} else {
			log.Warn("No kustomization found for resource, ", resourceName)
		}
	}

	log.Info("Generating kustomization")
	isDir, err := utils.IsDir(path)
	if err != nil {
		return fmt.Errorf("error determinig directory: %w", err)
	}
	if !isDir {
		return fmt.Errorf("path is not directory: %s", path)
	}

	kustomizationYAML, err := yaml.Marshal(k)
	if err != nil {
		return fmt.Errorf("cannot marshal kustomization: %w", err)
	}
	kustomizationFile := filepath.Join(path, "kustomization.yaml")
	err = utils.WriteFile(kustomizationFile, kustomizationYAML, uint32(0666), dryRun)
	if err != nil {
		return fmt.Errorf("cannot write kustomization: %w", err)
	}

	return nil
}
