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
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// mockDiscoveryClient simulates Browse responses for discoveryHandler tests.
// It only returns browseResults when the requested nodeID matches the root
// Objects folder (ns=0;i=85). For any other node (i.e. children being
// recursed into) it returns empty results to avoid duplication.
type mockDiscoveryClient struct {
	browseResults []*ua.BrowseResult
}

func (c *mockDiscoveryClient) Node(id *ua.NodeID) *opcua.Node { return nil }
func (c *mockDiscoveryClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	return nil, nil
}
func (c *mockDiscoveryClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	if c.browseResults != nil && len(req.NodesToBrowse) > 0 {
		rootObjectsID := ua.NewNumericNodeID(0, 85)
		if req.NodesToBrowse[0].NodeID.String() == rootObjectsID.String() {
			return &ua.BrowseResponse{Results: c.browseResults}, nil
		}
	}
	return &ua.BrowseResponse{Results: []*ua.BrowseResult{}}, nil
}

func Test_discoveryHandler_emptyResult(t *testing.T) {
	client := &mockDiscoveryClient{}
	got, err := discoveryHandler(context.Background(), client, map[string]string{})
	if err != nil {
		t.Fatalf("discoveryHandler returned error: %v", err)
	}
	gotStr, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", got)
	}
	if gotStr != "[]" {
		t.Fatalf("expected empty JSON array, got %s", gotStr)
	}
	var items []LLDItem
	if err := json.Unmarshal([]byte(gotStr), &items); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected zero items, got %d", len(items))
	}
}

func Test_discoveryHandler_nonEmptyResult(t *testing.T) {
	refs := []*ua.ReferenceDescription{
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 1001)},
			DisplayName: &ua.LocalizedText{Text: "Temperature"},
			BrowseName:  &ua.QualifiedName{Name: "Temperature"},
			NodeClass:   ua.NodeClassVariable,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 1002)},
			DisplayName: &ua.LocalizedText{Text: "Pressure"},
			BrowseName:  &ua.QualifiedName{Name: "Pressure"},
			NodeClass:   ua.NodeClassVariable,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 2001)},
			DisplayName: &ua.LocalizedText{Text: "DeviceFolder"},
			BrowseName:  &ua.QualifiedName{Name: "DeviceFolder"},
			NodeClass:   ua.NodeClassObject,
		},
	}
	browseResults := []*ua.BrowseResult{
		{
			StatusCode: ua.StatusGood,
			References: refs,
		},
	}

	client := &mockDiscoveryClient{browseResults: browseResults}
	got, err := discoveryHandler(context.Background(), client, map[string]string{})
	if err != nil {
		t.Fatalf("discoveryHandler returned error: %v", err)
	}

	gotStr, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", got)
	}

	var items []LLDItem
	if err := json.Unmarshal([]byte(gotStr), &items); err != nil {
		t.Fatalf("unmarshal failed: %v, content: %s", err, gotStr)
	}

	// Default filter is "" which matches only NodeClassVariable.
	// So we should get 2 variable items (Temperature, Pressure).
	if len(items) != 2 {
		t.Fatalf("expected 2 items (variables only by default), got %d: %v", len(items), items)
	}
	if items[0].NodeName != "Temperature" {
		t.Errorf("expected first item name 'Temperature', got %q", items[0].NodeName)
	}
	if items[1].NodeName != "Pressure" {
		t.Errorf("expected second item name 'Pressure', got %q", items[1].NodeName)
	}

	// Verify JSON format matches Zabbix LLD expectations.
	var raw []map[string]string
	if err := json.Unmarshal([]byte(gotStr), &raw); err != nil {
		t.Fatalf("unmarshal raw failed: %v", err)
	}
	if _, ok := raw[0]["{#NODEID}"]; !ok {
		t.Error("expected {#NODEID} key in JSON output")
	}
	if _, ok := raw[0]["{#NODENAME}"]; !ok {
		t.Error("expected {#NODENAME} key in JSON output")
	}
}

func Test_discoveryHandler_nodeClassFilter(t *testing.T) {
	refs := []*ua.ReferenceDescription{
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 1001)},
			DisplayName: &ua.LocalizedText{Text: "Temperature"},
			BrowseName:  &ua.QualifiedName{Name: "Temperature"},
			NodeClass:   ua.NodeClassVariable,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 2001)},
			DisplayName: &ua.LocalizedText{Text: "DeviceFolder"},
			BrowseName:  &ua.QualifiedName{Name: "DeviceFolder"},
			NodeClass:   ua.NodeClassObject,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 3001)},
			DisplayName: &ua.LocalizedText{Text: "Reset"},
			BrowseName:  &ua.QualifiedName{Name: "Reset"},
			NodeClass:   ua.NodeClassMethod,
		},
	}
	browseResults := []*ua.BrowseResult{
		{
			StatusCode: ua.StatusGood,
			References: refs,
		},
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedName  string
	}{
		{"filter_object", "object", 1, "DeviceFolder"},
		{"filter_method", "method", 1, "Reset"},
		{"filter_variable", "variable", 1, "Temperature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockDiscoveryClient{browseResults: browseResults}
			params := map[string]string{"NodeClassFilter": tt.filter}
			got, err := discoveryHandler(context.Background(), client, params)
			if err != nil {
				t.Fatalf("discoveryHandler returned error: %v", err)
			}
			var items []LLDItem
			if err := json.Unmarshal([]byte(got.(string)), &items); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if len(items) != tt.expectedCount {
				t.Fatalf("expected %d items for filter %q, got %d: %v",
					tt.expectedCount, tt.filter, len(items), items)
			}
			if items[0].NodeName != tt.expectedName {
				t.Errorf("expected item name %q, got %q", tt.expectedName, items[0].NodeName)
			}
		})
	}
}

func Test_discoveryHandler_filterAll(t *testing.T) {
	refs := []*ua.ReferenceDescription{
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 1001)},
			DisplayName: &ua.LocalizedText{Text: "Temperature"},
			NodeClass:   ua.NodeClassVariable,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 2001)},
			DisplayName: &ua.LocalizedText{Text: "DeviceFolder"},
			NodeClass:   ua.NodeClassObject,
		},
		{
			NodeID:      &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(2, 3001)},
			DisplayName: &ua.LocalizedText{Text: "Reset"},
			NodeClass:   ua.NodeClassMethod,
		},
	}
	browseResults := []*ua.BrowseResult{
		{
			StatusCode: ua.StatusGood,
			References: refs,
		},
	}

	client := &mockDiscoveryClient{browseResults: browseResults}
	params := map[string]string{"NodeClassFilter": "all"}
	got, err := discoveryHandler(context.Background(), client, params)
	if err != nil {
		t.Fatalf("discoveryHandler returned error: %v", err)
	}

	var items []LLDItem
	if err := json.Unmarshal([]byte(got.(string)), &items); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// "all" matches Variable, Object, and Method.
	if len(items) != 3 {
		t.Fatalf("expected 3 items for filter 'all', got %d: %v", len(items), items)
	}
}
