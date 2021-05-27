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
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
	toclient "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

type topologyTestCase struct {
	testCaseDescription string
	tc.Topology
}

func TestTopologies(t *testing.T) {
	WithObjs(t, []TCObj{Types, CacheGroups, CDNs, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, Servers, ServerCapabilities, ServerServerCapabilitiesForTopologies, Topologies, Tenants, DeliveryServices, TopologyBasedDeliveryServiceRequiredCapabilities}, func() {
		GetTestTopologies(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		rfcTime := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, rfcTime)
		header.Set(rfc.IfUnmodifiedSince, rfcTime)
		UpdateTestTopologies(t)
		UpdateTestTopologiesWithHeaders(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestTopologiesWithHeaders(t, header)
		ValidationTestTopologies(t)
		UpdateValidateTopologyORGServerCacheGroup(t)
		EdgeParentOfEdgeSucceedsWithWarning(t)
		UpdateTopologyName(t)
		GetTopologyWithNonExistentName(t)
		CreateTopologyWithInvalidCacheGroup(t)
		CreateTopologyWithInvalidParentNumber(t)
		CreateTopologyWithoutDescription(t)
		CreateTopologyWithoutName(t)
		CreateTopologyWithoutServers(t)
		CreateTopologyWithDuplicateParents(t)
		CreateTopologyWithNodeAsParentOfItself(t)
		CreateTopologyWithOrgLocAsChildNode(t)
		CreateTopologyWithExistingName(t)
		CreateTopologyWithMidLocTypeWithoutChild(t)
		CRUDTopologyReadOnlyUser(t)
		UpdateTopologyWithCachegroupAssignedToBecomeParentOfItself(t)
		UpdateTopologyWithNoServers(t)
		UpdateTopologyWithInvalidParentNumber(t)
		UpdateTopologyWithMidLocTypeWithoutChild(t)
		UpdateTopologyWithOrgLocAsChildNode(t)
		UpdateTopologyWithSameParentAndSecondaryParent(t)
		DeleteTopologyWithNonExistentName(t)
		DeleteTopologyBeingUsedByDeliveryService(t)
	})
}

func CreateTestTopologies(t *testing.T) {
	for _, topology := range testData.Topologies {
		postResponse, _, err := TOSession.CreateTopology(topology, toclient.RequestOptions{})
		if err != nil {
			t.Fatalf("could not create Topology: %v - alerts: %+v", err, postResponse.Alerts)
		}
		postResponse.Response.LastUpdated = nil
		if !reflect.DeepEqual(topology, postResponse.Response) {
			t.Fatalf("Topology in response should be the same as the one POSTed. expected: %v\nactual: %v", topology, postResponse.Response)
		}
	}
}

func GetTestTopologies(t *testing.T) {
	if len(testData.Topologies) < 1 {
		t.Fatalf("test data has no topologies, can't test")
	}
	topos, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("expected error to be nil, actual: %v - alerts: %+v", err, topos.Alerts)
	}
	if len(topos.Response) != len(testData.Topologies) {
		t.Errorf("expected %d Topologies to exist in Traffic Ops, actual: %d", len(testData.Topologies), len(topos.Response))
	}
}

func UpdateTestTopologiesWithHeaders(t *testing.T, header http.Header) {
	originalName := "top-used-by-cdn1-and-cdn2"
	newName := "blah"

	// Retrieve the Topology by name so we can get the id for Update()
	opts := toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", originalName)
	resp, _, err := TOSession.GetTopologies(opts)
	if err != nil {
		t.Errorf("cannot get Topology by name '%s': %v - alerts: %+v", originalName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Topology to exist with name '%s', found: %d", originalName, len(resp.Response))
	}
	resp.Response[0].Name = newName
	_, reqInf, err := TOSession.UpdateTopology(originalName, resp.Response[0], toclient.RequestOptions{Header: header})
	if err == nil {
		t.Errorf("Expected error about Precondition Failed, got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
	}
}

func EdgeParentOfEdgeSucceedsWithWarning(t *testing.T) {
	testCase := topologyTestCase{testCaseDescription: "an edge parenting a mid", Topology: tc.Topology{
		Name:        "edge-parent-of-edge",
		Description: "An edge is a parent, which is technically valid, but we will warn the user in case it was a mistake",
		Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{1}},
			{Cachegroup: "cachegroup2", Parents: []int{}},
		}}}
	response, _, err := TOSession.CreateTopology(testCase.Topology, toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error creating Topology for '%s': %v - alerts: %+v", testCase.testCaseDescription, err, response.Alerts)
	}
	containsWarning := false
	for _, alert := range response.Alerts.Alerts {
		if alert.Level == tc.WarnLevel.String() {
			containsWarning = true
		}
	}
	if !containsWarning {
		t.Fatalf("expected a warning-level alert message in the response, actual: %v", response.Alerts)
	}
	delResp, _, err := TOSession.DeleteTopology(testCase.Topology.Name, toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot delete Topology: %v - alerts: %+v", err, delResp.Alerts)
	}
}

func ValidationTestTopologies(t *testing.T) {
	invalidTopologyTestCases := []topologyTestCase{
		{testCaseDescription: "no nodes", Topology: tc.Topology{Name: "empty-top", Description: "Invalid because there are no nodes", Nodes: []tc.TopologyNode{}}},
		{testCaseDescription: "a node listing itself as a parent", Topology: tc.Topology{Name: "self-parent", Description: "Invalid because a node lists itself as a parent", Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{1}},
			{Cachegroup: "parentCachegroup", Parents: []int{1}},
		}}},
		{testCaseDescription: "duplicate parents", Topology: tc.Topology{}},
		{testCaseDescription: "too many parents", Topology: tc.Topology{Name: "duplicate-parents", Description: "Invalid because a node lists the same parent twice", Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{1, 1}},
			{Cachegroup: "parentCachegroup", Parents: []int{}},
		}}},
		{testCaseDescription: "too many parents", Topology: tc.Topology{Name: "too-many-parents", Description: "Invalid because a node has more than 2 parents", Nodes: []tc.TopologyNode{
			{Cachegroup: "parentCachegroup", Parents: []int{}},
			{Cachegroup: "secondaryCachegroup", Parents: []int{}},
			{Cachegroup: "parentCachegroup2", Parents: []int{}},
			{Cachegroup: "cachegroup1", Parents: []int{0, 1, 2}},
		}}},
		{testCaseDescription: "an edge parenting a mid", Topology: tc.Topology{Name: "edge-parent-of-mid", Description: "Invalid because an edge is a parent of a mid", Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{1}},
			{Cachegroup: "parentCachegroup", Parents: []int{2}},
			{Cachegroup: "cachegroup2", Parents: []int{}},
		}}},
		{testCaseDescription: "a leaf mid", Topology: tc.Topology{Name: "leaf-mid", Description: "Invalid because a mid is a leaf node", Nodes: []tc.TopologyNode{
			{Cachegroup: "parentCachegroup", Parents: []int{1}},
			{Cachegroup: "secondaryCachegroup", Parents: []int{}},
		}}},
		{testCaseDescription: "cyclical nodes", Topology: tc.Topology{Name: "cyclical-nodes", Description: "Invalid because it contains cycles", Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{1, 2}},
			{Cachegroup: "parentCachegroup", Parents: []int{2}},
			{Cachegroup: "secondaryCachegroup", Parents: []int{1}},
		}}},
		{testCaseDescription: "a cycle across topologies", Topology: tc.Topology{Name: "cycle-with-4-tier-topology", Description: `Invalid because it contains a cycle when combined with the "4-tiers" topology`, Nodes: []tc.TopologyNode{
			{Cachegroup: "parentCachegroup", Parents: []int{1}},
			{Cachegroup: "parentCachegroup2", Parents: []int{}},
			{Cachegroup: "cachegroup1", Parents: []int{0}},
		}}},
		{testCaseDescription: "a cycle across cache groups", Topology: tc.Topology{Name: "cycle-with-non-topology-cachegroups", Description: "Invalid because it contains a cycle when combined with a topology constructed from cache group parentage", Nodes: []tc.TopologyNode{
			{Cachegroup: "edge-parent1", Parents: []int{1}},
			{Cachegroup: "has-edge-parent1", Parents: []int{}},
		}}},
		{testCaseDescription: "a nonexistent cache group", Topology: tc.Topology{Name: "nonexistent-cg", Description: "Invalid because it references a cache group that does not exist", Nodes: []tc.TopologyNode{
			{Cachegroup: "legitcachegroup", Parents: []int{}},
		}}},
		{testCaseDescription: "an out-of-bounds parent index", Topology: tc.Topology{Name: "oob-parent", Description: "Invalid because it contains an out-of-bounds parent", Nodes: []tc.TopologyNode{
			{Cachegroup: "cachegroup1", Parents: []int{7}},
		}}},
		{testCaseDescription: "a cachegroup containing no servers", Topology: tc.Topology{Name: "empty-cg", Description: `Invalid because it contains a cachegroup, fallback3, that contains no servers`, Nodes: []tc.TopologyNode{
			{Cachegroup: "parentCachegroup", Parents: []int{}},
			{Cachegroup: "parentCachegroup2", Parents: []int{}},
			{Cachegroup: "fallback3", Parents: []int{0, 1}},
		}}},
	}

	for _, testCase := range invalidTopologyTestCases {
		_, reqInf, err := TOSession.CreateTopology(testCase.Topology, toclient.RequestOptions{})
		if err == nil {
			t.Fatalf("expected POST with %v to return an error, actual: nil", testCase.testCaseDescription)
		}
		statusCode := reqInf.StatusCode
		if statusCode < 400 || statusCode >= 500 {
			t.Fatalf("Expected a 400-level status code for topology %s but got %d", testCase.Topology.Name, statusCode)
		}
	}
}

func updateSingleTopology(topology tc.Topology) error {
	updateResponse, _, err := TOSession.UpdateTopology(topology.Name, topology, toclient.RequestOptions{})
	if err != nil {
		return fmt.Errorf("cannot put Topology: %v - alerts: %+v", err, updateResponse.Alerts)
	}
	updateResponse.Response.LastUpdated = nil
	if !reflect.DeepEqual(topology, updateResponse.Response) {
		return fmt.Errorf("Topologies should be equal after updating. expected: %v\nactual: %v", topology, updateResponse.Response)
	}
	return nil
}

func UpdateTestTopologies(t *testing.T) {
	for _, topology := range testData.Topologies {
		if err := updateSingleTopology(topology); err != nil {
			t.Fatalf(err.Error())
		}
	}

	opts := toclient.NewRequestOptions()

	// attempt to add cachegroup that doesn't meet DS required capabilities
	opts.QueryParameters.Set("name", "top-for-ds-req")
	resp, _, err := TOSession.GetTopologies(opts)
	if err != nil {
		t.Fatalf("cannot get Topology: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Topology to exist with name 'top-for-ds-req', found: %d", len(resp.Response))
	}
	top := resp.Response[0]

	top.Nodes = append(top.Nodes, tc.TopologyNode{Cachegroup: "cachegroup1", Parents: []int{0}})
	_, _, err = TOSession.UpdateTopology(top.Name, top, toclient.RequestOptions{})
	if err == nil {
		t.Errorf("making invalid update to topology - expected: error, actual: nil")
	}

	// attempt to add a cachegroup that only has caches in one CDN while the topology is assigned to DSes from multiple CDNs
	opts.QueryParameters.Set("name", "top-used-by-cdn1-and-cdn2")
	resp, _, err = TOSession.GetTopologies(opts)
	if err != nil {
		t.Fatalf("cannot get Topology: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Topology to exist with name 'top-for-ds-req', found: %d", len(resp.Response))
	}
	top = resp.Response[0]

	opts = toclient.NewRequestOptions()
	opts.QueryParameters.Set("topology", "top-used-by-cdn1-and-cdn2")
	dses, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	if len(dses.Response) < 2 {
		t.Fatalf("expected at least 2 delivery services assigned to topology top-used-by-cdn1-and-cdn2, actual: %d", len(dses.Response))
	}
	foundCDN1 := false
	foundCDN2 := false
	for _, ds := range dses.Response {
		if ds.CDNName == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service that had null or undefined CDN Name")
			continue
		}
		if *ds.CDNName == "cdn1" {
			foundCDN1 = true
		} else if *ds.CDNName == "cdn2" {
			foundCDN2 = true
		}
	}
	if !foundCDN1 || !foundCDN2 {
		t.Fatalf("expected delivery services assigned to topology top-used-by-cdn1-and-cdn2 to be assigned to cdn1 and cdn2")
	}
	opts = toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", "cdn1-only")
	cgs, _, err := TOSession.GetCacheGroups(opts)
	if err != nil {
		t.Fatalf("unable to GET cachegroup by name: %v", err)
	}
	if len(cgs.Response) != 1 {
		t.Fatalf("expected: to get 1 cachegroup named 'cdn1-only', actual: got %d", len(cgs.Response))
	}
	if cgs.Response[0].ID == nil {
		t.Fatal("Traffic Ops returned a representation for Cache Group 'cdn1-only' that had a null or undefined ID")
	}
	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Add("cachegroup", strconv.Itoa(*cgs.Response[0].ID))
	servers, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("unable to get servers by Cache Group ID: %v - alerts: %+v", err, servers.Alerts)
	}
	for _, s := range servers.Response {
		if s.Cachegroup == nil || s.CDNName == nil {
			t.Error("Traffic Ops returned a representation of a server with null or undefined Cache Group and/or CDN name")
			continue
		}
		if *s.Cachegroup != "cdn1-only" {
			t.Fatalf("GET servers by cachegroup 'cdn1-only' - expected: only servers in cachegroup 'cdn1-only', actual: got server in %s", *s.Cachegroup)
		}
		if *s.CDNName != "cdn1" {
			t.Fatalf("expected: servers in cachegroup 'cdn1-only' to only be in cdn1, actual: servers in cdn %s", *s.CDNName)
		}
	}
	top.Nodes = append(top.Nodes, tc.TopologyNode{
		Cachegroup: "cdn1-only",
		Parents:    []int{0},
	})
	_, _, err = TOSession.UpdateTopology(top.Name, top, toclient.RequestOptions{})
	if err == nil {
		t.Errorf("making invalid update to topology (cachegroup contains only servers from cdn1 while the topology is assigned to delivery services in cdn1 and cdn2) - expected: error, actual: nil")
	}
}

func UpdateValidateTopologyORGServerCacheGroup(t *testing.T) {
	opts := toclient.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "ds-top")

	//Get the correct DS
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) < 1 {
		t.Fatalf("Expected exactly one Delivery Service to exist with XMLID 'ds-top', found: %d", len(resp.Response))
	}
	remoteDS := resp.Response[0]
	if remoteDS.Topology == nil || remoteDS.ID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service that had null or undefined Topology and/or ID")
	}

	//Assign ORG server to DS
	assignServer := []string{"denver-mso-org-01"}
	assignResponse, _, err := TOSession.AssignServersToDeliveryService(assignServer, remoteDS.XMLID, toclient.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error assigning server 'denver-mso-org-01' to Delivery Service '%s': %v - alerts: %+v", remoteDS.XMLID, err, assignResponse.Alerts)
	}

	//Get Topology node to update and remove ORG server nodes
	origTopo := *remoteDS.Topology
	opts = toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", origTopo)
	topResp, _, err := TOSession.GetTopologies(opts)
	if err != nil {
		t.Fatalf("couldn't find any Topologies: %v - alerts: %+v", err, topResp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Topology to exist with name '%s', found: %d", origTopo, len(resp.Response))
	}
	topo := topResp.Response[0]

	// remove org server cachegroup
	var p []int
	newNodes := []tc.TopologyNode{{Id: 0, Cachegroup: "topology-edge-cg-01", Parents: p, LastUpdated: nil}}
	if *remoteDS.Topology == topo.Name {
		topo.Nodes = newNodes
	}
	updTopResp, _, err := TOSession.UpdateTopology(*remoteDS.Topology, topo, toclient.RequestOptions{})
	if err == nil {
		t.Fatalf("should not update Topology: %s to %s, but update was a success", *remoteDS.Topology, newNodes[0].Cachegroup)
	} else if !alertsHaveError(updTopResp.Alerts.Alerts, "ORG servers are assigned to delivery services that use this topology, and their cachegroups cannot be removed:") {
		t.Errorf("expected error messsage containing: \"ORG servers are assigned to delivery services that use this topology, and their cachegroups cannot be removed\", got: %v - alets: %+v", err, updTopResp.Alerts)
	}

	//Remove org server assignment and reset DS back to as it was for further testing
	opts = toclient.NewRequestOptions()
	opts.QueryParameters.Set("hostName", "denver-mso-org-01")
	serverResp, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Errorf("Unexpected error getting servers filtered by Host Name 'denver-mso-org-01': %v - alerts: %+v", err, serverResp.Alerts)
	}
	if len(serverResp.Response) == 0 {
		t.Fatal("no servers in response, quitting")
	}
	if serverResp.Response[0].ID == nil {
		t.Fatal("ID of the response server is nil, quitting")
	}
	alerts, _, err := TOSession.DeleteDeliveryServiceServer(*remoteDS.ID, *serverResp.Response[0].ID, toclient.RequestOptions{})
	if err != nil {
		t.Errorf("cannot remove server #%d from Delivery Service #%d: %v - alerts: %+v", *serverResp.Response[0].ID, *remoteDS.ID, err, alerts.Alerts)
	}
}

func UpdateTopologyName(t *testing.T) {
	currentTopologyName := "top-used-by-cdn1-and-cdn2"

	// Get details on existing topology
	opts := toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", currentTopologyName)
	resp, _, err := TOSession.GetTopologies(opts)
	if err != nil {
		t.Errorf("unable to get Topology filtered by name '%s': %v - alerts: %+v", currentTopologyName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Topology to exist with name '%s', found: %d", currentTopologyName, len(resp.Response))
	}
	newTopologyName := "test-topology"
	resp.Response[0].Name = newTopologyName

	// Update topology with new name
	updateResponse, _, err := TOSession.UpdateTopology(currentTopologyName, resp.Response[0], toclient.RequestOptions{})
	if err != nil {
		t.Errorf("cannot updated Topology: %v - alerts: %+v", err, updateResponse.Alerts)
	}
	if updateResponse.Response.Name != newTopologyName {
		t.Errorf("update topology name failed, expected: %v but got:%v", newTopologyName, updateResponse.Response.Name)
	}

	//To check whether the primary key change trickled down to DS table
	opts = toclient.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "top-ds-in-cdn2")
	resp1, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("failed to get details on DS: %v - alerts: %+v", err, resp1.Alerts)
	}
	if len(resp1.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service to exist with XMLID 'top-ds-in-cdn2', found: %d", len(resp1.Response))
	}
	if resp1.Response[0].Topology == nil {
		t.Fatal("Expected Delivery Service 'top-ds-in-cdn2' to have a Topology, but it was null or undefined in response from Traffic Ops")
	}
	if *resp1.Response[0].Topology != newTopologyName {
		t.Errorf("topology name change failed to trickle to delivery service table, expected: %s but got: %s", newTopologyName, *resp1.Response[0].Topology)
	}

	// Set everything back as it was for further testing.
	resp.Response[0].Name = currentTopologyName
	r, _, err := TOSession.UpdateTopology(newTopologyName, resp.Response[0], toclient.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Topology: %v - alerts: %+v", err, r.Alerts)
	}
}

func DeleteTestTopologies(t *testing.T) {
	for _, top := range testData.Topologies {
		delResp, _, err := TOSession.DeleteTopology(top.Name, toclient.RequestOptions{})
		if err != nil {
			t.Fatalf("cannot delete Topology: %v - alerts: %+v", err, delResp.Alerts)
		}
		opts := toclient.NewRequestOptions()
		opts.QueryParameters.Set("limit", "1")
		deleteLog, _, err := TOSession.GetLogs(opts)
		if err != nil {
			t.Fatalf("unable to get latest audit log entry: %v - alerts: %+v", err, deleteLog.Alerts)
		}
		if len(deleteLog.Response) != 1 {
			t.Fatalf("log entry length - expected: 1, actual: %d", len(deleteLog.Response))
		}
		if deleteLog.Response[0].Message == nil {
			t.Fatal("Traffic Ops responded with a representation of a log entry with null or undefined message")
		}
		if !strings.Contains(*deleteLog.Response[0].Message, top.Name) {
			t.Errorf("topology deletion audit log entry - expected: message containing topology name '%s', actual: %s", top.Name, *deleteLog.Response[0].Message)
		}

		opts.QueryParameters.Del("limit")
		opts.QueryParameters.Set("name", top.Name)
		resp, _, err := TOSession.GetTopologies(opts)
		if err != nil {
			t.Errorf("Unexpected error trying to fetch Topologies after deletion: %v - alerts: %+v", err, resp.Alerts)
		}
		if len(resp.Response) != 0 {
			t.Fatalf("expected not to find deleted Topology '%s' in Traffic Ops, but %d Topologies were found by that name", top.Name, len(resp.Response))
		}
	}
}

func GetTopologyWithNonExistentName(t *testing.T) {
	opts := toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", "non-existent-topology")
	resp, reqInf, _ := TOSession.GetTopologies(opts)
	if len(resp.Response) != 0 {
		t.Errorf("expected nothing in the response, but got %d Topologies", len(resp.Response))
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("expected a 200 response code, but got %d", reqInf.StatusCode)
	}
}

func CreateTopologyWithInvalidCacheGroup(t *testing.T) {
	nodes := make([]tc.TopologyNode, 0)
	node := tc.TopologyNode{
		Cachegroup: "non-existent-cachegroup",
		Parents:    nil,
	}
	nodes = append(nodes, node)
	top := tc.Topology{
		Description: "blah",
		Name:        "invalid-cachegroup-topology",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a '400 Bad Request' response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about the cachegroup name not being valid, but got none")
	}
}

func CreateTopologyWithInvalidParentNumber(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}
	if cachegroupName == "" {
		t.Fatal("No servers could be found in any Cache Groups - need at least one valid server to test creating a topology with an invalid parent number")
	}
	node := tc.TopologyNode{
		Cachegroup: cachegroupName,
		Parents:    []int{100},
	}
	nodes := make([]tc.TopologyNode, 1)
	nodes = append(nodes, node)
	top := tc.Topology{
		Description: "blah",
		Name:        "invalid-parent-topology",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a '400 Bad Request' response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about the parent not being valid, but got none")
	}
}

func CreateTopologyWithoutDescription(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}
	if cachegroupName == "" {
		t.Fatal("Failed to find a single Cache Group with any Servers in it")
	}

	node := tc.TopologyNode{
		Cachegroup: cachegroupName,
		Parents:    nil,
	}
	nodes := make([]tc.TopologyNode, 0, 1)
	nodes = append(nodes, node)
	top := tc.Topology{
		Name:  "topology-without-description",
		Nodes: nodes,
	}
	resp, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("expected a 200 response code, but got %d", reqInf.StatusCode)
	}
	if err != nil {
		t.Errorf("no error expected about description being empty, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	alerts, _, err := TOSession.DeleteTopology(top.Name, toclient.RequestOptions{})
	if err != nil {
		t.Errorf("couldn't delete Topology with name '%s': %v - alerts: %+v", top.Name, err, alerts.Alerts)
	}
}

func CreateTopologyWithoutName(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	node := tc.TopologyNode{
		Cachegroup: cachegroupName,
		Parents:    nil,
	}
	nodes := make([]tc.TopologyNode, 1)
	nodes = append(nodes, node)
	top := tc.Topology{
		Description: "description",
		Nodes:       nodes,
	}

	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology name being empty expected, but got none")
	}
}

func CreateTopologyWithoutServers(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) == 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	node := tc.TopologyNode{
		Cachegroup: cachegroupName,
		Parents:    nil,
	}
	nodes := make([]tc.TopologyNode, 1)
	nodes = append(nodes, node)
	top := tc.Topology{
		Name:        "topology_without_servers",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology containing no servers expected, but got none")
	}
}

func CreateTopologyWithDuplicateParents(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && parentName != *cg.Name && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil {
				parentName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
		{
			Cachegroup: cachegroupName,
			Parents:    []int{0, 0},
		},
	}
	top := tc.Topology{
		Name:        "topology_with_duplicate_parents",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having duplicate parent expected, but got none")
	}
}

func CreateTopologyWithNodeAsParentOfItself(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && parentName != *cg.Name {
				parentName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
		{
			Cachegroup: cachegroupName,
			Parents:    []int{0, 1},
		},
	}
	top := tc.Topology{
		Name:        "topology_with_node_as_parent_of_itself",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having node as parent of itself expected, but got none")
	}
}

func CreateTopologyWithOrgLocAsChildNode(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				parentName = *cg.Name
				break
			}
		}
	}

	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && parentName != *cg.Name && cg.Type != nil && *cg.Type == tc.CacheGroupOriginTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
		{
			Cachegroup: cachegroupName,
			Parents:    []int{0},
		},
	}
	top := tc.Topology{
		Name:        "topology_with_orgloc_as_child_node",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having ord_loc node as child expected, but got none")
	}
}

func CreateTopologyWithExistingName(t *testing.T) {
	resp, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("could not get Topologies: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("expected 1 or more topologies in response, but got 0")
	}
	_, reqInf, err := TOSession.CreateTopology(resp.Response[0], toclient.RequestOptions{})
	if err == nil {
		t.Errorf("expected error about creating topology with same name, but got none")
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
}

func CreateTopologyWithMidLocTypeWithoutChild(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting Cache Groups: %v - alerts: %+v", err, cacheGroups.Alerts)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatal("no cachegroups in response")
	}

	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupMidTypeName {
				parentName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
	}
	top := tc.Topology{
		Name:        "topology_with_midloc_and_no_child_nodes",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having mid_loc node and no children expected, but got none")
	}
}

func CRUDTopologyReadOnlyUser(t *testing.T) {
	opts := toclient.NewRequestOptions()
	opts.QueryParameters.Set("name", "root")
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Fatalf("couldn't get the root tenant ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to have the name 'root', found: %d", len(resp.Response))
	}

	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	user := tc.User{
		Username:             util.StrPtr("test_user"),
		RegistrationSent:     tc.TimeNoModFromTime(time.Now()),
		LocalPassword:        util.StrPtr("test_pa$$word"),
		ConfirmLocalPassword: util.StrPtr("test_pa$$word"),
		RoleName:             util.StrPtr("read-only user"),
	}
	user.Email = util.StrPtr("email@domain.com")
	user.TenantID = util.IntPtr(resp.Response[0].ID)
	user.FullName = util.StrPtr("firstName LastName")

	u, _, err := TOSession.CreateUser(user, client.RequestOptions{})
	if err != nil {
		t.Fatalf("could not create read-only user: %v - alerts: %+v", err, u.Alerts)
	}
	client, _, err := toclient.LoginWithAgent(TOSession.URL, "test_user", "test_pa$$word", true, "to-api-v4-client-tests/tenant4user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to log in with test_user: %v", err.Error())
	}
	nodes := []tc.TopologyNode{
		{
			Cachegroup: "parentName",
			Parents:    []int{},
		},
	}
	top := tc.Topology{
		Name:        "topology",
		Description: "description",
		Nodes:       nodes,
	}
	// Create
	_, reqInf, err := client.CreateTopology(top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 Forbidden error, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about Read-Only users not being able to create topologies, but got nothing")
		alerts, _, err := client.DeleteTopology(top.Name, toclient.RequestOptions{})
		if err != nil {
			t.Errorf("could not delete Topology '%s': %v - alerts: %+v", top.Name, err, alerts.Alerts)
		}
	}

	// Read
	tops, _, err := client.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get Topologies: %v - alerts: %+v", err, tops.Alerts)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}

	// Update
	updatedTop := tops.Response[0]
	updatedTop.Description = "updated description"
	_, reqInf, err = client.UpdateTopology(updatedTop.Name, updatedTop, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 Forbidden error, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about Read-Only users not being able to update topologies, but got nothing")
	}

	// Delete
	_, reqInf, err = client.DeleteTopology(tops.Response[0].Name, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 Forbidden error, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about Read-Only users not being able to delete topologies, but got nothing")
	}

	if u.Response.Username != nil {
		ForceDeleteTestUsersByUsernames(t, []string{"test_user"})
	}
}

func UpdateTopologyWithCachegroupAssignedToBecomeParentOfItself(t *testing.T) {
	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get Topologies: %v - alerts: %+v", err, tops.Alerts)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}
	tp := tops.Response[0]

	// create a list of indices consisting of all the node indices,
	// so that when we assign this parent list wile updating,
	// TO complains about the parent of a node being the same as itself
	parents := make([]int, 0, len(tp.Nodes))
	for i := range tp.Nodes {
		parents = append(parents, i)
	}
	nodes := tp.Nodes
	for i := range nodes {
		nodes[i].Parents = parents
	}
	tp.Nodes = nodes

	tops.Response[0] = tp
	_, reqInf, err := TOSession.UpdateTopology(tops.Response[0].Name, tops.Response[0], toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having parents the same as children expected, but got none")
	}
}

func UpdateTopologyWithSameParentAndSecondaryParent(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				parentName = *cg.Name
				break
			}
		}
	}

	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && parentName != *cg.Name && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
		{
			Cachegroup: cachegroupName,
			Parents:    []int{0, 0},
		},
	}

	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get Topologies: %v - alerts: %+v", err, tops.Alerts)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}

	top := tc.Topology{
		Description: tops.Response[0].Description,
		Name:        tops.Response[0].Name,
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.UpdateTopology(top.Name, top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology's cachegroup having same primary and secondary parents, but got none")
	}
}

func UpdateTopologyWithOrgLocAsChildNode(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	cachegroupName := ""
	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				parentName = *cg.Name
				break
			}
		}
	}

	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && parentName != *cg.Name && cg.Type != nil && *cg.Type == tc.CacheGroupOriginTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
		{
			Cachegroup: cachegroupName,
			Parents:    []int{0},
		},
	}

	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get topologies: %v", err)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}
	top := tc.Topology{
		Name:        "topology_with_orgloc_as_child_node",
		Description: "description",
		Nodes:       nodes,
	}
	_, reqInf, err := TOSession.UpdateTopology(tops.Response[0].Name, top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having ord_loc node as child expected, but got none")
	}
}

func UpdateTopologyWithMidLocTypeWithoutChild(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}

	parentName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) != 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupMidTypeName {
				parentName = *cg.Name
				break
			}
		}
	}

	nodes := []tc.TopologyNode{
		{
			Cachegroup: parentName,
			Parents:    []int{},
		},
	}
	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get topologies: %v", err)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}

	top := tc.Topology{
		Name:        tops.Response[0].Name,
		Description: tops.Response[0].Description,
		Nodes:       nodes,
	}

	_, reqInf, err := TOSession.UpdateTopology(tops.Response[0].Name, top, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having mid_loc node and no children expected, but got none")
	}
}

func UpdateTopologyWithInvalidParentNumber(t *testing.T) {
	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get Topologies: %v - alerts: %+v", err, tops.Alerts)
	}
	if len(tops.Response) == 0 {
		t.Fatal("expected to get one or more topologies in the response, but got none")
	}
	tp := tops.Response[0]
	parents := make([]int, 0)
	parents = append(parents, len(tp.Nodes)+1)
	for i := range tp.Nodes {
		tp.Nodes[i].Parents = parents
	}
	_, reqInf, err := TOSession.UpdateTopology(tp.Name, tp, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a '400 Bad Request' response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("expected error about the parent not being valid, but got none")
	}
}

func UpdateTopologyWithNoServers(t *testing.T) {
	cacheGroups, _, err := TOSession.GetCacheGroups(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error while getting cachegroups: %v", err)
	}
	if len(cacheGroups.Response) == 0 {
		t.Fatalf("no cachegroups in response")
	}
	nodes := make([]tc.TopologyNode, 0)

	cachegroupName := ""
	opts := toclient.NewRequestOptions()
	for _, cg := range cacheGroups.Response {
		if cg.ID == nil {
			t.Error("Traffic Ops returned a representation for a Cache Group with null or undefined ID")
			continue
		}
		opts.QueryParameters.Set("cachegroup", strconv.Itoa(*cg.ID))
		resp, _, _ := TOSession.GetServers(opts)
		if len(resp.Response) == 0 {
			if cg.Name != nil && cg.Type != nil && *cg.Type == tc.CacheGroupEdgeTypeName {
				cachegroupName = *cg.Name
				break
			}
		}
	}
	node := tc.TopologyNode{
		Cachegroup: cachegroupName,
		Parents:    nil,
	}
	nodes = append(nodes, node)

	tops, _, err := TOSession.GetTopologies(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("error getting Topologies: %v - alerts: %+v", err, tops.Alerts)
	}
	if len(tops.Response) == 0 {
		t.Fatalf("expected 1 or more topologies in response, but got none")
	}
	tops.Response[0].Nodes = nodes
	_, reqInf, err := TOSession.UpdateTopology(tops.Response[0].Name, tops.Response[0], toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology having no servers expected, but got none")
	}
}

func DeleteTopologyBeingUsedByDeliveryService(t *testing.T) {
	ds, _, err := TOSession.GetDeliveryServices(toclient.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get Delivery Services: %v - alerts: %+v", err, ds.Alerts)
	}
	if len(ds.Response) == 0 {
		t.Fatalf("expected one or more ds's in the response, got none")
	}
	topologyName := ""
	for _, d := range ds.Response {
		if d.Topology != nil && *d.Topology != "" {
			topologyName = *d.Topology
			break
		}
	}
	if topologyName == "" {
		t.Error("Expected at least one Delivery Service to have a Topology, but none did")
	}
	_, reqInf, err := TOSession.DeleteTopology(topologyName, toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology being used by a ds expected, but got none")
	}
}

func DeleteTopologyWithNonExistentName(t *testing.T) {
	_, reqInf, err := TOSession.DeleteTopology("non existent name", toclient.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 response code, but got %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("error about topology not being present expected, but got none")
	}
}
