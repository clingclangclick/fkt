package fkt

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	log "github.com/sirupsen/logrus"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"

	"gopkg.in/yaml.v3"
)

type Values map[string]interface{}

type K8S struct {
	Kind string `json:"kind"`
}

func ProcessValues(values ...*Values) Values {
	v := Values{}

	for _, sv := range values {
		for ik, iv := range *sv {
			v[ik] = iv
		}
	}

	return v
}

func (v *Values) template(sourcePath, destinationPath string, settings *Settings, secrets *Secrets) error {
	tfd, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("cannot read template file: %s; %w", sourcePath, err)
	}

	readAsYaml := false
	for _, yamlExtension := range []string{".yaml", ".yml"} {
		if strings.EqualFold(filepath.Ext(sourcePath), yamlExtension) {
			readAsYaml = true
		}
	}

	fileString := &strings.Builder{}
	if readAsYaml {
		reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader([]byte(tfd))))
		multipleDocs := false
		for {
			buf, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			k8sYaml := &K8S{}
			err = yaml.Unmarshal(buf, &k8sYaml)
			if err != nil {
				return err
			}
			log.Info(secrets)
			if k8sYaml.Kind == "Secret" && secrets.ageKey != "" {
				log.Info("Adding secrets to values for Secret k8s Kind")
				(*v)["Secrets"] = secrets.values
			}

			tpl, err := v.execute(sourcePath, string(buf), settings.Delimiters.Left, settings.Delimiters.Right)
			if err != nil {
				return err
			}

			if multipleDocs {
				_, err = fileString.WriteString("---\n")
				if err != nil {
					return err
				}
			}

			var yamlFile string
			if k8sYaml.Kind == "Secret" && secrets.ageKey != "" {
				yamlFileString, err := encrypt(tpl.String(), secrets.ageKey)
				if err != nil {
					return err
				}
				yamlFile = string(yamlFileString)
			} else {
				yamlFile = tpl.String()
			}

			_, err = fileString.WriteString(yamlFile)
			if err != nil {
				return err
			}
			multipleDocs = true
		}
	} else {
		tpl, err := v.execute(sourcePath, string(tfd), settings.Delimiters.Left, settings.Delimiters.Right)
		if err != nil {
			return err
		}
		_, err = fileString.WriteString(tpl.String())
		if err != nil {
			return err
		}
	}

	of, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("cannot write template to: %s; %w", destinationPath, err)
	}
	defer of.Close()

	_, err = of.WriteString(fileString.String())
	if err != nil {
		return err
	}

	return nil
}

func (v *Values) execute(name, text, leftDelimiter, rightDelimiter string) (*strings.Builder, error) {
	t, err := template.New(name).
		Delims(leftDelimiter, rightDelimiter).
		Funcs(sprig.FuncMap()).
		Parse(text)

	t = template.Must(t, err)
	if err != nil {
		return &strings.Builder{}, fmt.Errorf("cannot generate template: %s; %w", name, err)
	}

	var tpl strings.Builder
	if err := t.Execute(&tpl, v); err != nil {
		return &strings.Builder{}, err
	}

	return &tpl, nil
}

func encrypt(yamlString string, ageKey string) ([]byte, error) {
	var encrypted []byte

	_, err := exec.LookPath("sops")
	if err != nil {
		return encrypted, err
	}

	cmd := exec.Command(
		"sops",
		"--encrypt",
		"--age", ageKey,
		"--encrypted-regex", "^(data|stringData)$",
		"--input-type", "yaml",
		"--output-type", "yaml",
		"/dev/stdin",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return encrypted, err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, yamlString)
	}()

	encrypted, _ = cmd.Output()
	if err != nil {
		return encrypted, err
	}

	return encrypted, nil
}
