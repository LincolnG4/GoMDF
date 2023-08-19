package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	file, err := os.Open("sample3.mf4")

	if err != nil {
		if err != io.EOF {
			fmt.Println("Could not open the file")
			panic(err)
		}

	}

	defer file.Close()
	
}

