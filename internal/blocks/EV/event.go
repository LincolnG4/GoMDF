package EV

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Next        int64
	Parent      int64
	Range       int64
	TxName      int64
	MdComment   int64
	Scope       []int64
	ATReference []int64
	//Version 4.2
	TxGroupName int64
}

type Data struct {
	Type            uint8
	SyncType        uint8
	RangeType       uint8
	Cause           uint8
	Flags           uint8
	Reserved1       [3]byte
	ScopeCount      uint32
	AttachmentCount uint16
	CreatorIndex    uint16
	SyncBaseValue   int64
	SyncFactor      float64
}

func (b *Block) getID() [4]byte {
	return b.Header.ID
}

func (b *Block) getLinkCount() uint64 {
	return b.Header.LinkCount
}

// Creates a new Block struct and initializes it by reading data from
// the provided file.
func New(file *os.File, version uint16, startAdress int64) (*Block, error) {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	blockID := blocks.EvID
	b.Header = blocks.Header{}
	_, err := file.Seek(startAdress, 0)
	if err != nil {
		if err != io.EOF {
			return b.BlankBlock(), fmt.Errorf("memory addr out of size: %v", err)
		}
	}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	err = binary.Read(buf, binary.LittleEndian, &b.Header)
	if err != nil {
		return b.BlankBlock(), err
	}

	id := b.getID()
	if string(id[:]) != blockID {
		return b.BlankBlock(), err
	}

	fmt.Printf("\n%s\n", id)
	fmt.Printf("%+v\n", b.Header)

	//Calculates size of Link Block
	blockSize = blocks.CalculateLinkSize(b.getLinkCount())
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section: %v", err)
	}

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf = blocks.LoadBuffer(file, blockSize)

	// Create a buffer based on block size
	if err := binary.Read(buf, binary.LittleEndian, &b.Data); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data section: %v", err)
	}

	linkFields := []int64{}
	for i := 0; i < len(buffEach)/8; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(buffEach[i*8:(i+1)*8])))
	}

	b.Link = Link{
		Next:      linkFields[0],
		Parent:    linkFields[1],
		Range:     linkFields[2],
		TxName:    linkFields[3],
		MdComment: linkFields[4],
	}
	if b.Data.ScopeCount != 0 {
		b.Link.Scope = linkFields[5 : 5+b.Data.ScopeCount]
	}
	if b.Data.AttachmentCount != 0 {
		b.Link.ATReference = linkFields[5+b.Data.ScopeCount : 5+int(b.Data.ScopeCount)+int(b.Data.AttachmentCount)]
	}
	if version >= 420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
		b.Link.TxGroupName = linkFields[len(linkFields)-1]
	}

	fmt.Printf("%+v\n", b.Link)
	fmt.Printf("%+v\n", b.Data)
	return &b, nil
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'E', 'V'},
			Reserved:  [4]byte{},
			Length:    blocks.EvblockSize,
			LinkCount: 0,
		},
		Link: Link{},
		Data: Data{},
	}
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
