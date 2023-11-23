package blocks

import (
	"bytes"
	"encoding/binary"
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

type LinkType map[string]int64

func NewBuffer(file *os.File, startAdress int64, BLOCK_SIZE int) *bytes.Buffer {
	bytesValue := seekBinaryByAddress(file, startAdress, BLOCK_SIZE)
	return bytes.NewBuffer(bytesValue)
}

func seekBinaryByAddress(file *os.File, address int64, block_size int) []byte {
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

func GetText(file *os.File, startAdress int64, bufSize []byte, decode bool) []byte {
	if startAdress == 0 {
		return []byte{}
	}

	if decode {
		_, err := file.Seek(int64(startAdress)+HeaderSize, io.SeekStart)
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

func ReadInt64FromBinary(file *os.File) int64 {
	var value int64
	if err := binary.Read(file, binary.LittleEndian, &value); err != nil {
		fmt.Println("Error reading binary data:", err)
	}
	return value
}

func ReadAllFromBinary(file *os.File) int64 {
	var value int64
	if err := binary.Read(file, binary.LittleEndian, &value); err != nil {
		fmt.Println("Error reading binary data:", err)
	}
	return value
}

func IsBitSet(value int, bitPosition int) bool {
	// Create a bitmask with the target bit set (1) and all other bits unset (0)
	bitmask := 1 << (bitPosition) // 2

	// Use bitwise AND to check if the target bit is set
	return (value & bitmask) != 0 // 5 & 2 != 0  false
}
