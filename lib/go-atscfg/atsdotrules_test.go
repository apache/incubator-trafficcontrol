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

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestMakeATSDotRules(t *testing.T) {
	server := makeGenericServer()
	serverProfile := "myProfile"
	server.Profile = &serverProfile

	hdr := "myHeaderComment"

	serverParams := []tc.Parameter{
		{
			Name:       "Drive_Prefix",
			ConfigFile: ATSDotRulesFileName,
			Value:      "/dev/sd",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
		{
			Name:       "Drive_Letters",
			ConfigFile: ATSDotRulesFileName,
			Value:      "a,b,c,d,e",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
		{
			Name:       "RAM_Drive_Prefix",
			ConfigFile: ATSDotRulesFileName,
			Value:      "/dev/ra",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
		{
			Name:       "RAM_Drive_Letters",
			ConfigFile: ATSDotRulesFileName,
			Value:      "f,g,h",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
	}

	cfg, err := MakeATSDotRules(server, serverParams, &ATSDotRulesOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 9 { // one line for each drive letter, plus 1 comment
		t.Errorf("expected one line for each drive letter plus a comment, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "sda") {
		t.Errorf("expected sda for drive letter, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "rah") {
		t.Errorf("expected sda for drive letter, actual: '%v'", txt)
	}
}
