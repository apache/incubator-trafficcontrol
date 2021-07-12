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

func TestMakeServerUnknown(t *testing.T) {
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.HostName = util.StrPtr("server0")
	server.Profile = util.StrPtr("serverProfile")
	server.DomainName = util.StrPtr("example.test")

	fileName := "myconfig.config"

	params := makeParamsFromMapArr(*server.Profile, fileName, map[string][]string{
		"location":   []string{"locationshouldnotexist"},
		"param0name": []string{"param0val0", "param0val1"},
		"param1name": []string{"param1val0"},
		"header":     []string{"//hdr"},
	})

	cfg, err := MakeServerUnknown(fileName, server, params, &ServerUnknownOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if strings.Contains(txt, "#") {
		t.Errorf("expected: '%v' actual: '%v'", "no default header comment", txt)
	}

	if strings.Contains(txt, "param0name") {
		t.Errorf("expected: '%v' actual: '%v'", "no param name", txt)
	}

	if strings.Contains(txt, "param1name") {
		t.Errorf("expected: '%v' actual: '%v'", "no param name", txt)
	}

	if strings.Contains(txt, "location") {
		t.Errorf("expected: '%v' actual: '%v'", "no location param name or value", txt)
	}

	if strings.Contains(txt, "header") {
		t.Errorf("expected: '%v' actual: '%v'", "no header param name", txt)
	}

	if !strings.Contains(txt, "param0val0") {
		t.Errorf("expected: '%v' actual: '%v'", "param0val0", txt)
	}

	if !strings.Contains(txt, "param0val1") {
		t.Errorf("expected: '%v' actual: '%v'", "param0val1", txt)
	}

	if !strings.Contains(txt, "param1val0") {
		t.Errorf("expected: '%v' actual: '%v'", "param1val0", txt)
	}

	txt = strings.TrimSpace(txt)
	if !strings.HasPrefix(txt, "//hdr") {
		t.Errorf("expected: '%v' actual: '%v'", "header param prefix", txt)
	}

}
