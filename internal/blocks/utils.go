package blocks

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type Link int64

type Header struct {
	ID        [4]byte
	Reserved  [4]byte
	Length    uint64
	LinkCount uint64
}

type Group struct {
	DataGroup    *DG
	ChannelGroup []*CG
	Channels     *map[string]*CN
}

func NewBuffer(file *os.File, startAdress Link, BLOCK_SIZE int) *bytes.Buffer {
	bytesValue := seekBinaryByAddress(file, startAdress, BLOCK_SIZE)
	return bytes.NewBuffer(bytesValue)
}

func seekBinaryByAddress(file *os.File, address Link, block_size int) []byte {
	buf := make([]byte, block_size)
	_, errs := file.Seek(int64(address), 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs)
		}

	}
	_, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
	return buf
}

func getText(file *os.File, startAdress Link, bufSize []byte, decode bool) []byte {
	if startAdress == 0 {
		return []byte{}
	}

	if decode {
		_, err := file.Seek(int64(startAdress)+24, io.SeekStart)
		if err != nil {
			panic(err)
		}

		n, err := file.Read(bufSize[:cap(bufSize)])
		bufSize = bufSize[:n]
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
		}
		return bufSize
	}
	return []byte{}
}
