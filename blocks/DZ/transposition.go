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
	data, err := Deflate(*t.Datablock)
	if err != nil {
		return nil, err
	}

	// calculete Matrix row and column
	rows := uint64(t.Parameter)
	cols := t.DecompressedLength / rows

	t.Transpose(rows, cols, &data)

	return data, nil
}

// Transpose applies traposition in the data block
func (t *Transposition) Transpose(rows uint64, cols uint64, data *[]byte) {
	//TODO IF t.DecompressedLength % cols != 0
	if len(*data)%int(cols) != 0 {

	}

	arr := *data
	// Create a new array for the transposed data
	transposed := make([]byte, t.DecompressedLength)

	var i, j uint64
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			// Transpose the element from (i, j) in the original to (j, i)
			transposed[j*rows+i] = arr[i*cols+j]
		}
	}

	copy(arr, transposed)
}
