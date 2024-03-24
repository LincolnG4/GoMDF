package SD

import (
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAdress int64) {

}

func (b *Block) BlankBlock() *Block {
	return &Block{}
}
