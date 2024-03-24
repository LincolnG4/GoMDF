package mf4

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
)

func SampleFiles() []fs.DirEntry {
	items, _ := os.ReadDir("./samples")
	return items
}

func TestReadSample(t *testing.T) {
	file, err := os.Open("/home/lincolng/Documents/Code/GO/ASAMMDF/GoMDF/samples/sample4.mf4")
	if err != nil {
		t.Fatalf(`could not open file %v`, err)
	}

	// test read file
	m, err := ReadFile(file)
	if err != nil {
		t.Fatalf(`could not read file %v`, err)
	}

	//test read sample
	result, err := m.GetChannelSample(0, "channel_b")
	if err != nil {
		t.Fatalf(`could not read channel %v`, err)
	}

	expected := []int64{5, 10, 0, 10, 5, 10, 10, 5, 0, 0, 10, 5, 10, 5, 10, 0, 0, 5, 5, 0}
	// Check if the lengths of the two arrays are equal
	if len(result) != len(expected) {
		t.Errorf("Lengths mismatch. Expected: %d, Got: %d", len(expected), len(result))
		return
	}

	// Convert interface{} to int for comparison (using reflection)
	intValues := make([]int64, len(result))
	for i, v := range result {
		// Ensure each value is an int before conversion
		if intVal, ok := v.(int64); ok {
			intValues[i] = intVal
		} else {
			t.Errorf("Element %d in result is not an integer: %v", i, v)
			return
		}
	}

	// Compare the converted int slice with the expected slice
	if !reflect.DeepEqual(intValues, expected) {
		t.Errorf("Result %v does not match expected slice %v", intValues, expected)
	}
}
