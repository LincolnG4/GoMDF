package mf4

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/AT"
	"github.com/LincolnG4/GoMDF/internal/blocks/CG"
	"github.com/LincolnG4/GoMDF/internal/blocks/CN"
	"github.com/LincolnG4/GoMDF/internal/blocks/DG"
	"github.com/LincolnG4/GoMDF/internal/blocks/DT"
	"github.com/LincolnG4/GoMDF/internal/blocks/EV"
	"github.com/LincolnG4/GoMDF/internal/blocks/FH"
	"github.com/LincolnG4/GoMDF/internal/blocks/HD"
	"github.com/LincolnG4/GoMDF/internal/blocks/ID"
	"github.com/LincolnG4/GoMDF/internal/blocks/MD"
	"github.com/LincolnG4/GoMDF/internal/blocks/SI"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
	"github.com/davecgh/go-spew/spew"
)

type MF4 struct {
	File           *os.File
	Header         *HD.Block
	Identification *ID.Block
	//Address to First File History Block
	FileHistory  int64
	ChannelGroup []*ChannelGroup
}

type ChannelGroup struct {
	Block      *CG.Block
	Channels   map[string]*Channel
	DataGroup  *DG.Block
	SourceInfo SI.SourceInfo
	Comment    string
}

func ReadFile(file *os.File) (*MF4, error) {
	var address int64 = 0
	mf4File := MF4{
		File:           file,
		Identification: ID.New(file, address),
	}
	fileVersion := mf4File.MdfVersion()
	if fileVersion < 400 {
		return nil, fmt.Errorf("file version is not >= 4.00")
	}

	if fileVersion >= 400 {
		mf4File.loadHeader()
		mf4File.loadFirstFileHistory()
		mf4File.loadEvents()
		mf4File.read()
	}
	return &mf4File, nil
}

func (m *MF4) read() {
	var file *os.File = m.File

	if !m.IsFinalized() {
		panic("NOT FINALIZED MF4, PACKAGE IS NOT PREPARED")
	}

	version := m.MdfVersion()
	nextDataGroupAddress := m.firstDataGroup()

	comment := ""
	for nextDataGroupAddress != 0 {
		dataGroupBlock := DG.New(file, nextDataGroupAddress)
		mdCommentAddr := dataGroupBlock.MetadataComment()
		if mdCommentAddr != 0 {
			comment = MD.New(file, mdCommentAddr)
		}

		nextAddressCG := dataGroupBlock.FirstChannelGroup()
		for nextAddressCG != 0 {
			cgBlock := CG.New(file, version, nextAddressCG)

			channelGroup := &ChannelGroup{
				Block:      cgBlock,
				Channels:   make(map[string]*Channel),
				DataGroup:  dataGroupBlock,
				SourceInfo: SI.Get(file, version, cgBlock.Link.SiAcqSource),
				Comment:    comment,
			}

			nextAddressCN := cgBlock.FirstChannel()
			for nextAddressCN != 0 {
				cnBlock := CN.New(file, version, nextAddressCN)
				cn := &Channel{
					Name:         cnBlock.GetChannelName(m.File),
					ChannelGroup: cgBlock,
					DataGroup:    dataGroupBlock,
					Block:        cnBlock,
					SourceInfo:   SI.Get(file, version, cnBlock.Link.SiSource),
					Convertion:   cnBlock.GetConversion(m.File, version, cnBlock.GetDataType()),
				}

				channelGroup.Channels[cn.Name] = cn

				comment := ""
				mdCommentAdress := cnBlock.GetCommentMd()
				if mdCommentAdress != 0 {
					comment = MD.New(file, mdCommentAdress)
				}

				cn.Comment = comment

				nextAddressCN = cnBlock.Next()
			}
			m.ChannelGroup = append(m.ChannelGroup, channelGroup)
			nextAddressCG = cgBlock.Next()
		}

		nextDataGroupAddress = dataGroupBlock.Next()
	}
}

/*
ChannelNames returns a map of channels of each datagroup

	map[key]value
	key = Datagroup number
	value = array with channel names
*/
func (m *MF4) ChannelNames() map[int][]string {
	channelMap := make(map[int][]string, 0)
	for i, cg := range m.ChannelGroup {
		channelNames := make([]string, 0)
		for name := range cg.Channels {
			channelNames = append(channelNames, name)
		}
		channelMap[i] = channelNames
	}
	return channelMap
}

// GetChannelSample loads sample by DataGroupName and ChannelName
func (m *MF4) GetChannelSample(indexDataGroup int, channelName string) ([]interface{}, error) {
	var sample []interface{}
	var err error

	file := m.File
	cgrp := m.ChannelGroup[indexDataGroup]

	// Does channel exist in datagroup?
	cn, ok := cgrp.Channels[channelName]
	if !ok {
		return nil, fmt.Errorf("channel %s doens't exist", channelName)
	}

	dg := cgrp.DataGroup
	// cg := cgrp.Block

	//Get channel with compositon Structure or Array
	// if cn.IsComposed() {

	// }

	// if cn.IsAllValuesInvalid() {
	// 	return nil, fmt.Errorf("channel %s has invalid read", channelName)
	// }

	dataRecordBlock := DT.New(file, dg.Link.Data)

	if dataRecordBlock.DataBlockType() == blocks.DtID || dataRecordBlock.DataBlockType() == blocks.DvID {
		sample, err = cn.readSingleDataBlock(file)
	} else {
		fmt.Println("package not ready to read")
	}
	if err != nil {
		return nil, err
	}

	cn.applyConvertion(&sample)
	return sample, nil
}

// loadEvents loads and processes events from the given MF4 instance.
// Events are represented by EVBLOCK structures, providing synchronization details.
// The function iterates through the linked list of events, creating EV instances
// and handling event details such as names, comments, and scopes.
// If errors occur during EV instance creation, they are printed to the console.
//
// Parameters:
//
//	m: A pointer to the MF4 instance containing events.
func (m *MF4) loadEvents() {
	if m.getFirstEvent() == 0 {
		return
	}

	nextEvent := m.getFirstEvent()
	for nextEvent != 0 {
		event, err := EV.New(m.File, m.MdfVersion(), nextEvent)
		if err != nil {
			fmt.Println(err)
		}
		nextEvent = event.Next()
	}
}

func readArrayBlock(file *os.File, addr int64) {
	//debug(file,addr,400)
}

// GetAttachmemts iterates over all AT blocks and return to an array
func (m *MF4) GetAttachments() []AT.AttFile {
	return AT.Get(m.File, m.getFirstAttachment())
}

// Saves attachment file input to output path
func (m *MF4) SaveAttachment(attachment AT.AttFile, outputPath string) AT.AttFile {
	return attachment.Save(m.File, outputPath)
}

// Version method returns the MDF file version
func (m *MF4) Version() string {
	return string(m.Identification.Version[:])
}

// ID method returns the MDF file ID
func (m *MF4) ID() string {
	return string(m.Identification.File[:])
}

// CreatedBy method returns the MDF Program identifier
func (m *MF4) CreatedBy() string {
	return string(m.Identification.Program[:])
}

// VersionNumber method returns the Version number of the MDF format, i.e. 420
func (m *MF4) MdfVersion() uint16 {
	return m.Identification.VersionNumber
}

// isUnfinalized method returns Standard flags for unfinalized MDF
func (m *MF4) IsFinalized() bool {
	return m.Identification.UnfinalizedFlag == 0
}

func (m *MF4) firstDataGroup() int64 {
	return m.Header.Link.DgFirst
}

func (m *MF4) loadFirstFileHistory() {
	m.FileHistory = m.Header.Link.FhFirst
}

func (m *MF4) getFirstAttachment() int64 {
	return m.Header.Link.AtFirst
}

func (m *MF4) getFirstEvent() int64 {
	return m.Header.Link.EvFirst
}

// Start angle in radians at the beginning of the measurement serves as the
// reference point for angle synchronous measurements.
func (m *MF4) StartAngleRad() (float64, error) {
	if !m.isDistanceValid() {
		return 0, fmt.Errorf("start angle rad is not valid for this file")
	}
	return m.getStartAngleRad(), nil
}

// Start distance in meters in meters at the beginning of the measurement serves
// as the reference point for distance synchronous measurements.
func (m *MF4) StartDistanceM() (float64, error) {
	if m.isDistanceValid() {
		return 0, fmt.Errorf("start distance meters is not valid for this file")
	}
	return m.getStartDistanceM(), nil
}

func (m *MF4) GetMeasureComment() string {
	if m.getHeaderMdComment() == 0 {
		return ""
	}

	t, err := TX.GetText(m.File, m.getHeaderMdComment())
	if err != nil {
		return ""
	}

	return t
}

// ReadChangeLog reads and prints the change log entries from the MF4 file.
// The change log is stored in FHBLOCK structures, each representing a change made to the MDF file.
// The function iterates through the linked list of FHBLOCKs starting from the first one referenced
// by the HDBLOCK, printing the chronological change history.
//
// Parameters:
//
//	m: A pointer to the MF4 instance containing the file change log.
func (m *MF4) ReadChangeLog() {
	nextAddressFH := m.getFileHistory()
	for nextAddressFH != 0 {
		fhBlock := FH.New(m.File, nextAddressFH)
		c := fhBlock.GetChangeLog(m.File)
		t := fhBlock.GetTimeNs()
		f := fhBlock.GetTimeFlag()

		fmt.Println(m.formatLog(t, f, c))

		nextAddressFH = fhBlock.Next()
	}
}

// StartTimeNs returns the start timestamp of measurement in nanoseconds
func (m *MF4) GetStartTimeNs() int64 {
	t := m.getStartTimeNs()
	tzo := uint64(m.getHDTimezoneOffsetMin())
	dlo := uint64(m.getDaylightOffsetMin())
	tf := m.getTimeFlag()
	return m.GetTimeNs(t, tzo, dlo, tf)
}

func (m *MF4) GetStartTimeLT() time.Time {
	return m.formatTimeLT(m.GetStartTimeNs())
}

func (m *MF4) getFileHistory() int64 {
	return m.FileHistory
}

func (m *MF4) formatLog(t int64, f uint8, c string) string {
	ts := m.formatTimeLT(t)
	return fmt.Sprint(ts, c)
}

func (m *MF4) getHDTimezoneOffsetMin() int16 {
	return m.Header.Data.TZOffsetMin
}

func (m *MF4) getTimeFlag() uint8 {
	return m.Header.Data.TimeFlags
}

func (m *MF4) getStartTimeNs() uint64 {
	return m.Header.Data.StartTimeNs
}

func (m *MF4) getStartAngleRad() float64 {
	return m.Header.Data.StartAngleRad
}

func (m *MF4) getStartDistanceM() float64 {
	return m.Header.Data.StartDistM
}

func (m *MF4) isDistanceValid() bool {
	return m.Header.Data.Flags == 1
}

func (m *MF4) getTimeClass() uint8 {
	return m.Header.Data.TimeClass
}

func (m *MF4) loadHeader() {
	m.Header = HD.New(m.File, blocks.IdblockSize)
}

func (m *MF4) getHeaderMdComment() int64 {
	return m.Header.Link.MdComment
}

func seekRead(file *os.File, readAddr int64, data []byte) {
	_, errs := file.Seek(readAddr, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "memory Addr out of size")
		}
	}
	_, err := file.Read(data)
	if err != nil {
		if err != io.EOF {
			fmt.Println("loadBuffer error: ", err)
		}
	}
}

func debug(file *os.File, offset int64, size int) {
	_, err := file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, size)
	n, err := file.Read(buf[:cap(buf)])
	buf = buf[:n]
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
	spew.Dump(buf)
}
