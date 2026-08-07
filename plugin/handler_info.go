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
	"crypto/x509"
	"encoding/json"
	"math"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type CertInfo struct {
	Subject             string `json:"subject,omitempty"`
	Issuer              string `json:"issuer,omitempty"`
	NotBefore           string `json:"not_before,omitempty"`
	NotAfter            string `json:"not_after,omitempty"`
	DaysUntilExpiration *int   `json:"days_until_expiration,omitempty"`
}

type InfoPayload struct {
	State                    string    `json:"state,omitempty"`
	StateCode                *int32    `json:"state_code,omitempty"`
	StartTime                string    `json:"start_time,omitempty"`
	CurrentTime              string    `json:"current_time,omitempty"`
	CurrentSessionCount      *uint32   `json:"current_session_count,omitempty"`
	CurrentSubscriptionCount *uint32   `json:"current_subscription_count,omitempty"`
	Certificate              *CertInfo `json:"certificate,omitempty"`
}

func infoHandler(ctx context.Context,
	client clientLike,
	params map[string]string,
	_ ...string) (interface{}, error) {

	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: ua.NewNumericNodeID(0, 2257)}, // StartTime
			{NodeID: ua.NewNumericNodeID(0, 2258)}, // CurrentTime
			{NodeID: ua.NewNumericNodeID(0, 2259)}, // State
			{NodeID: ua.NewNumericNodeID(0, 2277)}, // CurrentSessionCount
			{NodeID: ua.NewNumericNodeID(0, 2285)}, // CurrentSubscriptionCount
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := client.Read(ctx, req)
	if err != nil {
		return nil, err
	}

	payload := InfoPayload{}

	if len(resp.Results) > 0 {
		if startTime, ok := getValue(resp.Results[0]).(time.Time); ok {
			payload.StartTime = startTime.Format(time.RFC3339)
		}
	}
	if len(resp.Results) > 1 {
		if currentTime, ok := getValue(resp.Results[1]).(time.Time); ok {
			payload.CurrentTime = currentTime.Format(time.RFC3339)
		}
	}
	if len(resp.Results) > 2 {
		val := getValue(resp.Results[2])
		if stateCode, ok := asInt32(val); ok {
			payload.StateCode = &stateCode
			payload.State = getServerStateString(int(stateCode))
		}
	}
	if len(resp.Results) > 3 {
		if sessCount, ok := asUint32(getValue(resp.Results[3])); ok {
			payload.CurrentSessionCount = &sessCount
		}
	}
	if len(resp.Results) > 4 {
		if subCount, ok := asUint32(getValue(resp.Results[4])); ok {
			payload.CurrentSubscriptionCount = &subCount
		}
	}

	eps, err := opcua.GetEndpoints(ctx, params[uriParam])
	if err == nil && len(eps) > 0 && len(eps[0].ServerCertificate) > 0 {
		x509Cert, err := x509.ParseCertificate(eps[0].ServerCertificate)
		if err == nil {
			now := time.Now()
			days := int(math.Round(x509Cert.NotAfter.Sub(now).Hours() / 24))
			payload.Certificate = &CertInfo{
				Subject:             x509Cert.Subject.String(),
				Issuer:              x509Cert.Issuer.String(),
				NotBefore:           x509Cert.NotBefore.Format(time.RFC3339),
				NotAfter:            x509Cert.NotAfter.Format(time.RFC3339),
				DaysUntilExpiration: &days,
			}
		}
	}

	bz, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return string(bz), nil
}

func getValue(val *ua.DataValue) interface{} {
	if val == nil || val.Status != ua.StatusGood || val.Value == nil {
		return nil
	}
	return val.Value.Value()
}

func asInt32(v interface{}) (int32, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case int32:
		return val, true
	case uint32:
		return int32(val), true
	case int64:
		return int32(val), true
	case uint64:
		return int32(val), true
	case int:
		return int32(val), true
	default:
		return 0, false
	}
}

func asUint32(v interface{}) (uint32, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case uint32:
		return val, true
	case int32:
		return uint32(val), true
	case int64:
		return uint32(val), true
	case uint64:
		return uint32(val), true
	case int:
		return uint32(val), true
	default:
		return 0, false
	}
}

func getServerStateString(state int) string {
	switch state {
	case 0:
		return "Running"
	case 1:
		return "Failed"
	case 2:
		return "NoConfiguration"
	case 3:
		return "Suspended"
	case 4:
		return "Shutdown"
	case 5:
		return "Test"
	case 6:
		return "CommunicationFault"
	default:
		return "Unknown"
	}
}
