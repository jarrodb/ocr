package config

// Fixture maps an image identifier (substring of upload_url or filename) to a
// fully-formed response. Used by tests and seeded local demos to return
// deterministic plate readings without invoking OCR.
type Fixture struct {
	Match      string  `mapstructure:"match"`
	Plate      string  `mapstructure:"plate"`
	Score      float64 `mapstructure:"score"`
	DScore     float64 `mapstructure:"dscore"`
	RegionCode string  `mapstructure:"region_code"`
	Make       string  `mapstructure:"make"`
	Model      string  `mapstructure:"model"`
	Color      string  `mapstructure:"color"`
	YearMin    int     `mapstructure:"year_min"`
	YearMax    int     `mapstructure:"year_max"`
	Type       string  `mapstructure:"type"`
}

// VehicleSeed feeds the generator when no fixture matches and no real OCR is
// available. Picked deterministically by hashing the plate so the same input
// produces the same vehicle metadata.
type VehicleSeed struct {
	Make    string `mapstructure:"make"`
	Model   string `mapstructure:"model"`
	Color   string `mapstructure:"color"`
	Type    string `mapstructure:"type"`
	YearMin int    `mapstructure:"year_min"`
	YearMax int    `mapstructure:"year_max"`
}

type Tesseract struct {
	Enabled       bool    `mapstructure:"enabled"`
	BinaryPath    string  `mapstructure:"binary_path"`
	PSM           int     `mapstructure:"psm"`
	Language      string  `mapstructure:"language"`
	MinConfidence float64 `mapstructure:"min_confidence"`
}

type Config struct {
	Port     int
	APIToken string
	// AuthRequired toggles the cloud-vs-on-prem auth contract.
	// Cloud (true, default): every request must carry `Authorization: Token <APIToken>`.
	// On-prem (false): no header required — mirrors the real Platerecognizer
	// SDK image, which is unauthenticated because the license is baked in.
	// The official Python client's `--sdk-url` flag uses the on-prem contract.
	AuthRequired         bool
	Mode                 string
	Delay                string
	FailureRate          float64
	CORSOrigins          []string
	DefaultRegion        string
	GeneratedPlatePrefix string
	Tesseract            Tesseract
	Fixtures             []Fixture
	Vehicles             []VehicleSeed
}
