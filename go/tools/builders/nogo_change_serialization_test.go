package main

import (
	"os"
	"reflect"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	// Create a temporary file
	file, err := os.CreateTemp("", "tmp_file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	// Initialize a Change struct with some edits and analyzers
	change := *NewChange()
	change.AddEdit("AnalyzerA", "file1.txt", Edit{New: "replacement1", Start: 0, End: 5})
	change.AddEdit("AnalyzerA", "file1.txt", Edit{New: "replacement2", Start: 10, End: 15})
	change.AddEdit("AnalyzerB", "file2.txt", Edit{New: "new text", Start: 20, End: 25})

	// Test saving to file
	err = SaveToFile(file.Name(), change)
	if err != nil {
		t.Fatalf("Failed to save Change struct to file: %v", err)
	}

	// Test loading from file
	loadedChange, err := LoadFromFile(file.Name())
	if err != nil {
		t.Fatalf("Failed to load Change struct from file: %v", err)
	}

	// Compare original and loaded Change structs
	if !reflect.DeepEqual(change, loadedChange) {
		t.Fatalf("Loaded Change struct does not match original.\nOriginal: %+v\nLoaded: %+v", change, loadedChange)
	}
}
