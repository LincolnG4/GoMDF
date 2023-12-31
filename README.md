# GoMDF - Read and Write ASAM MDF FILES
Go package for reading ASAM MDF files.

## Installation
⚠️ The package not finalized   !!! ⚠️
```
go get github.com/LincolnG4/GoMDF
```

## Usage

```Go
package main

import (
	"fmt"
	"os"

	mf4 "github.com/LincolnG4/GoMDF/mf4"
)

func main() {
	file, err := os.Open("sample3.mf4")
	if err != nil {
		panic(err)
	}

	m, err := mf4.ReadFile(file)
	if err != nil {
		panic(err)
	}
	// Access metadata
	fmt.Println(m.Version())
	fmt.Println("Version ID --> ", m.MdfVersion())
	fmt.Println("Start Time NS --> ", m.StartTimeNs())
	fmt.Println("Start StartTimeLT --> ", m.StartTimeLT())

	// Get channel samples
	fmt.Println(m.ChannelNames())
	samples, err := m.GetChannelSample(0, "ActlEngPrcntTorqueHighResolution")
	if err != nil {
		panic(err)
	}
	fmt.Println(samples)
	// Download attachments
	att := m.GetAttachments()[0]
	m.SaveAttachment(att, "/PATH/TO/BE/SAVE/")

	// Read Change logs
	m.ReadChangeLog()
}

```

## Features
- Parse MDF file format and load metadata
- Extract channel sample data 
- Support for attachments
- Support for Events
- Access to common metadata fields
- Documentation
- API documentation is available at https://godoc.org/github.com/LincolnG4/GoMDF

## Contributing
Pull requests are welcome! Please open any issues.

This provides a high-level overview of how to use the package from Go code along with installation instructions. Let me know if any part of the README explanation could be improved!

## References 

[ASAM MDF](https://github.com/danielhrisca/asammdf)  
[MDF Validator ](https://www.vector.com/int/en/products/application-areas/ecu-calibration/measurement/mdf/) 
