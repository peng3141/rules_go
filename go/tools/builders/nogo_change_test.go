package main

import (
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// Mock helper to create a mock file in the token.FileSet
func mockFileSet(fileName string, size int) *token.FileSet {
	fset := token.NewFileSet()
	f := fset.AddFile(fileName, fset.Base(), size)
	for i := 0; i < size; i++ {
		f.AddLine(i)
	}
	return fset
}

// Mock analyzers for the test
var (
	analyzer1 = &analysis.Analyzer{Name: "analyzer1"}
	analyzer2 = &analysis.Analyzer{Name: "analyzer2"}
)

// TestAddEdit_MultipleAnalyzers tests AddEdit with multiple analyzers and files using reflect.DeepEqual
func TestAddEdit_MultipleAnalyzers(t *testing.T) {
	// Step 1: Setup
	change := NewChange()

	// Mock data for analyzer 1
	file1 := "file1.go"
	edit1a := Edit{Start: 10, End: 20, New: "code1 from analyzer1"}
	edit1b := Edit{Start: 30, End: 40, New: "code2 from analyzer1"}

	// Mock data for analyzer 2
	edit2a := Edit{Start: 50, End: 60, New: "code1 from analyzer2"}
	edit2b := Edit{Start: 70, End: 80, New: "code2 from analyzer2"}

	// Expected map after all edits are added
	expected := map[string]map[string][]Edit{
		analyzer1.Name: {
			file1: {edit1a, edit1b},
		},
		analyzer2.Name: {
			file1: {edit2a, edit2b},
		},
	}

	// Step 2: Action - Add edits for both analyzers
	change.AddEdit(analyzer1.Name, file1, edit1a)
	change.AddEdit(analyzer1.Name, file1, edit1b)
	change.AddEdit(analyzer2.Name, file1, edit2a)
	change.AddEdit(analyzer2.Name, file1, edit2b)

	// Step 3: Verify that the actual map matches the expected map using reflect.DeepEqual
	if !reflect.DeepEqual(change.AnalyzerToFileToEdits, expected) {
		t.Fatalf("Change.AnalyzerToFileToEdits did not match the expected result.\nGot: %+v\nExpected: %+v", change.AnalyzerToFileToEdits, expected)
	}
}

// Test case for valid, successful cases
func TestNewChangeFromDiagnostics_SuccessCases(t *testing.T) {
	cwd, _ := os.Getwd()
	file1path := filepath.Join(cwd, "file1.go")

	tests := []struct {
		name              string
		fileSet           *token.FileSet
		diagnosticEntries []DiagnosticEntry
		expectedEdits     map[string]map[string][]Edit
	}{
		{
			name:    "ValidEdits",
			fileSet: mockFileSet(file1path, 100),
			diagnosticEntries: []DiagnosticEntry{
				{
					Analyzer: analyzer1,
					Diagnostic: analysis.Diagnostic{
						SuggestedFixes: []analysis.SuggestedFix{
							{
								TextEdits: []analysis.TextEdit{
									{Pos: token.Pos(5), End: token.Pos(10), NewText: []byte("new_text")},
								},
							},
						},
					},
				},
				{
					Analyzer: analyzer1,
					Diagnostic: analysis.Diagnostic{
						SuggestedFixes: []analysis.SuggestedFix{
							{
								TextEdits: []analysis.TextEdit{
									{Pos: token.Pos(60), End: token.Pos(67), NewText: []byte("new_text")},
								},
							},
						},
					},
				},
			},
			expectedEdits: map[string]map[string][]Edit{
				"analyzer1": {
					"file1.go": {
						{New: "new_text", Start: 4, End: 9},   // offset is 0-based, while Pos is 1-based
						{New: "new_text", Start: 59, End: 66}, // offset is 0-based, while Pos is 1-based
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change, err := NewChangeFromDiagnostics(tt.diagnosticEntries, tt.fileSet)

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if !reflect.DeepEqual(change.AnalyzerToFileToEdits, tt.expectedEdits) {
				t.Fatalf("expected edits: %+v, got: %+v", tt.expectedEdits, change.AnalyzerToFileToEdits)
			}
		})
	}
}

// Test case for error cases
func TestNewChangeFromDiagnostics_ErrorCases(t *testing.T) {
	cwd, _ := os.Getwd()
	file1path := filepath.Join(cwd, "file1.go")

	tests := []struct {
		name              string
		fileSet           *token.FileSet
		diagnosticEntries []DiagnosticEntry
		expectedErr       string
	}{
		{
			name:    "InvalidPosEnd",
			fileSet: mockFileSet(file1path, 100),
			diagnosticEntries: []DiagnosticEntry{
				{
					Analyzer: analyzer1,
					Diagnostic: analysis.Diagnostic{
						SuggestedFixes: []analysis.SuggestedFix{
							{
								TextEdits: []analysis.TextEdit{
									{Pos: token.Pos(15), End: token.Pos(10), NewText: []byte("new_text")},
								},
							},
						},
					},
				},
			},
			expectedErr: "invalid fix: pos (15) > end (10)",
		},
		{
			name:    "EndBeyondFile",
			fileSet: mockFileSet(file1path, 100),
			diagnosticEntries: []DiagnosticEntry{
				{
					Analyzer: analyzer1,
					Diagnostic: analysis.Diagnostic{
						SuggestedFixes: []analysis.SuggestedFix{
							{
								TextEdits: []analysis.TextEdit{
									{Pos: token.Pos(50), End: token.Pos(102), NewText: []byte("new_text")},
								},
							},
						},
					},
				},
			},
			expectedErr: "invalid fix: end (102) past end of file (101)", // Pos=101 holds the extra EOF token, note Pos is 1-based
		},
		{
			name:    "MissingFileInfo",
			fileSet: token.NewFileSet(), // No files added
			diagnosticEntries: []DiagnosticEntry{
				{
					Analyzer: analyzer1,
					Diagnostic: analysis.Diagnostic{
						SuggestedFixes: []analysis.SuggestedFix{
							{
								TextEdits: []analysis.TextEdit{
									{Pos: token.Pos(5), End: token.Pos(10), NewText: []byte("new_text")},
								},
							},
						},
					},
				},
			},
			expectedErr: "invalid fix: missing file info for pos (5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewChangeFromDiagnostics(tt.diagnosticEntries, tt.fileSet)

			if err == nil {
				t.Fatalf("expected an error, got none")
			}

			if err.Error() != tt.expectedErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectedErr, err)
			}
		})
	}
}
