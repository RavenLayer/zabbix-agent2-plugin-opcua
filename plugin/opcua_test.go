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
	"os"
	"testing"

	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/plugin"
)

func TestMain(m *testing.M) {
	var code int

	_ = log.Open(log.Console, log.Debug, "", 0)

	Impl.Init(Name)
	Impl.Configure(&plugin.GlobalOptions{Timeout: 30}, []byte(""))

	code = m.Run()
	if code != 0 {
		log.Critf("failed to run plugin tests")
		os.Exit(code)
	}
	os.Exit(code)
}
