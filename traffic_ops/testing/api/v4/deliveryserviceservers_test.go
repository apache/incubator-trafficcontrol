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
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestDeliveryServiceServers(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, DeliveryServices}, func() {
		DeleteTestDeliveryServiceServers(t)
		AssignServersToTopologyBasedDeliveryService(t)
		AssignOriginsToTopologyBasedDeliveryServices(t)
		TryToRemoveLastServerInDeliveryService(t)
		AssignServersToNonTopologyBasedDeliveryServiceThatUsesMidTier(t)
	})
}

func TestDeliveryServiceServersWithRequiredCapabilities(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, ServerCapabilities, Topologies, DeliveryServices, DeliveryServicesRequiredCapabilities, ServerServerCapabilities}, func() {
		CreateTestDeliveryServiceServersWithRequiredCapabilities(t)
		CreateTestMSODSServerWithReqCap(t)
	})
}

const dssaTestingXMLID = "test-ds-server-assignments"

func TryToRemoveLastServerInDeliveryService(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", dssaTestingXMLID)
	dses, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("Unexpected error trying to get Delivery service with XMLID '%s': %v - alerts: %+v", dssaTestingXMLID, err, dses.Alerts)
	}
	if len(dses.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery service with XMLID '%s', got: %d", dssaTestingXMLID, len(dses.Response))
	}
	ds := dses.Response[0]
	if ds.ID == nil {
		t.Fatalf("Delivery Service '%s' has no ID", dssaTestingXMLID)
	}

	statuses, _, err := TOSession.GetStatuses(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Could not fetch Statuses: %v - alerts: %+v", err, statuses.Alerts)
	}
	if len(statuses.Response) < 1 {
		t.Fatal("Need at least one Status")
	}

	var badStatusID int
	found := false
	for _, status := range statuses.Response {
		if status.Name != tc.CacheStatusOnline.String() && status.Name != tc.CacheStatusReported.String() {
			badStatusID = status.ID
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Need at least one status with a name other than '%s' or '%s'", tc.CacheStatusOnline, tc.CacheStatusReported)
	}

	// TODO: this isn't sufficient to define a single server, so there might be
	// a better way to do this.
	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("hostName", dssaTestingXMLID)
	opts.QueryParameters.Set("domainName", dssaTestingXMLID)
	servers, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("Unexpected error fetching server '%s.%s': %v - alerts: %+v", dssaTestingXMLID, dssaTestingXMLID, err, servers.Alerts)
	}
	if len(servers.Response) != 1 {
		t.Fatalf("Expected exactly one server with FQDN '%s.%s', got: %d", dssaTestingXMLID, dssaTestingXMLID, len(servers.Response))
	}
	server := servers.Response[0]
	if server.ID == nil {
		t.Fatal("Server had null/undefined ID after creation")
	}

	resp, _, err := TOSession.CreateDeliveryServiceServers(*ds.ID, []int{*server.ID}, true, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Failed to assign server to Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	}

	_, _, err = TOSession.CreateDeliveryServiceServers(*ds.ID, []int{}, true, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to remove the only server assigned to a Delivery Service")
	}

	_, _, err = TOSession.DeleteDeliveryServiceServer(*ds.ID, *server.ID, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to remove the only server assigned to a Delivery Service")
	}

	alerts, _, err := TOSession.DeleteServer(*server.ID, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to delete the only server assigned to a Delivery Service")
	} else {
		t.Logf("Got expected error trying to delete the only server assigned to a Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
	}

	alerts, _, err = TOSession.AssignDeliveryServiceIDsToServerID(*server.ID, []int{}, true, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to remove a Delivery Service from the only server to which it is assigned")
	} else {
		t.Logf("Got expected error trying to remove a Delivery Service from the only server to which it is assigned: %v - alerts: %+v", err, alerts.Alerts)
	}

	server.StatusID = &badStatusID
	putRequest := tc.ServerPutStatus{
		Status:        util.JSONNameOrIDStr{ID: &badStatusID},
		OfflineReason: util.StrPtr("test"),
	}
	alerts, _, err = TOSession.UpdateServerStatus(*server.ID, putRequest, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to put server into a bad state when it's the only one assigned to a Delivery Service")
	} else {
		t.Logf("Got expected error trying to put server into a bad state when it's the only one assigned to a Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
	}

	alerts, _, err = TOSession.UpdateServer(*server.ID, server, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to put server into a bad state when it's the only one assigned to a Delivery Service")
	} else {
		t.Logf("Got expected error trying to put server into a bad state when it's the only one assigned to a Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
	}

	server.HostName = util.StrPtr(dssaTestingXMLID + "-quest")
	server.ID = nil
	interfaces := server.Interfaces
	for interfaceIndex, i := range interfaces {
		ipAddresses := i.IPAddresses
		for index, ip := range ipAddresses {
			if ip.ServiceAddress {
				str := "100.100.100."
				ip.Address = str + strconv.Itoa(index)
			}
			ipAddresses[index] = ip
		}
		interfaces[interfaceIndex].IPAddresses = ipAddresses
	}
	server.Interfaces = interfaces
	alerts, _, err = TOSession.CreateServer(server, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Failed to create server: %v - alerts: %+v", err, alerts.Alerts)
	}
	opts.QueryParameters.Set("hostName", *server.HostName)
	servers, _, err = TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("Could not fetch server after creation: %v - alerts: %+v", err, servers.Alerts)
	}
	if len(servers.Response) != 1 {
		t.Fatalf("Expected exactly 1 server with hostname '%s'; got: %d", *server.HostName, len(servers.Response))
	}
	server = servers.Response[0]
	if server.ID == nil {
		t.Fatal("Server had null/undefined ID after creation")
	}

	_, _, err = TOSession.CreateDeliveryServiceServers(*ds.ID, []int{*server.ID}, true, client.RequestOptions{})
	if err == nil {
		t.Error("Didn't get expected error trying to replace the last server assigned to a Delivery Service with a server in a bad state")
	}

	// Cleanup
	setInactive(t, *ds.ID)
	alerts, _, err = TOSession.DeleteServer(*server.ID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Failed to delete server: %v - alerts: %+v", err, alerts.Alerts)
	}
}

func AssignServersToTopologyBasedDeliveryService(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "ds-top")
	ds, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service 'ds-top': %v - alerts: %+v", err, ds.Alerts)
	}
	if len(ds.Response) != 1 {
		t.Fatalf("expected one delivery service: 'ds-top', actual: %v", len(ds.Response))
	}
	d := ds.Response[0]
	if d.Topology == nil || d.ID == nil || d.CDNName == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service that had null or undefined Topology and/or CDN Name and/or ID")
	}
	serversResp, _, err := TOSession.GetServers(client.RequestOptions{})
	servers := []tc.ServerV4{}
	for _, s := range serversResp.Response {
		if s.CDNID != nil && *s.CDNID == d.CDNID && s.Type == tc.CacheTypeEdge.String() {
			servers = append(servers, s)
		}
	}
	if len(servers) < 1 {
		t.Fatalf("expected: at least one EDGE in cdn %s, actual: %v", *d.CDNName, servers)
	}
	if servers[0].ID == nil {
		t.Fatal("Traffic ops returned a representation of a Server that had a null or undefined ID")
	}
	serverNames := []string{}
	for _, s := range servers {
		if s.CDNID != nil && s.HostName != nil && *s.CDNID == d.CDNID && s.Type == tc.CacheTypeEdge.String() {
			serverNames = append(serverNames, *s.HostName)
		} else {
			t.Fatalf("expected only EDGE servers in cdn '%s', actual: %v", *d.CDNName, servers)
		}
	}
	_, reqInf, err := TOSession.AssignServersToDeliveryService(serverNames, "ds-top", client.RequestOptions{})
	if err == nil {
		t.Fatal("assigning servers to topology-based delivery service - expected: error, actual: nil error")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("assigning servers to topology-based delivery service - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}

	_, reqInf, err = TOSession.CreateDeliveryServiceServers(*d.ID, []int{*servers[0].ID}, false, client.RequestOptions{})
	if err == nil {
		t.Fatal("creating deliveryserviceserver assignment for topology-based delivery service - expected: error, actual: nil error")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("creating deliveryserviceserver assignment for topology-based delivery service - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}
}

func AssignOriginsToTopologyBasedDeliveryServices(t *testing.T) {
	// attempt to assign ORG server to a topology-based DS while the ORG server's cachegroup doesn't belong to the topology
	opts := client.NewRequestOptions()
	opts.QueryParameters.Add("hostName", "denver-mso-org-01")
	resp, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("unable to get server 'denver-mso-org-01': %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one server to exist with Host Name 'denver-mso-org-01': %d", len(resp.Response))
	}
	orgServer := resp.Response[0]
	_, reqInf, err := TOSession.AssignServersToDeliveryService([]string{*orgServer.HostName}, "ds-top-req-cap", client.RequestOptions{})
	if err == nil {
		t.Fatal("assigning ORG server to topology-based delivery service while the ORG server's cachegroup does not belong to the topology - expected: error, actual: nil error")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("assigning ORG server to topology-based delivery service while the ORG server's cachegroup does not belong to the topology - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}
	opts.QueryParameters.Del("hostName")
	opts.QueryParameters.Set("xmlId", "ds-top-req-cap")
	ds, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service 'ds-top-req-cap': %v - alerts: %+v", err, ds.Alerts)
	}
	if len(ds.Response) != 1 {
		t.Fatalf("expected one delivery service: 'ds-top-req-cap', actual: %v", len(ds.Response))
	}
	d := ds.Response[0]
	if d.Topology == nil || d.ID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service that had null or undefined Topology and/or ID")
	}
	_, reqInf, err = TOSession.CreateDeliveryServiceServers(*d.ID, []int{*orgServer.ID}, false, client.RequestOptions{})
	if err == nil {
		t.Fatal("creating deliveryserviceserver assignment for ORG server to topology-based delivery service while the ORG server's cachegroup does not belong to the topology - expected: error, actual: nil error")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("creating deliveryserviceserver assignment for ORG server to topology-based delivery service while the ORG server's cachegroup does not belong to the topology - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}

	// attempt to assign ORG server to a topology-based DS while the ORG server's cachegroup belongs to the topology
	assignResp, reqInf, err := TOSession.AssignServersToDeliveryService([]string{*orgServer.HostName}, "ds-top", client.RequestOptions{})
	if err != nil {
		t.Fatalf("assigning Origin server '%s' to Topology-based Delivery Service 'ds-top' while the ORG server's Cache Group belongs to the Topology - expected: no error, actual: %v - alerts: %+v", *orgServer.HostName, err, assignResp.Alerts)
	}
	if reqInf.StatusCode < http.StatusOK || reqInf.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("assigning ORG server to topology-based delivery service while the ORG server's cachegroup belongs to the topology - expected: 200-level status code, actual: %d", reqInf.StatusCode)
	}
	opts.QueryParameters.Set("xmlId", "ds-top")
	ds, _, err = TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service 'ds-top': %v - alerts: %+v", err, ds.Alerts)
	}
	if len(ds.Response) != 1 {
		t.Fatalf("expected one delivery service: 'ds-top', actual: %v", len(ds.Response))
	}
	d = ds.Response[0]
	if d.Topology == nil || d.ID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service that had null or undefined Topology and/or ID")
	}
	alerts, reqInf, err := TOSession.CreateDeliveryServiceServers(*d.ID, []int{*orgServer.ID}, true, client.RequestOptions{})
	if err != nil {
		t.Fatalf("assigning Origin server #%d to Topology-based Delivery Service #%d while the server's Cache Group belongs to the Topology - expected: no error, actual: %v - alerts: %+v", *orgServer.ID, *d.ID, err, alerts.Alerts)
	}
	if reqInf.StatusCode < http.StatusOK || reqInf.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("creating deliveryserviceserver assignment for ORG server to topology-based delivery service while the ORG server's cachegroup belongs to the topology - expected: 200-level status code, actual: %d", reqInf.StatusCode)
	}
}

func AssignServersToNonTopologyBasedDeliveryServiceThatUsesMidTier(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "ds1")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service 'ds1': %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("expected one delivery service: 'ds1', actual: %d", len(resp.Response))
	}
	dsWithMid := resp.Response[0]
	if dsWithMid.Topology != nil {
		t.Fatal("expected delivery service: 'ds1' to have a nil Topology, actual: non-nil")
	}
	if dsWithMid.CDNName == nil || dsWithMid.ID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service that had null or undefined CDN Name and/or ID")
	}
	serversResp, _, err := TOSession.GetServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to fetch all servers: %v - alerts: %+v", err, serversResp.Alerts)
	}
	serversIds := []int{}
	for _, s := range serversResp.Response {
		if s.CDNID != nil && *s.CDNID == dsWithMid.CDNID && s.Type == tc.CacheTypeEdge.String() {
			serversIds = append(serversIds, *s.ID)
		}
	}
	if len(serversIds) < 1 {
		t.Fatalf("expected: at least one EDGE in cdn %s, actual: 0", *dsWithMid.CDNName)
	}

	assignResp, _, err := TOSession.CreateDeliveryServiceServers(*dsWithMid.ID, serversIds, true, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error assigning servers %v to Delivery Service #%d: %v - alerts: %+v", serversIds, *dsWithMid.ID, err, assignResp.Alerts)
	}

	opts.QueryParameters = url.Values{"dsId": []string{strconv.Itoa(*dsWithMid.ID)}}
	dsServersResp, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("unable to fetch Delivery Service #%d servers: %v - alerts: %+v", *dsWithMid.ID, err, dsServersResp.Alerts)
	}
	dsServerIds := []int{}
	for _, dss := range dsServersResp.Response {
		dsServerIds = append(dsServerIds, *dss.ID)
	}
	if len(dsServerIds) <= len(serversIds) {
		t.Fatalf("delivery service servers (%d) expected to exceed directly assigned servers (%d) to account for implicitly assigned mid servers", len(dsServerIds), len(serversIds))
	}

	for _, dss := range dsServersResp.Response {
		if dss.CDNID == nil {
			t.Error("Traffic Ops returned a representation for a server with null or undefined CDN ID")
			continue
		}
		if *dss.CDNID != dsWithMid.CDNID {
			t.Error("a server for another cdn was returned for this delivery service")
		}
	}
}

func CreateTestDeliveryServiceServersWithRequiredCapabilities(t *testing.T) {
	sscs := testData.ServerServerCapabilities

	testCases := []struct {
		ds          string
		serverName  string
		ssc         tc.ServerServerCapability
		description string
		err         error
		capability  tc.DeliveryServicesRequiredCapability
	}{
		{
			serverName:  "atlanta-edge-01",
			description: "missing requirements for server -> DS assignment",
			err:         errors.New(`Caching server cannot be assigned to this delivery service without having the required delivery service capabilities`),
			ssc:         sscs[0],
			capability: tc.DeliveryServicesRequiredCapability{
				DeliveryServiceID:  helperGetDeliveryServiceID(t, util.StrPtr("ds-test-minor-versions")),
				RequiredCapability: sscs[1].ServerCapability,
			},
		},
		{
			serverName:  "atlanta-mid-01",
			description: "successful server -> DS assignment",
			err:         nil,
			ssc:         sscs[1],
			capability: tc.DeliveryServicesRequiredCapability{
				DeliveryServiceID:  helperGetDeliveryServiceID(t, util.StrPtr("ds3")),
				RequiredCapability: sscs[1].ServerCapability,
			},
		},
	}

	for _, ctc := range testCases {
		if ctc.capability.DeliveryServiceID == nil || ctc.capability.RequiredCapability == nil {
			t.Errorf("Bad hard-coded test case '%s' - MUST include non-nil DeliveryServiceID and RequiredCapability", ctc.description)
			continue
		}
		t.Run(ctc.description, func(t *testing.T) {
			opts := client.NewRequestOptions()
			opts.QueryParameters.Set("hostName", ctc.serverName)
			resp, _, err := TOSession.GetServers(opts)
			if err != nil {
				t.Fatalf("cannot get Server '%s' by hostname: %v - alerts: %+v", ctc.serverName, err, resp.Alerts)
			}
			servers := resp.Response
			if len(servers) < 1 {
				t.Fatalf("Expected at least one server to exist with Host Name '%s', found none", ctc.serverName)
			}
			server := servers[0]
			if server.ID == nil {
				t.Fatalf("server %s had nil ID", ctc.serverName)
			}

			alerts, _, err := TOSession.CreateDeliveryServicesRequiredCapability(ctc.capability, client.RequestOptions{})
			if err != nil {
				t.Fatalf("Unexpected error creating a relationship between a Delivery Service and a Capability it requires: %v - alerts: %+v", err, alerts.Alerts)
			}

			ctc.ssc.ServerID = server.ID
			sscResp, _, err := TOSession.CreateServerServerCapability(ctc.ssc, client.RequestOptions{})
			if err != nil {
				t.Fatalf("could not associate Capability '%s' to server #%d: %v - alerts: %+v", *ctc.ssc.ServerCapability, *ctc.ssc.ServerID, err, sscResp.Alerts)
			}

			assignResp, _, got := TOSession.CreateDeliveryServiceServers(*ctc.capability.DeliveryServiceID, []int{*server.ID}, true, client.RequestOptions{})
			if ctc.err == nil && got != nil {
				t.Fatalf("Unexpected error creating server-to-Delivery-Service assignments: %v - alerts: %+v", err, assignResp.Alerts)
			} else if ctc.err != nil {
				found := false
				for _, alert := range assignResp.Alerts.Alerts {
					if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, ctc.err.Error()) {
						found = true
					}
				}
				if !found {
					t.Fatalf("Expected to find an error-level alert relating to '%v', but it wasn't found", ctc.err)
				}
			}

			alerts, _, err = TOSession.DeleteDeliveryServicesRequiredCapability(*ctc.capability.DeliveryServiceID, *ctc.capability.RequiredCapability, client.RequestOptions{})
			if err != nil {
				t.Fatalf("Unexpected error deleting a relationship between a Delivery Service and a Capability it requires: %v - alerts: %+v", err, alerts.Alerts)
			}
		})
	}
}

func CreateTestMSODSServerWithReqCap(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlID", "msods1")
	dsReqCap, _, err := TOSession.GetDeliveryServicesRequiredCapabilities(opts)
	if err != nil {
		t.Fatalf("Unexpected error retrieving relationships between Delivery Services and Capabilities they require: %v - alerts: %+v", err, dsReqCap.Alerts)
	}

	if len(dsReqCap.Response) == 0 {
		t.Fatal("no delivery service required capabilites found for ds msods1")
	}
	dsrc := dsReqCap.Response[0]
	if dsrc.DeliveryServiceID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service/Required Capability relationship with null or undefined Delivery Service ID")
	}

	// Associate origin server to msods1 even though it does not have req cap
	// TODO: DON'T hard-code server hostnames!
	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("hostName", "denver-mso-org-01")
	resp, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("getting server denver-mso-org-01: %v - alerts: %+v", err, resp.Alerts)
	}
	servers := resp.Response
	if len(servers) != 1 {
		t.Fatal("expected 1 server with hostname denver-mso-org-01")
	}

	s := servers[0]
	if s.ID == nil {
		t.Fatal("server 'denver-mso-org-01' had nil ID")
	}

	// Make sure server has no caps to ensure test correctness
	sccsOpts := client.NewRequestOptions()
	sccsOpts.QueryParameters.Set("serverId", strconv.Itoa(*s.ID))
	sccs, _, err := TOSession.GetServerServerCapabilities(sccsOpts)
	if err != nil {
		t.Fatalf("Unexpected error getting Capabilities for server #%d ('denver-mso-org-01'): %v - alerts: %+v", *s.ID, err, sccs.Alerts)
	}
	if len(sccs.Response) != 0 {
		t.Fatal("expected 0 server server capabilities for server denver-mso-org-01")
	}

	// Is origin included in eligible servers even though it doesnt have required capability
	eServers, _, err := TOSession.GetDeliveryServicesEligible(*dsrc.DeliveryServiceID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("get delivery service msods1 eligible servers: %v - alerts: %+v", err, eServers.Alerts)
	}
	found := false
	for _, es := range eServers.Response {
		if es.HostName != nil && *es.HostName == "denver-mso-org-01" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find origin server denver-mso-org-01 to be in eligible server return even though it is missing a required capability")
	}

	if createResp, _, err := TOSession.CreateDeliveryServiceServers(*dsrc.DeliveryServiceID, []int{*s.ID}, true, client.RequestOptions{}); err != nil {
		t.Fatalf("Unexpected error creating server-to-Delivery-Service assignments: %v - alerts: %+v", err, createResp.Alerts)
	}

	// Create new bogus server capability
	if alerts, _, err := TOSession.CreateServerCapability(tc.ServerCapability{Name: "newfun"}, client.RequestOptions{}); err != nil {
		t.Fatalf("cannot create 'newfun' Server Capability: %v - alerts: %+v", err, alerts.Alerts)
	}

	// Attempt to assign to DS should not fail
	if alerts, _, err := TOSession.CreateDeliveryServicesRequiredCapability(tc.DeliveryServicesRequiredCapability{
		DeliveryServiceID:  dsrc.DeliveryServiceID,
		RequiredCapability: util.StrPtr("newfun"),
	}, client.RequestOptions{}); err != nil {
		t.Fatalf("Unexpected error adding Capability 'newfun' as requirement to Delivery Service 'msods1' (#%d): %v - alerts: %+v", *dsrc.DeliveryServiceID, err, alerts.Alerts)
	}

	// Remove required capablity
	if alerts, _, err := TOSession.DeleteDeliveryServicesRequiredCapability(*dsrc.DeliveryServiceID, "newfun", client.RequestOptions{}); err != nil {
		t.Fatalf("Unexpected error removing Capability 'newfun' as requirement from Delivery Service 'msods1' (#%d): %v - alerts: %+v", *dsrc.DeliveryServiceID, err, alerts.Alerts)
	}

	// Delete server capability
	if alerts, _, err := TOSession.DeleteServerCapability("newfun", client.RequestOptions{}); err != nil {
		t.Fatalf("Unexpected error deleteing the 'newfun' Server Capability: %v - alerts: %+v", err, alerts.Alerts)
	}
}

func DeleteTestDeliveryServiceServers(t *testing.T) {
	ds, server := getServerAndDSofSameCDN(t)
	if server.ID == nil {
		t.Fatalf("Got a server with a nil ID: %+v", server)
	}
	if ds.ID == nil {
		t.Fatalf("Got a delivery service with a nil ID %+v", ds)
	}

	resp, _, err := TOSession.CreateDeliveryServiceServers(*ds.ID, []int{*server.ID}, true, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error creating server-to-Delivery-Service assignments: %v - alerts: %+v", err, resp.Alerts)
	}

	dsServers, _, err := TOSession.GetDeliveryServiceServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error retrieving server-to-Delivery-Service assignments: %v - alerts: %+v", err, dsServers.Alerts)
	}

	found := false
	for _, dss := range dsServers.Response {
		if dss.DeliveryService != nil && *dss.DeliveryService == *ds.ID && dss.Server != nil && *dss.Server == *server.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("POST delivery service servers returned success, but ds-server not in GET")
	}

	if ds.Active == tc.DS_ACTIVE {
		ds.Active = tc.DS_INACTIVE
		_, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
		if err != nil {
			t.Errorf("Setting Delivery Service #%d to inactive", *ds.ID)
		}
	}

	if alerts, _, err := TOSession.DeleteDeliveryServiceServer(*ds.ID, *server.ID, client.RequestOptions{}); err != nil {
		t.Errorf("Unexpected error removing server-to-Delivery-Service assignments: %v - alerts: %+v", err, alerts.Alerts)
	}

	dsServers, _, err = TOSession.GetDeliveryServiceServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error retrieving server-to-Delivery-Service assignments: %v - alerts: %+v", err, dsServers.Alerts)
	}

	found = false
	for _, dss := range dsServers.Response {
		if dss.DeliveryService != nil && *dss.DeliveryService == *ds.ID && dss.Server != nil && *dss.Server == *server.ID {
			found = true
			break
		}
	}
	if found {
		t.Error("DELETE delivery service servers returned success, but still in GET")
	}
}

func getServerAndDSofSameCDN(t *testing.T) (tc.DeliveryServiceV4, tc.ServerV4) {
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot GET DeliveryServices: %v", err)
	}
	if len(dses.Response) < 1 {
		t.Fatal("GET DeliveryServices returned no dses, must have at least 1 to test ds-servers")
	}

	resp, _, err := TOSession.GetServers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Servers: %v - alerts: %+v", err, resp.Alerts)
	}
	servers := resp.Response
	if len(servers) < 1 {
		t.Fatal("GET Servers returned no dses, must have at least 1 to test ds-servers")
	}

	for _, ds := range dses.Response {
		for _, s := range servers {
			if ds.CDNName != nil && s.CDNName != nil && *ds.CDNName == *s.CDNName {
				return ds, s
			}
		}
	}
	t.Fatal("expected at least one delivery service and server in the same CDN")

	return tc.DeliveryServiceV4{}, tc.ServerV4{}
}
