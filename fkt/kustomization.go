package fkt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Kustomization struct {
	Cluster *Cluster
}

type kustomization struct {
	CommonAnnotations map[string]string `yaml:"commonAnnotations"`
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	Resources         []string          `yaml:"resources"`
}

func (k *Kustomization) generate(settings *Settings, commonAnnotations map[string]string, dryRun bool) error {
	resources, err := k.resources(settings)
	if err != nil {
		return fmt.Errorf("cannot generate kustomization resources: %w", err)
	}

	if len(resources) == 0 {
		log.Warn("No kustomization resources in: ", k.Cluster.path)
		return nil
	}

	targetPath := k.Cluster.pathClusters(settings)

	log.Info("Generating kustomization for: ", resources)
	isDir, err := utils.IsDir(targetPath)
	if err != nil {
		return fmt.Errorf("error determinig directory: %w", err)
	}
	if !isDir {
		return fmt.Errorf("path is not directory: %s", targetPath)
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
	kustomizationFile := filepath.Join(targetPath, "kustomization.yaml")
	err = utils.WriteFile(kustomizationFile, kustomizationYAML, uint32(0666), dryRun)
	if err != nil {
		return fmt.Errorf("cannot write kustomization: %w", err)
	}

	return nil
}

func (k *Kustomization) resources(settings *Settings) ([]string, error) {
	var resources []string

	cluster := k.Cluster

	for resourceName, resource := range cluster.Resources {
		if resource == nil {
			resource = &Resource{}
		}
		resource.load(resourceName)

		if !*resource.Managed {
			continue
		}

		resources = append(resources, resourceName)
	}

	clusterPath := cluster.pathClusters(settings)

	d, err := os.Open(clusterPath)
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
		de, err := utils.IsDir(filepath.Join(clusterPath, entry))
		if err != nil {
			return []string{}, err
		}
		if de {
			resources = append(resources, entry)
		}
	}

	return resources, nil
}
