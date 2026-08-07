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

	"github.com/gopcua/opcua/ua"
)

const (
	pingError = 0
	pingOk    = 1
)

// pingHandler checks if we can connect to the OPC UA server and read the server status state.
func pingHandler(ctx context.Context,
	client clientLike,
	_ map[string]string, _ ...string) (interface{}, error) {

	nodeID := ua.NewNumericNodeID(0, 2259) // ServerStatus.State
	req := &ua.ReadRequest{MaxAge: 2000, NodesToRead: []*ua.ReadValueID{{NodeID: nodeID}}, TimestampsToReturn: ua.TimestampsToReturnBoth}
	resp, err := client.Read(ctx, req)
	if err != nil {
		return pingError, nil
	}
	if len(resp.Results) == 0 || resp.Results[0].Status != ua.StatusGood {
		return pingError, nil
	}
	return pingOk, nil
}
