package fkt

//
// Secrets last encode date
// Template file with secret yaml last update date FS
// Save cluster secrets yaml doc from existing if exists and use if
//    secrets.yaml last update date <= generated file date
//    ~template file update date is <= generated file date
//
// Store resource file dates prior to removal
// ~Get secrets.yaml last update date~

import (
	"os"
	"time"

	utils "github.com/clingclangclick/fkt/utils"
	decrypt "github.com/getsops/sops/v3/decrypt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func (s *Secrets) read(path string) error {
	log.Info("Decrypting secrets from ", path)

	secretsFileExists, err := utils.IsFile(path)
	if secretsFileExists && err == nil {
		sopsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lastModified, err := utils.SOPSLastModified(sopsBytes)
		if err != nil {
			return err
		}
		s.lastModified = &lastModified

		contents, err := decrypt.File(path, "yaml")
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(contents, &s.values)
		return err
	} else {
		return err
	}
}

type Secrets struct {
	values       Values
	ageKey       string
	lastModified *time.Time
}
