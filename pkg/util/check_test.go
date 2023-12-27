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

package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckDuplicate(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected error
	}{
		{
			name:     "not duplicated",
			input:    []string{"hoge", "fuga"},
			expected: nil,
		},
		{
			name:     "duplicated",
			input:    []string{"hoge", "hoge"},
			expected: fmt.Errorf("duplicated value found %v\n", []string{"hoge"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDuplicate(tt.input)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestCheckTimeout(t *testing.T) {
	fromTime := int32(time.Now().Unix())
	time.Sleep(2 * time.Second)
	duration := int32(time.Now().Unix()) - fromTime
	tests := []struct {
		name     string
		timeout  int32
		expected error
	}{
		{
			name:     "duration exceeded timeout",
			timeout:  1,
			expected: fmt.Errorf("timeout %v execeeded", 1),
		},
		{
			name:     "duration equal timeout",
			timeout:  2,
			expected: nil,
		},
		{
			name:     "duration not exceeded timeout",
			timeout:  3,
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimeout(tt.timeout, duration)
			assert.Equal(t, tt.expected, err)
		})
	}
}
