/*
Copyright &copy; ZOZO, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package spreadsheet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/sheets/v4"
)

func TestFindSheet(t *testing.T) {
	existTitle := "exists"
	op := &spreadsheetOperator{
		spreadsheet: &sheets.Spreadsheet{
			Sheets: []*sheets.Sheet{
				&sheets.Sheet{
					Properties: &sheets.SheetProperties{
						Title: existTitle,
					},
				},
			},
		},
	}
	tests := []struct {
		name       string
		sheetTitle string
		expected   *sheets.Sheet
	}{
		{
			name:       "sheet title found",
			sheetTitle: existTitle,
			expected:   op.spreadsheet.Sheets[0],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundSheet, err := op.FindSheet(tt.sheetTitle)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, foundSheet)
		})
	}
}

func TestFindSheet_SheetNotExist(t *testing.T) {
	notExistTitle := "not exists"
	existTitle := "exists"
	op := &spreadsheetOperator{
		spreadsheet: &sheets.Spreadsheet{
			Sheets: []*sheets.Sheet{
				&sheets.Sheet{
					Properties: &sheets.SheetProperties{
						Title: existTitle,
					},
				},
			},
		},
	}
	tests := []struct {
		name       string
		sheetTitle string
		expected   error
	}{
		{
			name:       "sheet title not found",
			sheetTitle: notExistTitle,
			expected: &SheetNotFoundError{
				sheetName: notExistTitle,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := op.FindSheet(tt.sheetTitle)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestDoBatchUpdate_IncludeSpreadsheetInResponseFalse(t *testing.T) {
	op := &spreadsheetOperator{}
	tests := []struct {
		name     string
		req      *sheets.BatchUpdateSpreadsheetRequest
		expected error
	}{
		{
			name:     "IncludeSpreadsheetInResponse field value is false",
			req:      &sheets.BatchUpdateSpreadsheetRequest{},
			expected: fmt.Errorf("invalid input IncludeSpreadsheetInResponse is false"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := op.doBatchUpdate(tt.req)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestDoBatchUpdateFail(t *testing.T) {
	op := &spreadsheetOperator{}
	tests := []struct {
		name     string
		req      *sheets.BatchUpdateSpreadsheetRequest
		expected error
	}{
		{
			name:     "IncludeSpreadsheetInResponse field value is false",
			req:      &sheets.BatchUpdateSpreadsheetRequest{},
			expected: fmt.Errorf("invalid input IncludeSpreadsheetInResponse is false"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := op.doBatchUpdate(tt.req)
			assert.Equal(t, tt.expected, err)
		})
	}
}
