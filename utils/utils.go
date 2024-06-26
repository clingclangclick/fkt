package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func RelWD(path string) string {
	cwd, _ := os.Getwd()
	relPath, _ := filepath.Rel(cwd, path)

	return relPath
}

func RemoveExtraFilesAndDirectories(sourceDir, targetDir string, dryRun bool) error {
	sourceItems, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	targetItems := make(map[string]struct{})

	targetFiles, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}

	for _, item := range targetFiles {
		targetItems[item.Name()] = struct{}{}
	}

	for _, item := range sourceItems {
		if _, exists := targetItems[item.Name()]; !exists {
			itemPath := filepath.Join(sourceDir, item.Name())

			if dryRun && IsExist(itemPath) {
				return fmt.Errorf("dry-run, entry to be removed: %s", itemPath)
			}

			if item.IsDir() {
				if err := os.RemoveAll(itemPath); err != nil {
					return err
				}
				log.Debug("Removed target directory: ", itemPath)
			} else {
				if err := os.Remove(itemPath); err != nil {
					return err
				}
				log.Debug("Removed target file: ", itemPath)
			}
		}
	}

	return nil
}

func MkDir(path string, dryRun bool) error {
	exists, err := IsDir(path)
	if dryRun {
		if !exists {
			return fmt.Errorf("dry-run, %s does not exist or is not a directory", path)
		} else {
			return nil
		}
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0777)
		return err
	}

	return nil
}

func IsDir(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return s.IsDir(), nil
}

func IsFile(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return s.Mode().IsRegular(), nil
}

func IsExist(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func WriteFile(path string, b []byte, mode uint32, dryRun bool) error {
	if dryRun {
		existingData, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("dry-run, error reading existing file: %w", err)
		}

		if bytes.Equal(existingData, b) {
			return nil
		}

		return errors.New("dry-run, file contents would be changed")
	}

	err := os.WriteFile(path, b, os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func ContainsKustomization(path string) bool {
	log.Debug("Checking for kustomization.yaml at: ", RelWD(path))
	kustomizations := []string{
		"Kustomization",
		"kustomization.yaml",
		"kustomization.yml",
	}

	for _, kustomization := range kustomizations {
		kustomizationFile := filepath.Join(path, kustomization)
		ft, err := IsFile(kustomizationFile)
		if ft && err == nil {
			return true
		}
	}

	log.Trace("No kustomizations in ", path)
	return false
}

func SOPSLastModified(fileBytes []byte) (time.Time, error) {
	sopsStruct := struct {
		Sops struct {
			LastModified string `yaml:"lastmodified"`
		} `yaml:"sops"`
	}{}

	err := yaml.Unmarshal(fileBytes, &sopsStruct)
	if err != nil {
		return time.Time{}, err
	}
	iso8601Format := "2006-01-02T15:04:05Z"

	log.Trace("Found lastmodified for sops k8s yaml: ", sopsStruct.Sops.LastModified)

	t, err := time.Parse(iso8601Format, sopsStruct.Sops.LastModified)
	if err != nil {
		return time.Time{}, err
	}

	tUTC := t.UTC()

	log.Debug("Parsed lastmdified time as ", tUTC)
	return t, err
}
