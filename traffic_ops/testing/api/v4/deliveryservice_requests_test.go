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
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	tc "github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

const (
	dsrGood      = 0
	dsrBadTenant = 1
	dsrRequired  = 2
	dsrDraft     = 3
)

// this resets the IDs of things attached to a DS, which needs to be done
// because the WithObjs flow destroys and recreates those object IDs
// non-deterministically with each test - BUT, the client method permanently
// alters the DSR structures by adding these referential IDs. Older clients
// got away with it by not making 'DeliveryService' a pointer, but to add
// original/requested fields you need to sometimes allow each to be nil, so
// this is a problem that needs to be solved at some point.
// A better solution _might_ be to reload all the test fixtures every time
// to wipe any and all referential modifications made to any test data, but
// for now that's overkill.
func resetDS(ds *tc.DeliveryServiceV4) {
	if ds == nil {
		return
	}
	ds.CDNID = 0
	ds.ID = nil
	ds.ProfileID = nil
	ds.TenantID = 0
	ds.TypeID = 0
}

func TestDeliveryServiceRequests(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants, DeliveryServiceRequests}, func() {
		GetTestDeliveryServiceRequestsIMS(t)
		GetTestDeliveryServiceRequests(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, time)
		header.Set(rfc.IfUnmodifiedSince, time)
		UpdateTestDeliveryServiceRequests(t)
		UpdateTestDeliveryServiceRequestsWithHeaders(t, header)
		GetTestDeliveryServiceRequestsIMSAfterChange(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestDeliveryServiceRequestsWithHeaders(t, header)
	})
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func UpdateTestDeliveryServiceRequestsWithHeaders(t *testing.T, header http.Header) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test updating them using headers", dsrGood+1)
	}
	// Retrieve the DeliveryServiceRequest by name so we can get the id for the Update
	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}
	resetDS(ds)
	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no Delivery Service", dsrGood)
	}
	opts := client.NewRequestOptions()
	opts.Header = header
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Service Request by XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatal("Length of GET DeliveryServiceRequest is 0")
	}
	respDSR := resp.Response[0]
	if respDSR.ID == nil {
		t.Fatalf("Got a DSR for XML ID '%s' that had a nil ID", ds.XMLID)
	}
	if respDSR.ChangeType != dsr.ChangeType {
		t.Fatalf("remote representation of DSR with XMLID '%s' differed from stored data", ds.XMLID)
	}
	var respDS *tc.DeliveryServiceV4
	if respDSR.ChangeType == tc.DSRChangeTypeDelete {
		respDS = respDSR.Original
	} else {
		respDS = respDSR.Requested
	}

	respDS.DisplayName = "new display name"
	opts.QueryParameters.Del("xmlId")
	_, reqInf, err := TOSession.UpdateDeliveryServiceRequest(*respDSR.ID, respDSR, opts)
	if err == nil {
		t.Errorf("Expected error about precondition failed, but got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
	}
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func GetTestDeliveryServiceRequestsIMSAfterChange(t *testing.T, header http.Header) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test updating them with IMS", dsrGood+1)
	}
	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}

	resetDS(ds)
	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no Delivery Service", dsrGood)
	}

	opts := client.NewRequestOptions()
	opts.Header = header
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, reqInf, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}
	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)
	opts.Header.Set(rfc.IfModifiedSince, timeStr)
	resp, reqInf, err = TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func CreateTestDeliveryServiceRequests(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test creating Delivery Service Requests", dsrGood+1)
	}
	dsr := testData.DeliveryServiceRequests[dsrGood]
	resetDS(dsr.Original)
	resetDS(dsr.Requested)
	respDSR, _, err := TOSession.CreateDeliveryServiceRequest(dsr, client.RequestOptions{})
	if err != nil {
		t.Errorf("could not create Delivery Service Requests: %v - alerts: %+v", err, respDSR.Alerts)
	}

}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func TestDeliveryServiceRequestRequired(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		if len(testData.DeliveryServiceRequests) < dsrRequired+1 {
			t.Fatalf("Need at least %d Delivery Service Requests to test creating a Delivery Service Request missing required fields", dsrRequired+1)
		}
		dsr := testData.DeliveryServiceRequests[dsrRequired]
		resetDS(dsr.Original)
		resetDS(dsr.Requested)
		_, _, err := TOSession.CreateDeliveryServiceRequest(dsr, client.RequestOptions{})
		if err == nil {
			t.Error("expected: validation error, actual: nil")
		}
	})
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func TestDeliveryServiceRequestRules(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test creating Delivery Service Request rules", dsrGood+1)
	}
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		routingName := strings.Repeat("X", 1) + "." + strings.Repeat("X", 48)
		// Test the xmlId length and form
		XMLID := "X " + strings.Repeat("X", 46)
		displayName := strings.Repeat("X", 49)

		dsr := testData.DeliveryServiceRequests[dsrGood]
		var ds *tc.DeliveryServiceV4
		if dsr.ChangeType == tc.DSRChangeTypeDelete {
			ds = dsr.Original
		} else {
			ds = dsr.Requested
		}
		resetDS(ds)
		if ds == nil {
			t.Fatalf("the %dth DSR in the test data had no DeliveryService", dsrGood)
		}
		ds.DisplayName = displayName
		ds.RoutingName = routingName
		ds.XMLID = XMLID

		_, _, err := TOSession.CreateDeliveryServiceRequest(dsr, client.RequestOptions{})
		if err == nil {
			t.Error("expected: validation error, actual: nil")
		}
	})
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func TestDeliveryServiceRequestTypeFields(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrBadTenant+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test creating a Delivery Service Request with missing fields for its Type", dsrBadTenant+1)
	}

	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters}, func() {
		dsr := testData.DeliveryServiceRequests[dsrBadTenant]
		var ds *tc.DeliveryServiceV4
		if dsr.ChangeType == tc.DSRChangeTypeDelete {
			ds = dsr.Original
		} else {
			ds = dsr.Requested
		}
		resetDS(ds)
		if ds == nil {
			t.Fatalf("the %dth DSR in the test data had no DeliveryService", dsrBadTenant)
		}

		resp, _, err := TOSession.CreateDeliveryServiceRequest(dsr, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error creating a Delivery Service Request: %v - alerts: %+v", err, resp.Alerts)
		}

		found := false
		for _, alert := range resp.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() {
				t.Errorf("Expected only succuss-level alerts creating a DSR, got error-level alert: %s", alert.Text)
			} else if alert.Level == tc.SuccessLevel.String() {
				t.Logf("Got expected alert creating a DSR: %s", alert.Text)
				found = true
			}
		}

		if !found {
			t.Errorf("Expected a success-level alert creating a DSR, got none")
		}

		opts := client.NewRequestOptions()
		opts.QueryParameters.Set("xmlId", ds.XMLID)
		dsrs, _, err := TOSession.GetDeliveryServiceRequests(opts)
		if err != nil {
			t.Errorf("Unexpected error retriving Delivery Service Requests with XMLID '%s': %v - alerts: %+v", ds.XMLID, err, dsrs.Alerts)
		}
		if len(dsrs.Response) != 1 {
			t.Fatalf("expected exactly one Deliveryservice Request with XMLID '%s'; got %d", ds.XMLID, len(dsrs.Response))
		}
		if dsrs.Response[0].ID == nil {
			t.Fatalf("got a DSR with a null ID by XMLID '%s'", ds.XMLID)
		}

		alert, _, err := TOSession.DeleteDeliveryServiceRequest(*dsrs.Response[0].ID, client.RequestOptions{})
		if err != nil {
			t.Errorf("cannot delete Delivery Service Request #%d: %v - alerts: %+v", dsrs.Response[0].ID, err, alert.Alerts)
		}
	})
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func TestDeliveryServiceRequestBad(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrDraft+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test creating a non-draft Delivery Service Request", dsrDraft+1)
	}
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		// try to create non-draft/submitted
		src := testData.DeliveryServiceRequests[dsrDraft]
		resetDS(src.Original)
		resetDS(src.Requested)
		src.Status = tc.RequestStatusPending

		if _, _, err := TOSession.CreateDeliveryServiceRequest(src, client.RequestOptions{}); err == nil {
			t.Fatal("expected: validation error, actual: nil")
		}
	})
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func TestDeliveryServiceRequestWorkflow(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrDraft+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test Delivery Service Request workflow", dsrDraft+1)
	}

	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		// test empty request table
		dsrs, _, err := TOSession.GetDeliveryServiceRequests(client.RequestOptions{})
		if err != nil {
			t.Errorf("Error getting empty list of Delivery Service Requests: %v - alerts: %+v", err, dsrs.Alerts)
		}
		if dsrs.Response == nil {
			t.Error("Expected empty Delivery Service Request slice -- got nil")
		}
		if len(dsrs.Response) != 0 {
			t.Errorf("Expected no entries in Delivery Service Request slice, got %d", len(dsrs.Response))
		}

		// Create a draft request
		src := testData.DeliveryServiceRequests[dsrDraft]
		resetDS(src.Original)
		resetDS(src.Requested)

		alerts, _, err := TOSession.CreateDeliveryServiceRequest(src, client.RequestOptions{})
		if err != nil {
			t.Errorf("Error creating Delivery Service Request: %v - alerts: %+v", err, alerts.Alerts)
		}

		found := false
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() {
				t.Errorf("Expected only succuss-level alerts creating a DSR, got error-level alert: %s", alert.Text)
			} else if alert.Level == tc.SuccessLevel.String() {
				t.Logf("Got expected alert creating a DSR: %s", alert.Text)
				found = true
			}
		}

		if !found {
			t.Errorf("Expected a success-level alert creating a DSR, got none")
		}

		// Create a duplicate request -- should fail because xmlId is the same
		alerts, _, err = TOSession.CreateDeliveryServiceRequest(src, client.RequestOptions{})
		if err == nil {
			t.Fatal("expected: validation error, actual: nil")
		}

		found = false
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.SuccessLevel.String() {
				t.Errorf("Expected only error-level alerts creating a duplicate DSR, got success-level alert: %s", alert.Text)
			} else if alert.Level == tc.ErrorLevel.String() {
				t.Logf("Got expected alert creating a duplicate DSR: %s", alert.Text)
				found = true
			}
		}
		if !found {
			t.Errorf("Expected an error-level alert creating a duplicate DSR, got none")
		}

		opts := client.NewRequestOptions()
		opts.QueryParameters.Set("xmlId", "test-transitions")
		dsrs, _, err = TOSession.GetDeliveryServiceRequests(opts)
		if err != nil {
			t.Errorf("Unexpected error getting Delivery Service Requests with XMLID 'test-transitions': %v - alerts: %+v", err, dsrs.Alerts)
		}
		if len(dsrs.Response) != 1 {
			t.Errorf("Expected exactly one Delivery Service Request with XMLID 'test-transitions', got: %d", len(dsrs.Response))
			if len(dsrs.Response) == 0 {
				t.Fatal("Cannot proceed")
			}
		}

		alerts = updateDeliveryServiceRequestStatus(t, dsrs.Response[0], "submitted", nil)

		found = false
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() {
				t.Errorf("Expected only succuss-level alerts updating a DSR, got error-level alert: %s", alert.Text)
			} else if alert.Level == tc.SuccessLevel.String() {
				t.Logf("Got expected alert updating a DSR: %s", alert.Text)
				found = true
			}
		}
		if !found {
			t.Errorf("Expected a success-level alert updating a DSR, got none: %v", alerts.Alerts)
		}

		if alerts.Response.Status != tc.RequestStatus("submitted") {
			t.Errorf("expected status=submitted, got %s", alerts.Response.Status)
		}
	})
}

func updateDeliveryServiceRequestStatus(t *testing.T, dsr tc.DeliveryServiceRequestV4, newstate string, header http.Header) tc.DeliveryServiceRequestResponseV4 {
	var resp tc.DeliveryServiceRequestResponseV4
	ID := dsr.ID
	if ID == nil {
		t.Error("updateDeliveryServiceRequestStatus called with a DSR that has a nil ID")
		return resp
	}
	dsr.Status = tc.RequestStatus("submitted")
	resp, _, err := TOSession.UpdateDeliveryServiceRequest(*ID, dsr, client.RequestOptions{Header: header})
	if err != nil {
		t.Errorf("Unexpected error updating Delivery Service Request: %v - alerts: %+v", err, resp.Alerts)
		return resp
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("id", strconv.Itoa(*ID))
	d, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Errorf("Error fetching Delivery Service Request #%d after status update: %v - alerts: %+v", ID, err, d.Alerts)
		return resp
	}
	if len(d.Response) != 1 {
		t.Errorf("Expected exactly one Delivery Service Request to exist with ID %d, found: %d", *ID, len(d.Response))
	}

	resp.Response = d.Response[0]

	return resp
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func GetTestDeliveryServiceRequestsIMS(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test getting them with IMS", dsrGood+1)
	}

	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)

	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}
	resetDS(ds)
	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no DeliveryService - or that DeliveryService had no XMLID", dsrGood)
	}

	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, reqInf, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("Unexpected error getting Delivery Service Requests with XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func GetTestDeliveryServiceRequests(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test getting them", dsrGood+1)
	}
	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}
	resetDS(ds)

	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no Delivery Service", dsrGood)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Service Requests with XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func UpdateTestDeliveryServiceRequests(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test updating them", dsrGood+1)
	}

	// Retrieve the DeliveryServiceRequest by name so we can get the id for the Update
	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}

	resetDS(ds)
	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no Delivery Service", dsrGood)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Service Request with XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Deliver Service Request to exist with XMLID '%s', but none were found in Traffic Ops", ds.XMLID)
	}
	respDSR := resp.Response[0]
	if respDSR.ID == nil {
		t.Fatalf("got a DSR by XMLID '%s' with a null or undefined ID", ds.XMLID)
	}
	var respDS *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		respDS = dsr.Original
	} else {
		respDS = dsr.Requested
	}
	expDisplayName := "new display name"
	respDS.DisplayName = expDisplayName
	id := *respDSR.ID
	alerts, _, err := TOSession.UpdateDeliveryServiceRequest(id, respDSR, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Delivery Service Request #%d: %v - alerts: %+v", id, err, alerts.Alerts)
		return
	}

	// Retrieve the DeliveryServiceRequest to check DeliveryServiceRequest name got updated
	opts.QueryParameters.Del("xmlId")
	opts.QueryParameters.Set("id", strconv.Itoa(id))
	dsrResp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service Request #%d: %v - alerts: %+v", id, err, dsrResp.Alerts)
	}
	if len(dsrResp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service Request to have id %d, found: %d", id, len(dsrResp.Response))
	}

	respDSR = dsrResp.Response[0]
	if respDSR.ChangeType == tc.DSRChangeTypeDelete {
		respDS = dsr.Original
	} else {
		respDS = dsr.Requested
	}

	if respDS == nil {
		t.Fatalf("Got DSR by ID '%d' that had no Delivery Service", id)
	}
	if respDS.DisplayName != expDisplayName {
		t.Errorf("results do not match actual: %s, expected: %s", respDS.DisplayName, expDisplayName)
	}
}

// Note that this test is suceptible to breaking if the structure of the test
// data's DSRs is altered at all.
func DeleteTestDeliveryServiceRequests(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < dsrGood+1 {
		t.Fatalf("Need at least %d Delivery Service Requests to test deleting them", dsrGood+1)
	}

	// Retrieve the DeliveryServiceRequest by name so we can get the id for the Update
	dsr := testData.DeliveryServiceRequests[dsrGood]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}

	resetDS(ds)
	if ds == nil {
		t.Fatalf("the %dth DSR in the test data had no Delivery Service", dsrGood)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service Requests with XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
	if len(resp.Response) < 1 {
		t.Fatalf("expected at least one Delivery Service Request to have XMLID '%s', got none", ds.XMLID)
	}
	respDSR := resp.Response[0]
	if respDSR.ID == nil {
		t.Fatalf("Got a DSR by XMLID '%s' that had no ID", ds.XMLID)
	}
	alert, _, err := TOSession.DeleteDeliveryServiceRequest(*respDSR.ID, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot delete Delivery Service Request #%d: %v - alerts: %+v", respDSR.ID, err, alert.Alerts)
	}

	// Retrieve the DeliveryServiceRequest to see if it got deleted
	dsrs, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Service Request #%d after deletion: %v - alerts: %+v", *respDSR.ID, err, dsrs.Alerts)
	}
	if len(dsrs.Response) > 0 {
		t.Errorf("expected Delivery Service Request #%d to be deleted, but it was found in Traffic Ops", *respDSR.ID)
	}
}
