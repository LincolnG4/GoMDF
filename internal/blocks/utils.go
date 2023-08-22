package blocks

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type Header struct {
	ID        [4]byte
	Reserved  [4]byte
	Length    uint64
	LinkCount uint64
}

func seekBinaryByAddress(file *os.File, address int64, block_size int) []byte {
	buf := make([]byte, block_size)
	_, errs := file.Seek(address, 0)
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

func NewBuffer(file *os.File, startAdress int64, BLOCK_SIZE int) *bytes.Buffer {
	bytesValue := seekBinaryByAddress(file, startAdress, BLOCK_SIZE)
	return bytes.NewBuffer(bytesValue)
}

func getText(file *os.File, startAdress int64, bufSize []byte, decode bool) []byte {
	if startAdress == 0 {
		return []byte{}
	}

	if decode {
		_, err := file.Seek(startAdress+24, io.SeekStart)
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

func CalculateLinkSize(linkCount uint64) int {
	return int(linkCount * LinkSize)
}

func CalculateDataSize(length uint64, linkCount uint64) int {
	return (int(length) - 24 - int(linkCount)*LinkSize)
}
