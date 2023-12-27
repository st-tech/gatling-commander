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

package cloudstorages

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

// GoogleCloudStorageOperator implements exec.cloudStorageOperator interface.
type GoogleCloudStorageOperator struct {
	client *storage.Client
}

// NewGoogleCloudStorageOperator returns initialized GoogleCloudStorageOperator value.
func NewGoogleCloudStorageOperator(ctx context.Context) (*GoogleCloudStorageOperator, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return &GoogleCloudStorageOperator{}, fmt.Errorf("storage.NewClient: %w", err)
	}
	return &GoogleCloudStorageOperator{
		client: client,
	}, nil
}

// Fetch returns bytes of object in GCS.
func (op *GoogleCloudStorageOperator) Fetch(ctx context.Context, path string) ([]byte, error) {
	client := op.client
	defer client.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	bucket, object := parsePath(path)
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll from gcs object: %w", err)
	}
	return data, nil
}
