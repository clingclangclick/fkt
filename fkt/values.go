package fkt

import (
	"encoding/json"
	"os"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	log "github.com/sirupsen/logrus"
)

type Values map[string]interface{}

func (v *Values) Dump() string {
	dump, err := json.Marshal(v)
	if err != nil {
		log.Panic(err)
	}

	return string(dump[:])
}

func (v *Values) ProcessValues(values ...Values) Values {
	for _, sv := range values {
		for ik, iv := range sv {
			(*v)[ik] = iv
		}
	}

	return *v
}

func (v *Values) Template(sourcePath, destinationPath string, settings *Settings) error {
	tfd, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("cannot read template file: %s; %w", sourcePath, err)
	}

	t, err := template.New(sourcePath).Delims(settings.Delimiters.Left, settings.Delimiters.Right).Funcs(sprig.FuncMap()).Parse(string(tfd))
	t = template.Must(t, err)
	if err != nil {
		return fmt.Errorf("cannot generate template for source: %s; %w", sourcePath, err)
	}

	of, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("cannot write template to: %s; %w", destinationPath, err)
	}
	defer of.Close()

	return t.Execute(of, v)
}
