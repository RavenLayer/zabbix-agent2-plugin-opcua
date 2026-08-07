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

type mockPingClient struct {
	err error
}

func (c *mockPingClient) Node(id *ua.NodeID) *opcua.Node { return nil }

func (c *mockPingClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	dv := &ua.DataValue{Status: ua.StatusGood, Value: &ua.Variant{}}
	return &ua.ReadResponse{Results: []*ua.DataValue{dv}}, nil
}

func (c *mockPingClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return &ua.BrowseResponse{}, nil
}

func Test_pingHandler_success(t *testing.T) {
	client := &mockPingClient{}
	got, err := pingHandler(context.Background(), client, nil)
	if err != nil {
		t.Fatalf("pingHandler() error = %v", err)
	}
	if got != pingOk {
		t.Fatalf("pingHandler() = %v, want %v", got, pingOk)
	}
}

func Test_pingHandler_readError(t *testing.T) {
	client := &mockPingClient{err: errors.New("connection refused")}
	got, err := pingHandler(context.Background(), client, nil)
	if err != nil {
		t.Fatalf("pingHandler() error = %v", err)
	}
	if got != pingError {
		t.Fatalf("pingHandler() = %v, want %v", got, pingError)
	}
}

func Test_pingHandler_badStatus(t *testing.T) {
	client := &mockPingBadStatusClient{}
	got, err := pingHandler(context.Background(), client, nil)
	if err != nil {
		t.Fatalf("pingHandler() error = %v", err)
	}
	if got != pingError {
		t.Fatalf("pingHandler() = %v, want %v", got, pingError)
	}
}

// mockPingBadStatusClient returns a DataValue with a bad status code.
type mockPingBadStatusClient struct{}

func (c *mockPingBadStatusClient) Node(id *ua.NodeID) *opcua.Node { return nil }
func (c *mockPingBadStatusClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	dv := &ua.DataValue{Status: ua.StatusBadNodeIDUnknown, Value: &ua.Variant{}}
	return &ua.ReadResponse{Results: []*ua.DataValue{dv}}, nil
}
func (c *mockPingBadStatusClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return &ua.BrowseResponse{}, nil
}

func Test_pingHandler_emptyResults(t *testing.T) {
	client := &mockPingEmptyClient{}
	got, err := pingHandler(context.Background(), client, nil)
	if err != nil {
		t.Fatalf("pingHandler() error = %v", err)
	}
	if got != pingError {
		t.Fatalf("pingHandler() = %v, want %v", got, pingError)
	}
}

// mockPingEmptyClient returns an empty results slice.
type mockPingEmptyClient struct{}

func (c *mockPingEmptyClient) Node(id *ua.NodeID) *opcua.Node { return nil }
func (c *mockPingEmptyClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	return &ua.ReadResponse{Results: []*ua.DataValue{}}, nil
}
func (c *mockPingEmptyClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return &ua.BrowseResponse{}, nil
}
