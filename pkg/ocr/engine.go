package ocr

import (
	"context"
	"fmt"
	"log"

	"github.com/jarrodb/ocr/pkg/config"
)

// Engine is the OCR ladder: fixture → real OCR → generated.
// Each tier is permitted to fail; the next tier covers it. The final tier
// (generator) cannot fail.
type Engine struct {
	cfg       *config.Config
	fixtures  *fixtureMatcher
	tesseract *tesseractEngine
	generator *generator
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		cfg:       cfg,
		fixtures:  newFixtureMatcher(cfg.Fixtures, cfg.DefaultRegion),
		tesseract: newTesseractEngine(cfg.Tesseract),
		generator: newGenerator(cfg.GeneratedPlatePrefix, cfg.DefaultRegion, cfg.Vehicles),
	}
}

// Read walks the ladder. Returns at most one Reading per call (the public PR
// API can return multiple plates per image, but for the mock a single plate is
// the realistic case and keeps test assertions tight).
func (e *Engine) Read(ctx context.Context, req Request) Reading {
	if r, ok := e.fixtures.match(req); ok {
		return r
	}

	if e.cfg.Tesseract.Enabled && len(req.Image) > 0 {
		r, err := e.tesseract.read(ctx, req)
		switch {
		case err != nil:
			log.Printf("ocr: tesseract failed (%v) — falling back to generated", err)
		case r.Score*100 < e.cfg.Tesseract.MinConfidence:
			log.Printf("ocr: tesseract confidence %.1f < min %.1f — falling back to generated", r.Score*100, e.cfg.Tesseract.MinConfidence)
		case r.Plate == "":
			log.Printf("ocr: tesseract returned empty plate — falling back to generated")
		default:
			e.generator.fillVehicle(&r)
			return r
		}
	}

	return e.generator.generate(req)
}

// EngineError is returned for unrecoverable engine-level failures (not used by
// Read, which always returns a Reading by design — kept for future endpoints
// that may need to surface engine availability).
type EngineError struct{ Err error }

func (e *EngineError) Error() string { return fmt.Sprintf("ocr engine: %v", e.Err) }
