package fkt

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	utils "github.com/clingclangclick/fkt/utils"
)

var settingsDefaults = map[string]string{
	"directory_overlays": "overlays",
	"directory_sources":  "sources",
	"delimiter_left":     "[[[",
	"delimiter_right":    "]]]",
}

type Settings struct {
	Delimiters struct {
		Left  string `yaml:"left"`
		Right string `yaml:"right"`
	} `yaml:"delimiters"`
	Directories struct {
		Sources       string `yaml:"sources"`
		Overlays      string `yaml:"overlays"`
		baseDirectory string
	} `yaml:"directories"`
	DryRun    bool       `yaml:"dry_run"`
	LogConfig *LogConfig `yaml:"log"`
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

	if settings.Directories.Overlays == "" {
		log.Trace("Settings default directory overlay: ", settingsDefaults["directory_overlays"])
		settings.Directories.Overlays = settingsDefaults["directory_overlays"]
	}
	log.Info("Overlays Directory: ", settings.Directories.Overlays)

	if settings.Directories.Sources == "" {
		log.Trace("Settings default directory source: ", settingsDefaults["directory_sources"])
		settings.Directories.Sources = settingsDefaults["directory_sources"]
	}
	log.Info("Sources Directory: ", settings.Directories.Sources)

	if settings.Directories.baseDirectory == "" {
		if baseDirectory == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %w", err)
			}
			log.Trace("Settings default directory base: ", utils.RelWD(cwd))
			settings.Directories.baseDirectory = cwd
		} else {
			log.Trace("Settings default directory base: ", utils.RelWD(baseDirectory), ", creating.")
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

	if settings.Directories.Overlays == "" {
		return fmt.Errorf("overlays directory not set")
	} else {
		exist, err := utils.IsDir(settings.pathOverlays())
		if !exist || err != nil {
			log.Error("Overlays directory does not exist at ", utils.RelWD(settings.pathOverlays()))
			err = os.MkdirAll(settings.pathOverlays(), 0777)
			if err != nil {
				return fmt.Errorf("overlays directory cannot be created: %w", err)
			}
		}
	}

	if settings.Directories.Sources == "" {
		return fmt.Errorf("sources directory not set")
	} else {
		exist, err := utils.IsDir(settings.pathSources())
		if !exist || err != nil {
			return fmt.Errorf("sources directory does not exist: %w", err)
		}
	}

	return nil
}

func (settings *Settings) pathOverlays() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Overlays)
}

func (settings *Settings) pathSources() string {
	return filepath.Join(settings.Directories.baseDirectory, settings.Directories.Sources)
}
