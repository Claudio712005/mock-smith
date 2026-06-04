// Package config carrega um arquivo mocksmith.yaml e o converte nas Options do
// runtime. O config é a base; quem chama (a CLI) sobrepõe com as flags.
package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/app"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"gopkg.in/yaml.v3"
)

// DefaultFile é o nome procurado no diretório atual quando --config não recebe
// um caminho.
const DefaultFile = "mocksmith.yaml"

// Config é a forma declarativa do mocksmith.yaml.
type Config struct {
	Spec        string              `yaml:"spec"`
	Addr        string              `yaml:"addr"`
	Profile     string              `yaml:"profile"`
	ForceStatus int                 `yaml:"forceStatus"`
	Endpoints   map[string]Endpoint `yaml:"endpoints"`
	Overrides   map[string]Override `yaml:"overrides"`
}

// Endpoint reúne os comportamentos por endpoint declarados no config.
type Endpoint struct {
	Fail     int    `yaml:"fail"`
	Slow     string `yaml:"slow"`
	Timeout  string `yaml:"timeout"`
	Corrupt  string `yaml:"corrupt"`
	Sequence []int  `yaml:"sequence"`
}

// Override é o estado inicial de runtime de um endpoint (Admin API).
type Override struct {
	Status    int     `yaml:"status"`
	LatencyMs int     `yaml:"latencyMs"`
	Rate      float64 `yaml:"rate"`
}

// Load lê e faz o parse do arquivo de config no caminho informado.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return &c, nil
}

// ToOptions converte o config nas Options do runtime, traduzindo os blocos por
// endpoint nas listas "path=valor" que o app já entende. Os endpoints são
// percorridos em ordem estável.
func (c *Config) ToOptions() app.Options {
	opts := app.Options{
		SpecPath:    c.Spec,
		Addr:        c.Addr,
		Profile:     c.Profile,
		ForceStatus: c.ForceStatus,
	}

	for _, path := range sortedKeys(c.Endpoints) {
		ep := c.Endpoints[path]
		if ep.Slow != "" {
			opts.Slow = append(opts.Slow, path+"="+ep.Slow)
		}
		if ep.Fail != 0 {
			opts.Fail = append(opts.Fail, path+"="+strconv.Itoa(ep.Fail))
		}
		if ep.Timeout != "" {
			opts.Timeout = append(opts.Timeout, path+"="+ep.Timeout)
		}
		if ep.Corrupt != "" {
			opts.Corrupt = append(opts.Corrupt, path+"="+ep.Corrupt)
		}
		if len(ep.Sequence) > 0 {
			opts.Sequence = append(opts.Sequence, path+"="+joinInts(ep.Sequence))
		}
	}

	if len(c.Overrides) > 0 {
		opts.Overrides = make(map[string]interceptor.Override, len(c.Overrides))
		for path, ov := range c.Overrides {
			opts.Overrides[path] = interceptor.Override{
				Status:    ov.Status,
				LatencyMs: ov.LatencyMs,
				Rate:      ov.Rate,
			}
		}
	}

	return opts
}

func sortedKeys(m map[string]Endpoint) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ",")
}
