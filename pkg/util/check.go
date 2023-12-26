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

// Package util implements utility in gatling-commander.
package util

import (
	"fmt"
)

// CheckDuplicate checks target string list items are unique.
func CheckDuplicate(target []string) error {
	var duplicates []string
	unique := make(map[string]interface{})
	for _, name := range target {
		if _, exist := unique[name]; !exist {
			unique[name] = nil
			continue
		}
		duplicates = append(duplicates, name)
		if len(duplicates) > 0 {
			return fmt.Errorf("duplicated value found %v\n", duplicates)
		}
	}
	return nil
}

// CheckTimeout checks duration value is not exceeded timeout value.
func CheckTimeout(timeout int32, duration int32) error {
	if duration > timeout {
		return fmt.Errorf("timeout %v execeeded", timeout)
	}
	return nil
}
