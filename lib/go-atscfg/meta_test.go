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

func TestMakeMetaConfig(t *testing.T) {
	server := &Server{}
	server.CachegroupID = util.IntPtr(42)
	server.Cachegroup = util.StrPtr("cg0")
	server.CDNName = util.StrPtr("mycdn")
	server.CDNID = util.IntPtr(43)
	server.DomainName = util.StrPtr("myserverdomain.invalid")
	server.HostName = util.StrPtr("myserver")
	server.HTTPSPort = util.IntPtr(443)
	server.ID = util.IntPtr(44)
	ip := "192.168.2.9"
	setIP(server, ip)
	// server.ParentCacheGroupID=            45
	// server.ParentCacheGroupType=          "MID_LOC"
	server.ProfileID = util.IntPtr(46)
	server.Profile = util.StrPtr("myserverprofile")
	server.TCPPort = util.IntPtr(80)
	// server.SecondaryParentCacheGroupID=   47
	// server.SecondaryParentCacheGroupType= "MID_LOC"
	server.Type = "EDGE"

	// uriSignedDSes := []tc.DeliveryServiceName{"mydsname"}
	// dses := map[tc.DeliveryServiceName]DeliveryService{"mydsname": {}}

	cgs := []tc.CacheGroupNullable{}
	topologies := []tc.Topology{}

	cfgPath := "/etc/foo/trafficserver"

	deliveryServices := []DeliveryService{}
	dss := []DeliveryServiceServer{}
	globalParams := []tc.Parameter{}

	makeLocationParam := func(name string) tc.Parameter {
		return tc.Parameter{
			Name:       "location",
			ConfigFile: name,
			Value:      "/my/location/",
			Profiles:   []byte(`["` + *server.Profile + `"]`),
		}
	}

	serverParams := []tc.Parameter{
		makeLocationParam("ssl_multicert.config"),
		makeLocationParam("volume.config"),
		makeLocationParam("ip_allow.config"),
		makeLocationParam("cache.config"),
		makeLocationParam("regex_revalidate.config"),
		makeLocationParam("uri_signing_mydsname.config"),
		makeLocationParam("uri_signing_nonexistentds.config"),
		makeLocationParam("regex_remap_nonexistentds.config"),
		makeLocationParam("url_sig_nonexistentds.config"),
		makeLocationParam("hdr_rw_nonexistentds.config"),
		makeLocationParam("hdr_rw_mid_nonexistentds.config"),
		makeLocationParam("unknown.config"),
		makeLocationParam("custom.config"),
		makeLocationParam("external.config"),
	}

	cfg, _, err := MakeConfigFilesList(cfgPath, server, serverParams, deliveryServices, dss, globalParams, cgs, topologies, &ConfigFilesListOpts{})
	if err != nil {
		t.Fatalf("MakeConfigFilesList: " + err.Error())
	}

	expectedConfigs := map[string]func(cf CfgMeta){
		"cache.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"ip_allow.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"volume.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"ssl_multicert.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"uri_signing_mydsname.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"unknown.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"custom.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"remap.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"regex_revalidate.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"external.config": func(cf CfgMeta) {
			if expected := "/my/location/"; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"hosting.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"parent.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"plugin.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"records.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"storage.config": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
		"ssl_server_name.yaml": func(cf CfgMeta) {
			if expected := cfgPath; cf.Path != expected {
				t.Errorf("expected location '%v', actual '%v'", expected, cf.Path)
			}
		},
	}

	for _, cfgFile := range cfg {
		if testF, ok := expectedConfigs[cfgFile.Name]; !ok {
			t.Errorf("unexpected config '" + cfgFile.Name + "'")
		} else {
			testF(cfgFile)
			delete(expectedConfigs, cfgFile.Name)
		}
	}

	server.Type = "MID"
	cfg, _, err = MakeConfigFilesList(cfgPath, server, serverParams, deliveryServices, dss, globalParams, cgs, topologies, &ConfigFilesListOpts{})
	if err != nil {
		t.Fatalf("MakeConfigFilesList: " + err.Error())
	}
	for _, cfgFile := range cfg {
		if cfgFile.Name != "cache.config" {
			continue
		}
		break
	}
	for _, fi := range cfg {
		if strings.Contains(fi.Name, "nonexistentds") {
			t.Errorf("expected location parameters for nonexistent delivery services to not be added to config, actual '%v'", fi.Name)
		}
	}
}
