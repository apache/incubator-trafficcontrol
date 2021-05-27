package v4

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
	"net/http"
	"strconv"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestCacheGroupsDeliveryServices(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, DeliveryServices, CacheGroupsDeliveryServices}, func() {})
}

// TODO this is the name hard-coded in the create servers test; change to be dynamic
// TODO this test assumes that a CDN named "cdn1" exists, has at least one Delivery Service, and also
// assumes that ALL SERVERS IN "cachegroup3" ARE EDGE-TIER CACHE SERVERS IN "cdn1". If that EVER changes,
// this WILL break.
const TestEdgeServerCacheGroupName = "cachegroup3"

func CreateTestCachegroupsDeliveryServices(t *testing.T) {
	dss, _, err := TOSession.GetDeliveryServiceServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error retrieving server-to-Delivery-Service assignments: %v - alerts: %+v", err, dss.Alerts)
	}
	if len(dss.Response) > 0 {
		t.Fatalf("cannot test cachegroups delivery services: expected no initial delivery service servers, actual %v", len(dss.Response))
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot GET DeliveryServices: %v - %v", err, dses)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", TestEdgeServerCacheGroupName)
	clientCGs, _, err := TOSession.GetCacheGroups(opts)
	if err != nil {
		t.Fatalf("getting cachegroup: %v", err)
	}
	if len(clientCGs.Response) != 1 {
		t.Fatalf("getting cachegroup expected 1, got %v", len(clientCGs.Response))
	}

	clientCG := clientCGs.Response[0]

	if clientCG.ID == nil {
		t.Fatalf("Cachegroup has a nil ID")
	}
	cgID := *clientCG.ID

	dsIDs := []int{}
	topologyDsIDs := []int{}
	for _, ds := range dses.Response {
		if *ds.CDNName == "cdn1" && ds.Topology == nil {
			dsIDs = append(dsIDs, *ds.ID)
		} else if *ds.CDNName == "cdn1" && ds.Topology != nil {
			topologyDsIDs = append(topologyDsIDs, *ds.ID)
		}
	}
	if len(dsIDs) < 1 {
		t.Fatal("No Delivery Services found in CDN 'cdn1', cannot continue.")
	}

	if len(topologyDsIDs) < 1 {
		t.Fatal("No Topology-based Delivery Services found in CDN 'cdn1', cannot continue.")
	}

	_, reqInf, err := TOSession.SetCacheGroupDeliveryServices(cgID, topologyDsIDs, client.RequestOptions{})
	if err == nil {
		t.Fatal("assigning Topology-based delivery service to cachegroup - expected: error, actual: nil")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("assigning Topology-based delivery service to cachegroup - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}

	resp, _, err := TOSession.SetCacheGroupDeliveryServices(cgID, dsIDs, client.RequestOptions{})
	if err != nil {
		t.Fatalf("setting cachegroup delivery services returned error: %v", err)
	}
	if len(resp.Response.ServerNames) == 0 {
		t.Fatal("setting cachegroup delivery services returned success, but no servers set")
	}

	// Note this second post of the same cg-dses specifically tests a previous bug, where the query
	// failed if any servers with location parameters were already assigned, due to a foreign key
	// violation. See https://github.com/apache/trafficcontrol/pull/3199
	resp, _, err = TOSession.SetCacheGroupDeliveryServices(cgID, dsIDs, client.RequestOptions{})
	if err != nil {
		t.Fatalf("setting cachegroup delivery services returned error: %v", err)
	}
	if len(resp.Response.ServerNames) == 0 {
		t.Fatal("setting cachegroup delivery services returned success, but no servers set")
	}

	opts.QueryParameters.Del("name")
	for _, serverName := range resp.Response.ServerNames {
		opts.QueryParameters.Set("hostName", string(serverName))
		resp, _, err := TOSession.GetServers(opts)
		if err != nil {
			t.Fatalf("getting server: %v - alerts: %+v", err, resp.Alerts)
		}
		servers := resp.Response
		if len(servers) != 1 {
			t.Fatalf("getting servers: expected 1 got %v", len(servers))
		}
		server := servers[0]
		serverID := server.ID

		if serverID == nil {
			t.Fatalf("got a nil server ID in response, quitting")
		}
		serverDSes, _, err := TOSession.GetDeliveryServicesByServer(*serverID, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error getting servers for Delivery Service #%d: %v - alerts: %+v", *serverID, err, serverDSes.Alerts)
		}

		for _, dsID := range dsIDs {
			found := false
			for _, serverDS := range serverDSes.Response {
				if *serverDS.ID == int(dsID) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("post succeeded, but didn't assign delivery service %v to server", dsID)
			}
		}
	}
}

func setInactive(t *testing.T, dsID int) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("id", strconv.Itoa(dsID))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Failed to fetch details for Delivery Service #%d: %v - alerts: %+v", dsID, err, resp.Alerts)
		return
	}
	if len(resp.Response) != 1 {
		t.Errorf("Expected exactly one Delivery Service to exist with ID %d, found: %d", dsID, len(resp.Response))
		return
	}

	ds := resp.Response[0]
	if ds.Active != tc.DS_INACTIVE {
		ds.Active = tc.DS_INACTIVE
		_, _, err = TOSession.UpdateDeliveryService(dsID, ds, client.RequestOptions{})
		if err != nil {
			t.Errorf("Failed to set Delivery Service #%d to inactive: %v", dsID, err)
		}
	}
}

func DeleteTestCachegroupsDeliveryServices(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("limit", "1000000")
	dss, _, err := TOSession.GetDeliveryServiceServers(opts)
	if err != nil {
		t.Fatalf("Unexpected error retrieving server-to-Delivery-Service assignments: %v - alerts: %+v", err, dss.Alerts)
	}
	for _, ds := range dss.Response {
		if ds.DeliveryService == nil {
			t.Error("Got deliveryserviceserver with no Delivery Service")
			continue
		}
		if ds.Server == nil {
			t.Error("Got deliveryserviceserver with no server")
			continue
		}

		setInactive(t, *ds.DeliveryService)

		alerts, _, err := TOSession.DeleteDeliveryServiceServer(*ds.DeliveryService, *ds.Server, client.RequestOptions{})
		if err != nil {
			t.Errorf("deleting delivery service servers: %v - alerts: %+v", err, alerts.Alerts)
		}
	}

	dss, _, err = TOSession.GetDeliveryServiceServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error retrieving server-to-Delivery-Service assignments: %v - alerts: %+v", err, dss.Alerts)
	}
	if len(dss.Response) > 0 {
		t.Errorf("deleting delivery service servers: delete succeeded, expected empty subsequent get, actual %v", len(dss.Response))
	}
}
