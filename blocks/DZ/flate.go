package DZ

import (
	"bytes"
	"compress/zlib"
	"fmt"
)

type Flate struct {
	//Decompressed ID
	DecompressedID string

	//Zip algorithm (Deflate==0 or Transposition + Deflate==1)
	CompressType uint8

	//Decompressed size
	DecompressedLength uint64

	//Compressed size
	CompressedLength uint64

	//DataBlock
	Datablock *[]byte
}

func (f *Flate) Decompress() ([]byte, error) {
	// Create a zlib reader directly from the file's limited reader
	zr, err := zlib.NewReader(bytes.NewReader(*f.Datablock))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zr.Close()

	// Use a bytes.Buffer for more controlled incremental reads
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to read from zlib reader: %w", err)
	}

	return buf.Bytes(), nil
}
