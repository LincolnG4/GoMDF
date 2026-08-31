package mf4_test

import (
	"strings"
	"testing"
)

func TestAttachment(t *testing.T) {
	f := openFile(t, "sample3.mf4")
	atts, err := f.Attachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 {
		t.Fatalf("got %d attachments, want 1", len(atts))
	}
	a := atts[0]
	if a.Filename != "user_embedded_display.dspf" {
		t.Errorf("filename = %q", a.Filename)
	}
	if a.MimeType != "application/x-dspf" {
		t.Errorf("mimetype = %q", a.MimeType)
	}
	if !a.Embedded {
		t.Fatal("not embedded")
	}
	data, err := a.Data()
	if err != nil {
		t.Fatal(err)
	}
	if uint64(len(data)) != a.Size {
		t.Errorf("data len %d, want %d", len(data), a.Size)
	}
}

func TestHistory(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	hist, err := f.History()
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) == 0 {
		t.Fatal("no history")
	}
	if hist[0].Time.Year() < 2000 {
		t.Errorf("implausible history time %v", hist[0].Time)
	}
}

func TestEvents(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	evs, err := f.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	for _, e := range evs {
		if e.Name == "" && !strings.Contains(e.Comment, "") {
			t.Errorf("empty event %+v", e)
		}
	}
}
