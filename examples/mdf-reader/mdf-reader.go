package main

import (
	"fmt"
	"io"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
)

func main() {
	file, err := os.Open("../../samples/sample3.mf4")

	if err != nil {
		if err != io.EOF {
			fmt.Println("Could not open the file")
			panic(err)
		}

	}

	defer file.Close()
	
	mf4 := mf4.ReadFile(file,true)
	version := mf4.Version()
	fmt.Print(version)



}

