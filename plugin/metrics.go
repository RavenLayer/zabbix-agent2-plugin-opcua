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
	"strings"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/uri"
)

const (
	keyPing      = "opcua.ping"
	keyInfo      = "opcua.info"
	keyDiscovery = "opcua.discovery"
	keyGet       = "opcua.get"

	uriParam            = "Uri"
	userParam           = "User"
	passwordParam       = "Password"
	securityModeParam   = "SecurityMode"
	securityPolicyParam = "SecurityPolicy"
	certFileParam       = "CertFile"
	keyFileParam        = "KeyFile"
)

var uriDefaults = &uri.Defaults{Scheme: "opc.tcp", Port: "4840"}

var (
	maxPasswordLen = 512
)

var (
	// ConnParams
	paramURI = metric.NewConnParam(uriParam, "OPC UA endpoint URI or session name.").
			WithDefault(uriDefaults.Scheme + "://localhost:" + uriDefaults.Port).WithSession().
			WithValidator(AgentURIValidator{
			Defaults:       uriDefaults,
			AllowedSchemes: []string{"opc.tcp"},
		})
	paramUser     = metric.NewConnParam(userParam, "OPC UA username.").WithDefault("")
	paramPassword = metric.NewConnParam(passwordParam, "OPC UA password.").
			WithDefault("").
			WithValidator(metric.LenValidator{Max: &maxPasswordLen})

	// SessionOnlyParams
	paramSecurityMode   = metric.NewSessionOnlyParam(securityModeParam, "OPC UA security mode.").WithDefault("")
	paramSecurityPolicy = metric.NewSessionOnlyParam(securityPolicyParam, "OPC UA security policy.").WithDefault("")
	paramCertFile       = metric.NewSessionOnlyParam(certFileParam, "OPC UA client certificate file path.").WithDefault("")
	paramKeyFile        = metric.NewSessionOnlyParam(keyFileParam, "OPC UA client private key file path.").WithDefault("")
)

var metrics = metric.MetricSet{
	keyPing: metric.New(
		"Checks OPC UA server connectivity.",
		getCommonParams(), false,
	),
	keyInfo: metric.New(
		"Fetches OPC UA server diagnostic and certificate information.",
		getCommonParams(), false,
	),
	keyDiscovery: metric.New(
		"Recursively browses standard nodes to generate Zabbix LLD JSON.",
		getDiscoveryParams(), false,
	),
	keyGet: metric.New(
		"Reads a single NodeID's raw value.",
		getGetParams(), false,
	),
}

type AgentURIValidator struct {
	Defaults       *uri.Defaults
	AllowedSchemes []string
}

type clientLike interface {
	Node(*ua.NodeID) *opcua.Node
	Read(context.Context, *ua.ReadRequest) (*ua.ReadResponse, error)
	Browse(context.Context, *ua.BrowseRequest) (*ua.BrowseResponse, error)
}

type handlerFunc func(ctx context.Context,
	client clientLike,
	params map[string]string, _ ...string) (res interface{}, err error)

func init() {
	// Register with the internal Agent 2 manager
	err := plugin.RegisterMetrics(&Impl, Name, metrics.List()...)
	if err != nil {
		panic(err)
	}
}

// getHandlerFunc returns a handlerFunc related to a given key.
func getHandlerFunc(key string) handlerFunc {
	switch key {
	case keyPing:
		return pingHandler
	case keyInfo:
		return infoHandler
	case keyDiscovery:
		return discoveryHandler
	case keyGet:
		return getHandler
	default:
		return nil
	}
}

func (v AgentURIValidator) Validate(value *string) error {
	if value == nil {
		return nil
	}

	u, err := uri.New(*value, v.Defaults)
	if err != nil {
		return err
	}

	isValidScheme := false
	if v.AllowedSchemes != nil {
		for _, s := range v.AllowedSchemes {
			if u.Scheme() == s {
				isValidScheme = true
				break
			}
		}

		if !isValidScheme {
			return fmt.Errorf("allowed schemes: %s", strings.Join(v.AllowedSchemes, ", "))
		}
	}

	return nil
}

func getCommonParams() []*metric.Param {
	return []*metric.Param{
		paramURI,
		paramUser,
		paramPassword,
		paramSecurityMode,
		paramSecurityPolicy,
		paramCertFile,
		paramKeyFile,
	}
}

func getGetParams() []*metric.Param {
	return []*metric.Param{
		paramURI,
		paramUser,
		paramPassword,
		paramSecurityMode,
		paramSecurityPolicy,
		paramCertFile,
		paramKeyFile,
		metric.NewParam("NodeID", "OPC UA NodeID to read.").SetRequired(),
	}
}

func getDiscoveryParams() []*metric.Param {
	return []*metric.Param{
		paramURI,
		paramUser,
		paramPassword,
		paramSecurityMode,
		paramSecurityPolicy,
		paramCertFile,
		paramKeyFile,
		metric.NewParam("RootNodeID", "OPC UA root NodeID to start browsing.").WithDefault("ns=0;i=85"),
		metric.NewParam("NodeClassFilter", "OPC UA NodeClass filter (Variable, Object, etc.).").WithDefault(""),
	}
}
