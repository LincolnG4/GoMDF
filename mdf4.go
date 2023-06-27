package main

import (
	"fmt"
	"io"
	"os"
)

func main() {

	file, err := os.Open("samples/sample1.mf4")
	errorHandler(err)
	defer file.Close()

	//Create IDBLOCK
	idBlock := IDBlock{}
	idBlock.init(file)

	fmt.Printf("%+v\n", idBlock)

	if idBlock.IDVersionNumber > 400 {
		//Create HDBLOCK
		hdBlock := HDBlock{}
		hdBlock.init(file)

		fmt.Printf("%+v\n", hdBlock)
		fmt.Printf("%d \n", hdBlock.HDFHFirst)
		buf := seekBinaryByAddress(file, hdBlock.HDFHFirst, 56)
		fmt.Println(string(buf))

	}

}

func errorHandler(err error) {
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
}
