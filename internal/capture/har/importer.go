package har

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	DefaultMaxUploadBytes = 50 << 20
	DefaultMaxBodyBytes   = 1 << 20
	DefaultMaxEntries     = 20000
)

type Importer struct {
	MaxUploadBytes int64
	MaxBodyBytes   int
	MaxEntries     int
}

func NewImporter(maxUploadBytes int64) *Importer {
	if maxUploadBytes <= 0 {
		maxUploadBytes = DefaultMaxUploadBytes
	}
	return &Importer{
		MaxUploadBytes: maxUploadBytes,
		MaxBodyBytes:   DefaultMaxBodyBytes,
		MaxEntries:     DefaultMaxEntries,
	}
}

func (i *Importer) Import(r io.Reader) ([]Entry, error) {
	maxBytes := i.MaxUploadBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxUploadBytes
	}

	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read HAR: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("HAR exceeds %d bytes", maxBytes)
	}

	var file File
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&file); err != nil {
		return nil, fmt.Errorf("decode HAR: %w", err)
	}
	if len(file.Log.Entries) == 0 {
		return nil, errors.New("HAR contains no entries")
	}

	maxEntries := i.MaxEntries
	if maxEntries <= 0 {
		maxEntries = DefaultMaxEntries
	}
	if len(file.Log.Entries) > maxEntries {
		return nil, fmt.Errorf("HAR has %d entries; limit is %d", len(file.Log.Entries), maxEntries)
	}

	maxBody := i.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = DefaultMaxBodyBytes
	}
	for n := range file.Log.Entries {
		entry := &file.Log.Entries[n]
		if entry.Request.PostData != nil {
			entry.Request.PostData.Text = trim(entry.Request.PostData.Text, maxBody)
		}
		if entry.Response.Content != nil {
			entry.Response.Content.Text = trim(entry.Response.Content.Text, maxBody)
		}
	}

	return file.Log.Entries, nil
}

func trim(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
