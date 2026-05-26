package ocr

import (
	"strings"

	"github.com/jarrodb/ocr/pkg/config"
)

type fixtureMatcher struct {
	items         []fixtureItem
	defaultRegion string
}

type fixtureItem struct {
	match   string
	reading Reading
}

func newFixtureMatcher(fxs []config.Fixture, defaultRegion string) *fixtureMatcher {
	m := &fixtureMatcher{defaultRegion: defaultRegion}
	for _, fx := range fxs {
		if fx.Match == "" || fx.Plate == "" {
			continue
		}
		score := fx.Score
		if score == 0 {
			score = 0.92
		}
		dscore := fx.DScore
		if dscore == 0 {
			dscore = 0.99
		}
		region := fx.RegionCode
		if region == "" {
			region = defaultRegion
		}
		m.items = append(m.items, fixtureItem{
			match: strings.ToLower(fx.Match),
			reading: Reading{
				Plate:      strings.ToLower(fx.Plate),
				Score:      score,
				DScore:     dscore,
				RegionCode: region,
				Make:       fx.Make,
				Model:      fx.Model,
				Color:      fx.Color,
				Type:       fx.Type,
				YearMin:    fx.YearMin,
				YearMax:    fx.YearMax,
				Source:     SourceFixture,
			},
		})
	}
	return m
}

func (m *fixtureMatcher) match(req Request) (Reading, bool) {
	hay := strings.ToLower(req.UploadURL + " " + req.Filename)
	for _, it := range m.items {
		if strings.Contains(hay, it.match) {
			return it.reading, true
		}
	}
	return Reading{}, false
}
