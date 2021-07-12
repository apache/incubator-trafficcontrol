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

func TestMakeLogsXMLDotConfig(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := makeParamsFromMap("serverProfile", LogsXMLFileName, map[string]string{
		"LogFormat.Name":           "myFormatName",
		"LogFormat.Format":         "myFormat",
		"LogObject.Filename":       "myFilename",
		"LogObject.RollingEnabled": "myRollingEnabled",
		"LogFormat.Invalid":        "ShouldNotBeHere",
		"LogObject.Invalid":        "ShouldNotBeHere",
	})

	server := makeGenericServer()
	server.Profile = &profileName

	cfg, err := MakeLogsXMLDotConfig(server, paramData, &LogsXMLDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testXMLComment(t, txt, hdr)

	if !strings.Contains(txt, "myFormatName") {
		t.Errorf("expected config to contain LogFormat.Name 'myFormatName', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myFormat") {
		t.Errorf("expected config to contain LogFormat.Format 'myFormat', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myFilename") {
		t.Errorf("expected config to contain LogFormat.Filename 'myFilename', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myRollingEnabled") {
		t.Errorf("expected config to contain LogFormat.RollingEnabled 'myRollingEnabled', actual: '%v'", txt)
	}
	if strings.Contains(txt, "ShouldNotBeHere") {
		t.Errorf("expected config to omit unknown config 'ShouldNotBeHere', actual: '%v'", txt)
	}
}

func testXMLComment(t *testing.T, txt string, hdr string) {
	commentLine := strings.SplitN(txt, "\n", 2)[0] // SplitN always returns at least 1 element, no need to check len before indexing

	if !strings.HasPrefix(strings.TrimSpace(commentLine), "<!--") {
		t.Errorf("expected comment on first line, actual: '" + commentLine + "'")
	}
	if !strings.HasSuffix(strings.TrimSpace(commentLine), "-->") {
		t.Errorf("expected ending comment on first line, actual: '" + commentLine + "'")
	}
	if !strings.Contains(commentLine, hdr) {
		t.Errorf("expected header comment '" + hdr + "' in comment, actual: '" + commentLine + "'")
	}
}
