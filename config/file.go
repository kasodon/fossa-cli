package config

import (
	"fmt"
	"io/ioutil"
	"os"

	git "gopkg.in/src-d/go-git.v4"
	yaml "gopkg.in/yaml.v2"
)

type configFileV1 struct {
	Version int `yaml:"version"`

	CLI     configFileCLIV1
	Analyze configFileAnalyzeV1
}

type configFileCLIV1 struct {
	// Upload configuration.
	APIKey  string `yaml:"api_key,omitempty"`
	Server  string `yaml:"server,omitempty"`
	Project string `yaml:"project,omitempty"`
	Locator string `yaml:"locator,omitempty"`
}

type configFileAnalyzeV1 struct {
	Modules []ModuleConfig `yaml:"modules,omitempty"`
}

func readConfigFile(path string) (string, configFileV1, error) {
	if _, err := os.Stat(path); path != "" && err != nil && os.IsNotExist(err) {
		return path, configFileV1{}, fmt.Errorf("invalid config file specified")
	} else if _, err := os.Stat(".fossa.yml"); err == nil {
		path = ".fossa.yml"
	} else if _, err = os.Stat(".fossa.yaml"); err == nil {
		path = ".fossa.yaml"
	}

	if path == "" {
		conf, err := setDefaultValues(configFileV1{})
		return path, conf, err
	}
	conf, err := parseConfigFile(path)
	return path, conf, err
}

func parseConfigFile(filename string) (configFileV1, error) {
	// Read configuration file.
	var config configFileV1

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}

	config, err = setDefaultValues(config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func setDefaultValues(c configFileV1) (configFileV1, error) {
	// Set config version
	c.Version = 1

	// Set default endpoint.
	if c.CLI.Server == "" {
		c.CLI.Server = os.Getenv("FOSSA_ENDPOINT")
		if c.CLI.Server == "" {
			c.CLI.Server = "https://app.fossa.io"
		}
	}

	// Load API key from environment variable.
	if c.CLI.APIKey == "" {
		c.CLI.APIKey = os.Getenv("FOSSA_API_KEY")
	}

	// Infer default locator and project from `git`.
	if c.CLI.Locator == "" {
		// TODO: this needs to happen in the module directory, not the working
		// directory
		repo, err := git.PlainOpen(".")
		if err == nil {
			project := c.CLI.Project
			if project == "" {
				origin, err := repo.Remote("origin")
				if err == nil && origin != nil {
					project = origin.Config().URLs[0]
					c.CLI.Project = project
				}
			}

			revision, err := repo.Head()
			if err == nil {
				c.CLI.Locator = "git+" + project + "$" + revision.Hash().String()
			}
		}
	}

	return c, nil
}

// WriteConfigFile writes a config state to yaml
func WriteConfigFile(conf *CLIConfig) error {
	if conf.ConfigFilePath == "" {
		conf.ConfigFilePath = ".fossa.yml"
	}

	writeConfig := configFileV1{
		Version: 1,
		CLI: configFileCLIV1{
			APIKey:  conf.APIKey,
			Server:  conf.Endpoint,
			Project: conf.Project,
		},
		Analyze: configFileAnalyzeV1{
			Modules: conf.Modules,
		},
	}

	yamlConfig, err := yaml.Marshal(writeConfig)
	if err != nil {
		return err
	}

	configHeader := []byte(`# Generated by FOSSA CLI (https://github.com/fossas/fossa-cli)
# Visit https://fossa.io to learn more
`)

	return ioutil.WriteFile(conf.ConfigFilePath, append(configHeader, yamlConfig...), 0777)
}
