package mf4

import (
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/datasection"
	"github.com/LincolnG4/GoMDF/internal/records"
)

// resolveVLSD turns a column of byte offsets (as extracted from the
// records of a VLSD channel) into the actual variable-length values from
// the channel's signal-data section.
func (c *Channel) resolveVLSD(offsets *records.Column) (*records.Column, error) {
	sd, err := c.signalDataReader()
	if err != nil {
		return nil, err
	}
	textual := c.cn.DataType >= blocks.DTStringLatin && c.cn.DataType <= blocks.DTStringUTF16BE
	out := &records.Column{Invalid: offsets.Invalid}
	if textual {
		out.Kind = records.KindString
		out.S = make([]string, 0, len(offsets.U))
	} else {
		out.Kind = records.KindBytes
		out.B = make([][]byte, 0, len(offsets.U))
	}
	size := sd.Size()
	var lenBuf [4]byte
	for _, off := range offsets.U {
		if int64(off)+4 > size {
			return nil, fmt.Errorf("VLSD offset %d out of signal data (size %d)", off, size)
		}
		if _, err := sd.ReadAt(lenBuf[:], int64(off)); err != nil {
			return nil, err
		}
		n := int64(uint32(lenBuf[0]) | uint32(lenBuf[1])<<8 | uint32(lenBuf[2])<<16 | uint32(lenBuf[3])<<24)
		if int64(off)+4+n > size {
			return nil, fmt.Errorf("VLSD value at %d (%d bytes) exceeds signal data (size %d)", off, n, size)
		}
		payload := make([]byte, n)
		if n > 0 {
			if _, err := sd.ReadAt(payload, int64(off)+4); err != nil {
				return nil, err
			}
		}
		if textual {
			out.S = append(out.S, records.DecodeString(payload, c.cn.DataType))
		} else {
			out.B = append(out.B, payload)
		}
	}
	return out, nil
}

// signalDataReader returns the data section holding this channel's
// variable-length values. cn_data may point at SD/DZ/DL/HL blocks, or —
// in unsorted files — at a VLSD channel group whose de-interleaved
// records form the stream.
func (c *Channel) signalDataReader() (*datasection.Reader, error) {
	addr := c.cn.Data
	if addr == 0 {
		return nil, fmt.Errorf("VLSD channel without signal data link")
	}
	id, err := blocks.PeekID(c.group.file.src, addr)
	if err != nil {
		return nil, err
	}
	if id == blocks.IDCG {
		cg, err := blocks.DecodeCG(c.group.file.src, addr)
		if err != nil {
			return nil, err
		}
		return c.group.dg.deinterleaved(cg.RecordID)
	}
	return datasection.New(c.group.file.src, addr, c.group.file.cfg.cacheBytes)
}
