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
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	Resources         []string          `yaml:"resources"`
	CommonAnnotations map[string]string `yaml:"commonAnnotations"`
	Patches           []interface{}     `yaml:"patches"`
}

func (k *Kustomization) generate(settings *Settings, cluster *Cluster, dryRun bool) error {
	err := k.resources(settings, cluster)
	if err != nil {
		return fmt.Errorf("cannot generate kustomization resources: %w", err)
	}

	targetPath := cluster.pathTargets(settings)

	log.Info("Generating kustomization")
	isDir, err := utils.IsDir(targetPath)
	if err != nil {
		return fmt.Errorf("error determinig directory: %w", err)
	}
	if !isDir {
		return fmt.Errorf("path is not directory: %s", targetPath)
	}

	kustomizationYAML, err := yaml.Marshal(k)
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

func (k *Kustomization) resources(settings *Settings, cluster *Cluster) error {
	var resources []string

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

	clusterPath := cluster.pathTargets(settings)

	d, err := os.Open(clusterPath)
	if err != nil {
		return err
	}
	defer d.Close()

	entries, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		found := slices.Contains(resources, entry)
		if found {
			continue
		}
		de, err := utils.IsDir(filepath.Join(clusterPath, entry))
		if err != nil {
			return err
		}
		if de {
			resources = append(resources, entry)
		}
	}

	if len(resources) == 0 {
		log.Warn("No kustomization resources in: ", cluster.path)
	}

	k.Resources = resources

	return nil
}
