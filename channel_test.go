package mf4_test

import (
	"os"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

var ZipFile, _ = os.Open("./samples/Discrete_deflate.mf4")
var ZipTestCase = TestCase{
	file:   ZipFile,
	Sample: []any{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26, -16, -6, 4, 14, 24, 34, 44, 54, 64, 74, 84, 94, 104, 114, 124, -122, -112, -102, -92, -82, -72, -62, -52, -42, -32, -22, -12, -2, 8, 18, 28, 38, 48, 58, 68, 78, 88, 98, 108, 118, -128, -118, -108, -98, -88, -78, -68, -58, -48, -38, -28, -18, -8, 2, 12, 22, 32, 42, 52, 62, 72, 82, 92, 102, 112, 122, -124, -114, -104, -94, -84, -74, -64, -54, -44, -34, 0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26},
}

var DataListFile, _ = os.Open("./samples/ASAP2_Demo_V171.mf4")
var DataListTestCase = TestCase{
	file:   DataListFile,
	Sample: []any{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26, -16, -6, 4, 14, 24, 34, 44, 54, 64, 74, 84, 94, 104, 114, 124, -122, -112, -102, -92, -82, -72, -62, -52, -42, -32, -22, -12, -2, 8, 18, 28, 38, 48, 58, 68, 78, 88, 98, 108, 118, -128, -118, -108, -98, -88, -78, -68, -58, -48, -38, -28, -18, -8, 2, 12, 22, 32, 42, 52, 62, 72, 82, 92, 102, 112, 122, -124, -114, -104, -94, -84, -74, -64, -54, -44, -34, 0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26},
}

var HLCompressedDataListFile, _ = os.Open("./samples/sample_compressed.mf4")
var HLCompressedDataListTestCase = TestCase{
	file:   DataListFile,
	Sample: []any{},
}

func TestZip(t *testing.T) {
	testcase := ZipTestCase
	m, _ := mf4.ReadFile(testcase.file, &mf4.ReadOptions{})
	channel, err := m.GetChannelSample(0, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE")
	if err != nil {
		t.Error("Channel not loaded correctly.")
	}

	for index, value := range channel {
		if int(value.(int8)) != testcase.Sample[index].(int) {
			t.Errorf("Sample value %d diff %d from test file", value, testcase.Sample[index])
			t.FailNow()
		}
	}
}

func TestDataList(t *testing.T) {
	testcase := DataListTestCase
	m, _ := mf4.ReadFile(testcase.file, &mf4.ReadOptions{})

	channel, err := m.GetChannelSample(0, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE")
	if err != nil {
		t.Error("Channel not loaded correctly.")
	}

	for index, value := range channel {
		if int(value.(int8)) != testcase.Sample[index].(int) {
			t.Errorf("Sample value %d diff %d from test file", value, testcase.Sample[index])
			t.FailNow()
		}
	}
}
