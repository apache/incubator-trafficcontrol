package v14

/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestParentDotConfig(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, DeliveryServices}, func() {
		defer DeleteTestDeliveryServiceServersCreated(t)
		CreateTestDeliveryServiceServers(t)
		GetTestParentDotConfig(t)
	})
}

func GetTestParentDotConfig(t *testing.T) {
	dsServers, _, err := TOSession.GetDeliveryServiceServers()
	if err != nil {
		t.Fatalf("GET delivery service servers: %v\n", err)
	} else if len(dsServers.Response) == 0 {
		t.Fatalf("GET delivery service servers: no servers found\n")
	} else if dsServers.Response[0].Server == nil {
		t.Fatalf("GET delivery service servers: returned nil server\n")
	} else if dsServers.Response[0].DeliveryService == nil {
		t.Fatalf("GET delivery service servers: returned nil ds\n")
	}
	serverID := *dsServers.Response[0].Server

	ds, _, err := TOSession.GetDeliveryService(strconv.Itoa(*dsServers.Response[0].DeliveryService))
	if err != nil {
		t.Fatalf("Getting ds %+v: "+err.Error()+"\n", *dsServers.Response[0].DeliveryService)
	} else if ds == nil {
		t.Fatalf("Getting ds %+v: "+"got nil response"+"\n", *dsServers.Response[0].DeliveryService)
	} else if ds.OrgServerFQDN == "" {
		t.Fatalf("Getting ds %+v: "+"got empty ds.OrgServerFQDN"+"\n", *dsServers.Response[0].DeliveryService)
	}

	originURI, err := url.Parse(ds.OrgServerFQDN)
	if err != nil {
		t.Fatalf("Getting ds %+v: "+" ds.OrgServerFQDN '%+v' failed to parse as a URL: %+v\n", *dsServers.Response[0].DeliveryService, ds.OrgServerFQDN, err)
	}
	originHost := originURI.Hostname()

	parentDotConfig, _, err := TOSession.GetATSServerConfig(serverID, "parent.config")
	if err != nil {
		t.Fatalf("Getting server %+v config parent.config: "+err.Error()+"\n", serverID)
	}

	if !strings.Contains(parentDotConfig, originHost) {
		t.Errorf("expected: parent.config to contain delivery service origin FQDN '%+v' host '%+v', actual: '''%+v'''", ds.OrgServerFQDN, originHost, parentDotConfig)
	}
}

func CreateTestDeliveryServiceServers(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices()
	if err != nil {
		t.Errorf("cannot GET DeliveryServices: %v\n", err)
	}
	ds := tc.DeliveryService{}
	for _, tods := range dses {
		if !tods.Type.IsHTTP() && !tods.Type.IsDNS() {
			continue
		}
		if !tods.Type.UsesMidCache() {
			continue
		}
		if ds.ID == 0 || tods.ID < ds.ID {
			ds = tods
		}
	}
	if ds.ID == 0 {
		t.Fatalf("GET DeliveryServices returned no DNS or HTTP dses, must have at least 1 to test")
	}

	servers, _, err := TOSession.GetServers()
	if err != nil {
		t.Errorf("cannot GET Servers: %v\n", err)
	}

	server := tc.Server{}
	for _, toserver := range servers {
		if toserver.Type != string(tc.CacheTypeEdge) {
			continue
		}
		if server.ID == 0 || toserver.ID > server.ID {
			server = toserver
		}
	}
	if server.ID == 0 {
		t.Fatalf("GET Servers returned no EDGE server, must have at least 1 to test")
	}

	for _, ds := range dses {
		serverIDs := []int{}
		for _, server := range servers {
			serverIDs = append(serverIDs, server.ID)
		}

		_, err = TOSession.CreateDeliveryServiceServers(ds.ID, serverIDs, true)
		if err != nil {
			t.Errorf("POST delivery service servers: %v\n", err)
		}
	}
}

// DeleteTestDeliveryServiceServersCreated deletes the dss assignments created by CreateTestDeliveryServiceServers.
func DeleteTestDeliveryServiceServersCreated(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices()
	if err != nil {
		t.Errorf("cannot GET DeliveryServices: %v\n", err)
	}
	ds := tc.DeliveryService{}
	for _, tods := range dses {
		if !tods.Type.IsHTTP() && !tods.Type.IsDNS() {
			continue
		}
		if ds.ID == 0 || tods.ID < ds.ID {
			ds = tods
		}
	}
	if ds.ID == 0 {
		t.Fatalf("GET DeliveryServices returned no DNS or HTTP dses, must have at least 1 to test")
	}

	servers, _, err := TOSession.GetServers()
	if err != nil {
		t.Errorf("cannot GET Servers: %v\n", err)
	}

	server := tc.Server{}
	for _, toserver := range servers {
		if toserver.Type != string(tc.CacheTypeEdge) {
			continue
		}
		if server.ID == 0 || toserver.ID > server.ID {
			server = toserver
		}
	}
	if server.ID == 0 {
		t.Fatalf("GET Servers returned no EDGE server, must have at least 1 to test")
	}

	dsServers, _, err := TOSession.GetDeliveryServiceServersN(999999999)
	if err != nil {
		t.Errorf("GET delivery service servers: %v\n", err)
	}

	found := false
	for _, dss := range dsServers.Response {
		if *dss.DeliveryService == ds.ID && *dss.Server == server.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("POST delivery service servers returned success, but ds-server not in GET")
	}

	if _, _, err := TOSession.DeleteDeliveryServiceServer(ds.ID, server.ID); err != nil {
		t.Errorf("DELETE delivery service server: %v\n", err)
	}

	dsServers, _, err = TOSession.GetDeliveryServiceServers()
	if err != nil {
		t.Errorf("GET delivery service servers: %v\n", err)
	}

	found = false
	for _, dss := range dsServers.Response {
		if *dss.DeliveryService == ds.ID && *dss.Server == server.ID {
			found = true
			break
		}
	}
	if found {
		t.Errorf("DELETE delivery service servers returned success, but still in GET")
	}
}
