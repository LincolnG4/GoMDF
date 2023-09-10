# GoMDF - ASAM MDF for Golang
ASAM MDF / MF4 (Measurement Data Format) files editor in GoLang
Package based on <https://github.com/danielhrisca/asammdf>

⚠️ The package is still under development  !!! ⚠️

## **Targets / Tasks**:
- [ ] Read MF4 Files
- [ ] Write MF4 Files
- [ ] Optimize  
- [ ] Read/Write any version of MDF file  
- [ ] Optimize  

## Getting Started

### Installation  

Use go get to retrieve the package to add it to your GOPATH workspace, or project's Go module dependencies.

```go
go get github.com/LincolnG4/GoMDF@main
```

## Quick Examples

⚠️ The package is still under development !!! ⚠️

```go
package main

import (
 "fmt"
 "io"
 "os"

 mf4 "github.com/LincolnG4/GoMDF"
 "github.com/LincolnG4/GoMDF/app"
)

func main() {
 file, err := os.Open("sample.MF4")

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
  fa= append(fa,value.ExtractAttachment(file, "/home/"))
 }
 fmt.Println(fa)
}


```