package server

// Response shapes mirror the Platerecognizer Snapshot Cloud API:
// https://guides.platerecognizer.com/docs/snapshot/api-reference/
//
// Keep field names and JSON tags exact — the Go client SDK in pkg/client
// unmarshals these, and so will any third-party caller pointed at the mock.

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

type InfoResponse struct {
	Version    string `json:"version"`
	LicenseKey string `json:"license_key"`
	TotalCalls int    `json:"total_calls"`
	Usage      struct {
		Calls int `json:"calls"`
	} `json:"usage"`
	Webhooks []any `json:"webhooks"`
}

type ErrorBody struct {
	Detail     string `json:"detail"`
	StatusCode int    `json:"status_code"`
}
