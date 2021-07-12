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
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-util"
)

func TestMakeSetDSCPDotConfig(t *testing.T) {
	server := makeGenericServer()
	server.CDNName = util.StrPtr("mycdn")

	hdr := "myHeaderComment"
	fileName := "set_dscp_42.config"

	cfg, err := MakeSetDSCPDotConfig(fileName, server, &SetDSCPDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(txt, hdr) {
		t.Errorf("expected: header comment text '" + hdr + "', actual: missing")
	}
	if !strings.HasPrefix(strings.TrimSpace(txt), "#") {
		t.Errorf("expected: header comment, actual: missing")
	}

	if !strings.Contains(txt, "42") {
		t.Errorf("expected: dscp number '42' in config, actual '%v'", txt)
	}
}

func TestMakeSetDSCPDotConfigNonNumber(t *testing.T) {
	server := makeGenericServer()
	server.CDNName = util.StrPtr("mycdn")

	hdr := "myHeaderComment"
	fileName := "set_dscp_42a.config"

	cfg, err := MakeSetDSCPDotConfig(fileName, server, &SetDSCPDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(strings.ToLower(txt), "error") {
		t.Errorf("expected: error from non-number dscp, actual '%v'", txt)
	}
}
