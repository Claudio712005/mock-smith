// Package config carrega um arquivo mocksmith.yaml e o converte nas Options do
// runtime. O config é a base; quem chama (a CLI) sobrepõe com as flags. Suporta
// um único server (campos no topo) ou vários (bloco "servers").
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

// Config é a forma declarativa do mocksmith.yaml. Os campos de server ficam
// inline (modo single); o bloco "servers" liga o modo multi.
type Config struct {
	ServerConfig `yaml:",inline"`
	Servers      []ServerConfig `yaml:"servers"`
}

// ServerConfig descreve um server: spec, escuta, profile e comportamentos e
// regras de valor por endpoint.
type ServerConfig struct {
	Spec        string              `yaml:"spec"`
	Addr        string              `yaml:"addr"`
	Profile     string              `yaml:"profile"`
	ForceStatus int                 `yaml:"forceStatus"`
	Endpoints   map[string]Endpoint `yaml:"endpoints"`
	Overrides   map[string]Override `yaml:"overrides"`
	Values      map[string]any      `yaml:"values"`
	Count       map[string]any      `yaml:"count"`
}

// Endpoint reúne, por endpoint, os comportamentos (fail/slow/...) e as regras de
// valor (values/count).
type Endpoint struct {
	Fail     int            `yaml:"fail"`
	Slow     string         `yaml:"slow"`
	Timeout  string         `yaml:"timeout"`
	Corrupt  string         `yaml:"corrupt"`
	Sequence []int          `yaml:"sequence"`
	Values   map[string]any `yaml:"values"`
	Count    map[string]any `yaml:"count"`
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

// ServerList resolve os servers declarados: o bloco "servers" se presente, senão
// o server único inline.
func (c *Config) ServerList() ([]ServerConfig, error) {
	if len(c.Servers) > 0 {
		if c.ServerConfig.Spec != "" {
			return nil, fmt.Errorf("config has both a top-level 'spec' and a 'servers' block; use one or the other")
		}
		return c.Servers, nil
	}
	return []ServerConfig{c.ServerConfig}, nil
}

// ToOptions converte um server nas Options do runtime, traduzindo comportamentos
// por endpoint nas listas "path=valor" e normalizando as regras de valor.
func (s ServerConfig) ToOptions() (app.Options, error) {
	opts := app.Options{
		SpecPath:    s.Spec,
		Addr:        s.Addr,
		Profile:     s.Profile,
		ForceStatus: s.ForceStatus,
	}

	for _, key := range sortedKeys(s.Endpoints) {
		ep := s.Endpoints[key]
		if ep.Slow != "" {
			opts.Slow = append(opts.Slow, key+"="+ep.Slow)
		}
		if ep.Fail != 0 {
			opts.Fail = append(opts.Fail, key+"="+strconv.Itoa(ep.Fail))
		}
		if ep.Timeout != "" {
			opts.Timeout = append(opts.Timeout, key+"="+ep.Timeout)
		}
		if ep.Corrupt != "" {
			opts.Corrupt = append(opts.Corrupt, key+"="+ep.Corrupt)
		}
		if len(ep.Sequence) > 0 {
			opts.Sequence = append(opts.Sequence, key+"="+joinInts(ep.Sequence))
		}
	}

	if len(s.Overrides) > 0 {
		opts.Overrides = make(map[string]interceptor.Override, len(s.Overrides))
		for key, ov := range s.Overrides {
			opts.Overrides[key] = interceptor.Override{
				Status:    ov.Status,
				LatencyMs: ov.LatencyMs,
				Rate:      ov.Rate,
			}
		}
	}

	global, err := toRuleSet(s.Values, s.Count)
	if err != nil {
		return app.Options{}, fmt.Errorf("global rules: %w", err)
	}
	opts.GlobalRules = global

	for _, key := range sortedKeys(s.Endpoints) {
		ep := s.Endpoints[key]
		if len(ep.Values) == 0 && len(ep.Count) == 0 {
			continue
		}
		rs, err := toRuleSet(ep.Values, ep.Count)
		if err != nil {
			return app.Options{}, fmt.Errorf("rules for %q: %w", key, err)
		}
		if opts.EndpointRules == nil {
			opts.EndpointRules = map[string]app.ValueRuleSet{}
		}
		opts.EndpointRules[key] = rs
	}

	return opts, nil
}

func toRuleSet(values, count map[string]any) (app.ValueRuleSet, error) {
	rs := app.ValueRuleSet{}
	if len(values) > 0 {
		rs.Values = make(map[string][]any, len(values))
		for matcher, v := range values {
			rs.Values[matcher] = asList(v)
		}
	}
	if len(count) > 0 {
		rs.Counts = make(map[string][]any, len(count))
		for matcher, v := range count {
			items, err := normalizeCount(v)
			if err != nil {
				return app.ValueRuleSet{}, fmt.Errorf("count %q: %w", matcher, err)
			}
			rs.Counts[matcher] = items
		}
	}
	return rs, nil
}

// asList normaliza um valor em uma lista (escalar vira lista de um item).
func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return []any{v}
}

// normalizeCount valida que cada item é int (quantidade) ou nil (null no lugar
// da lista). Aceita um int solto como lista de um item.
func normalizeCount(v any) ([]any, error) {
	raw := asList(v)
	out := make([]any, 0, len(raw))
	for _, item := range raw {
		switch n := item.(type) {
		case nil:
			out = append(out, nil)
		case int:
			if n < 0 {
				return nil, fmt.Errorf("negative count %d", n)
			}
			out = append(out, n)
		default:
			return nil, fmt.Errorf("invalid count item %v (want int or null)", item)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty count")
	}
	return out, nil
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
