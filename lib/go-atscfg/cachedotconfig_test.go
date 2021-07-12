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

func TestMakeCacheDotConfig(t *testing.T) {
	server := makeGenericServer()
	serverProfile := "myProfile"
	server.Profile = &serverProfile
	servers := []Server{*server}

	ds0 := makeGenericDS()
	ds0.ID = util.IntPtr(420)
	ds0.XMLID = util.StrPtr("ds0")
	ds0.OrgServerFQDN = util.StrPtr("http://my.fqdn.example.net")
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type

	ds1 := makeGenericDS()
	ds1.ID = util.IntPtr(421)
	ds1.XMLID = util.StrPtr("ds1")
	ds1.OrgServerFQDN = util.StrPtr("http://my.other.fqdn.example.net")
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type

	ds2 := makeGenericDS()
	ds2.ID = util.IntPtr(422)
	ds2.XMLID = util.StrPtr("ds2")
	ds2.OrgServerFQDN = util.StrPtr("http://nocache-fqn.example.net")
	ds2Type := tc.DSTypeHTTPNoCache
	ds2.Type = &ds2Type

	dses := []DeliveryService{*ds0, *ds1, *ds2}

	dss := makeDSS(servers, dses)

	hdr := "myHeaderComment"

	cfg, err := MakeCacheDotConfig(server, servers, dses, dss, &CacheDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if strings.Contains(txt, "my.fqdn.example.net") {
		t.Errorf("expected cached DS type 'my.fqdn.example.net' omitted, actual: '%v'", txt)
	}
	if strings.Contains(txt, "my.other.fqdn.example.net") {
		t.Errorf("expected cached DS type 'my.fqdn.example.net' omitted, actual: '%v'", txt)
	}
	if strings.Contains(txt, "nocache-fqn-should-not-exist.example.net") {
		t.Errorf("expected config include NoCache DS origin 'nocache-fqn-should-not-exist.example.net', actual: '%v'", txt)
	}

}
