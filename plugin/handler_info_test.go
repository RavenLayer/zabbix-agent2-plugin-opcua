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
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// mockInfoClient simulates OPC UA Read responses for infoHandler tests.
// It returns fixed DataValues corresponding to StartTime, CurrentTime, State,
// CurrentSessionCount, and CurrentSubscriptionCount (in that order).
type mockInfoClient struct {
	startTime     time.Time
	currentTime   time.Time
	stateCode     int32
	sessionCount  uint32
	subscriptions uint32
}

func (c *mockInfoClient) Node(id *ua.NodeID) *opcua.Node { return nil }

func (c *mockInfoClient) Read(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	results := []*ua.DataValue{
		{Status: ua.StatusGood, Value: ua.MustVariant(c.startTime)},
		{Status: ua.StatusGood, Value: ua.MustVariant(c.currentTime)},
		{Status: ua.StatusGood, Value: ua.MustVariant(c.stateCode)},
		{Status: ua.StatusGood, Value: ua.MustVariant(c.sessionCount)},
		{Status: ua.StatusGood, Value: ua.MustVariant(c.subscriptions)},
	}
	return &ua.ReadResponse{Results: results}, nil
}

func (c *mockInfoClient) Browse(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return &ua.BrowseResponse{}, nil
}

func Test_getServerStateString(t *testing.T) {
	tests := []struct {
		state int
		want  string
	}{
		{0, "Running"},
		{1, "Failed"},
		{2, "NoConfiguration"},
		{3, "Suspended"},
		{4, "Shutdown"},
		{5, "Test"},
		{6, "CommunicationFault"},
		{99, "Unknown"},
	}
	for _, tt := range tests {
		if got := getServerStateString(tt.state); got != tt.want {
			t.Errorf("getServerStateString(%d) = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func Test_infoHandler_payload(t *testing.T) {
	startTime := time.Date(2026, 9, 10, 7, 38, 8, 0, time.UTC)
	currentTime := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)

	client := &mockInfoClient{
		startTime:     startTime,
		currentTime:   currentTime,
		stateCode:     0,
		sessionCount:  5,
		subscriptions: 3,
	}

	// infoHandler also calls opcua.GetEndpoints for certificate info.
	// With no real server, that call will fail gracefully and certificate
	// fields will be omitted. We test the non-cert fields here.
	got, err := infoHandler(context.Background(), client, map[string]string{
		uriParam: "opc.tcp://localhost:4840",
	})
	if err != nil {
		t.Fatalf("infoHandler() error = %v", err)
	}

	str, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", got)
	}

	var payload InfoPayload
	if err := json.Unmarshal([]byte(str), &payload); err != nil {
		t.Fatalf("failed to unmarshal info JSON: %v, content: %s", err, str)
	}

	if payload.State != "Running" {
		t.Errorf("expected state 'Running', got %q", payload.State)
	}
	if payload.StateCode == nil || *payload.StateCode != 0 {
		t.Errorf("expected state_code 0, got %v", payload.StateCode)
	}
	if payload.StartTime != startTime.Format(time.RFC3339) {
		t.Errorf("expected start_time %q, got %q", startTime.Format(time.RFC3339), payload.StartTime)
	}
	if payload.CurrentTime != currentTime.Format(time.RFC3339) {
		t.Errorf("expected current_time %q, got %q", currentTime.Format(time.RFC3339), payload.CurrentTime)
	}
	if payload.CurrentSessionCount == nil || *payload.CurrentSessionCount != 5 {
		t.Errorf("expected current_session_count 5, got %v", payload.CurrentSessionCount)
	}
	if payload.CurrentSubscriptionCount == nil || *payload.CurrentSubscriptionCount != 3 {
		t.Errorf("expected current_subscription_count 3, got %v", payload.CurrentSubscriptionCount)
	}
}

func Test_infoHandler_failedState(t *testing.T) {
	client := &mockInfoClient{
		startTime:     time.Now().Add(-1 * time.Hour),
		currentTime:   time.Now(),
		stateCode:     1, // Failed
		sessionCount:  0,
		subscriptions: 0,
	}

	got, err := infoHandler(context.Background(), client, map[string]string{
		uriParam: "opc.tcp://localhost:4840",
	})
	if err != nil {
		t.Fatalf("infoHandler() error = %v", err)
	}

	var payload InfoPayload
	if err := json.Unmarshal([]byte(got.(string)), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.State != "Failed" {
		t.Errorf("expected state 'Failed', got %q", payload.State)
	}
	if payload.StateCode == nil || *payload.StateCode != 1 {
		t.Errorf("expected state_code 1, got %v", payload.StateCode)
	}
}
