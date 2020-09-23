package v3

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
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"

	tc "github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/testing/api/utils"
)

const (
	dsrGood      = 0
	dsrBadTenant = 1
	dsrRequired  = 2
	dsrDraft     = 3
)

func TestDeliveryServiceRequests(t *testing.T) {
	ReloadFixtures() // resets IDs
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants, DeliveryServiceRequests}, func() {
		GetTestDeliveryServiceRequestsIMS(t)
		GetTestDeliveryServiceRequests(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, time)
		UpdateTestDeliveryServiceRequests(t)
		GetTestDeliveryServiceRequestsIMSAfterChange(t, header)
	})
}

func GetTestDeliveryServiceRequestsIMSAfterChange(t *testing.T, header http.Header) {
	// dsr := testData.DeliveryServiceRequests[dsrGood]
	_, reqInf, err := TOSession.GetDeliveryServiceRequestsV30(header, nil)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err.Error())
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}
	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)
	header.Set(rfc.IfModifiedSince, timeStr)
	_, reqInf, err = TOSession.GetDeliveryServiceRequestsV30(header, nil)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err.Error())
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func CreateTestDeliveryServiceRequests(t *testing.T) {
	t.Log("CreateTestDeliveryServiceRequests")

	dsr := testData.DeliveryServiceRequests[dsrGood]
	respDSR, _, err := TOSession.CreateDeliveryServiceRequestV30(dsr, nil)
	t.Log("Response: ", respDSR)
	if err != nil {
		t.Errorf("could not CREATE DeliveryServiceRequests: %v", err)
	}

}

func TestDeliveryServiceRequestRequired(t *testing.T) {
	ReloadFixtures()
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		dsr := testData.DeliveryServiceRequests[dsrRequired]
		alerts, _, err := TOSession.CreateDeliveryServiceRequestV30(dsr, nil)
		if err == nil {
			t.Fatal("Expected an error creating a DSR missing required fields, but didn't get one")
		}
		t.Logf("Received expected error creating DSR missing required fields %v", err)

		if len(alerts.Alerts) == 0 {
			t.Errorf("Expected: validation error alerts, actual: %+v", alerts)
		}
	})
}

func TestDeliveryServiceRequestGetAssignee(t *testing.T) {
	ReloadFixtures()
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		if len(testData.DeliveryServiceRequests) < 1 {
			t.Fatal("Need at least one DSR for testing")
		}
		dsr := testData.DeliveryServiceRequests[0]
		me, _, err := TOSession.GetUserCurrent()
		if err != nil {
			t.Fatalf("Fetching current user: %v", err)
		}
		if me.UserName == nil {
			t.Fatal("Current user has no username")
		}
		if me.ID == nil {
			t.Fatal("Current user has no ID")
		}
		dsr.Assignee = me.UserName
		dsr.AssigneeID = me.ID
		alerts, _, err := TOSession.CreateDeliveryServiceRequestV30(dsr, nil)
		if err != nil {
			t.Fatalf("Creating DSR: %v - %v", err, alerts)
		}

		dsrs, _, err := TOSession.GetDeliveryServiceRequests()
		if err != nil {
			t.Fatalf("Fetching DSRs: %v", err)
		}
		if len(dsrs) < 1 {
			t.Fatal("No DSRs returned after creating one")
		}
		d := dsrs[0]
		if len(dsrs) > 1 {
			t.Errorf("Too many DSRs returned after creating only one: %d", len(dsrs))
			t.Logf("Testing will proceed with DSR: %v", d)
		}

	})
}

func TestDeliveryServiceRequestRules(t *testing.T) {
	ReloadFixtures()
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		routingName := strings.Repeat("X", 1) + "." + strings.Repeat("X", 48)
		// Test the xmlId length and form
		XMLID := "X " + strings.Repeat("X", 46)
		displayName := strings.Repeat("X", 49)

		dsr := testData.DeliveryServiceRequests[dsrGood]
		dsr.Requested.DisplayName = &displayName
		dsr.Requested.RoutingName = &routingName
		dsr.Requested.XMLID = &XMLID

		alerts, _, err := TOSession.CreateDeliveryServiceRequestV30(dsr, nil)
		if err == nil {
			t.Error("Expected creating DSR with fields that fail validation to fail, but it didn't")
		} else {
			t.Logf("Received expected error creating DSR with fields that fail validation: %v", err)
		}
		if len(alerts.Alerts) == 0 {
			t.Errorf("Expected: validation error alerts, actual: %+v", alerts)
		}
	})
}

func TestDeliveryServiceRequestBad(t *testing.T) {
	ReloadFixtures()
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants}, func() {
		// try to create non-draft/submitted
		src := testData.DeliveryServiceRequests[dsrDraft]
		s, err := tc.RequestStatusFromString("pending")
		if err != nil {
			t.Errorf(`unable to create Status from string "pending"`)
		}
		src.Status = s

		alerts, _, err := TOSession.CreateDeliveryServiceRequestV30(src, nil)
		if err == nil {
			t.Error("Expected an error creating a bad DSR, but didn't get one")
		} else {
			t.Logf("Received expected error creating DSR: %v", err)
		}

		found := false
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.SuccessLevel.String() {
				t.Errorf("Unexpected success creating bad DSR: %v", alert.Text)
			} else if alert.Level == tc.ErrorLevel.String() {
				found = true
			}
		}

		if !found {
			t.Error("Didn't find an error alert when creating a bad DSR")
		}
	})
}

// TestDeliveryServiceRequestWorkflow tests that transitions of Status are
func TestDeliveryServiceRequestWorkflow(t *testing.T) {
	ReloadFixtures()
	WithObjs(t, []TCObj{Types, CDNs, Tenants, CacheGroups, Topologies, DeliveryServices, Parameters}, func() {
		// test empty request table
		dsrs, _, err := TOSession.GetDeliveryServiceRequestsV30(nil, nil)
		if err != nil {
			t.Errorf("Error getting empty list of DeliveryServiceRequests %v++", err)
		}
		if dsrs == nil {
			t.Error("Expected empty DeliveryServiceRequest slice -- got nil")
		}
		if len(dsrs) != 0 {
			t.Errorf("Expected no entries in DeliveryServiceRequest slice -- got %d", len(dsrs))
		}

		// Create a draft request
		src := testData.DeliveryServiceRequests[dsrDraft]
		src.SetXMLID()

		alerts, _, err := TOSession.CreateDeliveryServiceRequestV30(src, nil)
		if err != nil {
			t.Errorf("Error creating DeliveryServiceRequest %v", err)
		}

		expected := []string{`deliveryservice_request was created.`}
		utils.Compare(t, expected, alerts.ToStrings())

		// Create a duplicate request -- should fail because xmlId is the same
		alerts, _, err = TOSession.CreateDeliveryServiceRequestV30(src, nil)
		if err == nil {
			t.Error("Expected an error creating duplicate request - didn't get one")
		} else {
			t.Logf("Received expected error creating Delivery Service Request: %v", err)
		}

		found := false
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.SuccessLevel.String() {
				t.Errorf("Unexpected success message creating duplicate DSR: %v", alert.Text)
			} else if alert.Level == tc.ErrorLevel.String() {
				found = true
			}
		}
		if !found {
			t.Error("Didn't find expected error-level alert when creating duplicate DSR")
		}

		params := url.Values{}
		params.Set("xmlId", src.XMLID)
		dsrs, _, err = TOSession.GetDeliveryServiceRequestsV30(nil, params)
		if len(dsrs) != 1 {
			t.Errorf("Expected 1 deliveryServiceRequest -- got %d", len(dsrs))
			if len(dsrs) == 0 {
				t.Fatal("Cannot proceed")
			}
		}

		alerts, dsr := updateDeliveryServiceRequestStatus(t, dsrs[0], "submitted")
		found = false
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() {
				t.Errorf("Unexpected error-level alert: %s", alert.Text)
			} else if alert.Level == tc.SuccessLevel.String() && strings.Contains(alert.Text, "updated") {
				found = true
			}
		}

		if !found {
			t.Error("Didn't find success-level alert after updating")
		}

		if dsr.Status != tc.RequestStatus("submitted") {
			t.Errorf("expected status=submitted,  got %s", string(dsr.Status))
		}
	})
}

func updateDeliveryServiceRequestStatus(t *testing.T, dsr tc.DeliveryServiceRequestV30, newstate string) (tc.Alerts, tc.DeliveryServiceRequestV30) {
	if dsr.ID == nil {
		t.Error("Cannot update DSR with no ID")
		return tc.Alerts{}, tc.DeliveryServiceRequestV30{}
	}

	ID := *dsr.ID
	dsr.Status = tc.RequestStatus("submitted")

	alerts, _, err := TOSession.UpdateDeliveryServiceRequest(ID, dsr, nil)
	if err != nil {
		t.Errorf("Error updating deliveryservice_request: %v", err)
		return alerts, dsr
	}

	params := url.Values{}
	params.Set("id", strconv.Itoa(ID))
	d, _, err := TOSession.GetDeliveryServiceRequestsV30(nil, params)
	if err != nil {
		t.Errorf("Error updating deliveryservice_request %d: %v", ID, err)
		return alerts, dsr
	}

	if len(d) != 1 {
		t.Errorf("Expected 1 deliveryservice_request, got %d", len(d))
	}
	return alerts, d[0]
}

func GetTestDeliveryServiceRequestsIMS(t *testing.T) {
	var header http.Header
	header = make(map[string][]string)
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)
	header.Set(rfc.IfModifiedSince, time)
	// dsr := testData.DeliveryServiceRequests[dsrGood]
	_, reqInf, err := TOSession.GetDeliveryServiceRequestsV30(header, nil)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err.Error())
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestDeliveryServiceRequests(t *testing.T) {
	dsr := testData.DeliveryServiceRequests[dsrGood]
	resp, _, err := TOSession.GetDeliveryServiceRequestByXMLID(*dsr.Requested.XMLID)
	if err != nil {
		t.Errorf("cannot GET DeliveryServiceRequest by XMLID: %v - %v", err, resp)
	}
}

func UpdateTestDeliveryServiceRequests(t *testing.T) {

	// Retrieve the DeliveryServiceRequest by name so we can get the id for the Update
	dsr := testData.DeliveryServiceRequests[dsrGood]
	dsr.SetXMLID()
	params := url.Values{}
	params.Set("xmlId", dsr.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequestsV30(nil, params)
	if err != nil {
		t.Errorf("cannot GET DeliveryServiceRequest by XMLID '%s': %v - %v", dsr.XMLID, *dsr.Requested.XMLID, err)
	}
	if len(resp) == 0 {
		t.Fatal("Length of GET DeliveryServiceRequest is 0")
	}
	respDSR := resp[0]
	if respDSR.Requested == nil {
		t.Fatalf("Got back DSR without 'requested' (changetype: '%s')", respDSR.ChangeType)
	}
	if respDSR.ID == nil {
		t.Fatal("Got back DSR without ID")
	}
	expDisplayName := "new display name"
	respDSR.Requested.DisplayName = &expDisplayName
	var alert tc.Alerts
	alert, _, err = TOSession.UpdateDeliveryServiceRequest(*respDSR.ID, respDSR, nil)
	t.Log("Response: ", alert)
	if err != nil {
		t.Fatalf("cannot UPDATE DeliveryServiceRequest by id: %v - %v", err, alert)
	}

	// Retrieve the DeliveryServiceRequest to check DeliveryServiceRequest name got updated
	params.Del("xmlId")
	params.Set("id", strconv.Itoa(*respDSR.ID))
	resp, _, err = TOSession.GetDeliveryServiceRequestsV30(nil, params)
	if err != nil {
		t.Fatalf("cannot GET DeliveryServiceRequest by ID: %v - %v", respDSR.ID, err)
	}
	if len(resp) < 1 {
		t.Fatalf("No DSR by ID %d after updating that DSR", *respDSR.ID)
	}
	respDSR = resp[0]
	if respDSR.Requested == nil {
		t.Fatal("Got back DSR without 'requested' after update")
	}
	if respDSR.Requested.DisplayName == nil {
		t.Fatal("Got back DSR with null 'requested.displayName' after updating that field to non-null value")
	}
	if *respDSR.Requested.DisplayName != expDisplayName {
		t.Errorf("results do not match actual: '%s', expected: '%s'", *respDSR.Requested.DisplayName, expDisplayName)
	}

}

func DeleteTestDeliveryServiceRequests(t *testing.T) {

	// Retrieve the DeliveryServiceRequest by name so we can get the id for the Update
	dsr := testData.DeliveryServiceRequests[dsrGood]
	resp, _, err := TOSession.GetDeliveryServiceRequestByXMLID(*dsr.Requested.XMLID)
	if err != nil || len(resp) < 1 {
		t.Fatalf("cannot GET DeliveryServiceRequest by XMLID: %v - %v", *dsr.Requested.XMLID, err)
	}
	respDSR := resp[0]
	alert, _, err := TOSession.DeleteDeliveryServiceRequestByID(respDSR.ID)
	t.Log("Response: ", alert)
	if err != nil {
		t.Errorf("cannot DELETE DeliveryServiceRequest by id: %d - %v - %v", respDSR.ID, err, alert)
	}

	// Retrieve the DeliveryServiceRequest to see if it got deleted
	dsrs, _, err := TOSession.GetDeliveryServiceRequestByXMLID(*dsr.Requested.XMLID)
	if err != nil {
		t.Errorf("error deleting DeliveryServiceRequest name: %s", err.Error())
	}
	if len(dsrs) > 0 {
		t.Errorf("expected DeliveryServiceRequest XMLID: %s to be deleted", *dsr.Requested.XMLID)
	}
}
