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
	utils "github.com/clingclangclick/fkt/utils"
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

func (v *Values) template(templatePath, targetPath string, settings *Settings, secrets *Secrets) error {
	tfd, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("cannot read template file: %s; %w", templatePath, err)
	}

	readAsYaml := false
	for _, yamlExtension := range []string{".yaml", ".yml"} {
		if strings.EqualFold(filepath.Ext(templatePath), yamlExtension) {
			readAsYaml = true
		}
	}

	// Non-YAML files not read in as mulitdoc for k8s kind processing for secrets
	if !readAsYaml {
		tpl, err := v.execute(templatePath, string(tfd), settings.Delimiters.Left, settings.Delimiters.Right)
		if err != nil {
			return err
		}

		err = os.RemoveAll(targetPath)
		if err != nil {
			return err
		}

		err = utils.WriteFile(targetPath, []byte(tpl.String()), 0666, false)
		if err != nil {
			return err
		}
		return nil
	}

	// Do not update secrets when:
	// - targetPath file exists (already written)
	// - targetPath last modified time after modified times of:
	//   - template file
	//   - configuration file
	//   - secrets file, if file exists
	targetPathIsFile, err := utils.IsFile(targetPath)
	if targetPathIsFile && err == nil {
		targetPathInfo, err := os.Stat(targetPath)
		if err != nil {
			return err
		}
		targetPathModified := targetPathInfo.ModTime().UTC()

		templatePathInfo, err := os.Stat(templatePath)
		if err != nil {
			return err
		}
		templatePathModified := templatePathInfo.ModTime().UTC()
		if err != nil {
			return err
		}
		if targetPathModified.After(templatePathModified) &&
			targetPathModified.After(settings.configFileModifiedTime) &&
			secrets.lastModified != nil && targetPathModified.After(*secrets.lastModified) {
			log.Trace(utils.RelWD(targetPath), " modified after template, config, and secrets file, not modifying")
			return nil
		} else {
			log.Trace("Regenerating ", utils.RelWD(targetPath))
		}
	}

	fileString := &strings.Builder{}
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

		if multipleDocs {
			_, err = fileString.WriteString("---\n")
			if err != nil {
				return err
			}
		}

		k8sYaml := &K8S{}
		err = yaml.Unmarshal(buf, &k8sYaml)
		if err != nil {
			return err
		}
		if k8sYaml.Kind == "Secret" && secrets.ageKey != "" {
			log.Info("Adding secrets to values for Secret k8s Kind")
			(*v)["Secrets"] = secrets.values
		}

		tpl, err := v.execute(templatePath, string(buf), settings.Delimiters.Left, settings.Delimiters.Right)
		if err != nil {
			return err
		}

		var yamlFile string
		if k8sYaml.Kind == "Secret" {
			if secrets.ageKey == "" {
				return fmt.Errorf("secret templated but no age public key exists for cluster")
			}
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

	err = os.RemoveAll(targetPath)
	if err != nil {
		return err
	}

	err = utils.WriteFile(targetPath, []byte(fileString.String()), 0666, false)
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

	errs := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, err := io.WriteString(stdin, yamlString)
		errs <- err
	}()

	if err := <-errs; err != nil {
		return encrypted, err
	}

	encrypted, _ = cmd.Output()
	if err != nil {
		return encrypted, err
	}

	return encrypted, nil
}
