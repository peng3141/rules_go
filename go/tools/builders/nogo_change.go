package main


import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
)

// DiagnosticEntry represents a diagnostic entry with the corresponding analyzer.
type DiagnosticEntry struct {
	analysis.Diagnostic
	*analysis.Analyzer
}

// An Edit describes the replacement of a portion of a text file.
type Edit struct {
	New   string `json:"new"`   // the replacement
	Start int    `json:"start"` // starting byte offset of the region to replace
	End   int    `json:"end"`   // ending byte offset of the region to replace
}


// Change represents a set of edits to be applied to a set of files.
type Change struct {
	AnalyzerToFileToEdits map[string]map[string][]Edit `json:"analyzer_file_to_edits"`
}



// NewChange creates a new Change object.
func NewChange() *Change {
	return &Change{
		AnalyzerToFileToEdits: make(map[string]map[string][]Edit),
	}
}

// NewChangeFromDiagnostics builds a Change from a set of diagnostics.
// Unlike Diagnostic, Change is independent of the FileSet given it uses perf-file offsets instead of token.Pos.
// This allows Change to be used in contexts where the FileSet is not available, e.g., it remains applicable after it is saved to disk and loaded back.
// See https://github.com/golang/tools/blob/master/go/analysis/diagnostic.go for details.
func NewChangeFromDiagnostics(entries []DiagnosticEntry, fileSet *token.FileSet) (*Change, error) {
	c := NewChange()
	cwd, err := os.Getwd() // workspace root
	if err != nil {
		return c, fmt.Errorf("Error getting current working directory: (%v)", err)
	}
	for _, entry := range entries {
		analyzer := entry.Analyzer.Name
		for _, sf := range entry.Diagnostic.SuggestedFixes {
			for _, edit := range sf.TextEdits {
				file := fileSet.File(edit.Pos)

				if file == nil {
					return c, fmt.Errorf("invalid fix: missing file info for pos (%v)", edit.Pos)
				}
				if edit.Pos > edit.End {
					return c, fmt.Errorf("invalid fix: pos (%v) > end (%v)", edit.Pos, edit.End)
				}
				if eof := token.Pos(file.Base() + file.Size()); edit.End > eof {
					return c, fmt.Errorf("invalid fix: end (%v) past end of file (%v)", edit.End, eof)
				}
				edit := Edit{Start: file.Offset(edit.Pos), End: file.Offset(edit.End), New: string(edit.NewText)}
				fileRelativePath, err := filepath.Rel(cwd, file.Name())
				if err != nil {
					fileRelativePath = file.Name() // fallback logic
				}
				c.AddEdit(analyzer, fileRelativePath, edit)
			}
		}
	}
	return c, nil
}

// AddEdit adds an edit to the change.
func (c *Change) AddEdit(analyzer string, file string, edit Edit) {
	// Check if the analyzer exists in the map
	if _, ok := c.AnalyzerToFileToEdits[analyzer]; !ok {
		// Initialize the map for the analyzer if it doesn't exist
		c.AnalyzerToFileToEdits[analyzer] = make(map[string][]Edit)
	}

	// Append the edit to the list of edits for the specific file under the analyzer
	c.AnalyzerToFileToEdits[analyzer][file] = append(c.AnalyzerToFileToEdits[analyzer][file], edit)
}
