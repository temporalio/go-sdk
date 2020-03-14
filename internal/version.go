// Copyright (c) 2017 Uber Technologies, Inc.
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

package internal

// below are the metadata which will be embedded as part
// of headers in every rpc call made by this client to
// temporal server.

// Update to the metadata below is typically done
// by the temporal team as part of a major feature or
// behavior change

// SDKVersion is a semver that represents
// the version of this temporal client library.
// This represents API changes visible to Temporal
// client side library consumers, i.e. developers
// that are writing workflows. So every time we change API
// that can affect them we have to change this number.
// Format: MAJOR.MINOR.PATCH
const SDKVersion = "0.20.0"

// SDKFeatureVersion is a semver that represents the
// feature set of this temporal client library support.
// This can be used for client capability check, on
// Temporal server, for backward compatibility
// Format: MAJOR.MINOR.PATCH
const SDKFeatureVersion = "0.20.0"
