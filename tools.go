package mf4

import "fmt"

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

func (m *MF4) firstFileHistory() int64 {
	return m.Header.Link.FhFirst
}

func (m *MF4) firstAttachment() int64 {
	return m.Header.Link.AtFirst
}

func (m *MF4) StartTimeNS() uint64 {
	return m.Header.Data.StartTimeNs
}

// Time zone offset in minutes.
// The value is not necessarily a multiple of 60
// and can be negative! For the current time zone
// definitions, it is expected to be in the range [-840,840] min.
// For example a value of 60 (min) means UTC+1
// time zone = Central European Time (CET).
func (m *MF4) TimezoneOffsetMin() (int16, error) {
	if !m.isTimeOffsetValid() {
		return 0, fmt.Errorf("timezone is not valid for this file")
	}
	return m.Header.Data.TZOffsetMin, nil
}

// [False]: Local time flag
// If set, the start time stamp in nanoseconds
// represents the local time instead of the UTC
// time. In this case, time zone and DST offset
// must not be considered (time offsets flag must
// not be set). Should only be used if UTC time is
// unknown.
// If the bit is not set (default), the start time
// stamp represents the UTC time.
//
// [True]: Time offsets valid flag
// If set, the time zone and DST offsets are valid.
// Must not be set together with "local time" flag
// (mutually exclusive).
// If the offsets are valid, the locally displayed
// time at start of recording can be determined
// (after conversion of offsets to ns) by
// Local time = UTC time + time zone offset +
// DST offset.
func (m *MF4) isTimeOffsetValid() bool {
	return m.Header.Data.TimeFlags == 1
}
