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
	"fmt"

	"github.com/gopcua/opcua/ua"
)

// getHandler reads a single OPC UA NodeID and returns its raw value.
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

	// Reuse helper from handler_info.go to extract the Go value.
	val := getValue(resp.Results[0])
	if val == nil {
		return nil, fmt.Errorf("no value returned for NodeID %s", nodeIDStr)
	}
	return val, nil
}
