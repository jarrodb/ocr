package client

// These mirror the server-side response types in pkg/server/types.go.
// They're duplicated here so consumers can import this SDK without pulling in
// the mock's server package.

type Box struct {
	XMin int `json:"xmin"`
	YMin int `json:"ymin"`
	XMax int `json:"xmax"`
	YMax int `json:"ymax"`
}

type Region struct {
	Code  string  `json:"code"`
	Score float64 `json:"score"`
}

type Candidate struct {
	Plate string  `json:"plate"`
	Score float64 `json:"score"`
}

type Vehicle struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
	Box   Box     `json:"box"`
}

type ModelMake struct {
	Make  string  `json:"make"`
	Model string  `json:"model"`
	Score float64 `json:"score"`
}

type ColorScore struct {
	Color string  `json:"color"`
	Score float64 `json:"score"`
}

type OrientationScore struct {
	Orientation string  `json:"orientation"`
	Score       float64 `json:"score"`
}

type YearRange struct {
	YearRange [2]int  `json:"year_range"`
	Score     float64 `json:"score"`
}

type Result struct {
	Plate      string             `json:"plate"`
	Score      float64            `json:"score"`
	DScore     float64            `json:"dscore"`
	Box        Box                `json:"box"`
	Region     Region             `json:"region"`
	Candidates []Candidate        `json:"candidates"`
	Vehicle    Vehicle            `json:"vehicle"`
	ModelMake  []ModelMake        `json:"model_make,omitempty"`
	Color      []ColorScore       `json:"color,omitempty"`
	Orient     []OrientationScore `json:"orientation,omitempty"`
	Year       *YearRange         `json:"year,omitempty"`
}

type PlateReaderResponse struct {
	ProcessingTime float64  `json:"processing_time"`
	Results        []Result `json:"results"`
	Filename       string   `json:"filename"`
	Version        int      `json:"version"`
	CameraID       *string  `json:"camera_id"`
	Timestamp      string   `json:"timestamp"`
}

type Usage struct {
	Month    int    `json:"month"`
	Calls    int    `json:"calls"`
	Year     int    `json:"year"`
	ResetsOn string `json:"resets_on"`
}

type StatisticsResponse struct {
	Usage      Usage `json:"usage"`
	TotalCalls int   `json:"total_calls"`
}

// ReadParams configures a single POST /v1/plate-reader/ call.
// Provide exactly one of Image+ContentType, UploadURL.
type ReadParams struct {
	Image       []byte
	Filename    string
	ContentType string
	UploadURL   string
	Regions     []string
	MMC         bool
	CameraID    string
	Timestamp   string
}

// APIError is returned for non-2xx responses.
type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	return e.Detail
}
