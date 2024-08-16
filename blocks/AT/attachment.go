package AT

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/MD"
	"github.com/LincolnG4/GoMDF/blocks/TX"
)

type Block struct {
	Header  blocks.Header
	Link    Link
	Data    Data
	Address int64
}

type Link struct {
	Next       int64 //Link to next ATBLOCK
	TxFilename int64 //Link to TXBLOCK
	TxMimetype int64 //LINK to TXBLOCK
	MDComment  int64 //Link to MDBLOCK
}

type Data struct {
	Flags uint16
	// Creator index, i.e. zero-based index of FHBLOCK in global list
	// of FHBLOCKs that specifies which application has created this attachment,
	// or changed it most recently.
	CreatorIndex uint16
	Reserved     [4]byte
	// 128-bit value for MD5 check sum. Only valid if "MD5 check sum valid"
	// flag (bit 2) is set.
	MD5Checksum [16]byte
	// Original data size in Bytes, i.e. either for external file or for
	// compressed data.
	OriginalSize uint64

	// Embedded data size N, i.e. number of Bytes for binary embedded data
	// following this element.
	EmbeddedSize uint64

	//Contains binary embedded data
	EmbeddedData []byte
}

type AttFile struct {
	Name         string
	Type         string
	Comment      string
	Path         string
	CreatorIndex string
	block        *Block
}

func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to seek to address %d: %w", startAddress, err)
	}

	// Read the header
	b.Header = blocks.Header{}
	headerBuf := make([]byte, blocks.HeaderSize)
	if _, err := io.ReadFull(file, headerBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read header: %w", err)
	}
	if err := binary.Read(bytes.NewReader(headerBuf), binary.LittleEndian, &b.Header); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode header: %w", err)
	}

	// Validate the header ID
	if string(b.Header.ID[:]) != blocks.AtID {
		return b.BlankBlock(), fmt.Errorf("invalid block ID: expected %s, got %s", blocks.AtID, b.Header.ID)
	}

	// Read the link block
	linkSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	linkBuf := make([]byte, linkSize)
	if _, err := io.ReadFull(file, linkBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read link block: %w", err)
	}
	if err := binary.Read(bytes.NewReader(linkBuf), binary.LittleEndian, &b.Link); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode link block: %w", err)
	}

	return &b, nil
}

func (b *Block) LoadAttachmentFile(file *os.File) *AttFile {
	var fileName string
	var comment string

	if b.GetTxFilename() != 0 {
		fileName = b.GetFileName(file, b.GetTxFilename())
	}
	mimeType := b.GetMimeType(file, b.GetTxMimeType())

	//Read MDComment
	MdCommentAdress := b.GetMdComment()
	if MdCommentAdress != 0 {
		comment = MD.New(file, MdCommentAdress)
	}

	return &AttFile{
		Name:    fileName,
		Type:    mimeType,
		Comment: comment,
		block:   b,
	}
}

func (a AttFile) getBlock() *Block {
	return a.block
}

func (a AttFile) Save(file *os.File, outputPath string) AttFile {
	b := a.getBlock()
	//Load data to block
	addr := b.Address + int64(blocks.HeaderSize) + int64(blocks.CalculateLinkSize(a.block.Header.LinkCount))
	d, err := b.loadData(file, addr)
	if err != nil {
		fmt.Println(err)
	}
	flag := int(d.Flags)
	data := d.EmbeddedData
	t, err := TX.GetText(file, b.GetTxFilename())
	if err != nil {
		fmt.Println(err)
	}

	a.Path = strings.ReplaceAll(t, "\\", string(os.PathSeparator))
	filename := filepath.Base(a.Path)
	filetype, err := TX.GetText(file, b.Link.TxMimetype)
	if err != nil {
		fmt.Println(err)
	}
	a.CreatorIndex = fmt.Sprint(d.CreatorIndex)
	//If file has no extension, try to save it by mime
	if filepath.Ext(filename) == "" {
		ext, err := mime.ExtensionsByType(filetype)
		if len(ext) > 0 {
			if err != nil {
				fmt.Println(err)
			}
			filename = filename + ext[len(ext)-1]
		} else {
			fmt.Printf("\n%s file unknow format", filename)
		}
	}

	//External File
	if !blocks.IsBitSet(flag, 0) {
		MdCommentAdress := b.GetMdComment()
		if MdCommentAdress != 0 {
			a.Comment = string(MD.New(file, MdCommentAdress))
		}

		fmt.Printf("\n%s is external, the path to the file is %s", filename, a.Path)
		return a
	}

	//Embbeded file - Compressed Zip
	if blocks.IsBitSet(flag, 1) {
		data = decompressFile(d)
	}

	//Embbeded file - MD5 check sum
	if blocks.IsBitSet(flag, 2) {
		if md5.Sum(data) != d.MD5Checksum {
			fmt.Println("Checksums do not match. The file may be corrupted. File:", filename)
		}
	}

	p := filepath.Join(outputPath + filename)
	a.Path = p
	saveFile(file, p, &data)
	return a
}

// decompressFile uses zlib to decompress databyte
func decompressFile(d *Data) []byte {
	c := bytes.NewReader(d.EmbeddedData)
	r, err := zlib.NewReader(c)
	if err != nil {
		fmt.Println(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Println(err)
	}
	return data
}

// saveFile saves bytes to target file
func saveFile(file *os.File, outputPath string, data *[]byte) error {
	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("error to create the file output: ", err)
		return err
	}
	_, err = f.Write(*data)
	if err != nil {
		fmt.Println("error to write data to file output: ", err)
		return err
	}
	return nil
}

func (b *Block) loadData(file *os.File, adress int64) (*Data, error) {
	_, errs := file.Seek(adress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}
	//Calculates size of Data Block
	blockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		return &Data{}, fmt.Errorf("error reading link section: %v", err)
	}

	var fixedArray16 [16]byte

	d := Data{}
	d.Flags = binary.LittleEndian.Uint16(buffEach[0:2])
	d.CreatorIndex = binary.LittleEndian.Uint16(buffEach[2:4])

	//md5CheckSum
	md5CheckSum := buffEach[8:24]
	copy(fixedArray16[:], md5CheckSum[:])
	d.MD5Checksum = fixedArray16

	d.OriginalSize = binary.LittleEndian.Uint64(buffEach[24:32])
	d.EmbeddedSize = binary.LittleEndian.Uint64(buffEach[32:40])
	d.EmbeddedData = buffEach[40:]
	return &d, nil
}

func Get(f *os.File, a int64) ([]AttFile, error) {
	var fileName, comm string
	i := 0
	arr := make([]AttFile, 0)
	for a != 0 {
		atBlock, err := New(f, a)
		if err != nil {
			return arr, nil
		}

		if atBlock.GetTxFilename() != 0 {
			fileName = atBlock.GetFileName(f, atBlock.GetTxFilename())
		} else {
			fileName = fmt.Sprintf("file-%d", i)
			i++
		}

		mimeType := atBlock.GetMimeType(f, atBlock.GetTxMimeType())

		//Read MDComment
		MdCommentAdress := atBlock.GetMdComment()
		if MdCommentAdress != 0 {
			comm = MD.New(f, MdCommentAdress)
		}

		arr = append(arr, AttFile{
			Name:    fileName,
			Type:    mimeType,
			Comment: comm,
			block:   atBlock,
		})
		a = atBlock.Next()
	}
	return arr, nil
}

func (b *Block) GetTxFilename() int64 {
	return b.Link.TxFilename
}

func (b *Block) GetTxMimeType() int64 {
	return b.Link.TxMimetype
}

func GetTextString(file *os.File, a int64) string {
	t, err := TX.GetText(file, a)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) GetFileName(file *os.File, a int64) string {
	return GetTextString(file, a)
}

func (b *Block) GetMimeType(file *os.File, a int64) string {
	return GetTextString(file, a)
}

func (b *Block) GetMdComment() int64 {
	return b.Link.MDComment
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.AtID),
			Reserved:  [4]byte{},
			Length:    blocks.AtblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
