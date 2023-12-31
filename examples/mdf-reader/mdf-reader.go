package main

import (
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/mf4"
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

	m, err := mf4.ReadFile(file)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Version ID --> ", m.MdfVersion())
	fmt.Println("Start Time NS --> ", m.GetStartTimeNs())
	fmt.Println("Start StartTimeLT --> ", m.GetStartTimeLT())

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

	// Access metadata
	fmt.Println(m.Version())
	fmt.Println("Version ID --> ", m.MdfVersion())
	fmt.Println("Start Time NS --> ", m.GetStartTimeNs())
	fmt.Println("Start StartTimeLT --> ", m.GetStartTimeLT())

	// Get channel samples
	fmt.Println(m.ChannelNames())
	samples, err := m.GetChannelSample(0, "Signal")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(samples)
	// Download attachments
	// att := m.GetAttachments()[0]
	// m.SaveAttachment(att, "/PATH/TO/BE/SAVE/")

	// Read Change logs
	m.ReadChangeLog()
}
