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
	"strconv"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestDeliveryServiceRequestComments(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Tenants, DeliveryServiceRequests, DeliveryServiceRequestComments}, func() {
		GetTestDeliveryServiceRequestCommentsIMS(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfUnmodifiedSince, time)
		header.Set(rfc.IfModifiedSince, time)
		SortTestDeliveryServiceRequestComments(t)
		UpdateTestDeliveryServiceRequestComments(t)
		UpdateTestDeliveryServiceRequestCommentsWithHeaders(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestDeliveryServiceRequestCommentsWithHeaders(t, header)
		GetTestDeliveryServiceRequestComments(t)
		GetTestDeliveryServiceRequestCommentsIMSAfterChange(t, header)
	})
}

func UpdateTestDeliveryServiceRequestCommentsWithHeaders(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header
	comments, _, _ := TOSession.GetDeliveryServiceRequestComments(opts)

	if len(comments.Response) > 0 {
		firstComment := comments.Response[0]
		newFirstCommentValue := "new comment value"
		firstComment.Value = newFirstCommentValue

		_, reqInf, err := TOSession.UpdateDeliveryServiceRequestComment(firstComment.ID, firstComment, opts)
		if err == nil {
			t.Errorf("expected precondition failed error, but got none")
		}
		if reqInf.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
		}
	}
}

func GetTestDeliveryServiceRequestCommentsIMSAfterChange(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header
	resp, reqInf, err := TOSession.GetDeliveryServiceRequestComments(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Service Request Comments: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}
	opts.Header = make(map[string][]string)
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)
	opts.Header.Set(rfc.IfModifiedSince, time)
	resp, reqInf, err = TOSession.GetDeliveryServiceRequestComments(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Service Request Comments: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func CreateTestDeliveryServiceRequestComments(t *testing.T) {
	if len(testData.DeliveryServiceRequests) < 1 {
		t.Fatal("Need at least one Delivery Service Request to test creating Delivery Service Request Comments")
	}

	// Retrieve a delivery service request by xmlId so we can get the ID needed to create a dsr comment
	dsr := testData.DeliveryServiceRequests[0]
	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}
	resetDS(ds)
	if ds == nil {
		t.Fatal("first DSR in the test data had a nil Delivery Service")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", ds.XMLID)
	resp, _, err := TOSession.GetDeliveryServiceRequests(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service Request by XMLID '%s': %v - alerts: %+v", ds.XMLID, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("found %d Delivery Service request by XMLID '%s, expected exactly one", len(resp.Response), ds.XMLID)
	}
	respDSR := resp.Response[0]
	if respDSR.ID == nil {
		t.Fatalf("got Delivery Service Request with xml_id '%s' that had a null ID", ds.XMLID)
	}

	for _, comment := range testData.DeliveryServiceRequestComments {
		comment.DeliveryServiceRequestID = *respDSR.ID
		resp, _, err := TOSession.CreateDeliveryServiceRequestComment(comment, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create Delivery Service Request Comment: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func SortTestDeliveryServiceRequestComments(t *testing.T) {
	var sortedList []string
	resp, _, err := TOSession.GetDeliveryServiceRequestComments(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	for _, dsrc := range resp.Response {
		sortedList = append(sortedList, dsrc.XMLID)
	}

	res := sort.SliceIsSorted(sortedList, func(p, q int) bool {
		return sortedList[p] < sortedList[q]
	})
	if res != true {
		t.Errorf("list is not sorted by their names: %v", sortedList)
	}
}

func UpdateTestDeliveryServiceRequestComments(t *testing.T) {

	comments, _, err := TOSession.GetDeliveryServiceRequestComments(client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Service Request Comments: %v - alerts: %+v", err, comments.Alerts)
	}
	if len(comments.Response) < 1 {
		t.Fatal("Expected at least one Delivery Service Request Comment to exist in Traffic Ops - none did")
	}
	firstComment := comments.Response[0]
	newFirstCommentValue := "new comment value"
	firstComment.Value = newFirstCommentValue

	var alert tc.Alerts
	alert, _, err = TOSession.UpdateDeliveryServiceRequestComment(firstComment.ID, firstComment, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Delivery Service Request Comment #%d: %v - alerts: %+v", firstComment.ID, err, alert.Alerts)
	}

	// Retrieve the delivery service request comment to check that the value got updated
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("id", strconv.Itoa(firstComment.ID))
	resp, _, err := TOSession.GetDeliveryServiceRequestComments(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Service Request Comment #%d: %v - alerts: %+v", firstComment.ID, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service Request Comment to exist with ID %d, found: %d", firstComment.ID, len(resp.Response))
	}
	respDSRC := resp.Response[0]
	if respDSRC.Value != newFirstCommentValue {
		t.Errorf("results do not match actual: %s, expected: %s", respDSRC.Value, newFirstCommentValue)
	}

}

func GetTestDeliveryServiceRequestCommentsIMS(t *testing.T) {
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)
	resp, reqInf, err := TOSession.GetDeliveryServiceRequestComments(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Service Request Comments: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestDeliveryServiceRequestComments(t *testing.T) {
	comments, _, err := TOSession.GetDeliveryServiceRequestComments(client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Service Request Comments: %v - alerts: %+v", err, comments.Alerts)
	}

	opts := client.NewRequestOptions()
	for _, comment := range comments.Response {
		opts.QueryParameters.Set("id", strconv.Itoa(comment.ID))
		resp, _, err := TOSession.GetDeliveryServiceRequestComments(opts)
		if err != nil {
			t.Errorf("cannot get Delivery Service Request Comment by id %d: %v - alerts: %+v", comment.ID, err, resp.Alerts)
		}
	}
}

func DeleteTestDeliveryServiceRequestComments(t *testing.T) {
	comments, _, err := TOSession.GetDeliveryServiceRequestComments(client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Service Request Comments: %v - alerts: %+v", err, comments.Alerts)
	}

	opts := client.NewRequestOptions()
	for _, comment := range comments.Response {
		resp, _, err := TOSession.DeleteDeliveryServiceRequestComment(comment.ID, client.RequestOptions{})
		if err != nil {
			t.Errorf("cannot delete Delivery Service Request Comment #%d: %v - alerts: %+v", comment.ID, err, resp.Alerts)
		}

		// Retrieve the delivery service request comment to see if it got deleted
		opts.QueryParameters.Set("id", strconv.Itoa(comment.ID))
		comments, _, err := TOSession.GetDeliveryServiceRequestComments(opts)
		if err != nil {
			t.Errorf("Unexpected error fetching Delivery Service Request Comment %d after deletion: %v - alerts: %+v", comment.ID, err, comments.Alerts)
		}
		if len(comments.Response) > 0 {
			t.Errorf("expected Delivery Service Request Comment #%d to be deleted, but it was found in Traffic Ops", comment.ID)
		}
	}
}
