package atscfg

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestMakeURISigningConfig(t *testing.T) {
	fileName := "uri_signing_myds.config"
	keyBts := []byte("anything")
	keys := map[tc.DeliveryServiceName][]byte{
		"myds": keyBts,
	}

	cfg, err := MakeURISigningConfig(fileName, keys, nil)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	// URI Signing config is the verbatim bytes from Riak.
	if txt != string(keyBts) {
		t.Errorf("expected URI signing config to match input bytes, actual '%v'", txt)
	}
}

func TestGetDSFromURISigningConfigFileName(t *testing.T) {
	expecteds := map[string]string{
		"uri_signing_foo.config":                            "foo",
		"uri_signing_.config":                               "",
		"uri_signing.config":                                "",
		"uri_signing_foo.conf":                              "",
		"uri_signing_foo.confi":                             "",
		"uri_signing_foo_bar_baz.config":                    "foo_bar_baz",
		"uri_signing_uri_signing_foo_bar_baz.config.config": "uri_signing_foo_bar_baz.config",
	}

	for fileName, expected := range expecteds {
		actual := getDSFromURISigningConfigFileName(fileName)
		if expected != string(actual) {
			t.Errorf("GetDSFromURLSigConfigFileName('%v') expected '%v' actual '%v'\n", fileName, expected, actual)
		}
	}
}
