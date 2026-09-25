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
	"time"

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

func Test_getHandler_json_good(t *testing.T) {
	srcTime := time.Date(2026, 9, 24, 9, 15, 0, 0, time.UTC)
	srvTime := time.Date(2026, 9, 24, 9, 15, 1, 0, time.UTC)

	dv := &ua.DataValue{
		Status:          ua.StatusGood,
		Value:           ua.MustVariant(float64(42.5)),
		SourceTimestamp: srcTime,
		ServerTimestamp: srvTime,
	}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{
		"NodeID":       "ns=2;s=Temp",
		"OutputFormat": "json",
	}

	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result for json format, got %T", got)
	}

	var payload GetPayload
	if err := json.Unmarshal([]byte(str), &payload); err != nil {
		t.Fatalf("failed to unmarshal json payload: %v, content: %s", err, str)
	}

	if payload.Value.(float64) != 42.5 {
		t.Errorf("expected value 42.5, got %v", payload.Value)
	}
	if payload.Status != "good" {
		t.Errorf("expected status 'good', got %q", payload.Status)
	}
	if payload.StatusCode != 0 {
		t.Errorf("expected status_code 0, got %v", payload.StatusCode)
	}
	if payload.Severity != "good" {
		t.Errorf("expected severity 'good', got %q", payload.Severity)
	}
	if payload.DataType != "double" {
		t.Errorf("expected data_type 'double', got %q", payload.DataType)
	}
	if payload.SourceTimestamp != srcTime.Unix() {
		t.Errorf("expected source_timestamp %d, got %d", srcTime.Unix(), payload.SourceTimestamp)
	}
	if payload.ServerTimestamp != srvTime.Unix() {
		t.Errorf("expected server_timestamp %d, got %d", srvTime.Unix(), payload.ServerTimestamp)
	}
}

func Test_getHandler_json_badStatus(t *testing.T) {
	dv := &ua.DataValue{
		Status: ua.StatusBadSensorFailure,
	}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{
		"NodeID":       "ns=2;s=BadSensor",
		"OutputFormat": "json",
	}

	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("expected no Go error for bad status in json format, got: %v", err)
	}

	var payload GetPayload
	if err := json.Unmarshal([]byte(got.(string)), &payload); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}

	if payload.Value != nil {
		t.Errorf("expected value nil for bad status, got %v", payload.Value)
	}
	if payload.Severity != "bad" {
		t.Errorf("expected severity 'bad', got %q", payload.Severity)
	}
	if payload.Status != "badsensorfailure" {
		t.Errorf("expected status 'badsensorfailure', got %q", payload.Status)
	}
	if payload.StatusCode != uint32(ua.StatusBadSensorFailure) {
		t.Errorf("expected status_code %d, got %d", uint32(ua.StatusBadSensorFailure), payload.StatusCode)
	}
}

func Test_getHandler_json_uncertainStatus(t *testing.T) {
	dv := &ua.DataValue{
		Status: ua.StatusUncertain,
	}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}
	params := map[string]string{
		"NodeID":       "ns=2;s=UncertainSensor",
		"OutputFormat": "json",
	}

	got, err := getHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload GetPayload
	if err := json.Unmarshal([]byte(got.(string)), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if payload.Severity != "uncertain" {
		t.Errorf("expected severity 'uncertain', got %q", payload.Severity)
	}
	if payload.Status != "uncertain" {
		t.Errorf("expected status 'uncertain', got %q", payload.Status)
	}
}

func Test_getHandler_format_default(t *testing.T) {
	dv := &ua.DataValue{Status: ua.StatusGood, Value: ua.MustVariant(int32(100))}
	client := &mockGetClient{dataVals: []*ua.DataValue{dv}}

	// Explicit "value"
	gotVal, err := getHandler(context.Background(), client, map[string]string{
		"NodeID":       "ns=2;i=1",
		"OutputFormat": "value",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotVal.(int32) != 100 {
		t.Errorf("expected 100, got %v", gotVal)
	}

	// Omitted / empty OutputFormat defaults to "value"
	gotDefault, err := getHandler(context.Background(), client, map[string]string{
		"NodeID": "ns=2;i=1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDefault.(int32) != 100 {
		t.Errorf("expected 100, got %v", gotDefault)
	}
}
