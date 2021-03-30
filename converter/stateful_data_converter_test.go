// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package converter

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
)

type StatefulDataConverter struct {
	dataConverter DataConverter
	prefix        string
}

func (dc *StatefulDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	return dc.dataConverter.ToPayload(value)
}

func (dc *StatefulDataConverter) ToPayloads(values ...interface{}) (*commonpb.Payloads, error) {
	return dc.dataConverter.ToPayloads(values)
}

func (dc *StatefulDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	return dc.dataConverter.FromPayload(payload, valuePtr)
}

func (dc *StatefulDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	return dc.dataConverter.FromPayloads(payloads, valuePtrs...)
}

func (dc *StatefulDataConverter) ToString(payload *commonpb.Payload) string {
	if dc.prefix != "" {
		return dc.prefix + ": " + dc.dataConverter.ToString(payload)
	}

	return dc.dataConverter.ToString(payload)
}

func (dc *StatefulDataConverter) ToStrings(payloads *commonpb.Payloads) []string {
	var result []string
	for _, payload := range payloads.GetPayloads() {
		result = append(result, dc.ToString(payload))
	}

	return result
}

func (dc *StatefulDataConverter) WithValue(v interface{}) DataConverter {
	prefix, ok := v.(string)
	if !ok {
		return dc
	}

	return &StatefulDataConverter{
		dataConverter: dc.dataConverter,
		prefix:        prefix,
	}
}

func newStatefulDataConverter(dataConverter DataConverter) DataConverter {
	return &StatefulDataConverter{
		dataConverter: dataConverter,
	}
}

var statefulDataConverter = newStatefulDataConverter(defaultDataConverter)

func TestStatefulDataConverter(t *testing.T) {
	t.Parallel()
	t.Run("default", func(t *testing.T) {
		t.Parallel()
		payload, _ := statefulDataConverter.ToPayload("test")
		result := statefulDataConverter.ToString(payload)

		require.Equal(t, `"test"`, result)
	})
	t.Run("with state", func(t *testing.T) {
		t.Parallel()
		dc := WithValue(statefulDataConverter, "testing")

		payload, _ := dc.ToPayload("test")
		result := dc.ToString(payload)

		require.Equal(t, `testing: "test"`, result)
	})
}
