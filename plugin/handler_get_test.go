//go:build tests
// +build tests

/*
 * Copyright (c) 2026 RavenLayer <sasha@ravenlayer.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type mockGetClient struct {
	err      error
	dataVals []*ua.DataValue
}

func (c *mockGetClient) Node(id *ua.NodeID) *opcua.Node { return nil }

func (c *mockGetClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &ua.ReadResponse{Results: c.dataVals}, nil
}

func (c *mockGetClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return &ua.BrowseResponse{}, nil
}

func Test_getHandler_success(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(int32(42))}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;i=42"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.(int32) != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
}

func Test_getHandler_missingNodeID(t *testing.T) {
	client := &mockGetClient{}
	_, err := getHandler(context.Background(), client, map[string]string{})
	if err == nil {
		t.Fatalf("expected error for missing NodeID, got nil")
	}
}

func Test_getHandler_readError(t *testing.T) {
	client := &mockGetClient{err: errors.New("read failure")}
	params := map[string]string{"NodeID": "ns=2;i=42"}
	_, err := getHandler(context.Background(), client, params)
	if err == nil {
		t.Fatalf("expected read error, got nil")
	}
}

func Test_getHandler_badStatus(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusBadNodeIDUnknown, Value: ua.MustVariant(int32(42))}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;i=999"}
	_, err := getHandler(context.Background(), client, params)
	if err == nil {
		t.Fatalf("expected error for bad status code, got nil")
	}
}

func Test_getHandler_complexType(t *testing.T) {
	serverStatus := map[string]interface{}{
		"State": int32(0),
		"BuildInfo": map[string]interface{}{
			"ProductName": "C++ SDK OPC UA Demo Server",
		},
	}
	ext := &ua.ExtensionObject{Value: serverStatus}
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(ext)}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=0;i=2256"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result for complex type, got %T (%v)", got, got)
	}

	// Verify valid JSON without outer EncodingMask/TypeID/Value wrapper
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v, content: %s", err, str)
	}
	if parsed["State"].(float64) != 0 {
		t.Fatalf("expected State 0 directly on root, got %v", parsed["State"])
	}
	if _, hasMask := parsed["EncodingMask"]; hasMask {
		t.Fatalf("EncodingMask should not be present in output, got: %s", str)
	}
	if _, hasTypeID := parsed["TypeID"]; hasTypeID {
		t.Fatalf("TypeID should not be present in output, got: %s", str)
	}
}

func Test_getHandler_complexTypeWithBuildInfo(t *testing.T) {
	serverStatus := map[string]interface{}{
		"State": int32(0),
		"BuildInfo": map[string]interface{}{
			"ProductName":      "Test Server",
			"ManufacturerName": "Test Corp",
			"SoftwareVersion":  "1.0.0",
			"BuildNumber":      "100",
		},
		"CurrentTime": "2026-09-12T10:00:00Z",
		"StartTime":   "2026-09-12T07:00:00Z",
	}
	ext := &ua.ExtensionObject{Value: serverStatus}
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(ext)}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=0;i=2256"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str := got.(string)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	buildInfo, ok := parsed["BuildInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected BuildInfo to be a nested object, got %T", parsed["BuildInfo"])
	}
	if buildInfo["ProductName"] != "Test Server" {
		t.Errorf("expected ProductName 'Test Server', got %v", buildInfo["ProductName"])
	}
	if buildInfo["ManufacturerName"] != "Test Corp" {
		t.Errorf("expected ManufacturerName 'Test Corp', got %v", buildInfo["ManufacturerName"])
	}
}

func Test_getHandler_sliceType(t *testing.T) {
	arr := []string{"foo", "bar", "baz"}
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(arr)}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;s=Array"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result for slice type, got %T", got)
	}

	var parsed []string
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v, content: %s", err, str)
	}
	if len(parsed) != 3 || parsed[1] != "bar" {
		t.Fatalf("unexpected parsed content: %v", parsed)
	}
}

func Test_getHandler_stringValue(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant("hello world")}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;s=MyString"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("expected 'hello world', got %v", got)
	}
}

func Test_getHandler_boolValue(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(true)}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;s=MyBool"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != true {
		t.Fatalf("expected true, got %v", got)
	}
}

func Test_getHandler_floatValue(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(float64(3.14))}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{"NodeID": "ns=2;s=MyFloat"}
	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(3.14) {
		t.Fatalf("expected 3.14, got %v", got)
	}
}
