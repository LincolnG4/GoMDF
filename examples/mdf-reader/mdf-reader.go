package main

import (
	"fmt"
	"io"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
)

func main() {
	file, err := os.Open("./samples/sample3.mf4")

	if err != nil {
		if err != io.EOF {
			fmt.Println("Could not open the file")
			panic(err)
		}

	}

	defer file.Close()
	
	m, err := mf4.ReadFile(file,true)
	if err != nil{
		fmt.Println(err)
	}
	version := m.Version()
	fmt.Print(version)

	//Return []string with channels name e.g [time,EngSpeed, ...]
	channels := m.ChannelNames()
	fmt.Println(channels)

}

