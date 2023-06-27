package main

import (
	"fmt"
	"io"
	"os"
)

func main() {

	file, err := os.Open("samples/sample3.mf4")
	errorHandler(err)
	defer file.Close()

	//Create IDBLOCK
	idBlock := IDBlock{}
	idBlock.init(file)

	if idBlock.IDVersionNumber > 400 {
		//Create HDBLOCK
		hdBlock := HDBlock{}
		hdBlock.init(file)

	}

}

func errorHandler(err error) {
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
}
