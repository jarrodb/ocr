package ocr

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jarrodb/ocr/pkg/config"
)

type tesseractEngine struct {
	cfg config.Tesseract
}

func newTesseractEngine(cfg config.Tesseract) *tesseractEngine {
	return &tesseractEngine{cfg: cfg}
}

// read invokes tesseract via subprocess, asking for TSV output so we can read
// per-word confidence. Returns the highest-confidence non-empty word as the
// plate. The caller decides whether the confidence is high enough to keep.
//
// Tesseract is invoked with PSM 8 (single word) by default — appropriate for
// pre-cropped plate images. For full-frame photos, PSM 11 (sparse text) gives
// more candidates but slower.
func (t *tesseractEngine) read(ctx context.Context, req Request) (Reading, error) {
	if t.cfg.BinaryPath == "" {
		return Reading{}, fmt.Errorf("tesseract binary not configured")
	}

	args := []string{
		"stdin", "stdout",
		"-l", t.cfg.Language,
		"--psm", strconv.Itoa(t.cfg.PSM),
		"tsv",
	}
	cmd := exec.CommandContext(ctx, t.cfg.BinaryPath, args...) //nolint:gosec // binary path is operator-configured
	cmd.Stdin = bytes.NewReader(req.Image)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Reading{}, fmt.Errorf("tesseract run: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	plate, conf, err := parseTSV(&stdout)
	if err != nil {
		return Reading{}, fmt.Errorf("parse tsv: %w", err)
	}

	return Reading{
		Plate:  strings.ToLower(strings.TrimSpace(plate)),
		Score:  conf / 100.0,
		DScore: 0.99,
		Source: SourceOCR,
	}, nil
}

// parseTSV reads tesseract's TSV output and returns the highest-confidence
// non-empty word along with its confidence (0-100). Tesseract emits a header
// row then one row per "level" (page/block/par/line/word); we want level=5.
//
// Columns: level page_num block_num par_num line_num word_num left top width height conf text
func parseTSV(r io.Reader) (string, float64, error) {
	rdr := csv.NewReader(r)
	rdr.Comma = '\t'
	rdr.FieldsPerRecord = -1
	rdr.LazyQuotes = true

	var bestWord string
	var bestConf float64
	first := true
	for {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}
		if first {
			first = false
			continue
		}
		if len(rec) < 12 {
			continue
		}
		level, _ := strconv.Atoi(rec[0])
		if level != 5 {
			continue
		}
		conf, _ := strconv.ParseFloat(rec[10], 64)
		word := strings.TrimSpace(rec[11])
		if word == "" || conf < 0 {
			continue
		}
		if conf > bestConf {
			bestConf = conf
			bestWord = word
		}
	}
	return bestWord, bestConf, nil
}
