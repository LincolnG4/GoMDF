# GoMDF
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
  "github.com/LincolnG4/GoMDF/mf4"
)

func main() {
  file, err := os.Open("example.mdf")
  if err != nil {
    panic(err)
  }
  
  mdf := mf4.ReadFile(file, false)
  
  // Access metadata
  fmt.Println(mdf.Version()) 
  fmt.Println("Version ID --> ", m.MdfVersion())
  fmt.Println("Start Time NS --> ", m.StartTimeNs())
  fmt.Println("Start StartTimeLT --> ", m.StartTimeLT())

  // Get channel samples
  samples, err := mdf.GetChannelSample(0, "Channel1")
  if err != nil {
    panic(err) 
  }

  // Download attachments
  att := mdf.GetAttachments()[0]
  mdf.SaveAttachment(att, "/PATH/TO/BE/SAVE/")

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
