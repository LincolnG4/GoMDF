package SD

import (
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAdress int64) *Block {
	if blockHeader, err := blocks.GetHeaderID(file, startAdress); blockHeader != "##SD" {
		panic(err)
	}

	return &Block{}
}

func (b *Block) BlankBlock() *Block {
	return &Block{}
}
