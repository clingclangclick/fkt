package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	log "github.com/sirupsen/logrus"
)

func RelWD(path string) string {
	cwd, _ := os.Getwd()
	relPath, _ := filepath.Rel(cwd, path)

	return relPath
}

func MkCleanDir(path string, protected []string, dryRun bool) error {
	log.Debug("MkCleanDir (dry run: ", dryRun, "): ", RelWD(path))

	exists, err := IsDir(path)
	if dryRun {
		if !exists {
			return fmt.Errorf("%s does not exist or is not a directory", path)
		} else {
			return nil
		}
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0777)
		return err
	}
	if !exists {
		return fmt.Errorf("%s exists and is not a directory", path)
	}

	log.Debug("Cleaning ", RelWD(path))
	return RmDir(path, protected, dryRun)
}

func MkDir(path string, dryRun bool) error {
	exists, err := IsDir(path)
	if dryRun {
		if !exists {
			return fmt.Errorf("%s does not exist or is not a directory", path)
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
	log.Trace("IsDir: ", path)

	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return s.IsDir(), nil
}

func IsFile(path string) (bool, error) {
	log.Trace("IsRegular: ", path)
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

func ContainsKustomization(path string) bool {
	log.Debug("Checking for kustomization.yaml at: ", path)
	kustomizations := []string{
		"kustomization",
		"Kustomization",
		"kustomization.yaml",
		"Kustomization.yaml",
		"kustomization.yml",
		"Kustomization.yml",
	}

	for _, kustomization := range kustomizations {
		ft, err := IsFile(filepath.Join(path, kustomization))
		if ft && err == nil {
			log.Trace("Found one of ", kustomizations, " in ", path)
			return true
		}
	}

	log.Trace("Did not find kustomizations in ", path)
	return false
}

func RmDir(path string, protected []string, dryRun bool) error {
	log.Debug("RmDir (dry run: ", dryRun, "): ", RelWD(path))

	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	subDirs, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, subDir := range subDirs {
		_, found := slices.BinarySearch(protected, subDir)
		if found {
			log.Info("Refusing to remove protected path: ", RelWD(subDir))
			continue
		}
		log.Trace("Removing path: ", subDir)
		err = os.RemoveAll(filepath.Join(path, subDir))
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteFile(path string, b []byte, mode uint32, dryRun bool) error {
	log.Debug("WriteFile (dry run: ", dryRun, "): ", RelWD(path))

	if dryRun {
		existingData, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading existing file: %w", err)
		}

		if string(existingData) == string(b) {
			return nil
		}

		return errors.New("dry run: file contents would be changed")
	}

	err := os.WriteFile(path, b, os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}
