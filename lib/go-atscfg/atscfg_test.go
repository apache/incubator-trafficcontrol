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
	"github.com/apache/trafficcontrol/lib/go-util"
)

func TestGenericHeaderComment(t *testing.T) {
	commentTxt := "foo"
	txt := makeHdrComment(commentTxt)
	testComment(t, txt, commentTxt)
}

func testComment(t *testing.T, txt string, commentTxt string) {
	commentLine := strings.SplitN(txt, "\n", 2)[0] // SplitN always returns at least 1 element, no need to check len before indexing

	if !strings.HasPrefix(strings.TrimSpace(commentLine), "#") {
		t.Errorf("expected comment on first line, actual: '" + commentLine + "'")
	}
	if !strings.Contains(commentLine, commentTxt) {
		t.Errorf("expected comment text '" + commentTxt + "' in comment, actual: '" + commentLine + "'")
	}
}

func TestTrimParamUnderscoreNumSuffix(t *testing.T) {
	inputExpected := map[string]string{
		``:                         ``,
		`a`:                        `a`,
		`_`:                        `_`,
		`foo__`:                    `foo__`,
		`foo__1`:                   `foo`,
		`foo__1234567890`:          `foo`,
		`foo_1234567890`:           `foo_1234567890`,
		`foo__1234__1234567890`:    `foo__1234`,
		`foo__1234__1234567890_`:   `foo__1234__1234567890_`,
		`foo__1234__1234567890a`:   `foo__1234__1234567890a`,
		`foo__1234__1234567890__`:  `foo__1234__1234567890__`,
		`foo__1234__1234567890__a`: `foo__1234__1234567890__a`,
		`__`:                       `__`,
		`__9`:                      ``,
		`_9`:                       `_9`,
		`__35971234789124`:         ``,
		`a__35971234789124`:        `a`,
		`1234`:                     `1234`,
		`foo__asdf_1234`:           `foo__asdf_1234`,
	}

	for input, expected := range inputExpected {
		if actual := trimParamUnderscoreNumSuffix(input); expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
}

func TestGetATSMajorVersionFromATSVersion(t *testing.T) {
	inputExpected := map[string]int{
		`7.1.2-34.56abcde.el7.centos.x86_64`:    7,
		`8`:                                     8,
		`8.1`:                                   8,
		`10.1`:                                  10,
		`1234.1.2-34.56abcde.el7.centos.x86_64`: 1234,
	}
	errExpected := []string{
		"a7.1.2-34.56abcde.el7.centos.x86_64",
		`-7.1.2-34.56abcde.el7.centos.x86_64`,
		".7.1.2-34.56abcde.el7.centos.x86_64",
		"7a.1.2-34.56abcde.el7.centos.x86_64",
		"7-a.1.2-34.56abcde.el7.centos.x86_64",
		"7-2.1.2-34.56abcde.el7.centos.x86_64",
		"100-2.1.2-34.56abcde.el7.centos.x86_64",
		"7a",
		"",
		"-",
		".",
	}

	for input, expected := range inputExpected {
		if actual, err := GetATSMajorVersionFromATSVersion(input); err != nil {
			t.Errorf("expected %v actual: error '%v'", expected, err)
		} else if actual != expected {
			t.Errorf("expected %v actual: %v", expected, actual)
		}
	}
	for _, input := range errExpected {
		if actual, err := GetATSMajorVersionFromATSVersion(input); err == nil {
			t.Errorf("input %v expected: error, actual: nil error '%v'", input, actual)
		}
	}
}

func TestServerInfoIsTopLevelCache(t *testing.T) {
	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          "cgTypeUnknown",
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: "cgTypeUnknown",
		}
		if s.IsTopLevelCache() {
			t.Errorf("expected server with non-origin parent types, and non-InvalidID parent IDs to not be top level, actual top level")
		}
	}
	{
		s := &ServerInfo{
			ParentCacheGroupID:            -1,
			ParentCacheGroupType:          "cgTypeUnknown",
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: "cgTypeUnknown",
		}
		if s.IsTopLevelCache() {
			t.Errorf("expected server with secondary parent non-origin type and non-InvalidID to not be top level, actual top level")
		}
	}
	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          "cgTypeUnknown",
			SecondaryParentCacheGroupID:   -1,
			SecondaryParentCacheGroupType: "cgTypeUnknown",
		}
		if s.IsTopLevelCache() {
			t.Errorf("expected server with parent non-origin type and non-InvalidID to not be top level, actual top level")
		}
	}
	{
		s := &ServerInfo{
			ParentCacheGroupID:            -1,
			ParentCacheGroupType:          "cgTypeUnknown",
			SecondaryParentCacheGroupID:   -1,
			SecondaryParentCacheGroupType: "cgTypeUnknown",
		}
		if !s.IsTopLevelCache() {
			t.Errorf("expected server with parent and secondary parents with InvalidID IDs to be top level, actual not top level")
		}
	}

	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          tc.CacheGroupOriginTypeName,
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: tc.CacheGroupOriginTypeName,
		}
		if !s.IsTopLevelCache() {
			t.Errorf("expected server with parent and secondary parents with origin-type to be top level, actual not top level")
		}
	}

	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          "not origin",
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: tc.CacheGroupOriginTypeName,
		}
		if s.IsTopLevelCache() {
			t.Errorf("expected server with parent valid ID and origin-type to be top level, actual top level")
		}
	}

	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          tc.CacheGroupOriginTypeName,
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: "not origin",
		}
		if s.IsTopLevelCache() {
			t.Errorf("expected server with secondary parent valid ID and not origin-type to not be top level, actual top level")
		}
	}

	{
		s := &ServerInfo{
			ParentCacheGroupID:            1,
			ParentCacheGroupType:          tc.CacheGroupOriginTypeName,
			SecondaryParentCacheGroupID:   -1,
			SecondaryParentCacheGroupType: "not origin",
		}
		if !s.IsTopLevelCache() {
			t.Errorf("expected server with secondary parent invalid valid ID and parent origin type to be top level, actual not top level")
		}
	}

	{
		s := &ServerInfo{
			ParentCacheGroupID:            -1,
			ParentCacheGroupType:          "not origin",
			SecondaryParentCacheGroupID:   1,
			SecondaryParentCacheGroupType: tc.CacheGroupOriginTypeName,
		}
		if !s.IsTopLevelCache() {
			t.Errorf("expected server with parent invalid valid ID and secondary parent origin type to be top level, actual not top level")
		}
	}
}

func TestGetConfigFile(t *testing.T) {
	expected := "hdr_rw_my-xml-id.config"
	cfgFile := GetConfigFile(HeaderRewritePrefix, "my-xml-id")
	if cfgFile != expected {
		t.Errorf("Expected %s.   Got %s", expected, cfgFile)
	}
}

func setIP(sv *tc.ServerNullable, ipAddress string) {
	setIPInfo(sv, "", ipAddress, "")
}

func setIP6(sv *tc.ServerNullable, ip6Address string) {
	setIPInfo(sv, "", "", ip6Address)
}

func setIPInfo(sv *tc.ServerNullable, interfaceName string, ipAddress string, ip6Address string) {
	sv.Interfaces = []tc.ServerInterfaceInfo{
		tc.ServerInterfaceInfo{
			Name: interfaceName,
		},
	}
	if ipAddress != "" {
		sv.Interfaces[0].IPAddresses = append(sv.Interfaces[0].IPAddresses, tc.ServerIPAddress{
			Address:        ipAddress,
			Gateway:        nil,
			ServiceAddress: true,
		})
	}
	if ip6Address != "" {
		sv.Interfaces[0].IPAddresses = append(sv.Interfaces[0].IPAddresses, tc.ServerIPAddress{
			Address:        ip6Address,
			Gateway:        nil,
			ServiceAddress: true,
		})
	}
}

func makeGenericServer() *tc.ServerNullable {
	server := &tc.ServerNullable{}
	server.ProfileID = util.IntPtr(42)
	server.CDNName = util.StrPtr("myCDN")
	server.Cachegroup = util.StrPtr("cg0")
	server.CachegroupID = util.IntPtr(422)
	server.DomainName = util.StrPtr("mydomain.example.net")
	server.CDNID = util.IntPtr(43)
	server.HostName = util.StrPtr("myserver")
	server.HTTPSPort = util.IntPtr(12443)
	server.ID = util.IntPtr(44)
	setIP(server, "192.168.2.1")
	server.ProfileID = util.IntPtr(46)
	server.Profile = util.StrPtr("serverprofile")
	server.TCPPort = util.IntPtr(80)
	server.Type = "EDGE"
	server.TypeID = util.IntPtr(91)
	status := string(tc.CacheStatusReported)
	server.Status = &status
	server.StatusID = util.IntPtr(99)
	return server
}

func makeGenericDS() *tc.DeliveryServiceNullableV30 {
	ds := &tc.DeliveryServiceNullableV30{}
	ds.ID = util.IntPtr(42)
	ds.XMLID = util.StrPtr("ds1")
	ds.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	dsType := tc.DSTypeDNS
	ds.Type = &dsType
	ds.MultiSiteOrigin = util.BoolPtr(false)
	ds.Active = util.BoolPtr(true)
	return ds
}

// makeDSS creates DSS as an outer product of every server and ds given.
// The given servers and dses must all have non-nil, unique IDs.
func makeDSS(servers []tc.ServerNullable, dses []tc.DeliveryServiceNullableV30) []tc.DeliveryServiceServer {
	dss := []tc.DeliveryServiceServer{}
	for _, sv := range servers {
		for _, ds := range dses {
			dss = append(dss, tc.DeliveryServiceServer{
				Server:          util.IntPtr(*sv.ID),
				DeliveryService: util.IntPtr(*ds.ID),
			})
		}
	}
	return dss
}

func makeParamsFromMapArr(profile string, configFile string, paramM map[string][]string) []tc.Parameter {
	params := []tc.Parameter{}
	for name, vals := range paramM {
		for _, val := range vals {
			params = append(params, tc.Parameter{
				Name:       name,
				ConfigFile: configFile,
				Value:      val,
				Profiles:   []byte(`["` + profile + `"]`),
			})
		}
	}
	return params
}

func makeParamsFromMap(profile string, configFile string, paramM map[string]string) []tc.Parameter {
	params := []tc.Parameter{}
	for name, val := range paramM {
		params = append(params, tc.Parameter{
			Name:       name,
			ConfigFile: configFile,
			Value:      val,
			Profiles:   []byte(`["` + profile + `"]`),
		})
	}
	return params
}
