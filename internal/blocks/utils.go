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

type LinkType map[string]int64

func NewBuffer(file *os.File, startAdress int64, BLOCK_SIZE int) *bytes.Buffer {
	bytesValue := seekBinaryByAddress(file, startAdress, BLOCK_SIZE)
	return bytes.NewBuffer(bytesValue)
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
func GetText(file *os.File, startAdress int64, bufSize []byte, decode bool) []byte {
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

func CalculateLinkSize(linkCount uint64) uint64 {
	return linkCount * uint64(LinkSize)
}

func CalculateDataSize(length uint64, linkCount uint64) uint64 {
	return (length - uint64(HeaderSize) - linkCount*uint64(LinkSize))
}

// Create a buffer based on blocksize
func LoadBuffer(file *os.File, blockSize uint64) *bytes.Buffer {
	buf := make([]byte, blockSize)

	_, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println("LoadBuffer error: ", err)
		}
	}

	return bytes.NewBuffer(buf)
}
