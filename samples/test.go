package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type IDBlock struct {
	IDFile          [8]byte
	IDVersion       [8]byte
	IDProgram       [8]byte
	IDReserved1     [4]byte
	IDVersionNumber uint16
	IDReserved2     [34]byte
}

type HDBlock struct {
	ID          [4]byte
	Reserved    [4]byte
	Length      uint64
	LinkCount   uint64
	HDDGFirst   uint64
	HDFHFirst   uint64
	HDCHFirst   uint64
	HDATFirst   uint64
	HDEVFirst   uint64
	HDMDComment uint64
	StartTime   uint64
	TZOffset    int16
	DSTOffset   int16
	TimeFlags   uint8
	TimeClass   uint8
	Flags       uint8
	Reserved2   uint8
	StartAngle  float32
	StartDist   float32
	Reserved3   byte
}

func (idBlock *IDBlock) init(reader *bufio.Reader) {

	IDBLOCK_SIZE := 64

	buf := make([]byte, IDBLOCK_SIZE)

	_, err := reader.Read(buf)

	errorHandler(err)

	buffer := bytes.NewBuffer(buf)
	errs := binary.Read(buffer, binary.LittleEndian, &(*idBlock))

	if errs != nil {
		copy(idBlock.IDFile[:], []byte("MDF     "))
		copy(idBlock.IDVersion[:], []byte("4.00    "))
		copy(idBlock.IDProgram[:], []byte("GoMDF1.0"))
		copy(idBlock.IDReserved1[:], bytes.Repeat([]byte{0}, 4))
		idBlock.IDVersionNumber = 400
		copy(idBlock.IDReserved2[:], bytes.Repeat([]byte{0}, 34))
	}
}

func (hdBlock *HDBlock) init(reader *bufio.Reader) {

	HDBLOCK_SIZE := 104

	buf := make([]byte, HDBLOCK_SIZE)

	_, err := reader.Read(buf)

	errorHandler(err)

	buffer := bytes.NewBuffer(buf)
	fmt.Println(string(buf))
	errs := binary.Read(buffer, binary.LittleEndian, &(*hdBlock))

	if errs != nil {

		copy(hdBlock.ID[:], []byte("##HD"))
		hdBlock.Length = 0
		hdBlock.LinkCount = 0
		hdBlock.HDDGFirst = 0
		hdBlock.HDFHFirst = 0
		hdBlock.HDCHFirst = 0
		hdBlock.HDATFirst = 0
		hdBlock.HDEVFirst = 0
		hdBlock.HDMDComment = 0
		hdBlock.StartTime = 0
		hdBlock.TZOffset = 0
		hdBlock.DSTOffset = 0
		hdBlock.TimeFlags = 0
		hdBlock.TimeClass = 0
		hdBlock.Flags = 0
		hdBlock.Reserved2 = 0
		hdBlock.StartAngle = 0
		hdBlock.StartDist = 0
		hdBlock.Reserved3 = 0
	}

}

func main() {

	file, err := os.Open("samples/sample3.mf4")
	errorHandler(err)
	defer file.Close()

	//Create IDBLOCK
	offset, err := file.Seek(0x40, 0)
	buffer := make([]byte, 64) // Read 10 bytes
	numBytesRead, err := file.Read(buffer)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Printf("Read %d bytes from address %d: %s\n", numBytesRead, offset, string(buffer))
}

func errorHandler(err error) {
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
}
