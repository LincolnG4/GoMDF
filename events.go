package mf4

import (
	"time"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// EventType classifies an event block (ev_type).
type EventType uint8

const (
	EventRecording            EventType = 0
	EventRecordingInterrupt   EventType = 1
	EventAcquisitionInterrupt EventType = 2
	EventStartRecording       EventType = 3
	EventStopRecording        EventType = 4
	EventTrigger              EventType = 5
	EventMarker               EventType = 6
)

// Event is a marker, trigger or recording event of the measurement.
type Event struct {
	Name    string
	Comment string
	Type    EventType
	Cause   uint8
	// SyncValue is the event position on its synchronization axis
	// (seconds for time-synchronized events).
	SyncValue float64
}

// Events returns the file's events.
func (f *File) Events() ([]Event, error) {
	var out []Event
	for addr := f.hd.EVFirst; addr != 0; {
		ev, err := blocks.DecodeEV(f.src, addr)
		if err != nil {
			return nil, err
		}
		e := Event{
			Type:      EventType(ev.Type),
			Cause:     ev.Cause,
			SyncValue: ev.SyncValue(),
		}
		if e.Name, err = blocks.DecodeText(f.src, ev.TXName); err != nil {
			return nil, err
		}
		if e.Comment, err = blocks.CommentText(f.src, ev.MDComment); err != nil {
			return nil, err
		}
		out = append(out, e)
		addr = ev.EVNext
	}
	return out, nil
}

// HistoryEntry is one file-history record: who/what changed the file and
// when.
type HistoryEntry struct {
	Time time.Time
	// Comment is the fh_md_comment content (an XML fragment naming the
	// tool, vendor and user; the plain text part is extracted when
	// possible).
	Comment string
}

// History returns the file-history chain, oldest first.
func (f *File) History() ([]HistoryEntry, error) {
	var out []HistoryEntry
	for addr := f.hd.FHFirst; addr != 0; {
		fh, err := blocks.DecodeFH(f.src, addr)
		if err != nil {
			return nil, err
		}
		h := HistoryEntry{Time: time.Unix(0, int64(fh.TimeNS)).UTC()}
		if h.Comment, err = blocks.CommentText(f.src, fh.MDComment); err != nil {
			return nil, err
		}
		out = append(out, h)
		addr = fh.FHNext
	}
	return out, nil
}
