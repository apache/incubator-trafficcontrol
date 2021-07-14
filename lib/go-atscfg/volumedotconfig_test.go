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

func TestMakeVolumeDotConfig(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := map[string]string{
		"Drive_Prefix":      "/dev/sd",
		"Drive_Letters":     "a,b,c,d,e",
		"RAM_Drive_Prefix":  "/dev/ra",
		"RAM_Drive_Letters": "f,g,h",
		"SSD_Drive_Prefix":  "/dev/ss",
		"SSD_Drive_Letters": "i,j,k",
	}

	server := makeGenericServer()
	server.Profile = &profileName

	params := makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err := MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 5 { // one line for each volume, plus 2 comments
		t.Errorf("expected one line for each drive letter plus a comment, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "size=33%") {
		t.Errorf("expected size=33%% for three volumes, actual: '%v'", txt)
	}

	delete(paramData, "SSD_Drive_Prefix")
	delete(paramData, "SSD_Drive_Letters")
	params = makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err = MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt = cfg.Text

	if count := strings.Count(txt, "\n"); count != 4 { // one line for each volume, plus 2 comments
		t.Errorf("expected one line for each drive letter plus a comment, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "size=50%") {
		t.Errorf("expected size=50%% for two volumes, actual: '%v'", txt)
	}

	delete(paramData, "RAM_Drive_Prefix")
	delete(paramData, "RAM_Drive_Letters")
	params = makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err = MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt = cfg.Text

	if count := strings.Count(txt, "\n"); count != 3 { // one line for each volume, plus 2 comments
		t.Errorf("expected one line for each drive letter plus a comment, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "size=100%") {
		t.Errorf("expected size=100%% for one volume, actual: '%v'", txt)
	}
}

func TestMakeVolumeDotConfigNoParams(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := map[string]string{}

	server := makeGenericServer()
	server.Profile = &profileName

	params := makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err := MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 3 { // two comments, plus a blank line to prevent clients from thinking the file doesn't exist.
		t.Errorf("expected 2 comments and blank line, actual: '%v' count %v", txt, count)
	}

	lines := strings.Split(txt, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, 2 comments and blank line, actual: '%v' count %v", txt, len(lines))
	}
	line := strings.TrimSpace(lines[2])
	if line != "" {
		t.Fatalf("expected non-comment line to be blank, actual: '%v'", txt)
	}
}

func TestMakeVolumeDotConfigNoLetters(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := map[string]string{
		"Drive_Prefix":     "/dev/sd",
		"RAM_Drive_Prefix": "/dev/ra",
		"SSD_Drive_Prefix": "/dev/ss",
	}

	server := makeGenericServer()
	server.Profile = &profileName

	params := makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err := MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 5 { // one line for each volume, plus 2 comments
		// it's important we create volumes and sizes if the prefixes exist, even without letters,
		// because storage.config will still increment its volumes, even without letters.
		// If we didn't, the volume numbers wouldn't match.
		t.Errorf("expected one line for each drive plus 2 comments, even without letters, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "size=33%") {
		t.Errorf("expected size=33%% for three volumes even without letters, actual: '%v'", txt)
	}
}

func TestMakeVolumeDotConfigSomePrefixes(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"
	paramData := map[string]string{
		"Drive_Prefix":     "/dev/sd",
		"SSD_Drive_Prefix": "/dev/ss",
	}

	server := makeGenericServer()
	server.Profile = &profileName

	params := makeParamsFromMap(*server.Profile, VolumeFileName, paramData)

	cfg, err := MakeVolumeDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 4 { // one line for each volume, plus 2 comments
		t.Errorf("expected one line for each drive parameter plus 2 comment, actual: '%v' count %v", txt, count)
	}

	if !strings.Contains(txt, "size=50%") {
		t.Errorf("expected size=50%% for two volume parameters, actual: '%v'", txt)
	}
}
