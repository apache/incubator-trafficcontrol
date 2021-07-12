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

func TestMakeHostingDotConfig(t *testing.T) {
	cdnName := "cdn0"

	server := makeGenericServer()
	server.HostName = util.StrPtr("server0")
	server.CDNName = &cdnName
	server.ProfileID = util.IntPtr(46)
	server.Profile = util.StrPtr("serverprofile")
	hdr := "myHeaderComment"

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParamRAMDrivePrefix,
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "ParamRAMDrivePrefix-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
		tc.Parameter{
			Name:       ParamDrivePrefix,
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "ParamDrivePrefix-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
		tc.Parameter{
			Name:       "somethingelse",
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "somethingelse-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
	}

	origins := []string{
		"https://origin0.example.net",
		"http://origin1.example.net",
		"http://origin2.example.net/path0",
		"origin3.example.net/",
		"https://origin4.example.net/",
		"http://origin5.example.net/",
	}
	dses := []DeliveryService{}
	for _, origin := range origins {
		ds := makeGenericDS()
		ds.CDNName = &cdnName
		ds.OrgServerFQDN = util.StrPtr(origin)
		dses = append(dses, *ds)
	}

	servers := []Server{*server}
	dss := makeDSS(servers, dses)

	cfg, err := MakeHostingDotConfig(server, servers, serverParams, dses, dss, nil, &HostingDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	lines := strings.Split(txt, "\n")

	if len(lines) == 0 {
		t.Fatalf("expected: lines actual: no lines\n")
	}

	commentLine := lines[0]
	commentLine = strings.TrimSpace(commentLine)
	if !strings.HasPrefix(commentLine, "#") {
		t.Errorf("expected: comment line starting with '#', actual: '%v'\n", commentLine)
	}
	if !strings.Contains(commentLine, hdr) {
		t.Errorf("expected: comment line containing header comment '%v', actual: '%v'\n", hdr, commentLine)
	}

	lines = lines[1:] // remove comment line

	originFQDNs := getFQDNs(origins)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strArrContainsSubstr(originFQDNs, line) {
			t.Errorf("expected %+v actual '%v'\n", originFQDNs, line)
		}
		originFQDNs = strArrRemoveSubstr(originFQDNs, line)
	}

	if len(originFQDNs) > 0 {
		t.Errorf("expected %+v actual %v\n", originFQDNs, "missing")
	}
}

func TestMakeHostingDotConfigTopologiesIgnoreDSS(t *testing.T) {
	cdnName := "cdn0"

	server := makeGenericServer()
	server.HostName = util.StrPtr("server0")
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CDNName = &cdnName
	server.CDNID = util.IntPtr(400)
	server.ProfileID = util.IntPtr(46)
	server.ID = util.IntPtr(899)
	server.Profile = util.StrPtr("serverprofile")
	hdr := "myHeaderComment"

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParamRAMDrivePrefix,
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "ParamRAMDrivePrefix-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
		tc.Parameter{
			Name:       ParamDrivePrefix,
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "ParamDrivePrefix-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
		tc.Parameter{
			Name:       "somethingelse",
			ConfigFile: HostingConfigParamConfigFile,
			Value:      "somethingelse-shouldnotappearinconfig",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		},
	}

	dsTopology := makeGenericDS()
	dsTopology.OrgServerFQDN = util.StrPtr("https://origin0.example.net")
	dsTopology.XMLID = util.StrPtr("ds-topology")
	dsTopology.CDNID = util.IntPtr(400)
	dsTopology.ID = util.IntPtr(900)
	dsTopology.Topology = util.StrPtr("t0")
	dsTopology.Active = util.BoolPtr(true)
	dsType := tc.DSTypeHTTP
	dsTopology.Type = &dsType

	dsTopologyWithoutServer := makeGenericDS()
	dsTopologyWithoutServer.ID = util.IntPtr(901)
	dsTopologyWithoutServer.OrgServerFQDN = util.StrPtr("https://origin1.example.net")
	dsTopologyWithoutServer.XMLID = util.StrPtr("ds-topology-without-server")
	dsTopologyWithoutServer.CDNID = util.IntPtr(400)
	dsTopologyWithoutServer.Topology = util.StrPtr("t1")
	dsTopologyWithoutServer.Active = util.BoolPtr(true)
	dsType2 := tc.DSTypeHTTP
	dsTopologyWithoutServer.Type = &dsType2

	dses := []DeliveryService{*dsTopology, *dsTopologyWithoutServer}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "midCG",
				},
			},
		},
		tc.Topology{
			Name: "t1",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "otherEdgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "midCG",
				},
			},
		},
	}

	servers := []Server{*server}
	dss := makeDSS(servers, dses)

	cfg, err := MakeHostingDotConfig(server, servers, serverParams, dses, dss, topologies, &HostingDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(txt, "origin0") {
		t.Errorf("expected origin0 in topology, actual %v\n", txt)
	}
	if strings.Contains(txt, "origin1") {
		t.Errorf("expected no origin1 not in topology, actual %v\n", txt)
	}
}

func strArrContainsSubstr(arr []string, substr string) bool {
	for _, as := range arr {
		if strings.Contains(as, substr) {
			return true
		}
	}
	return false
}

func strArrRemoveSubstr(arr []string, substr string) []string {
	// this is terribly inefficient, but it's just for testing, so it doesn't matter
	newArr := []string{}
	for _, as := range arr {
		if strings.Contains(as, substr) {
			continue
		}
		newArr = append(newArr, as)
	}
	return newArr
}

func getFQDNs(origins []string) []string {
	newOrigins := []string{}
	for _, origin := range origins {
		origin = strings.TrimLeft(origin, "http://")
		origin = strings.TrimLeft(origin, "https://")
		origin = strings.TrimRight(origin, "/")
	}
	return newOrigins
}
