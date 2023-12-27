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

package slack

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSlackPayloadDataColor(t *testing.T) {
	cleaner := regexp.MustCompile(`[\n\t]`) // for remove tab and new line
	tests := []struct {
		name      string
		isSuccess bool
		expected  string
	}{
		{
			name:      "success",
			isSuccess: true,
			expected: `{
				"text": "",
				"attachments": [
					{
						"color": "good",
						"text": "monitor loadtest status",
						"fields": [
							{
								"title": "loadtest result",
								"value": "loadtest execution succeeded",
								"short": false,
							}
						]
					}
				]
			}`,
		},
		{
			name:      "failed",
			isSuccess: false,
			expected: `{
				"text": "",
				"attachments": [
					{
						"color": "danger",
						"text": "monitor loadtest status",
						"fields": [
							{
								"title": "loadtest result",
								"value": "loadtest execution failed, please check cli log",
								"short": false,
							}
						]
					}
				]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := GenerateSlackPayloadData("", tt.isSuccess)
			assert.Equal(t, cleaner.ReplaceAllString(tt.expected, ""), cleaner.ReplaceAllString(data, ""))
		})
	}
}

func TestGenerateSlackPayloadDataMention(t *testing.T) {
	cleaner := regexp.MustCompile(`[\n\t]`)
	tests := []struct {
		name     string
		mention  string
		expected string
	}{
		{
			name:    "specify mention",
			mention: "test",
			expected: `{
				"text": "test",
				"attachments": [
					{
						"color": "good",
						"text": "monitor loadtest status",
						"fields": [
							{
								"title": "loadtest result",
								"value": "loadtest execution succeeded",
								"short": false,
							}
						]
					}
				]
			}`,
		},
		{
			name:    "not specify mention",
			mention: "",
			expected: `{
				"text": "",
				"attachments": [
					{
						"color": "good",
						"text": "monitor loadtest status",
						"fields": [
							{
								"title": "loadtest result",
								"value": "loadtest execution succeeded",
								"short": false,
							}
						]
					}
				]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := GenerateSlackPayloadData(tt.mention, true)
			assert.Equal(t, cleaner.ReplaceAllString(tt.expected, ""), cleaner.ReplaceAllString(data, ""))
		})
	}
}
