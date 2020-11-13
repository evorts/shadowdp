package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

type App struct {
	Port int `yaml:"port"`
	RemoteCdp struct{
		Enabled bool `yaml:"enabled"`
		Address string `yaml:"address"`
	} `yaml:"remote_cdp"`
}

type Mapping struct {
	FromHost  string `yaml:"from_host"`
	ToBaseUrl string `yaml:"to_base_url"`
}

type Config struct {
	App      App       `yaml:"app"`
	Mappings []Mapping `yaml:"mappings"`
}

type config struct {
	dir      string
	filename []string
	data     *Config
}

type IManager interface {
	GetConfig() *Config
	Initiate() (IManager, error)
	GetMapByHost(host string) *Mapping
	Reload() error
}

func NewConfig(dir string, filename ...string) IManager {
	return &config{
		dir:      dir,
		filename: filename,
		data:     nil,
	}
}

func (c *config) GetConfig() *Config {
	if c.data == nil {
		c.data, _ = c.read()
	}
	return c.data
}

func (c *config) GetApp() App {
	return c.data.App
}

func (c *config) GetMapByHost(host string) *Mapping {
	for _, mapping := range c.GetConfig().Mappings {
		if mapping.FromHost == host {
			return &mapping
		}
	}
	return nil
}

func (c *config) Initiate() (IManager, error) {
	data, err := c.read()
	if err != nil {
		return nil, err
	}
	c.data = data
	return c, nil
}

func (c *config) Reload() (err error) {
	c.data, err = c.read()
	return
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func (c *config) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (c *config) read() (*Config, error) {
	fName := ""
	for _, f := range c.filename {
		if len(c.dir) > 0 {
			f = fmt.Sprintf("%s/%s", strings.TrimSuffix(c.dir, "/"), strings.TrimPrefix(f, "/"))
		}
		if c.fileExists(f) {
			fName = f
			break
		}
	}
	if len(fName) < 1 {
		return nil, errors.New("no configuration file found")
	}
	cfg, err := ioutil.ReadFile(fName)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(cfg, &config)
	if err != nil {
		return nil, err

	}
	return &config, nil
}
