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

func TestMakeParentDotConfig(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.HostName = util.StrPtr("mymid0")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{}
	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigCapabilities(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	dses := []DeliveryService{*ds0}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()

	mid0 := makeTestParentServer()
	mid0.HostName = util.StrPtr("my-parent-nocaps")
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.HostName = util.StrPtr("mymid0")
	mid0.ID = util.IntPtr(45)
	mid0.CachegroupID = util.IntPtr(423)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.HostName = util.StrPtr("my-parent-fooonly")
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.ID = util.IntPtr(46)
	mid1.CachegroupID = util.IntPtr(423)
	setIP(mid1, "192.168.2.3")

	mid2 := makeTestParentServer()
	mid2.HostName = util.StrPtr("my-parent-foobar")
	mid2.Cachegroup = util.StrPtr("midCG")
	mid2.ID = util.IntPtr(47)
	mid2.CachegroupID = util.IntPtr(423)
	setIP(mid1, "192.168.2.4")

	servers := []Server{*server, *mid0, *mid1, *mid2}

	topologies := []tc.Topology{}
	serverCapabilities := map[int]map[ServerCapability]struct{}{
		*mid1.ID: map[ServerCapability]struct{}{"FOO": {}},
		*mid2.ID: map[ServerCapability]struct{}{"FOO": {}, "BAR": {}},
	}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{
		*ds0.ID: map[ServerCapability]struct{}{"FOO": {}},
	}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	lines := strings.Split(txt, "\n")

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (comment, ds, dot remap, and empty newline), actual: '%+v' text %v", len(lines), txt)
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // skip empty newline
		}
		if strings.HasPrefix(line, `dest_domain=.`) {
			continue // skip dot remap, which has all parents irrespective of capability
		}
		if strings.HasPrefix(line, `#`) {
			continue // skip comment
		}

		if !strings.Contains(line, "dest_domain=ds0.example.net") {
			t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", line)
		}
		if !strings.Contains(line, "foobar") {
			t.Errorf("expected parent with all capabilities, actual: '%v'", line)
		}
		if !strings.Contains(line, "fooonly") {
			t.Errorf("expected parent with required capabilities, actual: '%v'", line)
		}
		if strings.Contains(line, "nocaps") {
			t.Errorf("expected not to contain parent with no capabilities, actual line: '%v'", line)
		}
	}
}

func TestMakeParentDotConfigMSOSecondaryParent(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.MultiSiteOrigin = util.BoolPtr(true)
	dses := []DeliveryService{*ds0}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG0")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("my-parent-0")
	mid0.DomainName = util.StrPtr("my-parent-0-domain")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG1")
	mid1.CachegroupID = util.IntPtr(501)
	mid1.HostName = util.StrPtr("my-parent-1")
	mid1.DomainName = util.StrPtr("my-parent-1-domain")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{}
	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCG.SecondaryParentName = mid1.Cachegroup
	eCG.SecondaryParentCachegroupID = mid1.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	mCG1 := &tc.CacheGroupNullable{}
	mCG1.Name = mid1.Cachegroup
	mCG1.ID = mid1.CachegroupID
	mCGType1 := tc.CacheGroupMidTypeName
	mCG1.Type = &mCGType1

	cgs := []tc.CacheGroupNullable{*eCG, *mCG, *mCG1}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	txtx := strings.Replace(txt, " ", "", -1)

	if !strings.Contains(txtx, `secondary_parent="my-parent-1.my-parent-1-domain`) {
		t.Errorf("expected secondary parent 'my-parent-1.my-parent-1-domain', actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologies(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.CachegroupID = util.IntPtr(500)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

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
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if strings.Contains(txt, "# topology") {
		// ATS doesn't support inline comments in parent.config
		t.Errorf("expected: no inline '# topology' comment, actual: '%v'", txt)
	}
}

// TestMakeParentDotConfigNotInTopologies tests when a given edge is NOT in a Topology, that it doesn't add a remap line.
func TestMakeParentDotConfigNotInTopologies(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.Topology = util.StrPtr("t0")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.HostName = util.StrPtr("mymid0")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
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

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net' to NOT contain Topology DS without this edge: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesCapabilities(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0.ID = util.IntPtr(42)
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.Topology = util.StrPtr("t0")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")

	ds2 := makeParentDS()
	ds2.ID = util.IntPtr(44)
	ds2Type := tc.DSTypeDNS
	ds2.Type = &ds2Type
	ds2.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds2.OrgServerFQDN = util.StrPtr("http://ds2.example.net")
	ds2.Topology = util.StrPtr("t0")

	dses := []DeliveryService{*ds0, *ds1, *ds2}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid0")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.CachegroupID = util.IntPtr(500)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

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
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{
		44: map[ServerCapability]struct{}{"FOO": {}},
		45: map[ServerCapability]struct{}{"FOO": {}},
		46: map[ServerCapability]struct{}{"FOO": {}},
	}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{
		*ds1.ID: map[ServerCapability]struct{}{"FOO": {}},
		*ds2.ID: map[ServerCapability]struct{}{"BAR": {}},
	}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds2.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net' without required capabilities: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net' with necessary required capabilities, actual: '%v'", txt)
	}
	if strings.Contains(txt, "dest_domain=ds2.example.net") {
		t.Errorf("expected no parent 'dest_domain=ds2.example.net' without necessary required capabilities, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesOmitOfflineParents(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid-should-omit")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")
	statusOffline := string(tc.CacheStatusOffline)
	mid0.Status = &statusOffline

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.CachegroupID = util.IntPtr(500)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

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
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}

	if strings.Contains(txt, "should-omit") {
		t.Errorf("Topology expected to omit OFFLINE mid, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesOmitDifferentCDNParents(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid-should-omit")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")
	mid0.CDNName = util.StrPtr("myCDN-different-than-edge")
	mid0CDNID := *server.CDNID + 1
	mid0.CDNID = &mid0CDNID

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.CachegroupID = util.IntPtr(500)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

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
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}

	if strings.Contains(txt, "should-omit") {
		t.Errorf("Topology expected to omit parent with a different CDN, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesMSO(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.MultiSiteOrigin = util.BoolPtr(true)

	dses := []DeliveryService{*ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	origin0 := makeTestParentServer()
	origin0.Cachegroup = util.StrPtr("originCG")
	origin0.CachegroupID = util.IntPtr(500)
	origin0.HostName = util.StrPtr("myorigin0")
	origin0.ID = util.IntPtr(45)
	setIP(origin0, "192.168.2.2")
	origin0.Type = tc.OriginTypeName
	origin0.TypeID = util.IntPtr(991)

	origin1 := makeTestParentServer()
	origin1.Cachegroup = util.StrPtr("originCG")
	origin1.CachegroupID = util.IntPtr(500)
	origin1.HostName = util.StrPtr("myorigin1")
	origin1.ID = util.IntPtr(46)
	setIP(origin1, "192.168.2.3")
	origin1.Type = tc.OriginTypeName
	origin1.TypeID = util.IntPtr(991)

	servers := []Server{*server, *origin0, *origin1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "originCG",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = origin0.Cachegroup
	eCG.ParentCachegroupID = origin0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	oCG := &tc.CacheGroupNullable{}
	oCG.Name = origin0.Cachegroup
	oCG.ID = origin0.CachegroupID
	oCGType := tc.CacheGroupOriginTypeName
	oCG.Type = &oCGType

	cgs := []tc.CacheGroupNullable{*eCG, *oCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *origin0.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myorigin0") {
		t.Errorf("expected origin0 with DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin1") {
		t.Errorf("expected no origin1 without DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesMSOWithCapabilities(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.MultiSiteOrigin = util.BoolPtr(true)

	dses := []DeliveryService{*ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)
	server.ID = util.IntPtr(44)

	origin0 := makeTestParentServer()
	origin0.Cachegroup = util.StrPtr("originCG")
	origin0.CachegroupID = util.IntPtr(500)
	origin0.HostName = util.StrPtr("myorigin0")
	origin0.ID = util.IntPtr(45)
	setIP(origin0, "192.168.2.2")
	origin0.Type = tc.OriginTypeName
	origin0.TypeID = util.IntPtr(991)

	origin1 := makeTestParentServer()
	origin1.Cachegroup = util.StrPtr("originCG")
	origin1.CachegroupID = util.IntPtr(500)
	origin1.HostName = util.StrPtr("myorigin1")
	origin1.ID = util.IntPtr(46)
	setIP(origin1, "192.168.2.3")
	origin1.Type = tc.OriginTypeName
	origin1.TypeID = util.IntPtr(991)

	servers := []Server{*server, *origin0, *origin1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "originCG",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{
		*server.ID: {
			ServerCapability("FOO"): struct{}{},
		},
	}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{
		*ds1.ID: {
			ServerCapability("FOO"): struct{}{},
		},
	}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = origin0.Cachegroup
	eCG.ParentCachegroupID = origin0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	oCG := &tc.CacheGroupNullable{}
	oCG.Name = origin0.Cachegroup
	oCG.ID = origin0.CachegroupID
	oCGType := tc.CacheGroupOriginTypeName
	oCG.Type = &oCGType

	cgs := []tc.CacheGroupNullable{*eCG, *oCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *origin0.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myorigin0") {
		t.Errorf("expected origin0 with DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin1") {
		t.Errorf("expected no origin1 without DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigMSOWithCapabilities(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.MultiSiteOrigin = util.BoolPtr(true)

	dses := []DeliveryService{*ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("midCG")
	server.Type = "MID"
	server.CachegroupID = util.IntPtr(400)
	server.ID = util.IntPtr(44)

	origin0 := makeTestParentServer()
	origin0.Cachegroup = util.StrPtr("originCG")
	origin0.CachegroupID = util.IntPtr(500)
	origin0.HostName = util.StrPtr("myorigin0")
	origin0.ID = util.IntPtr(45)
	setIP(origin0, "192.168.2.2")
	origin0.Type = tc.OriginTypeName
	origin0.TypeID = util.IntPtr(991)

	origin1 := makeTestParentServer()
	origin1.Cachegroup = util.StrPtr("originCG")
	origin1.CachegroupID = util.IntPtr(500)
	origin1.HostName = util.StrPtr("myorigin1")
	origin1.ID = util.IntPtr(46)
	setIP(origin1, "192.168.2.3")
	origin1.Type = tc.OriginTypeName
	origin1.TypeID = util.IntPtr(991)

	servers := []Server{*server, *origin0, *origin1}

	serverCapabilities := map[int]map[ServerCapability]struct{}{
		*server.ID: {
			ServerCapability("FOO"): struct{}{},
		},
	}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{
		*ds1.ID: {
			ServerCapability("FOO"): struct{}{},
		},
	}

	midCG := &tc.CacheGroupNullable{}
	midCG.Name = server.Cachegroup
	midCG.ID = server.CachegroupID
	midCG.ParentName = origin0.Cachegroup
	midCG.ParentCachegroupID = origin0.CachegroupID
	midCGType := tc.CacheGroupMidTypeName
	midCG.Type = &midCGType

	oCG := &tc.CacheGroupNullable{}
	oCG.Name = origin0.Cachegroup
	oCG.ID = origin0.CachegroupID
	oCGType := tc.CacheGroupOriginTypeName
	oCG.Type = &oCGType

	cgs := []tc.CacheGroupNullable{*midCG, *oCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *origin0.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}
	topologies := []tc.Topology{}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myorigin0") {
		t.Errorf("expected origin0 with DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin1") {
		t.Errorf("expected no origin1 without DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesMSOParams(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.ProfileName = util.StrPtr("ds1Profile")
	ds1.ProfileID = util.IntPtr(994)
	ds1.MultiSiteOrigin = util.BoolPtr(true)

	dses := []DeliveryService{*ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMSOAlgorithm,
			ConfigFile: "parent.config",
			Value:      "consistent_hash",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMSOParentRetry,
			ConfigFile: "parent.config",
			Value:      "both",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMSOUnavailableServerRetryResponses,
			ConfigFile: "parent.config",
			Value:      `"400,503"`,
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMSOMaxSimpleRetries,
			ConfigFile: "parent.config",
			Value:      "14",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMSOMaxUnavailableServerRetries,
			ConfigFile: "parent.config",
			Value:      "9",
			Profiles:   []byte(`["ds1Profile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "8",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	origin0 := makeTestParentServer()
	origin0.Cachegroup = util.StrPtr("originCG")
	origin0.CachegroupID = util.IntPtr(500)
	origin0.HostName = util.StrPtr("myorigin0")
	origin0.ID = util.IntPtr(45)
	setIP(origin0, "192.168.2.2")
	origin0.Type = tc.OriginTypeName
	origin0.TypeID = util.IntPtr(991)

	origin1 := makeTestParentServer()
	origin1.Cachegroup = util.StrPtr("originCG")
	origin1.CachegroupID = util.IntPtr(500)
	origin1.HostName = util.StrPtr("myorigin1")
	origin1.ID = util.IntPtr(46)
	setIP(origin1, "192.168.2.3")
	origin1.Type = tc.OriginTypeName
	origin1.TypeID = util.IntPtr(991)

	servers := []Server{*server, *origin0, *origin1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "originCG",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = origin0.Cachegroup
	eCG.ParentCachegroupID = origin0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	oCG := &tc.CacheGroupNullable{}
	oCG.Name = origin0.Cachegroup
	oCG.ID = origin0.CachegroupID
	oCGType := tc.CacheGroupOriginTypeName
	oCG.Type = &oCGType

	cgs := []tc.CacheGroupNullable{*eCG, *oCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *origin0.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myorigin0") {
		t.Errorf("expected origin0 with DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin1") {
		t.Errorf("expected no origin1 without DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "parent_retry=both") {
		t.Errorf("expected DS MSO parent_retry param 'both', actual: '%v'", txt)
	}
	if !strings.Contains(txt, `unavailable_server_retry_responses="400,503"`) {
		t.Errorf(`expected DS MSO unavailable_server_retry_responses param '"400,503"'', actual: '%v'`, txt)
	}
	if !strings.Contains(txt, "max_simple_retries=14") {
		t.Errorf("expected DS MSO max_simple_retries param '14', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "max_unavailable_server_retries=9") {
		t.Errorf("expected DS MSO max_unavailable_server_retries param '9', actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesParams(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.ProfileName = util.StrPtr("ds1Profile")
	ds1.ProfileID = util.IntPtr(994)
	ds1.MultiSiteOrigin = util.BoolPtr(true)

	dses := []DeliveryService{*ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      "consistent_hash",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamParentRetry,
			ConfigFile: "parent.config",
			Value:      "both",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamUnavailableServerRetryResponses,
			ConfigFile: "parent.config",
			Value:      `"400,503"`,
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMaxSimpleRetries,
			ConfigFile: "parent.config",
			Value:      "14",
			Profiles:   []byte(`["ds1Profile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamMaxUnavailableServerRetries,
			ConfigFile: "parent.config",
			Value:      "9",
			Profiles:   []byte(`["ds1Profile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "8",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	origin0 := makeTestParentServer()
	origin0.Cachegroup = util.StrPtr("originCG")
	origin0.CachegroupID = util.IntPtr(500)
	origin0.HostName = util.StrPtr("myorigin0")
	origin0.ID = util.IntPtr(45)
	setIP(origin0, "192.168.2.2")
	origin0.Type = tc.OriginTypeName
	origin0.TypeID = util.IntPtr(991)

	origin1 := makeTestParentServer()
	origin1.Cachegroup = util.StrPtr("originCG")
	origin1.CachegroupID = util.IntPtr(500)
	origin1.HostName = util.StrPtr("myorigin1")
	origin1.ID = util.IntPtr(46)
	setIP(origin1, "192.168.2.3")
	origin1.Type = tc.OriginTypeName
	origin1.TypeID = util.IntPtr(991)

	servers := []Server{*server, *origin0, *origin1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1},
				},
				tc.TopologyNode{
					Cachegroup: "originCG",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = origin0.Cachegroup
	eCG.ParentCachegroupID = origin0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	oCG := &tc.CacheGroupNullable{}
	oCG.Name = origin0.Cachegroup
	oCG.ID = origin0.CachegroupID
	oCGType := tc.CacheGroupOriginTypeName
	oCG.Type = &oCGType

	cgs := []tc.CacheGroupNullable{*eCG, *oCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *origin0.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "myorigin0") {
		t.Errorf("expected origin0 with DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if strings.Contains(txt, "myorigin1") {
		t.Errorf("expected no origin1 without DeliveryServiceServer assigned to this DS, actual: '%v'", txt)
	}
	if !strings.Contains(txt, "parent_retry=both") {
		t.Errorf("expected DS MSO parent_retry param 'both', actual: '%v'", txt)
	}
	if !strings.Contains(txt, `unavailable_server_retry_responses="400,503"`) {
		t.Errorf(`expected DS MSO unavailable_server_retry_responses param '"400,503"'', actual: '%v'`, txt)
	}
	if !strings.Contains(txt, "max_simple_retries=14") {
		t.Errorf("expected DS MSO max_simple_retries param '14', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "max_unavailable_server_retries=9") {
		t.Errorf("expected DS MSO max_unavailable_server_retries param '9', actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigTopologiesNonStandardServerTypes(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")
	mid0.Type = "MIDSOMETHING-RANDOM"

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.CachegroupID = util.IntPtr(500)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	mid1.Type = "MID_SOMETHING_ELSE"
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

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
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if strings.Contains(txt, "# topology") {
		// ATS doesn't support inline comments in parent.config
		t.Errorf("expected: no inline '# topology' comment, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigSecondaryMode(t *testing.T) {

	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.ProfileID = util.IntPtr(311)
	ds0.ProfileName = util.StrPtr("ds0Profile")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.ProfileID = util.IntPtr(312)
	ds1.ProfileName = util.StrPtr("ds1Profile")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamSecondaryMode,
			ConfigFile: "parent.config",
			Value:      "",
			Profiles:   []byte(`["ds0Profile","ds1Profile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "8",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG2")
	mid1.CachegroupID = util.IntPtr(501)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1, 2},
				},
				tc.TopologyNode{
					Cachegroup: "midCG",
				},
				tc.TopologyNode{
					Cachegroup: "midCG2",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCG.SecondaryParentName = mid1.Cachegroup
	eCG.SecondaryParentCachegroupID = mid1.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	mCG2 := &tc.CacheGroupNullable{}
	mCG2.Name = mid1.Cachegroup
	mCG2.ID = mid1.CachegroupID
	mCGType2 := tc.CacheGroupMidTypeName
	mCG2.Type = &mCGType2

	cgs := []tc.CacheGroupNullable{*eCG, *mCG, *mCG2}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if strings.Count(txt, "secondary_mode=2") != 2 {
		t.Errorf("expected secondary_mode=2 for both Topology and DSS DSes with ParentConfigParamSecondaryMode parameter and secondary parents, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigNoSecondaryMode(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: false, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.ProfileID = util.IntPtr(311)
	ds0.ProfileName = util.StrPtr("ds0Profile")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.ProfileID = util.IntPtr(312)
	ds1.ProfileName = util.StrPtr("ds1Profile")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		tc.Parameter{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		tc.Parameter{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		tc.Parameter{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "8",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG2")
	mid1.CachegroupID = util.IntPtr(501)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{
		tc.Topology{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				tc.TopologyNode{
					Cachegroup: "edgeCG",
					Parents:    []int{1, 2},
				},
				tc.TopologyNode{
					Cachegroup: "midCG",
				},
				tc.TopologyNode{
					Cachegroup: "midCG2",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCG.SecondaryParentName = mid1.Cachegroup
	eCG.SecondaryParentCachegroupID = mid1.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	mCG2 := &tc.CacheGroupNullable{}
	mCG2.Name = mid1.Cachegroup
	mCG2.ID = mid1.CachegroupID
	mCGType2 := tc.CacheGroupMidTypeName
	mCG2.Type = &mCGType2

	cgs := []tc.CacheGroupNullable{*eCG, *mCG, *mCG2}

	dss := []DeliveryServiceServer{
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		DeliveryServiceServer{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if strings.Contains(txt, "secondary_mode") {
		t.Errorf("expected no secondary_mode for DSes without ParentConfigParamSecondaryMode parameter, actual: '%v'", txt)
	}

	if strings.Contains(txt, `topology 't0'`) {
		t.Errorf("expected no comment with topology name, actual: '%v'", txt)
	}
	if strings.Contains(txt, `ds 'ds1'`) {
		t.Errorf("expected no comment with delivery service name, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigComments(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: true, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "7",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.HostName = util.StrPtr("mymid0")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG")
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{}
	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	cgs := []tc.CacheGroupNullable{*eCG, *mCG}

	dss := []DeliveryServiceServer{
		{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "# ds 'ds1'") {
		t.Errorf("expected comment with delivery service name, actual: '%v'", txt)
	}
}

func TestMakeParentDotConfigCommentTopology(t *testing.T) {
	hdr := &ParentConfigOpts{AddComments: true, HdrComment: "myHeaderComment"}

	ds0 := makeParentDS()
	ds0Type := tc.DSTypeHTTP
	ds0.Type = &ds0Type
	ds0.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreUseInCacheKeyAndPassUp))
	ds0.OrgServerFQDN = util.StrPtr("http://ds0.example.net")
	ds0.ProfileID = util.IntPtr(311)
	ds0.ProfileName = util.StrPtr("ds0Profile")

	ds1 := makeParentDS()
	ds1.ID = util.IntPtr(43)
	ds1Type := tc.DSTypeDNS
	ds1.Type = &ds1Type
	ds1.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds1.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	ds1.Topology = util.StrPtr("t0")
	ds1.ProfileID = util.IntPtr(312)
	ds1.ProfileName = util.StrPtr("ds1Profile")

	dses := []DeliveryService{*ds0, *ds1}

	parentConfigParams := []tc.Parameter{
		{
			Name:       ParentConfigParamQStringHandling,
			ConfigFile: "parent.config",
			Value:      "myQStringHandlingParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
		{
			Name:       ParentConfigParamAlgorithm,
			ConfigFile: "parent.config",
			Value:      tc.AlgorithmConsistentHash,
			Profiles:   []byte(`["serverprofile"]`),
		},
		{
			Name:       ParentConfigParamQString,
			ConfigFile: "parent.config",
			Value:      "myQstringParam",
			Profiles:   []byte(`["serverprofile"]`),
		},
	}

	serverParams := []tc.Parameter{
		{
			Name:       "trafficserver",
			ConfigFile: "package",
			Value:      "8",
			Profiles:   []byte(`["global"]`),
		},
	}

	server := makeTestParentServer()
	server.Cachegroup = util.StrPtr("edgeCG")
	server.CachegroupID = util.IntPtr(400)

	mid0 := makeTestParentServer()
	mid0.Cachegroup = util.StrPtr("midCG")
	mid0.CachegroupID = util.IntPtr(500)
	mid0.HostName = util.StrPtr("mymid")
	mid0.ID = util.IntPtr(45)
	setIP(mid0, "192.168.2.2")

	mid1 := makeTestParentServer()
	mid1.Cachegroup = util.StrPtr("midCG2")
	mid1.CachegroupID = util.IntPtr(501)
	mid1.HostName = util.StrPtr("mymid1")
	mid1.ID = util.IntPtr(46)
	setIP(mid1, "192.168.2.3")

	servers := []Server{*server, *mid0, *mid1}

	topologies := []tc.Topology{
		{
			Name: "t0",
			Nodes: []tc.TopologyNode{
				{
					Cachegroup: "edgeCG",
					Parents:    []int{1, 2},
				},
				{
					Cachegroup: "midCG",
				},
				{
					Cachegroup: "midCG2",
				},
			},
		},
	}

	serverCapabilities := map[int]map[ServerCapability]struct{}{}
	dsRequiredCapabilities := map[int]map[ServerCapability]struct{}{}

	eCG := &tc.CacheGroupNullable{}
	eCG.Name = server.Cachegroup
	eCG.ID = server.CachegroupID
	eCG.ParentName = mid0.Cachegroup
	eCG.ParentCachegroupID = mid0.CachegroupID
	eCG.SecondaryParentName = mid1.Cachegroup
	eCG.SecondaryParentCachegroupID = mid1.CachegroupID
	eCGType := tc.CacheGroupEdgeTypeName
	eCG.Type = &eCGType

	mCG := &tc.CacheGroupNullable{}
	mCG.Name = mid0.Cachegroup
	mCG.ID = mid0.CachegroupID
	mCGType := tc.CacheGroupMidTypeName
	mCG.Type = &mCGType

	mCG2 := &tc.CacheGroupNullable{}
	mCG2.Name = mid1.Cachegroup
	mCG2.ID = mid1.CachegroupID
	mCGType2 := tc.CacheGroupMidTypeName
	mCG2.Type = &mCGType2

	cgs := []tc.CacheGroupNullable{*eCG, *mCG, *mCG2}

	dss := []DeliveryServiceServer{
		{
			Server:          *server.ID,
			DeliveryService: *ds0.ID,
		},
		{
			Server:          *server.ID,
			DeliveryService: *ds1.ID,
		},
	}
	cdn := &tc.CDN{
		DomainName: "cdndomain.example",
		Name:       "my-cdn-name",
	}

	cfg, err := MakeParentDotConfig(dses, server, servers, topologies, serverParams, parentConfigParams, serverCapabilities, dsRequiredCapabilities, cgs, dss, cdn, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr.HdrComment)

	if !strings.Contains(txt, "dest_domain=ds0.example.net") {
		t.Errorf("expected parent 'dest_domain=ds0.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "dest_domain=ds1.example.net") {
		t.Errorf("expected parent 'dest_domain=ds1.example.net', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "qstring=myQStringHandlingParam") {
		t.Errorf("expected qstring from param 'qstring=myQStringHandlingParam', actual: '%v'", txt)
	}
	if strings.Contains(txt, "secondary_mode") {
		t.Errorf("expected no secondary_mode for DSes without ParentConfigParamSecondaryMode parameter, actual: '%v'", txt)
	}
	if !strings.Contains(txt, `# ds 'ds1' topology 't0'`) {
		t.Errorf("expected comment with delivery service and topology, actual: '%v'", txt)
	}
}

func makeTestParentServer() *Server {
	server := &Server{}
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

func makeParentDS() *DeliveryService {
	ds := &DeliveryService{}
	ds.ID = util.IntPtr(42)
	ds.XMLID = util.StrPtr("ds1")
	ds.QStringIgnore = util.IntPtr(int(tc.QStringIgnoreDrop))
	ds.OrgServerFQDN = util.StrPtr("http://ds1.example.net")
	dsType := tc.DSTypeDNS
	ds.Type = &dsType
	ds.MultiSiteOrigin = util.BoolPtr(false)
	return ds
}
