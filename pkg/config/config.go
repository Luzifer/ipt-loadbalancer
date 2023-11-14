// Package config defines the syntax of the configuration file
package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/Luzifer/go_helpers/v2/fieldcollection"
	"gopkg.in/yaml.v3"
)

type (
	// File wraps the whole config file content
	File struct {
		ManagedChain string    `yaml:"managedChain"`
		Services     []Service `yaml:"services"`
	}

	// Service represents a single service to be exposed
	Service struct {
		Name        string             `yaml:"name"`
		HealthCheck ServiceHealthCheck `yaml:"healthCheck"`
		BindAddr    string             `yaml:"bindAddr"`
		BindPort    int                `yaml:"bindPort"`
		Proto       string             `yaml:"proto"`
		Targets     []Target           `yaml:"targets"`
	}

	// ServiceHealthCheck defines type and settings for the health-
	// check to apply to the targets to deem them alive
	ServiceHealthCheck struct {
		Type     string                           `yaml:"type"`
		Interval time.Duration                    `yaml:"interval"`
		Settings *fieldcollection.FieldCollection `yaml:"settings"`
	}

	// Target represents a load-balancing target to route the traffic
	// to in case it is deemed alive
	Target struct {
		Addr      string `yaml:"addr"`
		LocalAddr string `yaml:"localAddr"`
		Port      int    `yaml:"port"`
		Weight    int    `yaml:"weight"`
	}
)

//go:embed default.yaml
var defaultConfig []byte

// Load reads the configuration file from disk and parses it over the
// included default configuration
func Load(fn string) (cf File, err error) {
	defConf := yaml.NewDecoder(bytes.NewReader(defaultConfig))
	defConf.KnownFields(true)
	if err = defConf.Decode(&cf); err != nil {
		return cf, fmt.Errorf("unmarshalling default config: %w", err)
	}

	f, err := os.Open(fn) //#nosec:G304 // This is intended to load a custom config file
	if err != nil {
		return cf, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	fileConf := yaml.NewDecoder(f)
	fileConf.KnownFields(true)
	if err = fileConf.Decode(&cf); err != nil {
		return cf, fmt.Errorf("unmarshalling config file: %w", err)
	}

	return cf, nil
}

// Protocol evaluates the Proto and returns tcp if empty
func (s Service) Protocol() string {
	if s.Proto == "" {
		return "tcp"
	}
	return s.Proto
}

func (t Target) String() string { return fmt.Sprintf("%s:%d", t.Addr, t.Port) }
