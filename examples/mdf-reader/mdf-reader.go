package main

import (
	"fmt"
	"io"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
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

	fmt.Println("Version ID --> ", m.MdfVersion())
	fmt.Println("Start Time NS --> ", m.StartTimeNs())
	fmt.Println("Start Start--> ", m.StartTimeLT())

	//Return all channels availables
	channels := m.ChannelNames()
	fmt.Println(channels)

	for dg, cn := range m.ChannelNames() {
		for _, ch := range cn {
			value, err := m.GetChannelSample(dg, ch)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("\nChannel %s ==> Value %v", ch, value)
		}
	}

	value, err := m.GetChannelSample(0, "dwordCounter")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("\nChannel %s ==> Value %v", "Triangle", value)
	//Extract embedded and compressed files from MF4
	fa := m.GetAttachmemts()
	fmt.Println(fa)
	d := m.SaveAttachment(fa[1], "/home/lincolng/Downloads/testFolder/")
	fmt.Println(d)

	m.ReadChangeLog()
}
