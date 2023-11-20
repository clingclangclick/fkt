package fkt

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

type Source struct {
	Origin    *string `yaml:"origin"`
	Namespace *string `yaml:"namespace"`
	Values    Values  `yaml:"values,flow"`
	Managed   *bool   `yaml:"managed"`
	Name      string
}

func (s *Source) config() Values {
	config := make(Values)

	config["name"] = s.Name
	config["origin"] = *s.Origin
	config["namespace"] = *s.Namespace

	return config
}

func (s *Source) load(name string) {
	s.Name = name
	log.Debug("Source name: ", s.Name)

	if s.Managed == nil {
		log.Debug("Source managed unset, setting to `true`")
		s.Managed = new(bool)
		*s.Managed = true
	}
	log.Debug("Source managed: ", *s.Managed)

	if s.Namespace == nil {
		log.Debug("Source namespace unset, setting to ", name)
		s.Namespace = &name
	}
	log.Debug("Source namespace: ", *s.Namespace)

	if s.Origin == nil {
		log.Debug("Source origin unset, setting to source name")
		s.Origin = &name
	}
	log.Debug("Source origin: ", *s.Origin)

	if s.Values == nil {
		s.Values = make(Values)
	}
	log.Trace("Source values: ", s.Values)
}

func (s *Source) pathDestination(settings *Settings, clusterPath string) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Overlays, clusterPath, s.Name)
}

func (s *Source) pathSource(settings *Settings) string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Sources, *s.Origin)
}

func (s *Source) process(settings *Settings, values Values, clusterPath string, subPaths ...string) error {
	subPath := ""

	if len(subPaths) > 0 {
		subPathSlice := []string{}
		subPathSlice = append(subPathSlice, subPaths...)
		subPath = filepath.Join(subPathSlice...)
	}

	sourcePath := filepath.Join(s.pathSource(settings), subPath)
	log.Debug("Source path: ", utils.RelWD(sourcePath))

	destinationPath := filepath.Join(s.pathDestination(settings, clusterPath), subPath)
	log.Debug("Destination path: ", utils.RelWD(destinationPath))

	err := utils.MkCleanDir(destinationPath, []string{}, settings.DryRun)
	if err != nil {
		return err
	}

	if !*s.Managed {
		log.Info("Unmanaged, skipping templating")
		return nil
	}

	de, err := utils.IsDir(sourcePath)
	if err != nil {
		return fmt.Errorf("is not a directory: %s; %w", sourcePath, err)
	}

	if !utils.ContainsKustomization(sourcePath) {
		return fmt.Errorf("kustomization file does not exist in: %s; %w", sourcePath, err)
	}

	if !de {
		return fmt.Errorf("source(%s) not a directory", sourcePath)
	}
	de, _ = utils.IsDir(destinationPath)
	if !de {
		err := utils.MkCleanDir(destinationPath, []string{}, settings.DryRun)
		if err != nil {
			return fmt.Errorf("failed to create directory %s; %w", destinationPath, err)
		}
	}

	sdh, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sdh.Close()

	entries, err := sdh.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourceEntryPath := filepath.Join(sourcePath, entry)
		dt, err := utils.IsDir(sourceEntryPath)
		if err != nil {
			return err
		}
		if !dt {
			destinationEntryPath := filepath.Join(destinationPath, entry)
			err := values.template(sourceEntryPath, destinationEntryPath, settings)
			if err != nil {
				return err
			}
		} else {
			err = s.process(settings, values, clusterPath, sourceEntryPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Source) validate(settings *Settings, name string) error {
	if *s.Managed {
		path := filepath.Join(settings.pathSources(), *s.Origin)
		_, err := utils.IsDir(path)
		if err != nil {
			return fmt.Errorf("source validation failed for: %s; %w", name, err)
		}

		if !utils.ContainsKustomization(path) {
			return fmt.Errorf("kustomization file does not exist in: %s; %w", utils.RelWD(path), err)
		}
	}

	return nil
}
