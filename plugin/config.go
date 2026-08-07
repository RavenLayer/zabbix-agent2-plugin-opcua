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
	"golang.zabbix.com/sdk/conf"
	"golang.zabbix.com/sdk/plugin"
)

// Session struct holds individual options for OPC UA connection for each session.
type Session struct {
	Uri            string `conf:"optional"`
	User           string `conf:"optional"`
	Password       string `conf:"optional"`
	SecurityMode   string `conf:"optional"`
	SecurityPolicy string `conf:"optional"`
	CertFile       string `conf:"optional"`
	KeyFile        string `conf:"optional"`
}

// PluginOptions are options for OPC UA connection.
type PluginOptions struct {
	plugin.SystemOptions `conf:"optional,name=System"`

	// Timeout is the maximum time in seconds for waiting when a connection has to be established.
	Timeout int `conf:"optional,range=1:30"`

	// Sessions stores pre-defined named sets of connections settings.
	Sessions map[string]Session `conf:"optional"`

	// Default stores default connection parameter values from configuration file.
	Default Session `conf:"optional"`
}

var DefaultOptions = PluginOptions{}

// Configure implements the Configurator interface.
func (p *Plugin) Configure(global *plugin.GlobalOptions, options interface{}) {
	p.options = DefaultOptions

	if err := conf.Unmarshal(options, &p.options); err != nil {
		p.Errf("cannot unmarshal configuration options: %s", err)
	}

	if p.options.Timeout == 0 {
		p.options.Timeout = global.Timeout
	}
}

// Validate implements the Configurator interface.
func (p *Plugin) Validate(options interface{}) error {
	var opts PluginOptions
	return conf.Unmarshal(options, &opts)
}
