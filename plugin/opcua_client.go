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
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type OpcUaClient struct {
	Client *opcua.Client
	mu     sync.Mutex
}

type ConnManager struct {
	mu      sync.Mutex
	clients map[string]*OpcUaClient
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		clients: make(map[string]*OpcUaClient),
	}
}

func parseSecurityMode(mode string) ua.MessageSecurityMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "none":
		return ua.MessageSecurityModeNone
	case "sign":
		return ua.MessageSecurityModeSign
	case "signandencrypt":
		return ua.MessageSecurityModeSignAndEncrypt
	default:
		return ua.MessageSecurityModeNone
	}
}

func parseSecurityPolicy(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "none", "":
		return ua.SecurityPolicyURINone
	case "basic128rsa15":
		return ua.SecurityPolicyURIBasic128Rsa15
	case "basic256":
		return ua.SecurityPolicyURIBasic256
	case "basic256sha256":
		return ua.SecurityPolicyURIBasic256Sha256
	case "aes128_sha256_rsaoaep":
		return ua.SecurityPolicyURIAes128Sha256RsaOaep
	case "aes256_sha256_rsapss":
		return ua.SecurityPolicyURIAes256Sha256RsaPss
	default:
		if strings.HasPrefix(policy, "http://") {
			return policy
		}
		return ua.FormatSecurityPolicyURI(policy)
	}
}

func getTokenType(user string) ua.UserTokenType {
	if user != "" {
		return ua.UserTokenTypeUserName
	}
	return ua.UserTokenTypeAnonymous
}

func (c *ConnManager) GetClient(ctx context.Context, params map[string]string, timeout int, defaultSecMode, defaultSecPolicy string) (*opcua.Client, error) {
	uri := params[uriParam]
	user := params[userParam]
	password := params[passwordParam]
	secMode := params[securityModeParam]
	if secMode == "" {
		secMode = defaultSecMode
	}
	secPolicy := params[securityPolicyParam]
	if secPolicy == "" {
		secPolicy = defaultSecPolicy
	}
	certFile := params[certFileParam]
	keyFile := params[keyFileParam]

	clientKey := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", uri, user, password, secMode, secPolicy, certFile, keyFile)

	c.mu.Lock()
	cli, exists := c.clients[clientKey]
	if !exists {
		cli = &OpcUaClient{}
		c.clients[clientKey] = cli
	}
	c.mu.Unlock()

	cli.mu.Lock()
	defer cli.mu.Unlock()

	if cli.Client != nil {
		if cli.Client.State() == opcua.Connected {
			return cli.Client, nil
		}
		_ = cli.Client.Close(ctx)
		cli.Client = nil
	}

	var opts []opcua.Option
	opts = append(opts, opcua.RequestTimeout(time.Duration(timeout)*time.Second))
	opts = append(opts, opcua.AutoReconnect(true))

	if user != "" {
		opts = append(opts, opcua.AuthUsername(user, password))
	} else {
		opts = append(opts, opcua.AuthAnonymous())
	}

	mSecurityMode := parseSecurityMode(secMode)
	mSecurityPolicy := parseSecurityPolicy(secPolicy)

	endpoints, err := opcua.GetEndpoints(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints from %s: %w", uri, err)
	}

	ep := opcua.SelectEndpoint(endpoints, mSecurityPolicy, mSecurityMode)

	ep.EndpointURL = uri

	opts = append(opts, opcua.SecurityFromEndpoint(ep, getTokenType(user)))

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair from %s and %s: %w", certFile, keyFile, err)
		}
		pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key in %s is not an RSA private key", keyFile)
		}
		opts = append(opts, opcua.Certificate(cert.Certificate[0]), opcua.PrivateKey(pk))
	}

	client, err := opcua.NewClient(uri, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPC UA client for %s: %w", uri, err)
	}
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to OPC UA server at %s: %w", uri, err)
	}

	cli.Client = client
	return client, nil
}

// CloseAll disconnects all cached OPC UA client connections.
func (c *ConnManager) CloseAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for key, cli := range c.clients {
		cli.mu.Lock()
		if cli.Client != nil {
			_ = cli.Client.Close(ctx)
			cli.Client = nil
		}
		cli.mu.Unlock()
		delete(c.clients, key)
	}
}
