package main

import (
	"fmt"
	"io"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
	"github.com/LincolnG4/GoMDF/app"
)

func main() {
	file, err := os.Open("sample3.MF4")

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

	for dg,cn := range m.ChannelNames(){
		for _,ch := range cn {
			value, err := m.GetChannelSample(dg, ch)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("\nChannel %s ==> Value %v",ch,value)
		}
	}
	

	//Extract embedded and compressed files from MF4
	fa := []app.AttFile{}
	for _, value := range m.Attachments {
		fa = append(fa, value.ExtractAttachment(file, "/home/lincolng/Downloads/"))
	}
	fmt.Println(fa)
}
