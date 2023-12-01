package fkt

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

var settingsDefaults = map[string]string{
	"directory_clusters":  "clusters",
	"directory_templates": "templates",
	"delimiter_left":      "[[[",
	"delimiter_right":     "]]]",
}

type Settings struct {
	DryRun     bool       `yaml:"dry_run"`
	LogConfig  *LogConfig `yaml:"log"`
	Delimiters struct {
		Left  string `yaml:"left"`
		Right string `yaml:"right"`
	} `yaml:"delimiters"`
	Directories struct {
		Templates     string `yaml:"templates"`
		Clusters      string `yaml:"clusters"`
		baseDirectory string
	} `yaml:"directories"`
}

func (settings *Settings) Defaults(
	baseDirectory string,
	dryRun bool,
	logConfig LogConfig,
) error {
	if settings.LogConfig == nil {
		settings.LogConfig = &LogConfig{}
	}
	err := settings.LogConfig.settings(logConfig)
	if err != nil {
		return fmt.Errorf("error setting log configuration: %w", err)
	}

	log.Info("Settings")
	log.Info("Dry run: ", settings.DryRun)

	if settings.Directories.Clusters == "" {
		log.Trace("Settings default clusters directory: ", settingsDefaults["directory_clusters"])
		settings.Directories.Clusters = settingsDefaults["directory_cluster"]
	}
	log.Info("Clusters Directory: ", settings.Directories.Clusters)

	if settings.Directories.Templates == "" {
		log.Trace("Settings default templates directory: ", settingsDefaults["directory_templates"])
		settings.Directories.Templates = settingsDefaults["directory_templates"]
	}
	log.Info("Templates Directory: ", settings.Directories.Templates)

	if settings.Directories.baseDirectory == "" {
		if baseDirectory == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %w", err)
			}
			log.Trace("Settings default base directory: ", utils.RelWD(cwd))
			settings.Directories.baseDirectory = cwd
		} else {
			log.Trace("Settings default base directory: ", utils.RelWD(baseDirectory))
			settings.Directories.baseDirectory = baseDirectory
		}
	}
	log.Info("Base Directory: ", settings.Directories.baseDirectory)

	if settings.Delimiters.Left == "" {
		log.Trace("Settings default delimiter left: ", settingsDefaults["delimiter_left"])
		settings.Delimiters.Left = settingsDefaults["delimiter_left"]
	}
	log.Info("Left Delimiter: ", settings.Delimiters.Left)

	if settings.Delimiters.Right == "" {
		log.Trace("Settings default delimiter right: ", settingsDefaults["delimiter_right"])
		settings.Delimiters.Right = settingsDefaults["delimiter_right"]
	}
	log.Info("Right Delimiter: ", settings.Delimiters.Right)

	return nil
}

func (settings *Settings) Validate() error {
	log.Info("Validating settings")

	if settings.Directories.baseDirectory == "" {
		return fmt.Errorf("base directory not set")
	} else {
		exist, err := utils.IsDir(settings.Directories.baseDirectory)
		if !exist || err != nil {
			return fmt.Errorf("base directory does not exist: %w", err)
		}
	}

	if settings.Directories.Clusters == "" {
		return fmt.Errorf("clusters directory unset")
	} else {
		exist, err := utils.IsDir(settings.pathClusters())
		if !exist || err != nil {
			log.Error("Clusters directory does not exist at ", utils.RelWD(settings.pathClusters()))
			err = os.MkdirAll(settings.pathClusters(), 0777)
			if err != nil {
				return fmt.Errorf("clusters directory cannot be created: %w", err)
			}
		}
	}

	if settings.Directories.Templates == "" {
		return fmt.Errorf("templates directory unset")
	} else {
		exist, err := utils.IsDir(settings.pathTemplates())
		if !exist || err != nil {
			return fmt.Errorf("templates directory does not exist: %w", err)
		}
	}

	return nil
}

func (settings *Settings) pathClusters() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Clusters)
}

func (settings *Settings) pathTemplates() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Templates)
}
