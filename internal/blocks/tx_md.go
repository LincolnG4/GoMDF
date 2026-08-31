package blocks

import (
	"bytes"
	"encoding/xml"

	"github.com/LincolnG4/GoMDF/internal/source"
)

// DecodeText decodes a TX or MD block at addr and returns its text with
// the trailing NUL padding removed. addr == 0 (NIL link) returns "".
func DecodeText(src source.Source, addr int64) (string, error) {
	if addr == 0 {
		return "", nil
	}
	h, err := DecodeHeader(src, addr, "")
	if err != nil {
		return "", err
	}
	if h.ID != IDTX && h.ID != IDMD {
		return "", blockErrf(h.ID, addr, "%w: expected ##TX or ##MD", ErrInvalidBlock)
	}
	if h.LinkCount != 0 {
		return "", blockErrf(h.ID, addr, "%w: text block with %d links", ErrInvalidBlock, h.LinkCount)
	}
	data, err := src.Slice(addr+HeaderSize, int64(h.DataLen()))
	if err != nil {
		return "", blockErr(h.ID, addr, err)
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		data = data[:i]
	}
	return string(data), nil
}

// CommentText decodes a comment link that may be either a plain TX string
// or an MD XML fragment; for MD it extracts the <TX> element text. Errors
// on malformed XML fall back to the raw text.
func CommentText(src source.Source, addr int64) (string, error) {
	s, err := DecodeText(src, addr)
	if err != nil || len(s) == 0 || s[0] != '<' {
		return s, err
	}
	// MD comment: XML with a TX element holding the plain-text comment.
	type txDoc struct {
		TX string `xml:"TX"`
	}
	var doc txDoc
	if xml.Unmarshal([]byte(s), &doc) == nil && doc.TX != "" {
		return doc.TX, nil
	}
	return s, nil
}
