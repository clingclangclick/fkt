package fkt

import (
	"fmt"
	"os"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

type Values map[string]interface{}

func ProcessValues(values ...*Values) Values {
	v := Values{}
	for _, sv := range values {
		for ik, iv := range *sv {
			v[ik] = iv
		}
	}

	return v
}

func (v *Values) template(sourcePath, destinationPath string, settings *Settings) error {
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
