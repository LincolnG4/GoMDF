package mf4

import (
	"fmt"
	"time"
)

func (m *MF4) GetTimeNs(t uint64, tzo uint64, dlo uint64, tf uint8) int64 {
	if m.isTimeOffsetValid(tf) {
		return int64(t)
	}
	return int64(t + tzo + dlo)
}

func (m *MF4) formatTimeLT(t int64) time.Time {
	return time.Unix(0, t)
}

// Time zone offset in minutes. Range (-840, 840) minutes. For instance,
// a value of 60 minutes implies UTC+1 time zone, corresponding to Central
// European Time (CET).
func (m *MF4) TimezoneOffsetMin(tzo int16, timeFlag uint8) (int16, error) {
	if !m.isTimeOffsetValid(timeFlag) {
		return 0, fmt.Errorf("timezone is not valid for this file")
	}
	return tzo, nil
}

// Daylight saving time (DST) offset in minutes for the starting timestamp.
// During the summer months, many regions observe a DST offset of 60 minutes
// (1 hour).
func (m *MF4) DaylightOffsetMin(tFlag uint8) (int16, error) {
	if !m.isTimeOffsetValid(tFlag) {
		return 0, fmt.Errorf("daylight is not valid for this file")
	}
	return m.getDaylightOffsetMin(), nil
}

func (m *MF4) getDaylightOffsetMin() int16 {
	return m.Header.Data.TZOffsetMin
}

// [False]: Local time flag
//
// [True]: Time offsets valid flag
func (m *MF4) isTimeOffsetValid(timeFlag uint8) bool {
	return timeFlag == 1
}
