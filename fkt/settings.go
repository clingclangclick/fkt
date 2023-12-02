package fkt

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		Target        string `yaml:"target"`
		baseDirectory string
	} `yaml:"directories"`
	configFileModifiedTime time.Time
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

	if settings.Directories.Target == "" {
		log.Trace("Settings default target directory: ", settingsDefaults["directory_target"])
		settings.Directories.Target = settingsDefaults["directory_target"]
	}
	log.Info("Clusters Directory: ", settings.Directories.Target)

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

	if settings.Directories.Target == "" {
		return fmt.Errorf("target directory unset")
	} else {
		exist, err := utils.IsDir(settings.pathTargets())
		if !exist || err != nil {
			log.Error("Target directory does not exist at ", utils.RelWD(settings.pathTargets()))
			err = os.MkdirAll(settings.pathTargets(), 0777)
			if err != nil {
				return fmt.Errorf("target directory cannot be created: %w", err)
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

func (settings *Settings) pathTargets() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Target)
}

func (settings *Settings) pathTemplates() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Templates)
}
