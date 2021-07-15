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
	"sort"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestServerCapabilities(t *testing.T) {
	WithObjs(t, []TCObj{ServerCapabilities}, func() {
		GetTestServerCapabilities(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		rfcTime := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, rfcTime)
		header.Set(rfc.IfUnmodifiedSince, rfcTime)
		SortTestServerCapabilities(t)
		CreateTestServerCapabilityAlreadyExist(t)
		GetTestServerCapabilitiesByInvalidName(t)
		UpdateTestServerCapabilities(t)
		UpdateTestServerCapabilitiesWithHeaders(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestServerCapabilitiesWithHeaders(t, header)
		ValidationTestServerCapabilities(t)
		UpdateTestServerCapabilititesInvalidData(t)
		DeleteTestServerCapabilitiesInvalidName(t)
	})
}

func CreateTestServerCapabilities(t *testing.T) {

	for _, sc := range testData.ServerCapabilities {
		resp, _, err := TOSession.CreateServerCapability(sc, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error creating Server Capability '%s': %v - alerts: %+v", sc.Name, err, resp.Alerts)
		}
	}
}

func CreateTestServerCapabilityAlreadyExist(t *testing.T) {
	if len(testData.ServerCapabilities) < 1 {
		t.Fatal("Need at least one Server Capabilities to test duplicate")
	}
	firstServerCapability := testData.ServerCapabilities[0]
	resp, reqInf, err := TOSession.CreateServerCapability(firstServerCapability, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected server_capability name '%s' already exists. - Alerts %v", firstServerCapability.Name, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
}

func SortTestServerCapabilities(t *testing.T) {
	resp, _, err := TOSession.GetServerCapabilities(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}

	sortedList := make([]string, 0, len(resp.Response))
	for _, sc := range resp.Response {
		sortedList = append(sortedList, sc.Name)
	}

	if !sort.StringsAreSorted(sortedList) {
		t.Errorf("list is not sorted by their names: %v", sortedList)
	}
}

func GetTestServerCapabilities(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, sc := range testData.ServerCapabilities {
		opts.QueryParameters.Set("name", sc.Name)
		resp, _, err := TOSession.GetServerCapabilities(opts)
		if err != nil {
			t.Errorf("cannot get Server Capability: %v - alerts: %+v", err, resp.Alerts)
		}
		if len(resp.Response) != 1 {
			t.Errorf("Expected exactly one Server Capability to exist with name '%s', found: %d", sc.Name, len(resp.Response))
		}
	}

	resp, _, err := TOSession.GetServerCapabilities(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Server Capabilities: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != len(testData.ServerCapabilities) {
		t.Errorf("expected to get %d Server Capabilities, actual: %d", len(testData.ServerCapabilities), len(resp.Response))
	}
}

func GetTestServerCapabilitiesByInvalidName(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "abcd")
	resp, _, err := TOSession.GetServerCapabilities(opts)
	if err != nil {
		t.Errorf("Expected no error: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) > 0 {
		t.Errorf("Expected no response for Get Server Capability by Invalid name, but found some response '%d'", len(resp.Response))
	}
}

func UpdateTestServerCapabilitiesWithHeaders(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header
	resp, _, err := TOSession.GetServerCapabilities(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatal("no server capability in response, quitting")
	}
	originalName := resp.Response[0].Name
	newSCName := "sc-test"
	resp.Response[0].Name = newSCName

	_, reqInf, err := TOSession.UpdateServerCapability(originalName, resp.Response[0], opts)
	if err == nil {
		t.Errorf("Expected error about Precondition Failed, got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
	}
}

func ValidationTestServerCapabilities(t *testing.T) {
	_, _, err := TOSession.CreateServerCapability(tc.ServerCapability{Name: "b@dname"}, client.RequestOptions{})
	if err == nil {
		t.Error("expected POST with invalid name to return an error, actual: nil")
	}
}

func UpdateTestServerCapabilities(t *testing.T) {
	// Get server capability name and edit it to a new name
	resp, _, err := TOSession.GetServerCapabilities(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("no server capability in response, quitting")
	}
	origName := resp.Response[0].Name
	newSCName := "sc-test"
	resp.Response[0].Name = newSCName

	// Update server capability with new name
	updateResponse, _, err := TOSession.UpdateServerCapability(origName, resp.Response[0], client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Server Capability: %v - alerts: %+v", err, updateResponse.Alerts)
	}

	// Get updated name
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", newSCName)
	getResp, _, err := TOSession.GetServerCapabilities(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, getResp.Alerts)
	}
	if len(getResp.Response) == 0 {
		t.Fatalf("no server capability in response, quitting")
	}
	if getResp.Response[0].Name != newSCName {
		t.Errorf("failed to update server capability name, expected: %v but got: %v", newSCName, updateResponse.Response.Name)
	}

	// Set everything back as it was for further testing.
	resp.Response[0].Name = origName
	r, _, err := TOSession.UpdateServerCapability(newSCName, resp.Response[0], client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update server Capability: %v - alerts: %+v", err, r.Alerts)
	}
}

func UpdateTestServerCapabilititesInvalidData(t *testing.T) {
	resp, _, err := TOSession.GetServerCapabilities(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("no server capability in response, quitting")
	}
	newSCName := "sc-test"
	resp.Response[0].Name = newSCName

	// Update server capability with new name
	updateResponse, reqInf, err := TOSession.UpdateServerCapability("invalid", resp.Response[0], client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected cannot find exactly one server capability with the query string provided: %v - alerts: %+v", err, updateResponse.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
}

func DeleteTestServerCapabilities(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, sc := range testData.ServerCapabilities {
		delResp, _, err := TOSession.DeleteServerCapability(sc.Name, client.RequestOptions{})
		if err != nil {
			t.Errorf("cannot delete Server Capability: %v - alerts: %+v", err, delResp.Alerts)
		}
		opts.QueryParameters.Set("name", sc.Name)
		serverCapability, _, err := TOSession.GetServerCapabilities(opts)
		if err != nil {
			t.Errorf("Unexpected error getting Server Capabilities filtered by name '%s' after deletion: %v - alerts: %+v", sc.Name, err, serverCapability.Alerts)
		}
		if len(serverCapability.Response) != 0 {
			t.Errorf("Expected an empty response when filtering for the name of a Server Capability that's been deleted, but found %d matching Server Capabilities", len(serverCapability.Response))
		}
	}
}

func DeleteTestServerCapabilitiesInvalidName(t *testing.T) {

	//invalid name
	delResp, reqInf, err := TOSession.DeleteServerCapability("invalid", client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected no server capability with that key found %v", delResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 error status code, but found %d", reqInf.StatusCode)
	}

	//no parameters
	delResp, reqInf, err = TOSession.DeleteServerCapability("", client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected missing key: name %v", delResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 error status code, but found %d", reqInf.StatusCode)
	}
}
