# GoMDF - ASAM MDF for Golang
ASAM MDF / MF4 (Measurement Data Format) files editor in GoLang
Package based on <https://github.com/danielhrisca/asammdf>

⚠️ The package not finalized   !!! ⚠️

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

⚠️ The package not finalized !!! ⚠️

```go
package main

import (
 "fmt"
 "io"
 "os"

 "github.com/LincolnG4/GoMDF/app"
 "github.com/LincolnG4/GoMDF/mf4"
)

func main() {
 file, err := os.Open("/PATH/TO/file.mf4")

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
 fmt.Println("Start StartTimeLT --> ", m.StartTimeLT())
 cc, _ := m.StartDistanceM()
 fmt.Println("Start Distance M --> ", cc)
 dd, _ := m.StartAngleRad()
 fmt.Println("Start Angle Rad --> ", dd)
 fmt.Println(m.GetMeasureComment())
 //Return all channels availables
 channels := m.ChannelNames()
 fmt.Println(channels)

 // for dg,cn := range m.ChannelNames(){
 // for _, ch := range cn {
 value, err := m.GetChannelSample(0, "dwordCounter")
 if err != nil {
  fmt.Println(err)
 }
 fmt.Printf("\nChannel %s ==> Value %v", "Triangle", value)
 // }
 // }


 //Extract embedded and compressed files from MF4
 fa := m.GetAttachmemts()
 fmt.Println(fa)
 d := m.SaveAttachment(fa[1], "/PATH/TO/BE/SAVE/")}
 fmt.Println(d)

 m.ReadChangeLog()
}

```
