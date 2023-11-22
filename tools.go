package mf4

import (
	"fmt"
	"time"
)

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

func (m *MF4) StartTimeNs() uint64 {
	tns := m.Header.Data.StartTimeNs
	if m.isTimeOffsetValid() {
		return tns
	}
	return tns + uint64(m.getTimezoneOffsetMin()) + uint64(m.getDaylightOffsetMin())
}

func (m *MF4) StartTimeString() time.Time {
	return time.Unix(0, int64(m.StartTimeNs()))
}

// Time zone offset in minutes. The value may not necessarily be a multiple
// of 60 and can be negative. As per current time zone definitions, it is
// expected to fall within the range [-840, 840] minutes. For instance, a
// value of 60 minutes implies UTC+1 time zone, corresponding to Central
// European Time (CET).
func (m *MF4) TimezoneOffsetMin() (int16, error) {
	if !m.isTimeOffsetValid() {
		return 0, fmt.Errorf("timezone is not valid for this file")
	}
	return m.getTimezoneOffsetMin(), nil
}

func (m *MF4) getTimezoneOffsetMin() int16 {
	return m.Header.Data.TZOffsetMin
}

// Daylight saving time (DST) offset in minutes for the starting timestamp.
// During the summer months, many regions observe a DST offset of 60 minutes (1 hour).
func (m *MF4) DaylightOffsetMin() (int16, error) {
	if !m.isTimeOffsetValid() {
		return 0, fmt.Errorf("daylight is not valid for this file")
	}
	return m.getDaylightOffsetMin(), nil
}

func (m *MF4) getDaylightOffsetMin() int16 {
	return m.Header.Data.TZOffsetMin
}

// [False]: Local time flag
// When activated, the starting timestamp in nanoseconds represents the loca
// time rather than UTC time. In this scenario, disregard time zone and DST
// offset considerations (ensure the time offsets flag is not set). This
// option should only be utilized when UTC time is unknown. If the bit is not
// activated (default), the starting timestamp signifies UTC time.
//
// [True]: Time offsets valid flag
// When activated, the time zone and DST offsets are deemed valid. It should
// not be activated simultaneously with the "local time" flag, as they are
// mutually exclusive. With valid offsets, the locally displayed time at the
// beginning of recording can be determined (after converting offsets to
// nanoseconds) using the formula:
// Local time = UTC time + time zone offset + DST offset.
func (m *MF4) isTimeOffsetValid() bool {
	return m.Header.Data.TimeFlags == 1
}

// Start angle in radians at the beginning of the measurement serves as the
// reference point for angle synchronous measurements. All subsequent angle
// values for master channels or events synchronized with angle are expressed
// relative to this starting angle.
func (m *MF4) StartAngleRad() (float64, error) {
	if !m.isDistanceValid() {
		return 0, fmt.Errorf("start angle rad is not valid for this file")
	}
	return m.getStartAngleRad(), nil
}

// Start distance in meters in meters at the beginning of the measurement serves
// as the reference point for distance synchronous measurements. All subsequent
// distance values for master channels or events synchronized with distance are
// expressed relative to this starting distance.
func (m *MF4) StartDistanceM() (float64, error) {
	if m.isDistanceValid() {
		return 0, fmt.Errorf("start distance meters is not valid for this file")
	}
	return m.getStartDistanceM(), nil
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
