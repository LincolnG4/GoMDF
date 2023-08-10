package main

import (
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/mdf"
)

func main() {
	file, err := os.Open("samples/sample3.mf4")
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
	
	defer file.Close()
	
	mdf.OpenFile(file)

}