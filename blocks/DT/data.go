package DT

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		if err != io.EOF {
			return b.BlankBlock(), fmt.Errorf("error seeking to address %d: %w", startAddress, err)
		}
		return b.BlankBlock(), fmt.Errorf("EOF reached while seeking to address %d", startAddress)
	}

	// Calculate the size of the Header and read it directly
	headerSize := blocks.HeaderSize
	headerBuffer := make([]byte, headerSize)
	if _, err := io.ReadFull(file, headerBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading header: %w", err)
	}

	// Create a reader from the buffer
	headerReader := bytes.NewReader(headerBuffer)

	// Read the header
	if err := binary.Read(headerReader, binary.LittleEndian, &b.Header); err != nil {
		return b.BlankBlock(), fmt.Errorf("error decoding header: %w", err)
	}

	return &b, nil
}

func (b *Block) DataBlockType() string {
	return string(b.Header.ID[:])
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.DtID),
			Reserved:  [4]byte{},
			Length:    24,
			LinkCount: 0,
		},
		Data: []byte{},
	}
}
