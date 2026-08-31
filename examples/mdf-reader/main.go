// Command mdf-reader demonstrates the GoMDF API: it prints the metadata
// tree of an MF4 file and the first samples of every channel.
package main

import (
	"fmt"
	"log"
	"os"

	mf4 "github.com/LincolnG4/GoMDF"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <file.mf4>", os.Args[0])
	}
	f, err := mf4.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Printf("MDF v%d, written by %s, started %s\n\n",
		f.Version(), f.Program(), f.StartTime())

	for gi, g := range f.Groups() {
		fmt.Printf("group %d %q: %d records\n", gi, g.Name, g.RecordCount)
		for _, ch := range g.Channels() {
			sig, err := ch.Read(mf4.WithRange(0, 5))
			if err != nil {
				fmt.Printf("  %-40s ERROR: %v\n", ch.Name, err)
				continue
			}
			var head any
			switch sig.Type {
			case mf4.SampleString:
				head = sig.Strings
			case mf4.SampleBytes:
				head = fmt.Sprintf("%d byte arrays", sig.Len())
			default:
				head = sig.Float64s()
			}
			fmt.Printf("  %-40s [%s] %v\n", ch.Name, ch.Unit, head)
		}
	}

	if atts, err := f.Attachments(); err == nil && len(atts) > 0 {
		fmt.Println("\nattachments:")
		for _, a := range atts {
			fmt.Printf("  %s (%s, %d bytes)\n", a.Filename, a.MimeType, a.Size)
		}
	}
}
