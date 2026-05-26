package ocr

import (
	"hash/fnv"
	"math/rand"
	"strings"

	"github.com/jarrodb/ocr/pkg/config"
)

type generator struct {
	prefix        string
	defaultRegion string
	vehicles      []config.VehicleSeed
}

func newGenerator(prefix, defaultRegion string, vehicles []config.VehicleSeed) *generator {
	if prefix == "" {
		prefix = "MCK"
	}
	return &generator{prefix: prefix, defaultRegion: defaultRegion, vehicles: vehicles}
}

// generate produces a synthetic reading. The plate uses the configured prefix
// (default "MCK") so callers/tests can identify generated results by string
// inspection without needing a side-channel field.
func (g *generator) generate(req Request) Reading {
	seed := seedFromRequest(req)
	r := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic mock, not security-sensitive

	plate := g.prefix + randomDigits(r, 4)

	out := Reading{
		Plate:      strings.ToLower(plate),
		Score:      0.85 + r.Float64()*0.1,
		DScore:     0.95 + r.Float64()*0.04,
		RegionCode: g.defaultRegion,
		Source:     SourceGenerated,
	}
	if len(req.Regions) > 0 {
		out.RegionCode = req.Regions[0]
	}
	g.fillVehicle(&out)
	return out
}

// fillVehicle picks a deterministic vehicle for a plate. Used by both the
// generated and OCR paths — once we have a plate string, the vehicle metadata
// is always synthesized from it (Tesseract reads text, not cars).
func (g *generator) fillVehicle(r *Reading) {
	if len(g.vehicles) == 0 {
		return
	}
	if r.Make != "" || r.Model != "" {
		return
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(r.Plate))
	v := g.vehicles[int(h.Sum64()%uint64(len(g.vehicles)))]
	r.Make = v.Make
	r.Model = v.Model
	r.Color = v.Color
	r.Type = v.Type
	r.YearMin = v.YearMin
	r.YearMax = v.YearMax
}

func seedFromRequest(req Request) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(req.UploadURL))
	_, _ = h.Write([]byte(req.Filename))
	if len(req.Image) > 0 {
		// Hash only the first 4KB to keep seeding cheap on large uploads.
		_, _ = h.Write(req.Image[:min(len(req.Image), 4096)])
	}
	return int64(h.Sum64()) //nolint:gosec // seed only
}

func randomDigits(r *rand.Rand, n int) string {
	var b strings.Builder
	for range n {
		b.WriteByte(byte('0' + r.Intn(10)))
	}
	return b.String()
}
