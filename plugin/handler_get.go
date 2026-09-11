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
	"fmt"
	"time"

	"github.com/gopcua/opcua/ua"
)

// getHandler reads a single OPC UA NodeID and returns its raw or JSON-formatted value.
func getHandler(ctx context.Context, client clientLike, params map[string]string, _ ...string) (interface{}, error) {
	nodeIDStr, ok := params["NodeID"]
	if !ok || nodeIDStr == "" {
		return nil, fmt.Errorf("missing required parameter: NodeID")
	}

	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid NodeID %s: %w", nodeIDStr, err)
	}

	req := &ua.ReadRequest{
		MaxAge:             2000,
		NodesToRead:        []*ua.ReadValueID{{NodeID: nodeID}},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := client.Read(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no read results for NodeID %s", nodeIDStr)
	}

	result := resp.Results[0]
	if result.Status != ua.StatusGood {
		return nil, fmt.Errorf("read failed with status %v for NodeID %s", result.Status, nodeIDStr)
	}

	// Reuse helper from handler_info.go to extract the Go value.
	val := getValue(result)
	if val == nil {
		return nil, fmt.Errorf("no value returned for NodeID %s", nodeIDStr)
	}
	return formatValue(val), nil
}

// formatValue unwraps OPC UA containers (like ExtensionObject) and formats scalars as primitive types
// and complex structures as JSON strings.
func formatValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	val = unwrapValue(val)

	switch v := val.(type) {
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string:
		return v
	case time.Time:
		return v.Format(time.RFC3339)
	case *ua.LocalizedText:
		if v != nil {
			return v.Text
		}
		return ""
	case *ua.QualifiedName:
		if v != nil {
			return v.Name
		}
		return ""
	default:
		bz, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bz)
	}
}

// unwrapValue extracts the underlying data payload from an OPC UA ExtensionObject
// to avoid returning wire-level metadata like EncodingMask, TypeID, etc.
func unwrapValue(val interface{}) interface{} {
	switch v := val.(type) {
	case *ua.ExtensionObject:
		if v != nil && v.Value != nil {
			return unwrapValue(v.Value)
		}
	case ua.ExtensionObject:
		if v.Value != nil {
			return unwrapValue(v.Value)
		}
	case []*ua.ExtensionObject:
		unwrapped := make([]interface{}, len(v))
		for i, item := range v {
			unwrapped[i] = unwrapValue(item)
		}
		return unwrapped
	case []ua.ExtensionObject:
		unwrapped := make([]interface{}, len(v))
		for i, item := range v {
			unwrapped[i] = unwrapValue(item)
		}
		return unwrapped
	case map[string]interface{}:
		if inner, ok := v["Value"]; ok {
			if _, hasMask := v["EncodingMask"]; hasMask {
				return unwrapValue(inner)
			}
		}
	}
	return val
}
