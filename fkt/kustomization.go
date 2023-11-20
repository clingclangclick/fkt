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
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	CommonAnnotations map[string]string `yaml:"commonAnnotations"`
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

	destinationPath := k.Cluster.pathOverlays(settings)

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

func (k *Kustomization) resources(settings *Settings) ([]string, error) {
	resources := []string{}

	for sourceName, source := range k.Cluster.Sources {
		if source == nil {
			source = &Source{}
		}
		source.load(sourceName)

		if !*source.Managed {
			continue
		}

		resources = append(resources, sourceName)
	}

	destinationPath := k.Cluster.pathOverlays(settings)

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

	return resources, nil
}
