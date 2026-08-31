package blocks

// Build composes one complete block: common header, link section, data
// section. The result length is NOT padded; blocks must be written at
// 8-aligned offsets (all sections here are multiples of 8 except TX/MD
// text, which callers pad).
func Build(id string, links []int64, data []byte) []byte {
	n := HeaderSize + len(links)*LinkSize + len(data)
	out := make([]byte, HeaderSize, n)
	copy(out[0:4], id)
	le.PutUint64(out[8:16], uint64(n))
	le.PutUint64(out[16:24], uint64(len(links)))
	for _, l := range links {
		out = le.AppendUint64(out, uint64(l))
	}
	return append(out, data...)
}

// BuildTX composes a ##TX block holding s (NUL-terminated, padded to 8).
func BuildTX(s string) []byte {
	data := append([]byte(s), 0)
	for len(data)%8 != 0 {
		data = append(data, 0)
	}
	return Build(IDTX, nil, data)
}

// BuildMD composes a ##MD block holding the XML fragment s.
func BuildMD(s string) []byte {
	data := append([]byte(s), 0)
	for len(data)%8 != 0 {
		data = append(data, 0)
	}
	return Build(IDMD, nil, data)
}

// BuildID composes the 64-byte identification block. unfinFlags != 0
// marks the file as unfinalized ("UnFinMF ") with the given standard
// finalization flags.
func BuildID(program string, version uint16, versionString string, unfinFlags uint16) []byte {
	out := make([]byte, IDSize)
	if unfinFlags != 0 {
		copy(out[0:8], "UnFinMF ")
	} else {
		copy(out[0:8], "MDF     ")
	}
	pad := func(dst []byte, s string) {
		for i := range dst {
			if i < len(s) {
				dst[i] = s[i]
			} else {
				dst[i] = ' '
			}
		}
	}
	pad(out[8:16], versionString)
	pad(out[16:24], program)
	le.PutUint16(out[28:30], version)
	le.PutUint16(out[60:62], unfinFlags)
	return out
}
