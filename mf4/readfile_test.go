package mf4_test

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/LincolnG4/GoMDF/mf4"
)

type TestCase struct {
	file                       *os.File
	expectedChannel            []string
	expectedSample             []any
	expectedVersion            uint16
	expectedNumberOfAttachment int
	expectedAttachmentName     string
	expectedType               string
}

func loadSimpleTestCase() TestCase {
	file, _ := os.Open("../samples/sample2.mf4")
	return TestCase{
		file:            file,
		expectedChannel: []string{"channel_b", "channel_c", "time", "channel_a"},
		expectedSample:  []any{5, 10, 0, 10, 5, 10, 10, 5, 0, 0, 10, 5, 10, 5, 10, 0, 0, 5, 5, 0},
		expectedVersion: 410,
	}
}

func loadAttachmentAndConnverstionTestCase() TestCase {
	file, _ := os.Open("../samples/sample3.mf4")
	return TestCase{
		file:                       file,
		expectedChannel:            []string{"t", "VehSpd_Cval_CPC"},
		expectedSample:             []any{99.74609375, 99.7578125, 99.69140625},
		expectedNumberOfAttachment: 1,
		expectedAttachmentName:     "user_embedded_display.dspf",
		expectedType:               "application/x-dspf",
	}
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
	if value != testcase.expectedVersion {
		t.Fatalf(`wrong version: expected %d, found %d`, testcase.expectedVersion, value)
	}
}

func TestReadChannels(t *testing.T) {
	testcase := loadSimpleTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	for _, expected := range m.ChannelNames()[0] {
		if !slices.Contains(testcase.expectedChannel, expected) {
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
	if len(result) != len(testcase.expectedSample) {
		t.Errorf("Lengths mismatch. Expected: %d, Got: %d", len(testcase.expectedSample), len(result))
		return
	}

	if ok, err := compareSlices(testcase.expectedSample, result); !ok {
		t.Error(err)
	}
}

func TestReadAttachment(t *testing.T) {
	testcase := loadAttachmentAndConnverstionTestCase()

	m, _ := mf4.ReadFile(testcase.file)

	att := m.GetAttachments()
	if len(att) != testcase.expectedNumberOfAttachment {
		t.Errorf("Wrong attachment size,  Expected: %d, Got: %d", testcase.expectedNumberOfAttachment, len(att))
	}
	if att[0].Name != testcase.expectedAttachmentName {
		t.Errorf("Wrong attachment size,  Expected: %s, Got: %s", testcase.expectedAttachmentName, att[0].Name)
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
	if len(result) != len(testcase.expectedSample) {
		t.Errorf("Lengths mismatch. Expected: %d, Got: %d", len(testcase.expectedSample), len(result))
		return
	}
	if ok, err := compareSlices(testcase.expectedSample, result); !ok {
		t.Error(err)
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
