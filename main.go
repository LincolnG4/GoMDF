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
			fmt.Println("Could not open the file")
			panic(err)
		}

	}

	defer file.Close()

	mdf.ReadFile(file)

}
