package mf4

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/LincolnG4/GoMDF/internal/dbc"
)

// BusDatabase is a parsed bus description (a CAN .dbc database) used to
// decode logged frame payloads into physical signals.
type BusDatabase struct {
	// Name identifies the database, e.g. the attachment file name.
	Name string
	db   *dbc.Database
}

// ParseDBC reads a CAN database in DBC format.
func ParseDBC(r io.Reader) (*BusDatabase, error) {
	db, err := dbc.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse DBC: %w", err)
	}
	return &BusDatabase{db: db}, nil
}

// LoadDBC reads a CAN database from a .dbc file.
func LoadDBC(path string) (*BusDatabase, error) {
	db, err := dbc.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load DBC %s: %w", path, err)
	}
	return &BusDatabase{Name: path, db: db}, nil
}

// MessageNames returns the message names defined by the database, in
// file order.
func (d *BusDatabase) MessageNames() []string {
	out := make([]string, 0, len(d.db.Messages))
	for _, m := range d.db.Messages {
		out = append(out, m.Name)
	}
	return out
}

// EmbeddedDatabases parses the bus databases attached to the MDF file
// (attachments with a DBC MIME type or a .dbc file name). Bus logging
// files usually embed the database they were recorded with.
func (f *File) EmbeddedDatabases() ([]*BusDatabase, error) {
	atts, err := f.Attachments()
	if err != nil {
		return nil, err
	}
	var out []*BusDatabase
	for i := range atts {
		a := &atts[i]
		isDBC := strings.Contains(strings.ToLower(a.MimeType), "dbc") ||
			strings.HasSuffix(strings.ToLower(a.Filename), ".dbc")
		if !isDBC || !a.Embedded {
			continue
		}
		data, err := a.Data()
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", a.Filename, err)
		}
		db, err := ParseDBC(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", a.Filename, err)
		}
		db.Name = a.Filename
		out = append(out, db)
	}
	return out, nil
}

// BusFrameGroup is a channel group holding logged bus frames.
type BusFrameGroup struct {
	Group *ChannelGroup
	// Kind is the bus type taken from the frame structure name, e.g.
	// "CAN" for CAN_DataFrame or "LIN" for LIN_Frame.
	Kind string

	prefix string // structure channel name, e.g. "CAN_DataFrame"
}

// BusFrameGroups returns the groups that hold raw bus frames. Use them
// to show raw traffic; use DecodeBus to get physical signals.
func (f *File) BusFrameGroups() []*BusFrameGroup {
	var out []*BusFrameGroup
	for _, g := range f.groups {
		for _, c := range g.channels {
			prefix, ok := strings.CutSuffix(c.Name, ".DataBytes")
			if !ok || prefix == "" {
				continue
			}
			if _, hasID := g.Channel(prefix + ".ID"); !hasID {
				continue
			}
			kind := prefix
			if i := strings.IndexByte(prefix, '_'); i > 0 {
				kind = prefix[:i]
			}
			out = append(out, &BusFrameGroup{Group: g, Kind: kind, prefix: prefix})
			break
		}
	}
	return out
}

// BusSignal is one signal decoded from logged bus frames. Time and the
// value slices are aligned 1:1, one entry per frame the signal appeared
// in (multiplexed signals only appear in their own frames).
type BusSignal struct {
	Name    string
	Unit    string
	Comment string

	// Message is the database message the signal belongs to.
	Message string
	// MessageID is the frame identifier (without the extended flag).
	MessageID uint32
	Extended  bool
	// BusChannel is the logged bus channel number (1-based), or 0 when
	// the log does not record one.
	BusChannel int

	Time   []float64
	Floats []float64
	// Strings holds the value-table text per sample for signals with a
	// VAL_ table; nil otherwise.
	Strings []string
}

// Len returns the number of samples.
func (s *BusSignal) Len() int { return len(s.Floats) }

// QualifiedName returns "Message.Signal", unique per bus channel.
func (s *BusSignal) QualifiedName() string { return s.Message + "." + s.Name }

// DecodeBus decodes every logged bus frame with the given databases and
// returns the physical signals, sorted by bus channel, message id and
// signal name. With no database given, the file's embedded databases are
// used.
//
// Frames whose identifier is not in any database are skipped; so are
// frames too short for a signal.
func (f *File) DecodeBus(dbs ...*BusDatabase) ([]*BusSignal, error) {
	if len(dbs) == 0 {
		embedded, err := f.EmbeddedDatabases()
		if err != nil {
			return nil, err
		}
		if len(embedded) == 0 {
			return nil, fmt.Errorf("%w: no bus database given and none embedded", ErrChannelNotFound)
		}
		dbs = embedded
	}
	groups := f.BusFrameGroups()
	if len(groups) == 0 {
		return nil, nil
	}
	acc := newBusAccumulator(dbs)
	for _, fg := range groups {
		if err := acc.addGroup(fg); err != nil {
			return nil, err
		}
	}
	return acc.signals(), nil
}

// busAccumulator collects decoded samples per (bus, message, signal).
type busAccumulator struct {
	dbs []*BusDatabase
	// msgs is keyed by bus channel and identifier.
	msgs map[busKey]*busMessage
}

type busKey struct {
	bus int
	id  uint32
	ext bool
}

type busMessage struct {
	msg     *dbc.Message
	key     busKey
	signals []*BusSignal
	defs    []*dbc.Signal
}

func newBusAccumulator(dbs []*BusDatabase) *busAccumulator {
	return &busAccumulator{dbs: dbs, msgs: make(map[busKey]*busMessage)}
}

// lookup finds (and caches) the decoder state for one frame identifier.
// It returns nil when no database describes the identifier.
func (a *busAccumulator) lookup(k busKey, capacity int) *busMessage {
	if bm, ok := a.msgs[k]; ok {
		return bm
	}
	var found *dbc.Message
	for _, d := range a.dbs {
		if m, ok := d.db.Message(k.id, k.ext); ok {
			found = m
			break
		}
	}
	if found == nil {
		a.msgs[k] = nil // negative cache
		return nil
	}
	bm := &busMessage{msg: found, key: k}
	for _, sd := range found.Signals {
		if sd.IsMultiplexer && sd.Multiplexed() {
			continue // extended multiplexing is not supported
		}
		sig := &BusSignal{
			Name:       sd.Name,
			Unit:       sd.Unit,
			Comment:    sd.Comment,
			Message:    found.Name,
			MessageID:  k.id,
			Extended:   k.ext,
			BusChannel: k.bus,
			Time:       make([]float64, 0, capacity),
			Floats:     make([]float64, 0, capacity),
		}
		if sd.ValueTable != nil {
			sig.Strings = make([]string, 0, capacity)
		}
		bm.signals = append(bm.signals, sig)
		bm.defs = append(bm.defs, sd)
	}
	a.msgs[k] = bm
	return bm
}

// addGroup decodes all frames of one frame group.
func (a *busAccumulator) addGroup(fg *BusFrameGroup) error {
	g := fg.Group
	read := func(suffix string) (*Signal, error) {
		ch, ok := g.Channel(fg.prefix + suffix)
		if !ok {
			return nil, nil
		}
		return ch.Read()
	}
	idSig, err := read(".ID")
	if err != nil || idSig == nil {
		return err
	}
	dataSig, err := read(".DataBytes")
	if err != nil || dataSig == nil {
		return err
	}
	lenSig, err := read(".DataLength")
	if err != nil {
		return err
	}
	busSig, err := read(".BusChannel")
	if err != nil {
		return err
	}
	ideSig, err := read(".IDE")
	if err != nil {
		return err
	}

	ids := idSig.Float64s()
	times := idSig.Time
	payloads := framePayloads(dataSig)
	var lengths, buses, ides []float64
	if lenSig != nil {
		lengths = lenSig.Float64s()
	}
	if busSig != nil {
		buses = busSig.Float64s()
	}
	if ideSig != nil {
		ides = ideSig.Float64s()
	}

	n := len(ids)
	if len(payloads) < n {
		n = len(payloads)
	}
	for i := 0; i < n; i++ {
		payload := payloads[i]
		if i < len(lengths) {
			if l := int(lengths[i]); l >= 0 && l < len(payload) {
				payload = payload[:l]
			}
		}
		k := busKey{id: uint32(ids[i])}
		if i < len(buses) {
			k.bus = int(buses[i])
		}
		if i < len(ides) {
			k.ext = ides[i] != 0
		}
		bm := a.lookup(k, n)
		if bm == nil {
			continue
		}
		var t float64
		if i < len(times) {
			t = times[i]
		}
		bm.decode(payload, t)
	}
	return nil
}

// decode appends one frame's signal values.
func (bm *busMessage) decode(payload []byte, t float64) {
	mux, hasMux := bm.msg.MuxValueOf(payload)
	for i, sd := range bm.defs {
		if sd.Multiplexed() && (!hasMux || int64(sd.MuxValue) != mux) {
			continue
		}
		raw, ok := sd.Raw(payload)
		if !ok {
			continue // frame too short for this signal
		}
		sig := bm.signals[i]
		sig.Time = append(sig.Time, t)
		sig.Floats = append(sig.Floats, sd.PhysicalFromRaw(raw))
		if sig.Strings != nil {
			sig.Strings = append(sig.Strings, sd.ValueTable[sd.SignedRaw(raw)])
		}
	}
}

// signals flattens the accumulated messages into a stable order.
func (a *busAccumulator) signals() []*BusSignal {
	var out []*BusSignal
	for _, bm := range a.msgs {
		if bm == nil {
			continue
		}
		for _, s := range bm.signals {
			if s.Len() > 0 {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.BusChannel != b.BusChannel:
			return a.BusChannel < b.BusChannel
		case a.MessageID != b.MessageID:
			return a.MessageID < b.MessageID
		default:
			return a.Name < b.Name
		}
	})
	return out
}

// framePayloads normalizes a DataBytes signal to per-frame byte slices.
func framePayloads(sig *Signal) [][]byte {
	switch sig.Type {
	case SampleBytes:
		return sig.Bytes
	case SampleString:
		out := make([][]byte, len(sig.Strings))
		for i, s := range sig.Strings {
			out[i] = []byte(s)
		}
		return out
	}
	return nil
}
