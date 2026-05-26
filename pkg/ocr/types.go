package ocr

// Source tags how a plate reading was produced. Callers (tests, debug
// endpoints) can read this to distinguish real OCR from fallbacks. Real
// Platerecognizer responses do not carry this field — the API JSON omits it.
type Source string

const (
	SourceFixture   Source = "fixture"
	SourceOCR       Source = "ocr"
	SourceGenerated Source = "generated"
)

// Reading is the engine's internal output. The server translates this into the
// public PR API response shape.
type Reading struct {
	Plate      string
	Score      float64
	DScore     float64
	RegionCode string
	Make       string
	Model      string
	Color      string
	Type       string
	YearMin    int
	YearMax    int
	Source     Source
}

// Request bundles the inputs an engine needs from a single HTTP call.
type Request struct {
	Image     []byte // raw image bytes (already fetched if upload_url was used)
	UploadURL string // original URL, used for fixture matching
	Filename  string // multipart filename, used for fixture matching
	Regions   []string
	WantMMC   bool
}
