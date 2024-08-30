package DZ

type Transposition struct {
	//Decompressed ID
	DecompressedID string

	//Zip algorithm (Deflate or Transposition + Deflate)
	CompressType uint8

	//Decompressed size
	DecompressedLength uint64

	//Compressed size
	CompressedLength uint64

	//Parameter for transposition type (CompressType==1)
	Parameter uint32

	//DataBlock
	Datablock *[]byte
}

func (t *Transposition) Decompress() ([]byte, error) {
	return nil, nil
}
