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
	"testing"
)

func Test_StringEnumValidator(t *testing.T) {
	v := StringEnumValidator{AllowedValues: []string{"value", "json"}}

	// Test valid values
	val1 := "value"
	if err := v.Validate(&val1); err != nil {
		t.Errorf("expected nil error for 'value', got: %v", err)
	}

	val2 := "json"
	if err := v.Validate(&val2); err != nil {
		t.Errorf("expected nil error for 'json', got: %v", err)
	}

	// Test nil (omitted parameter)
	if err := v.Validate(nil); err != nil {
		t.Errorf("expected nil error for nil, got: %v", err)
	}

	// Test invalid values
	for _, bad := range []string{"xml", "JSON", "Value", "csv", "invalid"} {
		vBad := bad
		if err := v.Validate(&vBad); err == nil {
			t.Errorf("expected error for %q, got nil", bad)
		}
	}
}

func Test_AgentURIValidator(t *testing.T) {
	v := AgentURIValidator{
		Defaults:       uriDefaults,
		AllowedSchemes: []string{"opc.tcp"},
	}

	// Valid URI
	uriValid := "opc.tcp://localhost:4840"
	if err := v.Validate(&uriValid); err != nil {
		t.Errorf("expected nil error for %q, got: %v", uriValid, err)
	}

	// Nil URI (omitted parameter)
	if err := v.Validate(nil); err != nil {
		t.Errorf("expected nil error for nil, got: %v", err)
	}

	// Disallowed scheme
	uriBadScheme := "http://localhost:4840"
	if err := v.Validate(&uriBadScheme); err == nil {
		t.Errorf("expected error for invalid scheme %q, got nil", uriBadScheme)
	}

	// Malformed URI
	uriMalformed := "://invalid"
	if err := v.Validate(&uriMalformed); err == nil {
		t.Errorf("expected error for malformed URI, got nil")
	}
}
