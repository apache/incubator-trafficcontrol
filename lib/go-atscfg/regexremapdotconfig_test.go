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

func TestMakeRegexRemapDotConfig(t *testing.T) {
	cdnName := "mycdn"
	hdr := "myHeaderComment"

	dsName := "myds"

	server := makeGenericServer()
	server.CDNName = &cdnName

	fileName := "regex_remap_myds.config"

	ds := makeGenericDS()
	ds.XMLID = &dsName
	ds.OrgServerFQDN = util.StrPtr("https://myorigin.example.net") // DS "origin_server_fqdn" is actually a URL including the scheme, the name is wrong.
	ds.RegexRemap = util.StrPtr("myregexremap")

	dses := []DeliveryService{*ds}

	cfg, err := MakeRegexRemapDotConfig(fileName, server, dses, &RegexRemapDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(txt, hdr) {
		t.Errorf("expected: header comment '" + hdr + "', actual: missing")
	}
	if !strings.HasPrefix(strings.TrimSpace(txt), "#") {
		t.Errorf("expected: header comment, actual: missing")
	}

	if strings.Contains(txt, "mycacheurl") {
		t.Errorf("expected: regex remap to not contain cacheurl, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin") {
		t.Errorf("expected: regex remap to not contain org server fqdn, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myregexremap") {
		t.Errorf("expected: regex remap to contain regex remap, actual: '%v'", txt)
	}
}

func TestMakeRegexRemapDotConfigUnusedDS(t *testing.T) {
	cdnName := "mycdn"
	hdr := "myHeaderComment"

	dsName := "myds"

	server := makeGenericServer()
	server.CDNName = &cdnName

	fileName := "regex_remap_myds.config"

	ds := makeGenericDS()
	ds.XMLID = &dsName
	ds.OrgServerFQDN = util.StrPtr("https://myorigin.example.net") // DS "origin_server_fqdn" is actually a URL including the scheme, the name is wrong.
	ds.QStringIgnore = util.IntPtr(0)
	ds.RegexRemap = util.StrPtr("myregexremap")

	ds1 := makeGenericDS()
	ds1.XMLID = util.StrPtr("otherds")
	ds1.OrgServerFQDN = util.StrPtr("https://otherorigin.example.net") // DS "origin_server_fqdn" is actually a URL including the scheme, the name is wrong.
	ds1.QStringIgnore = util.IntPtr(0)
	ds1.RegexRemap = util.StrPtr("otherregexremap")

	dses := []DeliveryService{*ds, *ds1}

	cfg, err := MakeRegexRemapDotConfig(fileName, server, dses, &RegexRemapDotConfigOpts{HdrComment: hdr})
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

	if strings.Contains(txt, "mycacheurl") {
		t.Errorf("expected: regex remap to not contain cacheurl, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin") {
		t.Errorf("expected: regex remap to not contain org server fqdn, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myregexremap") {
		t.Errorf("expected: regex remap to contain regex remap, actual: '%v'", txt)
	}

	if strings.Contains(txt, "othercacheurl") {
		t.Errorf("expected: regex remap to not contain other cacheurl, actual: '%v'", txt)
	}
	if strings.Contains(txt, "otherorigin") {
		t.Errorf("expected: regex remap to not contain other org server fqdn, actual: '%v'", txt)
	}
	if strings.Contains(txt, "otherregexremap") {
		t.Errorf("expected: regex remap to not contain other regex remap, actual: '%v'", txt)
	}
}

func TestMakeRegexRemapDotConfigReplaceReturns(t *testing.T) {
	cdnName := "mycdn"
	hdr := "myHeaderComment"

	dsName := "myds"

	server := makeGenericServer()
	server.CDNName = &cdnName

	fileName := "regex_remap_myds.config"

	ds := makeGenericDS()
	ds.XMLID = &dsName
	ds.OrgServerFQDN = util.StrPtr("https://myorigin.example.net") // DS "origin_server_fqdn" is actually a URL including the scheme, the name is wrong.
	ds.QStringIgnore = util.IntPtr(0)
	ds.RegexRemap = util.StrPtr("myregexremap__RETURN__mypostnewline")

	dses := []DeliveryService{*ds}

	cfg, err := MakeRegexRemapDotConfig(fileName, server, dses, &RegexRemapDotConfigOpts{HdrComment: hdr})
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

	if strings.Contains(txt, "mycacheurl") {
		t.Errorf("expected: regex remap to not contain cacheurl, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin") {
		t.Errorf("expected: regex remap to not contain org server fqdn, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myregexremap") {
		t.Errorf("expected: regex remap to contain regex remap, actual: '%v'", txt)
	}

	if strings.Contains(txt, "__RETURN__") {
		t.Errorf("expected: regex remap to replace __RETURN__, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myregexremap\nmypostnewline") {
		t.Errorf("expected: regex remap to replace __RETURN__ with newline, actual: '%v'", txt)
	}
}
