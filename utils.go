package main

import "os"

func seekBinaryByAddress(file *os.File, address int64, block_size int) []byte {
	buf := make([]byte, block_size)
	_, errs := file.Seek(address, 0)
	errorHandler(errs)
	_, err := file.Read(buf)
	errorHandler(err)
	return buf
}
