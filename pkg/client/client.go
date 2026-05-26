// Package client is a small Go SDK for the Platerecognizer Snapshot Cloud API.
//
// Platerecognizer does not publish a first-party Go SDK; their official SDK is
// Python (marcbelmont/deep-license-plate-recognition). This package mirrors
// that SDK's surface area so tests of the mock service can exercise the same
// call patterns a production caller would use.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.platerecognizer.com"

type Config struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type Client struct {
	baseURL string
	token   string
	hc      *http.Client
}

func New(cfg Config) (*Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("client: Token is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(base, "/"), token: cfg.Token, hc: hc}, nil
}

// Read calls POST /v1/plate-reader/.
func (c *Client) Read(ctx context.Context, p ReadParams) (*PlateReaderResponse, error) {
	if len(p.Image) == 0 && p.UploadURL == "" {
		return nil, fmt.Errorf("Read: Image or UploadURL is required")
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	if len(p.Image) > 0 {
		ct := p.ContentType
		if ct == "" {
			ct = "image/jpeg"
		}
		fname := p.Filename
		if fname == "" {
			fname = "upload.jpg"
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="upload"; filename=%q`, fname))
		h.Set("Content-Type", ct)
		part, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(p.Image); err != nil {
			return nil, err
		}
	}
	if p.UploadURL != "" {
		_ = mw.WriteField("upload_url", p.UploadURL)
	}
	if len(p.Regions) > 0 {
		_ = mw.WriteField("regions", strings.Join(p.Regions, ","))
	}
	if p.MMC {
		_ = mw.WriteField("mmc", "true")
	}
	if p.CameraID != "" {
		_ = mw.WriteField("camera_id", p.CameraID)
	}
	if p.Timestamp != "" {
		_ = mw.WriteField("timestamp", p.Timestamp)
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/plate-reader/", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))

	var out PlateReaderResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Statistics calls GET /v1/statistics/.
func (c *Client) Statistics(ctx context.Context) (*StatisticsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/statistics/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+c.token)
	var out StatisticsResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(b, &e)
		if e.Detail == "" {
			e.Detail = strings.TrimSpace(string(b))
		}
		return &APIError{StatusCode: resp.StatusCode, Detail: e.Detail}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(b, out)
}
