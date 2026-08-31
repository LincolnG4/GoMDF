// Package dbc parses CAN database (.dbc) files and decodes bus payloads
// into physical signal values.
//
// Only the parts needed to interpret logged bus traffic are parsed:
// messages (BO_), signals (SG_, including multiplexing), value tables
// (VAL_), comments (CM_) and IEEE float signal types (SIG_VALTYPE_).
// Everything else — node lists, attribute definitions, environment
// variables — is skipped.
package dbc

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ExtendedFlag marks an extended (29-bit) identifier in a DBC message id.
const ExtendedFlag = 0x8000_0000

// Signal is one signal inside a message.
type Signal struct {
	Name    string
	Comment string
	Unit    string

	// StartBit is the bit position as written in the DBC: the least
	// significant bit for little-endian signals, the most significant
	// one for big-endian signals.
	StartBit  int
	Length    int
	BigEndian bool
	Signed    bool

	Factor float64
	Offset float64
	Min    float64
	Max    float64

	// Float marks IEEE 754 raw encoding (SIG_VALTYPE_): the raw bits
	// are a float32 (Length 32) or float64 (Length 64).
	Float bool

	// IsMultiplexer marks the message's multiplexer switch signal ("M").
	IsMultiplexer bool
	// MuxValue is the switch value this signal is present for ("m<n>"),
	// or -1 when the signal is always present.
	MuxValue int

	Receivers []string
	// ValueTable maps raw values to texts (VAL_); nil when absent.
	ValueTable map[int64]string
}

// Multiplexed reports whether the signal is only present for a specific
// multiplexer value.
func (s *Signal) Multiplexed() bool { return s.MuxValue >= 0 }

// Message is one CAN/LIN message definition.
type Message struct {
	ID          uint32 // identifier without the extended flag
	Extended    bool
	Name        string
	Size        int // payload bytes (DLC)
	Transmitter string
	Comment     string
	Signals     []*Signal
}

// Signal returns the named signal of the message.
func (m *Message) Signal(name string) (*Signal, bool) {
	for _, s := range m.Signals {
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

// multiplexer returns the message's multiplexer switch signal, if any.
func (m *Message) multiplexer() *Signal {
	for _, s := range m.Signals {
		if s.IsMultiplexer {
			return s
		}
	}
	return nil
}

// Database is a parsed CAN database.
type Database struct {
	Version  string
	Messages []*Message

	byKey map[uint64]*Message
}

func key(id uint32, extended bool) uint64 {
	k := uint64(id)
	if extended {
		k |= 1 << 32
	}
	return k
}

// Message looks up a message by identifier.
func (d *Database) Message(id uint32, extended bool) (*Message, bool) {
	m, ok := d.byKey[key(id, extended)]
	if !ok && !extended {
		// Some databases store 11-bit ids with the extended flag unset
		// while the log marks the frame extended (or vice versa); fall
		// back to an id-only match.
		m, ok = d.byKey[key(id, true)]
	}
	return m, ok
}

var (
	reMessage    = regexp.MustCompile(`^BO_\s+(\d+)\s+([^\s:]+)\s*:\s*(\d+)\s*(\S*)`)
	reSignal     = regexp.MustCompile(`^\s*SG_\s+([^\s:]+)\s*(M|m\d+M?)?\s*:\s*(\d+)\|(\d+)@([01])([-+])\s*\(([^,]*),([^)]*)\)\s*\[([^|]*)\|([^\]]*)\]\s*"([^"]*)"\s*(.*)$`)
	reValTable   = regexp.MustCompile(`^VAL_\s+(\d+)\s+([^\s]+)\s+(.*?);?\s*$`)
	reValPair    = regexp.MustCompile(`(-?\d+)\s+"([^"]*)"`)
	reCommentSig = regexp.MustCompile(`^CM_\s+SG_\s+(\d+)\s+([^\s]+)\s+"(.*)"\s*;?\s*$`)
	reCommentMsg = regexp.MustCompile(`^CM_\s+BO_\s+(\d+)\s+"(.*)"\s*;?\s*$`)
	reValType    = regexp.MustCompile(`^SIG_VALTYPE_\s+(\d+)\s+([^\s]+)\s*:?\s*(\d+)\s*;?\s*$`)
	reVersion    = regexp.MustCompile(`^VERSION\s+"(.*)"`)
)

// Load parses the DBC file at path.
func Load(path string) (*Database, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads a DBC database.
func Parse(r io.Reader) (*Database, error) {
	db := &Database{byKey: make(map[uint64]*Message)}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8<<20)

	var cur *Message
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// A quoted string may span lines (comments); join until the
		// quotes are balanced.
		for strings.Count(trimmed, `"`)%2 == 1 && sc.Scan() {
			trimmed += "\n" + strings.TrimRight(sc.Text(), "\r")
		}

		switch {
		case strings.HasPrefix(trimmed, "BO_ "):
			m := parseMessage(trimmed)
			if m != nil {
				db.Messages = append(db.Messages, m)
				db.byKey[key(m.ID, m.Extended)] = m
				cur = m
			} else {
				cur = nil
			}
		case strings.HasPrefix(trimmed, "SG_ "):
			if cur == nil {
				continue
			}
			if s := parseSignal(line); s != nil {
				cur.Signals = append(cur.Signals, s)
			}
		case strings.HasPrefix(trimmed, "VAL_ "):
			db.applyValueTable(trimmed)
		case strings.HasPrefix(trimmed, "CM_ "):
			db.applyComment(trimmed)
		case strings.HasPrefix(trimmed, "SIG_VALTYPE_ "):
			db.applyValueType(trimmed)
		case strings.HasPrefix(trimmed, "VERSION"):
			if m := reVersion.FindStringSubmatch(trimmed); m != nil {
				db.Version = m[1]
			}
		default:
			// BU_, BA_*, NS_, SIG_GROUP_, EV_ ... not needed here.
			if !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t") {
				cur = nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(db.Messages) == 0 {
		return nil, fmt.Errorf("no messages found (not a DBC file?)")
	}
	return db, nil
}

func parseMessage(line string) *Message {
	m := reMessage.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	raw, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil {
		return nil
	}
	size, _ := strconv.Atoi(m[3])
	return &Message{
		ID:          uint32(raw) &^ ExtendedFlag,
		Extended:    uint32(raw)&ExtendedFlag != 0,
		Name:        m[2],
		Size:        size,
		Transmitter: m[4],
	}
}

func parseSignal(line string) *Signal {
	m := reSignal.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	start, _ := strconv.Atoi(m[3])
	length, _ := strconv.Atoi(m[4])
	s := &Signal{
		Name:      m[1],
		StartBit:  start,
		Length:    length,
		BigEndian: m[5] == "0", // @0 = Motorola, @1 = Intel
		Signed:    m[6] == "-",
		Unit:      m[11],
		MuxValue:  -1,
	}
	s.Factor = parseFloat(m[7], 1)
	s.Offset = parseFloat(m[8], 0)
	s.Min = parseFloat(m[9], 0)
	s.Max = parseFloat(m[10], 0)
	if rx := strings.TrimSpace(m[12]); rx != "" {
		s.Receivers = strings.Split(rx, ",")
		for i := range s.Receivers {
			s.Receivers[i] = strings.TrimSpace(s.Receivers[i])
		}
	}
	switch mux := m[2]; {
	case mux == "M":
		s.IsMultiplexer = true
	case strings.HasPrefix(mux, "m"):
		v := strings.TrimSuffix(strings.TrimPrefix(mux, "m"), "M")
		if n, err := strconv.Atoi(v); err == nil {
			s.MuxValue = n
		}
		if strings.HasSuffix(mux, "M") {
			s.IsMultiplexer = true // multiplexed multiplexer (extended)
		}
	}
	return s
}

func parseFloat(s string, def float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return v
}

func (d *Database) messageByRawID(s string) *Message {
	raw, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return nil
	}
	id := uint32(raw)
	return d.byKey[key(id&^ExtendedFlag, id&ExtendedFlag != 0)]
}

func (d *Database) applyValueTable(line string) {
	m := reValTable.FindStringSubmatch(line)
	if m == nil {
		return
	}
	msg := d.messageByRawID(m[1])
	if msg == nil {
		return
	}
	sig, ok := msg.Signal(m[2])
	if !ok {
		return
	}
	for _, pair := range reValPair.FindAllStringSubmatch(m[3], -1) {
		v, err := strconv.ParseInt(pair[1], 10, 64)
		if err != nil {
			continue
		}
		if sig.ValueTable == nil {
			sig.ValueTable = make(map[int64]string)
		}
		sig.ValueTable[v] = pair[2]
	}
}

func (d *Database) applyComment(line string) {
	if m := reCommentSig.FindStringSubmatch(line); m != nil {
		if msg := d.messageByRawID(m[1]); msg != nil {
			if sig, ok := msg.Signal(m[2]); ok {
				sig.Comment = m[3]
			}
		}
		return
	}
	if m := reCommentMsg.FindStringSubmatch(line); m != nil {
		if msg := d.messageByRawID(m[1]); msg != nil {
			msg.Comment = m[2]
		}
	}
}

func (d *Database) applyValueType(line string) {
	m := reValType.FindStringSubmatch(line)
	if m == nil {
		return
	}
	msg := d.messageByRawID(m[1])
	if msg == nil {
		return
	}
	if sig, ok := msg.Signal(m[2]); ok {
		// 1 = IEEE float (32 bit), 2 = IEEE double (64 bit)
		if m[3] == "1" || m[3] == "2" {
			sig.Float = true
		}
	}
}
