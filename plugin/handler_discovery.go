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
	"strings"

	"github.com/gopcua/opcua/ua"
)

type LLDItem struct {
	NodeID   string `json:"{#NODEID}"`
	NodeName string `json:"{#NODENAME}"`
}

func discoveryHandler(ctx context.Context,
	client clientLike,
	params map[string]string,
	_ ...string) (interface{}, error) {

	rootNodeIDStr := params["RootNodeID"]
	if rootNodeIDStr == "" {
		rootNodeIDStr = "ns=0;i=85"
	}
	nodeClassFilter := params["NodeClassFilter"]

	rootNodeID, err := ua.ParseNodeID(rootNodeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid root NodeID %s: %w", rootNodeIDStr, err)
	}

	var results []LLDItem
	visited := make(map[string]bool)
	maxDepth := 3 // Safe default depth limit

	browseRecursive(ctx, client, rootNodeID, nodeClassFilter, visited, 1, maxDepth, &results)

	if results == nil {
		results = []LLDItem{}
	}

	bz, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	return string(bz), nil
}

func browseRecursive(ctx context.Context, client clientLike, nodeID *ua.NodeID, filter string, visited map[string]bool, currentDepth int, maxDepth int, results *[]LLDItem) {
	nodeStr := nodeID.String()
	if visited[nodeStr] {
		return
	}
	visited[nodeStr] = true

	if currentDepth > maxDepth {
		return
	}

	req := &ua.BrowseRequest{
		NodesToBrowse: []*ua.BrowseDescription{
			{
				NodeID:          nodeID,
				BrowseDirection: ua.BrowseDirectionForward,
				ReferenceTypeID: ua.NewNumericNodeID(0, 33), // HierarchicalReferences (ns=0;i=33)
				IncludeSubtypes: true,
				NodeClassMask:   uint32(ua.NodeClassAll),
				ResultMask:      uint32(ua.BrowseResultMaskAll),
			},
		},
	}

	resp, err := client.Browse(ctx, req)
	if err != nil {
		return
	}

	for _, result := range resp.Results {
		if result.StatusCode != ua.StatusGood {
			continue
		}
		for _, ref := range result.References {
			if ref.NodeID == nil || ref.NodeID.NodeID == nil {
				continue
			}

			childNodeID := ref.NodeID.NodeID

			if matchNodeClass(ref.NodeClass, filter) {
				name := ""
				if ref.DisplayName != nil && ref.DisplayName.Text != "" {
					name = ref.DisplayName.Text
				} else if ref.BrowseName != nil {
					name = ref.BrowseName.Name
				}
				*results = append(*results, LLDItem{
					NodeID:   childNodeID.String(),
					NodeName: name,
				})
			}

			if ref.NodeClass == ua.NodeClassObject {
				browseRecursive(ctx, client, childNodeID, filter, visited, currentDepth+1, maxDepth, results)
			}
		}
	}
}

func matchNodeClass(class ua.NodeClass, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "all" {
		return class == ua.NodeClassVariable || class == ua.NodeClassObject || class == ua.NodeClassMethod
	}
	if filter == "" {
		return class == ua.NodeClassVariable
	}
	switch filter {
	case "variable":
		return class == ua.NodeClassVariable
	case "object":
		return class == ua.NodeClassObject
	case "method":
		return class == ua.NodeClassMethod
	default:
		return false
	}
}
