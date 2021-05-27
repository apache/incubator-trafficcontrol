package v4

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
	"net/http"
	"strconv"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

type topologiesQueueUpdateTestCase struct {
	Description string
	tc.TopologiesQueueUpdateRequest
}

func TestTopologiesQueueUpdate(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Users, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, DeliveryServices}, func() {
		const topologyName = "mso-topology"
		cdnID, dsID := getCDNIDAndDSID(t)
		InvalidCDNIDIsRejected(t, topologyName)
		InvalidActionIsRejected(t, topologyName, cdnID)
		NonexistentTopologyIsRejected(t, cdnID)
		UpdatesAreQueued(t, topologyName, cdnID, dsID)
	})
}

func getCDNIDAndDSID(t *testing.T) (int64, int) {
	xmlID := "ds-top"
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", xmlID)
	dses, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("unable to get Delivery Service '%s': %v - alerts: %+v", xmlID, err, dses.Alerts)
	}
	if len(dses.Response) < 1 {
		t.Fatalf("deliveryservice with xmlId '%s' not found!", xmlID)
	}
	ds := dses.Response[0]
	if ds.ID == nil {
		t.Fatalf("Traffic Ops returned a representation of a Delivery Service that had null or undefined ID")
	}
	return int64(ds.CDNID), *ds.ID
}

func InvalidCDNIDIsRejected(t *testing.T, topologyName string) {
	testCase := topologiesQueueUpdateTestCase{
		Description:                  "invalid CDN ID",
		TopologiesQueueUpdateRequest: tc.TopologiesQueueUpdateRequest{Action: "queue", CDNID: -1},
	}
	_, reqInf, _ := TOSession.TopologiesQueueUpdate(topologyName, testCase.TopologiesQueueUpdateRequest, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code %d for request with %s, got status code %d", http.StatusBadRequest, testCase.Description, reqInf.StatusCode)
	}
}

func InvalidActionIsRejected(t *testing.T, topologyName string, cdnID int64) {
	testCase := topologiesQueueUpdateTestCase{
		Description:                  "invalid update action",
		TopologiesQueueUpdateRequest: tc.TopologiesQueueUpdateRequest{Action: "requeue", CDNID: cdnID},
	}
	_, reqInf, _ := TOSession.TopologiesQueueUpdate(topologyName, testCase.TopologiesQueueUpdateRequest, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code %d for request with %s, got status code %d", http.StatusBadRequest, testCase.Description, reqInf.StatusCode)
	}
}

func NonexistentTopologyIsRejected(t *testing.T, cdnID int64) {
	const topologyName = "nonexistent"
	testCase := topologiesQueueUpdateTestCase{
		Description:                  "nonexistent topology",
		TopologiesQueueUpdateRequest: tc.TopologiesQueueUpdateRequest{Action: "queue", CDNID: cdnID},
	}
	_, reqInf, _ := TOSession.TopologiesQueueUpdate(topologyName, testCase.TopologiesQueueUpdateRequest, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code %d for request with %s, got status code %d", http.StatusBadRequest, testCase.Description, reqInf.StatusCode)
	}
}

func UpdatesAreQueued(t *testing.T, topologyName string, cdnID int64, dsID int) {
	const action = "queue"
	testCase := topologiesQueueUpdateTestCase{
		Description:                  "invalid update action",
		TopologiesQueueUpdateRequest: tc.TopologiesQueueUpdateRequest{Action: action, CDNID: cdnID},
	}
	resp, _, err := TOSession.TopologiesQueueUpdate(topologyName, testCase.TopologiesQueueUpdateRequest, client.RequestOptions{})
	if err != nil {
		t.Fatalf("received error queueing server updates on Topology '%s': %v - alerts: %+v", topologyName, err, resp.Alerts)
	}
	if resp.Action != action {
		t.Fatalf("expected action %s, received action %s", action, resp.Action)
	}
	if resp.CDNID != cdnID {
		t.Fatalf("expected CDN ID %d, received CDN ID %d", cdnID, resp.CDNID)
	}
	if topologyName != string(resp.Topology) {
		t.Fatalf("expected topology %s, received topology %s", topologyName, resp.Topology)
	}

	opts := client.NewRequestOptions()
	dsIDString := strconv.Itoa(dsID)
	opts.QueryParameters.Set("dsId", dsIDString)
	serversResponse, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("getting servers for Delivery Service with id %d: %v - alerts: %+v", dsID, err, serversResponse.Alerts)
	}
	servers := serversResponse.Response
	for _, server := range servers {
		if server.CDNID == nil || server.HostName == nil || server.UpdPending == nil {
			t.Error("Traffic Ops returned a representation of a server with null or undefined CDN ID and/or HostName and/or Update Pending flag")
			continue
		}
		if *server.CDNID != int(cdnID) {
			continue
		}
		if !*server.UpdPending {
			t.Fatalf("expected UpdPending = %t for server with hostname %s, got UpdPending = %t", true, *server.HostName, *server.UpdPending)
		}
	}
}
