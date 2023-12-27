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

// Package slack implements operator to notify message to slack.
package slack

import (
	"bytes"
	"fmt"
	"net/http"
)

type slackOperator struct {
	webhookURL string
}

// NewSlackOperator creates slackOperator with arguments webhookURL.
func NewSlackOperator(webhookURL string) *slackOperator {
	return &slackOperator{
		webhookURL: webhookURL,
	}
}

// Notify post http request to specified webhookURL.
func (op *slackOperator) Notify(msg string) error {
	payload := []byte(msg)
	res, err := http.Post(op.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to post message to slack webhook url\n")
	}
	defer res.Body.Close()
	return nil
}

/*
GenerateSlackPayloadData generates slack payload data with arguments mention and isSuccess.

The argument mention specifies the target of the mentions. The format of string is <@memberID>.
The argument isSuccess is condition which decide message color and its fields value.
*/
func GenerateSlackPayloadData(mention string, isSuccess bool) string {
	var color string
	var msg string

	if isSuccess {
		color = "good"
		msg = "loadtest execution succeeded"
	} else {
		color = "danger"
		msg = "loadtest execution failed, please check cli log"
	}

	data := fmt.Sprintf(`{
		"text": "%v",
		"attachments": [
			{
				"color": "%v",
				"text": "monitor loadtest status",
				"fields": [
					{
						"title": "loadtest result",
						"value": "%v",
						"short": false,
					}
				]
			}
		]
	}`, mention, color, msg)
	return data
}
