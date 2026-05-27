package ocr

import (
	"context"

	"github.com/jarrodb/ocr/pkg/config"
)

// Engine returns a plate reading via fixture match → deterministic generator.
// Both tiers always succeed; Read never errors.
type Engine struct {
	cfg       *config.Config
	fixtures  *fixtureMatcher
	generator *generator
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		cfg:       cfg,
		fixtures:  newFixtureMatcher(cfg.Fixtures, cfg.DefaultRegion),
		generator: newGenerator(cfg.GeneratedPlatePrefix, cfg.DefaultRegion, cfg.Vehicles),
	}
}

func (e *Engine) Read(_ context.Context, req Request) Reading {
	if r, ok := e.fixtures.match(req); ok {
		return r
	}
	return e.generator.generate(req)
}
