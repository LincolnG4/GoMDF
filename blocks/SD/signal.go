package SD

import (
	"log"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAdress int64) *Block {
	if blockHeader, err := blocks.GetHeaderID(file, startAdress); blockHeader != blocks.SdID {
		log.Fatalf("header of signal data %v != ##SD: %s", startAdress, err)
	}

	return &Block{}
}

func (b *Block) BlankBlock() *Block {
	return &Block{}
}
