package mf4_test

import (
	"os"
	"slices"
	"testing"

	"github.com/LincolnG4/GoMDF/mf4"
)

type TestCase struct {
	file            *os.File
	expectedChannel []string
	expectedSample  []int64
	expectedVersion uint16
}

func loadSimpleTestCase() TestCase {
	file, _ := os.Open("../samples/sample4.mf4")
	return TestCase{
		file:            file,
		expectedChannel: []string{"channel_b", "channel_c", "time", "channel_a"},
		expectedSample:  []int64{5, 10, 0, 10, 5, 10, 10, 5, 0, 0, 10, 5, 10, 5, 10, 0, 0, 5, 5, 0},
		expectedVersion: 410,
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

	for index, value := range result {
		if testcase.expectedSample[index] != value.(int64) {
			t.Errorf("Mismatch sample. Expected: %d, Got: %d", testcase.expectedSample[index], value)
			return
		}
	}
}
