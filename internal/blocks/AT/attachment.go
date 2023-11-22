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

	"github.com/LincolnG4/GoMDF/app"
	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/MD"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
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
	Flags        uint16
	CreatorIndex uint16 //Creator index, i.e. zero-based index of FHBLOCK in global list of FHBLOCKs that specifies which application has created this attachment, or changed it most recently.
	Reserved     [4]byte
	MD5Checksum  [16]byte //128-bit value for MD5 check sum (of the uncompressed data if data is embedded and compressed). Only valid if "MD5 check sum valid" flag (bit 2) is set.
	OriginalSize uint64   //Original data size in Bytes, i.e. either for external file or for compressed data.
	EmbeddedSize uint64   //Embedded data size N, i.e. number of Bytes for binary embedded data following this element.
	EmbeddedData []byte   //Contains binary embedded data
}

const blockID string = blocks.AtID

func New(file *os.File, startAdress int64) *Block {
	var b Block
	var blockSize uint64 = blocks.HeaderSize
	b.Address = startAdress

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	b.Header = blocks.Header{}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	if string(b.Header.ID[:]) != blockID {
		fmt.Printf("ERROR NOT %s", blockID)
	}

	fmt.Printf("\n%s\n", b.Header.ID)
	fmt.Printf("%+v\n", b.Header)

	//Calculates size of Link Block
	blockSize = blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = Link{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Link)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	fmt.Printf("%+v\n", b.Link)

	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'A', 'T'},
			Reserved:  [4]byte{},
			Length:    blocks.AtblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}

//ExtractAttachment 
func (b *Block) ExtractAttachment(file *os.File, outputPath string) app.AttFile {
	var comment string

	//Load data to block
	addr := b.Address + int64(blocks.HeaderSize) + int64(blocks.CalculateLinkSize(b.Header.LinkCount))
	d := b.loadData(file, addr)
	flag := int(d.Flags)
	data := d.EmbeddedData

	f := strings.ReplaceAll(string(*TX.GetText(file, b.Link.TxFilename)), "\\", string(os.PathSeparator))
	filename := filepath.Base(f)
	filetype := *TX.GetText(file, b.Link.TxMimetype)

	ci := fmt.Sprint(d.CreatorIndex)
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

	//Create file output
	p := filepath.Join(outputPath + filename)

	//External File
	if !blocks.IsBitSet(flag, 0) {
		MdCommentAdress := b.Link.MDComment
		if MdCommentAdress != 0 {
			comment = string(*MD.New(file, MdCommentAdress))
		}

		fmt.Printf("\n%s is external, the path to the file is %s", filename, f)
		return app.AttFile{
			Name:    filename,
			Type:    filetype,
			Comment: comment,
			Path:    f,
			CreatorIndex:ci,
		}
	}
	fmt.Println("### Embbeded")

	//Embbeded file - Compressed Zip
	if blocks.IsBitSet(flag, 1) {
		fmt.Println("### COMPRESSED")
		data = decompressFile(d)
	}

	//Embbeded file - MD5 check sum
	if blocks.IsBitSet(flag, 2) {
		if md5.Sum(data) != d.MD5Checksum{
			fmt.Println("Checksums do not match. The file may be corrupted. File:", filename)	
		}
	}

	saveFile(file, p, &data)

	return app.AttFile{
		Name:    filename,
		Type:    filetype,
		Comment: comment,
		Path:    p,
		CreatorIndex:ci,
	}
}

//decompressFile uses zlib to decompress databyte
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

//saveFile saves bytes to target file
func saveFile(file *os.File, outputPath string, data *[]byte) error {
	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Error to create the file output: ", err)
		return err
	}
	_, err = f.Write(*data)
	if err != nil {
		fmt.Println("Error to write data to file output: ", err)
		return err
	}

	return nil
}

func (b *Block) loadData(file *os.File, adress int64) *Data {
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
		fmt.Println("Error reading Link section:", err)
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

	return &d
}
