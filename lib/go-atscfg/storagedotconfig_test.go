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

func TestMakeStorageDotConfig(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.Profile = &profileName

	paramData := map[string]string{
		"Drive_Prefix":      "/dev/sd",
		"Drive_Letters":     "a,b,c,d,e",
		"RAM_Drive_Prefix":  "/dev/ra",
		"RAM_Drive_Letters": "f,g,h",
		"SSD_Drive_Prefix":  "/dev/ss",
		"SSD_Drive_Letters": "i,j,k",
	}

	params := makeParamsFromMap(*server.Profile, StorageFileName, paramData)

	/*
	   # DO NOT EDIT - Generated for myProfile by myToolName (https://myto.example.net) on Thu
	   Aug 8 08:58:54 MDT 2019
	           /dev/sda volume=1
	           /dev/sdb volume=1
	           /dev/sdc volume=1
	           /dev/sdd volume=1
	           /dev/sde volume=1
	           /dev/raf volume=2
	           /dev/rag volume=2
	           /dev/rah volume=2
	           /dev/ssi volume=3
	           /dev/ssj volume=3
	   	/dev/ssk volume=3
	*/

	cfg, err := MakeStorageDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Warnings) > 0 {
		t.Fatalf("expected no warnings, actual: %+v\n", cfg.Warnings)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 12 { // one line for each drive letter, plus the comment
		t.Errorf("expected one line for each drive letter plus a comment, actual: '"+txt+"' count %v", count)
	}

	if !strings.Contains(txt, paramData["Drive_Prefix"]) {
		t.Errorf("expected to contain Drive_Prefix '" + paramData["Drive_Prefix"] + "', actual: '" + txt + "'")
	}
	if !strings.Contains(txt, paramData["Ram_Drive_Prefix"]) {
		t.Errorf("expected to contain Ram_Drive_Prefix '" + paramData["Ram_Drive_Prefix"] + "', actual: '" + txt + "'")
	}
	if !strings.Contains(txt, paramData["SSD_Drive_Prefix"]) {
		t.Errorf("expected to contain SSD_Drive_Prefix '" + paramData["SSD_Drive_Prefix"] + "', actual: '" + txt + "'")
	}
	if !strings.Contains(txt, paramData["SSD_Drive_Prefix"]) {
		t.Errorf("expected to contain SSD_Drive_Prefix '" + paramData["SSD_Drive_Prefix"] + "', actual: '" + txt + "'")
	}
}

func TestMakeStorageDotConfigNoParams(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.Profile = &profileName

	paramData := map[string]string{}

	params := makeParamsFromMap(*server.Profile, StorageFileName, paramData)

	cfg, err := MakeStorageDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Warnings) > 0 {
		t.Fatalf("expected no warnings, actual: %+v\n", cfg.Warnings)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 2 { // comment plus a blank line
		t.Errorf("expected one line for comment, plus one blank line (it's important to send a blank line, to prevent many callers from returning a 404), actual: '"+txt+"' count %v", count)
	}

	lines := strings.Split(txt, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, actual: '"+txt+"' count %v", len(lines))
	}
	line := strings.TrimSpace(lines[1])
	if line != "" {
		t.Errorf("expected line after comment to be blank, actual: '" + txt + "'")
	}
}

func TestMakeStorageDotConfigNoDriveLetters(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.Profile = &profileName

	paramData := map[string]string{
		"Drive_Prefix":     "/dev/sd",
		"RAM_Drive_Prefix": "/dev/ra",
		"SSD_Drive_Prefix": "/dev/ss",
	}

	params := makeParamsFromMap(*server.Profile, StorageFileName, paramData)

	cfg, err := MakeStorageDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Warnings) != 3 {
		t.Fatalf("expected 3 warnings for drive letters of each type not existing, actual: %+v\n", cfg.Warnings)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 2 { // comment plus a blank line
		t.Errorf("expected one line for comment, plus one blank line (it's important to send a blank line, to prevent many callers from returning a 404), actual: '"+txt+"' count %v", count)
	}

	lines := strings.Split(txt, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, actual: '"+txt+"' count %v", len(lines))
	}
	line := strings.TrimSpace(lines[1])
	if line != "" {
		t.Errorf("expected line after comment to be blank, actual: '" + txt + "'")
	}
}

func TestMakeStorageDotConfigSomeDriveLetters(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.Profile = &profileName

	paramData := map[string]string{
		"Drive_Prefix":     "/dev/sd",
		"RAM_Drive_Prefix": "/dev/ra",
		"SSD_Drive_Prefix": "/dev/ss",
		"Drive_Letters":    "a,b,c,d,e",
	}

	params := makeParamsFromMap(*server.Profile, StorageFileName, paramData)

	cfg, err := MakeStorageDotConfig(server, params, hdr)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Warnings) != 2 {
		t.Fatalf("expected 2 warnings for 2 prefixes with no letters, actual: %+v\n", cfg.Warnings)
	}

	txt := cfg.Text

	testComment(t, txt, hdr)

	if count := strings.Count(txt, "\n"); count != 6 { // comment plus each letter
		t.Errorf("expected one line for comment, plus one line for each drive letter, actual: '"+txt+"' count %v", count)
	}
}
