package server_test

// This file exercises the mock through the official Platerecognizer client
// (parkpow/deep-license-plate-recognition: plate_recognition.py), shelled out
// as a subprocess. There is no pip-published Python SDK — the script itself is
// the canonical Python client per their README.
//
// Run via `just test-python-sdk`, which executes the test inside the
// Dockerfile.tests image where PYTHON_BIN and OCR_PYTHON_SDK_PATH are pre-set
// and `requests` is preinstalled in a venv at /opt/sdk-venv.
//
// The test self-skips when those env vars aren't set (e.g. someone runs
// `go test` directly on the host), so this file is safe to leave in the
// regular test set.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jarrodb/ocr/pkg/config"
	"github.com/jarrodb/ocr/pkg/server"
)

// minimal valid JPEG SOI + EOI markers; the script reads bytes and uploads them
// as-is, the mock doesn't decode them, and Tesseract is disabled in this test
// config — so any non-empty file works. The fixture/generator paths match on
// the filename, not the bytes.
var fakeJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0xFF, 0xD9}

func locatePythonSDK(t *testing.T) (pythonBin, scriptPath string, ok bool) {
	t.Helper()

	py := os.Getenv("PYTHON_BIN")
	script := os.Getenv("OCR_PYTHON_SDK_PATH")
	if py == "" || script == "" {
		t.Skip("PYTHON_BIN / OCR_PYTHON_SDK_PATH not set; run `just test-python-sdk` to execute this test inside the test container")
		return "", "", false
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("plate_recognition.py not at %q: %v", script, err)
		return "", "", false
	}
	if err := exec.Command(py, "-c", "import requests").Run(); err != nil {
		t.Skipf("python interpreter %q lacks `requests`: %v", py, err)
		return "", "", false
	}
	return py, script, true
}

func startMock(t *testing.T, authRequired bool) *httptest.Server {
	t.Helper()
	cfg := &config.Config{
		Port:                 0,
		AuthRequired:         authRequired,
		APIToken:             "test-token",
		Mode:                 "test",
		Delay:                "0s",
		CORSOrigins:          []string{"*"},
		DefaultRegion:        "us-ca",
		GeneratedPlatePrefix: "MCK",
		Tesseract:            config.Tesseract{Enabled: false},
		Fixtures: []config.Fixture{
			{Match: "tesla-model3", Plate: "tsla123", RegionCode: "us-ca", Make: "Tesla", Model: "Model 3", Color: "red", Type: "Sedan", YearMin: 2021, YearMax: 2023},
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
	return ts
}

// pythonResult mirrors the per-file result object inside the JSON list that
// plate_recognition.py prints. It contains the upstream API response wrapped
// in a `results` key — the Snapshot response shape we already emit.
type pythonResult struct {
	ProcessingTime float64 `json:"processing_time"`
	Filename       string  `json:"filename"`
	Results        []struct {
		Plate     string `json:"plate"`
		ModelMake []struct {
			Make  string `json:"make"`
			Model string `json:"model"`
		} `json:"model_make"`
		Color []struct {
			Color string `json:"color"`
		} `json:"color"`
		Region struct {
			Code string `json:"code"`
		} `json:"region"`
	} `json:"results"`
}

// pythonInvocation describes how the Python script should be invoked:
//   - mode "sdk":   pass --sdk-url <mock>, no Authorization header sent.
//     Matches the real on-prem SDK contract.
//   - mode "cloud": no --sdk-url, but OCR_CLOUD_URL env points the script's
//     patched cloud endpoint at the mock. Authorization: Token
//     <apiKey> is sent. Matches the real Cloud API contract.
type pythonInvocation struct {
	mode   string
	apiKey string // sent as `Authorization: Token <apiKey>` in cloud mode
}

func runPythonSDK(t *testing.T, ts *httptest.Server, inv pythonInvocation, filename string, extraArgs ...string) (pythonResult, []byte, error) {
	t.Helper()
	py, script, ok := locatePythonSDK(t)
	if !ok {
		t.FailNow()
	}

	dir := t.TempDir()
	img := filepath.Join(dir, filename)
	if err := os.WriteFile(img, fakeJPEG, 0o644); err != nil {
		t.Fatalf("write fixture image: %v", err)
	}

	key := inv.apiKey
	if key == "" {
		key = "test-token"
	}
	args := []string{script, "-a", key}
	env := append(os.Environ(), "PYTHONUNBUFFERED=1")

	switch inv.mode {
	case "cloud":
		env = append(env, "OCR_CLOUD_URL="+ts.URL+"/v1/plate-reader/")
	case "sdk", "":
		args = append(args, "-s", ts.URL)
	default:
		t.Fatalf("unknown invocation mode %q", inv.mode)
	}

	args = append(args, extraArgs...)
	args = append(args, img)

	cmd := exec.CommandContext(context.Background(), py, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return pythonResult{}, out, err
	}

	// The script prints `json.dumps(results, indent=2)` where results is a list.
	// On some versions warnings or non-JSON lines may precede — find the JSON.
	jsonStart := strings.Index(string(out), "[")
	if jsonStart < 0 {
		return pythonResult{}, out, fmt.Errorf("no JSON in python output")
	}
	var results []pythonResult
	if err := json.Unmarshal(out[jsonStart:], &results); err != nil {
		return pythonResult{}, out, fmt.Errorf("decode python output: %w", err)
	}
	if len(results) != 1 {
		return pythonResult{}, out, fmt.Errorf("expected 1 result, got %d", len(results))
	}
	return results[0], out, nil
}

// mustRunSDK is the happy-path wrapper that t.Fatals on any failure.
func mustRunSDK(t *testing.T, ts *httptest.Server, inv pythonInvocation, filename string, extraArgs ...string) pythonResult {
	t.Helper()
	got, out, err := runPythonSDK(t, ts, inv, filename, extraArgs...)
	if err != nil {
		t.Fatalf("plate_recognition.py failed (%v):\n%s", err, out)
	}
	return got
}

// --- on-prem SDK contract (--sdk-url, no auth header) ----------------------

func TestPythonSDK_SDKMode_FixtureMatch(t *testing.T) {
	ts := startMock(t, false)
	got := mustRunSDK(t, ts, pythonInvocation{mode: "sdk"}, "tesla-model3.jpg", "--mmc")

	if len(got.Results) != 1 {
		t.Fatalf("expected 1 plate in response, got %d", len(got.Results))
	}
	r := got.Results[0]
	if r.Plate != "tsla123" {
		t.Errorf("plate = %q, want %q", r.Plate, "tsla123")
	}
	if len(r.ModelMake) == 0 || r.ModelMake[0].Make != "Tesla" {
		t.Errorf("make = %+v, want Tesla", r.ModelMake)
	}
	if r.Region.Code != "us-ca" {
		t.Errorf("region = %q, want us-ca", r.Region.Code)
	}
}

func TestPythonSDK_SDKMode_GeneratedFallback(t *testing.T) {
	ts := startMock(t, false)
	got := mustRunSDK(t, ts, pythonInvocation{mode: "sdk"}, "unknown-car.jpg", "--mmc")

	if len(got.Results) != 1 {
		t.Fatalf("expected 1 plate, got %d", len(got.Results))
	}
	r := got.Results[0]
	if !strings.HasPrefix(strings.ToUpper(r.Plate), "MCK") {
		t.Errorf("expected MCK-prefix plate from generator path, got %q", r.Plate)
	}
	if len(r.ModelMake) == 0 {
		t.Error("expected synthesized make/model when --mmc was passed")
	}
}

func TestPythonSDK_SDKMode_RegionsPassthrough(t *testing.T) {
	ts := startMock(t, false)
	got := mustRunSDK(t, ts, pythonInvocation{mode: "sdk"}, "unknown-car.jpg", "-r", "us-tx")

	if len(got.Results) != 1 {
		t.Fatalf("expected 1 plate, got %d", len(got.Results))
	}
	if got.Results[0].Region.Code != "us-tx" {
		t.Errorf("region passthrough failed: got %q, want us-tx", got.Results[0].Region.Code)
	}
}

// --- cloud contract (auth_required=true, Authorization: Token <key>) -------

func TestPythonSDK_CloudMode_AuthAccepts(t *testing.T) {
	ts := startMock(t, true)
	got := mustRunSDK(t, ts, pythonInvocation{mode: "cloud", apiKey: "test-token"}, "tesla-model3.jpg", "--mmc")

	if len(got.Results) != 1 {
		t.Fatalf("expected 1 plate, got %d", len(got.Results))
	}
	if got.Results[0].Plate != "tsla123" {
		t.Errorf("plate = %q, want tsla123", got.Results[0].Plate)
	}
	if len(got.Results[0].ModelMake) == 0 || got.Results[0].ModelMake[0].Make != "Tesla" {
		t.Errorf("expected Tesla make in cloud-mode response, got %+v", got.Results[0].ModelMake)
	}
}

func TestPythonSDK_CloudMode_RejectsBadToken(t *testing.T) {
	ts := startMock(t, true)
	_, out, err := runPythonSDK(t, ts, pythonInvocation{mode: "cloud", apiKey: "wrong-token"}, "tesla-model3.jpg")

	if err == nil {
		t.Fatalf("expected failure with bad token; got success:\n%s", out)
	}
	if !strings.Contains(string(out), "403") && !strings.Contains(string(out), "valid API token") {
		t.Errorf("expected 403 / token error in script output, got:\n%s", out)
	}
}
