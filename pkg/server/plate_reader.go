package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jarrodb/ocr/pkg/ocr"
)

const apiVersion = 1

func (s *Server) handlePlateReader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.recordCall()
	start := time.Now()

	req, err := parsePlateReaderRequest(r, s.httpClient)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	reading := s.engine.Read(r.Context(), req)
	s.simulateDelay()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildResponse(reading, req, start)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// parsePlateReaderRequest extracts the image and parameters from any of the
// three intake formats PR supports: multipart upload, base64 upload field,
// upload_url. Returns an ocr.Request ready for the engine.
func parsePlateReaderRequest(r *http.Request, httpClient *http.Client) (ocr.Request, error) {
	ct := r.Header.Get("Content-Type")
	var (
		img       []byte
		filename  string
		uploadURL string
		regions   []string
		wantMMC   bool
	)

	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return ocr.Request{}, fmt.Errorf("parse multipart: %w", err)
		}
		uploadURL = r.FormValue("upload_url")
		regions = splitCSV(r.FormValue("regions"))
		wantMMC = boolForm(r.FormValue("mmc"))

		if fh, _, err := r.FormFile("upload"); err == nil {
			defer fh.Close()
			b, err := io.ReadAll(fh)
			if err != nil {
				return ocr.Request{}, fmt.Errorf("read upload: %w", err)
			}
			img = b
		} else if v := r.FormValue("upload"); v != "" {
			b, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return ocr.Request{}, fmt.Errorf("decode base64 upload: %w", err)
			}
			img = b
		}
		if fhs := r.MultipartForm.File["upload"]; len(fhs) > 0 {
			filename = fhs[0].Filename
		}

	default:
		if err := r.ParseForm(); err != nil {
			return ocr.Request{}, fmt.Errorf("parse form: %w", err)
		}
		uploadURL = r.FormValue("upload_url")
		regions = splitCSV(r.FormValue("regions"))
		wantMMC = boolForm(r.FormValue("mmc"))
		if v := r.FormValue("upload"); v != "" {
			b, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return ocr.Request{}, fmt.Errorf("decode base64 upload: %w", err)
			}
			img = b
		}
	}

	if len(img) == 0 && uploadURL == "" {
		return ocr.Request{}, fmt.Errorf("either upload or upload_url is required")
	}

	if len(img) == 0 && uploadURL != "" {
		b, name, err := fetchURL(r.Context(), httpClient, uploadURL)
		if err != nil {
			// Non-fatal: engine can still match fixture/generate by URL.
			log.Printf("fetch upload_url failed (%v); engine will fall back", err)
		} else {
			img = b
			if filename == "" {
				filename = name
			}
		}
	}

	return ocr.Request{
		Image:     img,
		UploadURL: uploadURL,
		Filename:  filename,
		Regions:   regions,
		WantMMC:   wantMMC,
	}, nil
}

func fetchURL(ctx context.Context, c *http.Client, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("upstream %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", err
	}
	name := url
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return b, name, nil
}

func buildResponse(reading ocr.Reading, req ocr.Request, start time.Time) PlateReaderResponse {
	box := Box{XMin: 143, YMin: 481, XMax: 282, YMax: 575}
	vbox := Box{XMin: 67, YMin: 113, XMax: 908, YMax: 653}

	res := Result{
		Plate:  reading.Plate,
		Score:  round3(reading.Score),
		DScore: round3(reading.DScore),
		Box:    box,
		Region: Region{Code: reading.RegionCode, Score: 0.85},
		Candidates: []Candidate{
			{Plate: reading.Plate, Score: round3(reading.Score)},
		},
		Vehicle: Vehicle{
			Type:  ifEmpty(reading.Type, "Sedan"),
			Score: 0.82,
			Box:   vbox,
		},
	}

	if req.WantMMC || reading.Make != "" {
		res.ModelMake = []ModelMake{{Make: reading.Make, Model: reading.Model, Score: 0.76}}
		res.Color = []ColorScore{{Color: reading.Color, Score: 0.91}}
		res.Orient = []OrientationScore{{Orientation: "Front", Score: 0.94}}
		if reading.YearMin > 0 && reading.YearMax > 0 {
			res.Year = &YearRange{YearRange: [2]int{reading.YearMin, reading.YearMax}, Score: 0.88}
		}
	}

	filename := req.Filename
	if filename == "" {
		filename = "image.jpg"
	}

	return PlateReaderResponse{
		ProcessingTime: float64(time.Since(start).Microseconds()) / 1000.0,
		Results:        []Result{res},
		Filename:       filename,
		Version:        apiVersion,
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func boolForm(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true
	}
	return false
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000.0
}
