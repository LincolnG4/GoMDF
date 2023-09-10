package main

import (
	"fmt"
	"io"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
	"github.com/LincolnG4/GoMDF/app"
)

func main() {
	file, err := os.Open("/home/lincolng/Downloads/MDF/BaseStandard/Examples/Attachments/EmbeddedCompressed/Vector_EmbeddedCompressed.MF4")

	if err != nil {
		if err != io.EOF {
			fmt.Println("Could not open the file")
			panic(err)
		}

	}

	defer file.Close()

	m, err := mf4.ReadFile(file, true)
	if err != nil {
		fmt.Println(err)
	}

	version := m.Version()
	fmt.Println(version)

	//Return all channels availables
	channels := m.ChannelNames()
	fmt.Println(channels)

	value, err := m.GetChannelSample(0,channels[0][0])
	if err != nil {
	fmt.Println(err)
	}
	fmt.Println(value)
	
	//Extract embedded and compressed files from MF4 
	fa := []app.AttFile{}
	for _, value := range m.Attachments {
		fa= append(fa,value.ExtractAttachment(file, "/home/lincolng/Downloads/"))
	}
	fmt.Println(fa)
}
