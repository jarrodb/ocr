package server_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jarrodb/ocr/pkg/client"
	"github.com/jarrodb/ocr/pkg/config"
	"github.com/jarrodb/ocr/pkg/server"
)

func setup(t *testing.T) (*httptest.Server, *client.Client) {
	t.Helper()
	cfg := &config.Config{
		Port:                 0,
		APIToken:             "test-token",
		AuthRequired:         true,
		Mode:                 "test",
		Delay:                "0s",
		CORSOrigins:          []string{"*"},
		DefaultRegion:        "us-ca",
		GeneratedPlatePrefix: "MCK",
		Fixtures: []config.Fixture{
			{Match: "tesla", Plate: "tsla123", RegionCode: "us-ca", Make: "Tesla", Model: "Model 3", Color: "red", Type: "Sedan", YearMin: 2021, YearMax: 2023},
		},
		Vehicles: []config.VehicleSeed{
			{Make: "Toyota", Model: "Camry", Color: "silver", Type: "Sedan", YearMin: 2018, YearMax: 2022},
		},
	}
	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	c, err := client.New(client.Config{BaseURL: ts.URL, Token: "test-token"})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	return ts, c
}

func TestPlateReader_FixtureMatch(t *testing.T) {
	_, c := setup(t)

	resp, err := c.Read(context.Background(), client.ReadParams{
		UploadURL: "https://example.com/photos/tesla-front.jpg",
		MMC:       true,
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	got := resp.Results[0]
	if got.Plate != "tsla123" {
		t.Errorf("plate = %q, want %q", got.Plate, "tsla123")
	}
	if len(got.ModelMake) == 0 || got.ModelMake[0].Make != "Tesla" {
		t.Errorf("expected make Tesla, got %+v", got.ModelMake)
	}
	if got.Region.Code != "us-ca" {
		t.Errorf("region = %q, want us-ca", got.Region.Code)
	}
}

func TestPlateReader_GeneratedFallback(t *testing.T) {
	_, c := setup(t)

	resp, err := c.Read(context.Background(), client.ReadParams{
		UploadURL: "https://example.com/photos/unknown-car.jpg",
		MMC:       true,
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := resp.Results[0]
	if !strings.HasPrefix(strings.ToUpper(got.Plate), "MCK") {
		t.Errorf("expected MCK-prefix on generated plate, got %q", got.Plate)
	}
	if len(got.ModelMake) == 0 {
		t.Fatal("expected synthesized make/model when MMC=true")
	}
}

func TestPlateReader_GeneratedIsDeterministic(t *testing.T) {
	_, c := setup(t)

	a, err := c.Read(context.Background(), client.ReadParams{UploadURL: "https://example.com/x.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Read(context.Background(), client.ReadParams{UploadURL: "https://example.com/x.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Results[0].Plate != b.Results[0].Plate {
		t.Errorf("same upload_url produced different plates: %q vs %q", a.Results[0].Plate, b.Results[0].Plate)
	}
}

func TestPlateReader_AuthRequired(t *testing.T) {
	_, _ = setup(t)
	ts, _ := setup(t)
	bad, err := client.New(client.Config{BaseURL: ts.URL, Token: "wrong"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bad.Read(context.Background(), client.ReadParams{UploadURL: "https://example.com/x.jpg"})
	var apiErr *client.APIError
	if err == nil {
		t.Fatal("expected error with wrong token")
	}
	if !errorsAs(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Errorf("expected 403 APIError, got %v", err)
	}
}

func TestPlateReader_RequiresImageOrURL(t *testing.T) {
	_, c := setup(t)
	_, err := c.Read(context.Background(), client.ReadParams{})
	if err == nil {
		t.Fatal("expected client-side validation error")
	}
}

func TestStatistics(t *testing.T) {
	_, c := setup(t)
	_, _ = c.Read(context.Background(), client.ReadParams{UploadURL: "https://example.com/a.jpg"})
	_, _ = c.Read(context.Background(), client.ReadParams{UploadURL: "https://example.com/b.jpg"})

	stats, err := c.Statistics(context.Background())
	if err != nil {
		t.Fatalf("Statistics: %v", err)
	}
	if stats.TotalCalls < 2 {
		t.Errorf("expected total_calls >= 2, got %d", stats.TotalCalls)
	}
	if stats.Usage.Calls < 2 {
		t.Errorf("expected usage.calls >= 2, got %d", stats.Usage.Calls)
	}
}

// errorsAs is a tiny re-implementation of errors.As to keep the test file's
// import list compact.
func errorsAs(err error, target **client.APIError) bool {
	for err != nil {
		if ae, ok := err.(*client.APIError); ok {
			*target = ae
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
