package mf4_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// TestCorpus opens and fully reads every MF4 file under $GOMDF_CORPUS
// (e.g. the ASAM base-standard Examples directory). Skipped when unset.
func TestCorpus(t *testing.T) {
	root := os.Getenv("GOMDF_CORPUS")
	if root == "" {
		t.Skip("GOMDF_CORPUS not set")
	}
	var files []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".mf4") {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		t.Fatalf("no MF4 files under %s", root)
	}
	for _, path := range files {
		rel, _ := filepath.Rel(root, path)
		t.Run(rel, func(t *testing.T) {
			f, err := mf4.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer f.Close()
			for _, ch := range f.Channels() {
				if _, err := ch.Read(); err != nil {
					t.Errorf("%s: %v", ch.Name, err)
				}
			}
			if _, err := f.Attachments(); err != nil {
				t.Errorf("attachments: %v", err)
			}
			if _, err := f.Events(); err != nil {
				t.Errorf("events: %v", err)
			}
			if _, err := f.Hierarchy(); err != nil {
				t.Errorf("hierarchy: %v", err)
			}
		})
	}
}
