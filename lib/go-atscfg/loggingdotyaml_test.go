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
	"fmt"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestMakeLoggingDotYAML(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.Profile = &profileName

	params := makeParamsFromMap("serverProfile", LoggingYAMLFileName, map[string]string{
		"LogFormat.Name":           "myFormatName",
		"LogFormat.Format":         "myFormat",
		"LogObject.Filename":       "myFilename",
		"LogObject.RollingEnabled": "myRollingEnabled",
		"LogFormat.Invalid":        "ShouldNotBeHere",
		"LogObject.Invalid":        "ShouldNotBeHere",
	})

	cfg, err := MakeLoggingDotYAML(server, params, &LoggingDotYAMLOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

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

func TestMakeLoggingDotYAMLMultiFormat(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := makeParamsFromMap("serverProfile", LoggingYAMLFileName, map[string]string{
		"LogFormat.Name":           "myFormatName0",
		"LogFormat.Format":         "myFormat0",
		"LogFormat1.Name":          "myFormatName1",
		"LogFormat1.Format":        "myFormat1",
		"LogFormat1.Filters":       "myFilter",
		"LogFormat9.Name":          "myFormatName9",
		"LogFormat9.Format":        "myFormat9",
		"LogFormat2.Name":          "myFormatName2",
		"LogFormat2.Format":        "myFormat2",
		"LogFormat11.Name":         "shouldNotBeHere11",
		"LogFormat11.Format":       "shouldNotBeHere11",
		"LogObject.Filename":       "myFilename0",
		"LogObject.Format":         "myFormatName0",
		"LogObject.RollingEnabled": "myRollingEnabled",
		"LogFormat.Invalid":        "ShouldNotBeHere",
		"LogObject.Invalid":        "ShouldNotBeHere",
		"LogObject2.Filename":      "myFilename2",
		"LogObject2.Format":        "myFormatName2",
		"LogObject11.Filename":     "shouldNotBeHere11",
		"LogObject11.Format":       "shouldNotBeHere11",
		"LogObject9.Filename":      "myFilename9",
		"LogObject9.Format":        "myFormatName9",
		"LogObject1.Filename":      "myFilename1",
		"LogObject1.Format":        "myFormatName1",
		"LogFilter.Name":           "myFilterName",
		"LogFilter.Filter":         "myFilter",
	})

	server := makeGenericServer()
	server.Profile = &profileName

	cfg, err := MakeLoggingDotYAML(server, paramData, &LoggingDotYAMLOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

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

	var v struct {
		Formats []struct {
			Name   string
			Format string
		}
		Logs []struct {
			Mode                 string
			Filename             string
			Format               string
			Rolling_enabled      string
			Rolling_interval_sec int
			Rolling_offset_hr    int
			Rolling_size_mb      int
		}
	}
	err = yaml.Unmarshal([]byte(txt), &v)
	if err != nil {
		t.Errorf("expected config to parse as yaml document '%v', actual: '%v'", err, txt)
	}
	if len(v.Formats) != 4 {
		t.Errorf("expected config to contain 4 'format' elements: '%v', actual: '%v'", v, txt)
		return
	}
	if len(v.Logs) != 4 {
		t.Errorf("expected config to contain 4 'logs' elements: '%v', actual: '%v'", v, txt)
		return
	}
	for i, n := range []int{0, 1, 2, 9} {
		if v.Formats[i].Name != fmt.Sprintf("myFormatName%d", n) {
			t.Errorf("expected config to contain formats.name 'myFormatName%d' in position %d, actual: '%v', full: '%v'", n, i, v.Formats[i].Name, txt)
		}
		if v.Formats[i].Format != fmt.Sprintf("myFormat%d", n) {
			t.Errorf("expected config to contain formats.format 'myFormat%d' in position %d, actual: '%v', full: '%v'", n, i, v.Formats[i].Format, txt)
		}
		if v.Logs[i].Format != fmt.Sprintf("myFormatName%d", n) {
			t.Errorf("expected config to contain logs.format 'myFormatName%d' in position %d, actual: '%v', full: '%v'", n, i, v.Logs[i].Format, txt)
		}
		if v.Logs[i].Filename != fmt.Sprintf("myFilename%d", n) {
			t.Errorf("expected config to contain logs.filename 'myFilename%d' in position %d, actual: '%v', full: '%v'", n, i, v.Logs[i].Filename, txt)
		}
		if v.Logs[i].Mode != "ascii" {
			t.Errorf("expected config to contain logs.mode 'ascii' in position %d, actual: '%v', full: '%v'", i, v.Logs[i].Mode, txt)
		}
	}
}
