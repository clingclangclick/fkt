package fkt

import (
	utils "github.com/clingclangclick/fkt/utils"
	decrypt "github.com/getsops/sops/v3/decrypt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func (s *Secrets) read(path string) error {
	log.Info("Decrypting secrets at ", path)

	secretsFileExists, err := utils.IsFile(path)
	if secretsFileExists && err == nil {
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
	values Values
	ageKey string
}
