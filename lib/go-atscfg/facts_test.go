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
)

func TestMake12MFacts(t *testing.T) {
	server := makeGenericServer()
	profileName := "myProfile"
	server.Profile = &profileName

	hdr := "myHeaderComment"

	cfg, err := Make12MFacts(server, &Config12MFactsOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	lines := strings.SplitN(txt, "\n", 2) // SplitN always returns at least 1 element, no need to check len before indexing
	if len(lines) < 2 {
		t.Fatalf("expected at least one line after the comment, found: 0")
	}
	afterCommentLines := lines[1]

	if !strings.Contains(afterCommentLines, profileName) {
		t.Errorf("expected profile name '"+profileName+"' in config, actual: '%v'", txt)
	}
}
