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

func (s *Source) Defaults(name string) {
	s.Name = name

	if s.Managed == nil {
		log.Debug("Managed unset, setting to `true`")
		s.Managed = new(bool)
		*s.Managed = true
	}

	if s.Namespace == nil {
		log.Debug("Namespace unset, setting to `default`")
		s.Namespace = new(string)
		*s.Namespace = "default"
	}

	if s.Origin == nil {
		log.Debug("Origin unset, setting to source name")
		s.Origin = new(string)
		*s.Origin = name
	}

	log.Debug("Source name: ", s.Name)
	log.Debug("Source origin: ", *s.Origin)
	if s.Values == nil {
		s.Values = make(Values)
	}
}

func (s *Source) Config() map[string]string {
	config := make(map[string]string)

	config["name"] = s.Name
	config["origin"] = *s.Origin
	config["namespace"] = *s.Namespace

	return config
}

func (s *Source) Validate(settings *Settings, name string) error {
	s.Defaults(name)
	if *s.Managed {
		path := filepath.Join(settings.sourcesPath(), *s.Origin)
		_, err := utils.IsDir(path)
		if err != nil {
			return fmt.Errorf("source validation failed for: %s", name)
		}
	}

	return nil
}

func (s *Source) DestinationPath(settings *Settings, clusterPath string) string {
	return filepath.Join(settings.Directories.BaseDirectory, settings.Directories.Overlays, clusterPath, s.Name)
}

func (s *Source) SourcePath(settings *Settings) string {
	return filepath.Join(settings.Directories.BaseDirectory, settings.Directories.Sources, *s.Origin)
}

func (s *Source) Process(settings *Settings, values Values, clusterPath string, subPaths ...string) error {
	subPath := ""
	if len(subPaths) > 0 {
		subPathSlice := []string{}
		subPathSlice = append(subPathSlice, subPaths...)
		subPath = filepath.Join(subPathSlice...)
	}
	sourcePath := filepath.Join(s.SourcePath(settings), subPath)
	destinationPath := filepath.Join(s.DestinationPath(settings, clusterPath), subPath)

	log.Debug("Source path: ", utils.RelWD(sourcePath))
	log.Debug("Destination path: ", utils.RelWD(destinationPath))

	err := utils.MkCleanDir(destinationPath, []string{}, settings.DryRun)
	if err != nil {
		log.Panic(err)
	}

	if !*s.Managed {
		log.Info("Unmanaged, skipping templating")
		return nil
	}

	de, err := utils.IsDir(sourcePath)
	if err != nil {
		return fmt.Errorf("is not a directory: %s", sourcePath)
	}

	if !utils.ContainsKustomization(sourcePath) {
		return fmt.Errorf("kustomization file does not exist in: %s", sourcePath)
	}

	if !de {
		return fmt.Errorf("source(%s) not a directory", sourcePath)
	}
	de, _ = utils.IsDir(destinationPath)
	if !de {
		err := utils.MkCleanDir(destinationPath, []string{}, settings.DryRun)
		if err != nil {
			return fmt.Errorf("failed to create directory %s (%s)", destinationPath, err)
		}
	}

	sdh, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
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
			err := values.Template(sourceEntryPath, destinationEntryPath, settings)
			if err != nil {
				return err
			}
		} else {
			err = s.Process(settings, values, clusterPath, sourceEntryPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
