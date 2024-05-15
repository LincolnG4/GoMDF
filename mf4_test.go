package mf4_test

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

type TestCase struct {
	file               *os.File
	Channel            []string
	Sample             []any
	Version            uint16
	NumberOfAttachment int
	AttachmentName     string
	AttachmentType     string
}

func loadSimpleTestCase() TestCase {
	file, _ := os.Open("./samples/sample2.mf4")
	return TestCase{
		file:    file,
		Channel: []string{"channel_b", "channel_c", "time", "channel_a"},
		Sample:  []any{5, 10, 0, 10, 5, 10, 10, 5, 0, 0, 10, 5, 10, 5, 10, 0, 0, 5, 5, 0},
		Version: 410,
	}
}

func loadAttachmentAndConnverstionTestCase() TestCase {
	file, _ := os.Open("./samples/sample3.mf4")
	return TestCase{
		file:               file,
		Channel:            []string{"t", "VehSpd_Cval_CPC"},
		Sample:             []any{99.74609375, 99.7578125, 99.69140625},
		NumberOfAttachment: 1,
		AttachmentName:     "user_embedded_display.dspf",
		AttachmentType:     "application/x-dspf",
	}
}

func compareSlices(s1, s2 interface{}) (bool, string) {
	v1 := reflect.ValueOf(s1)
	v2 := reflect.ValueOf(s2)

	if v1.Kind() != reflect.Slice || v2.Kind() != reflect.Slice {
		return false, "result is not slice"
	}

	for i := 0; i < v1.Len(); i++ {
		val1 := v1.Index(i).Interface()
		val2 := v2.Index(i).Interface()

		switch val1 := val1.(type) {
		case int:
			switch val2 := val2.(type) {
			case int:
				if val1 != val2 {
					return false, fmt.Sprintf("Expected: %d, Got: %d", val1, val2)
				}
			case int64:
				if int64(val1) != val2 {
					return false, fmt.Sprintf("Expected: %d, Got: %d", val1, val2)
				}
			default:
				return false, fmt.Sprintf("Expected: %d, Got: %d", val1, val2)
			}
		case int64:
			if val1 != val2.(int64) {
				return false, fmt.Sprintf("Expected: %d, Got: %d", val1, val2)
			}
		case float32:
			if val1 != val2.(float32) {
				return false, fmt.Sprintf("Expected: %f, Got: %f", val1, val2)
			}
		case float64:
			if val1 != val2.(float64) {
				return false, fmt.Sprintf("Expected: %f, Got: %f", val1, val2)
			}
		default:
			return false, fmt.Sprintf("Expected: %f, Got: %f", val1, val2)
		}
	}

	return true, ""
}

func TestReadFile(t *testing.T) {
	testcase := loadSimpleTestCase()

	_, err := mf4.ReadFile(testcase.file)
	if err != nil {
		t.Fatalf(`could not read file: %v`, err)
	}
}

func TestReadBasicInformations(t *testing.T) {
	testcase := loadSimpleTestCase()

	m, _ := mf4.ReadFile(testcase.file)
	value := m.MdfVersion()
	if value != testcase.Version {
		t.Fatalf(`wrong version: expected %d, found %d`, testcase.Version, value)
	}

}

func TestReadChannels(t *testing.T) {
	testcase := loadSimpleTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	for _, expected := range m.ListAllChannelsNames() {
		if !slices.Contains(testcase.Channel, expected) {
			t.Fatalf(`could not find %s`, expected)
		}
	}
}

func TestReadSampleSimpleDTblockINT(t *testing.T) {
	testcase := loadSimpleTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	result, err := m.GetChannelSample(0, "channel_b")
	if err != nil {
		t.Fatalf(`could not read samples from file %v`, err)
	}

	// Check if the lengths of the two arrays are equal
	if len(result) != len(testcase.Sample) {
		t.Errorf("Lengths mismatch. Expected: %d, Got: %d", len(testcase.Sample), len(result))
		return
	}

	if ok, err := compareSlices(testcase.Sample, result); !ok {
		t.Error(err)
	}
}

func TestReadAttachment(t *testing.T) {
	testcase := loadAttachmentAndConnverstionTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	att := m.GetAttachments()
	if len(att) != testcase.NumberOfAttachment {
		t.Errorf("Wrong attachment size,  Expected: %d, Got: %d", testcase.NumberOfAttachment, len(att))
	}
	if att[0].Name != testcase.AttachmentName {
		t.Errorf("Wrong attachment size,  Expected: %s, Got: %s", testcase.AttachmentName, att[0].Name)
	}

}

func TestNestedConversion(t *testing.T) {
	testcase := loadAttachmentAndConnverstionTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	result, err := m.GetChannelSample(0, "VehSpd_Cval_CPC")
	if err != nil {
		t.Fatalf(`could not read samples from file %v`, err)
	}

	// Check if the lengths of the two arrays are equal
	if len(result) != len(testcase.Sample) {
		t.Errorf("Lengths mismatch. Expected: %d, Got: %d", len(testcase.Sample), len(result))
		return
	}
	if ok, err := compareSlices(testcase.Sample, result); !ok {
		t.Error(err)
	}
}

func TestEvents(t *testing.T) {

}
