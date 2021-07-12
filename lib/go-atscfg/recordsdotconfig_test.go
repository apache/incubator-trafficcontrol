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

func TestMakeRecordsDotConfig(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	paramData := makeParamsFromMap("serverProfile", RecordsFileName, map[string]string{
		"param0":                    "val0",
		"param1":                    "val1",
		"param2":                    "val2",
		"test-hostname-replacement": "fooSTRING __HOSTNAME__",
	})

	server := makeTestRemapServer()
	server.Interfaces = nil
	ipStr := "192.163.2.99"
	ipCIDR := ipStr + "/30" // set the ip to a cidr, to make sure addr logic removes it
	setIP(server, ipCIDR)
	ip6Str := "2001:db8::9"
	ip6CIDR := ip6Str + "/48" // set the ip to a cidr, to make sure addr logic removes it
	setIP6(server, ip6CIDR)
	server.Profile = util.StrPtr(profileName)
	opt := &RecordsConfigOpts{}
	opt.DNSLocalBindServiceAddr = true
	opt.HdrComment = hdr
	cfg, err := MakeRecordsDotConfig(server, paramData, opt)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if !strings.Contains(txt, "param0 val0") {
		t.Errorf("expected config to contain paramData 'param0 val0', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "param1 val1") {
		t.Errorf("expected config to contain paramData 'param1 val1', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "param2 val2") {
		t.Errorf("expected config to contain paramData 'param2 val2', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "test-hostname-replacement fooSTRING __FULL_HOSTNAME__") {
		t.Errorf("expected config to replace 'STRING __HOSTNAME__' with 'STRING __FULL_HOSTNAME__', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "LOCAL proxy.local.outgoing_ip_to_bind STRING "+ipStr) {
		t.Errorf("expected config to contain outgoing_ip_to_bind from server, actual: '%v' warnings '%+v'", txt, cfg.Warnings)
	}
	if !strings.Contains(txt, "CONFIG proxy.config.dns.local_ipv4 STRING "+ipStr) {
		t.Errorf("expected config to contain dns.local_ipv4 from server, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "CONFIG proxy.config.dns.local_ipv6 STRING ["+ip6Str+"]") {
		t.Errorf("expected config to contain dns.local_ipv6 from server, actual: '%v'", txt)
	}
}

func TestReplaceLineSuffixes(t *testing.T) {
	{
		input := `
foo STRING __HOSTNAME__
bar
baz
`
		expected := `
foo STRING __FULL_HOSTNAME__
bar
baz
`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := `STRING __HOSTNAME__`
		expected := `STRING __FULL_HOSTNAME__`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := `
STRING __HOSTNAME__
`
		expected := `
STRING __FULL_HOSTNAME__
`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := `
  
STRING __HOSTNAME__
`
		expected := `
  
STRING __FULL_HOSTNAME__
`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := `
STRING __HOSTNAME__
  STRING __HOSTNAME__
`
		expected := `
STRING __FULL_HOSTNAME__
  STRING __FULL_HOSTNAME__
`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := `
`
		expected := `
`
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
	{
		input := ``
		expected := ``
		actual := replaceLineSuffixes(input, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
		if expected != actual {
			t.Errorf("Expected '%v' Actual '%v'", expected, actual)
		}
	}
}
