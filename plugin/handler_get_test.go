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
