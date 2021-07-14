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
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	tc "github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestCDNs(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Parameters}, func() {
		GetTestCDNsIMS(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, time)
		header.Set(rfc.IfUnmodifiedSince, time)
		SortTestCDNs(t)
		UpdateTestCDNs(t)
		UpdateTestCDNsWithHeaders(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestCDNsWithHeaders(t, header)
		GetTestCDNs(t)
		t.Run("get CDNs filtered by Domain Name", GetTestCDNsbyDomainName)
		GetTestCDNsbyDnssec(t)
		GetTestCDNsIMSAfterChange(t, header)
		CreateTestCDNEmptyName(t)
		CreateTestCDNEmptyDomainName(t)
		GetTestPaginationSupportCdns(t)
		SortTestCdnDesc(t)
		CreateTestCDNsAlreadyExist(t)
		DeleteTestCDNsInvalidId(t)
		if includeSystemTests {
			UpdateDeleteCDNWithLocks(t)
		}
	})
}

func UpdateDeleteCDNWithLocks(t *testing.T) {
	// Create a new user with operations level privileges
	user1 := tc.UserV40{
		User: tc.User{
			Username:             util.StrPtr("lock_user1"),
			RegistrationSent:     tc.TimeNoModFromTime(time.Now()),
			LocalPassword:        util.StrPtr("test_pa$$word"),
			ConfirmLocalPassword: util.StrPtr("test_pa$$word"),
			RoleName:             util.StrPtr("operations"),
		},
	}
	user1.Email = util.StrPtr("lockuseremail@domain.com")
	user1.TenantID = util.IntPtr(1)
	user1.FullName = util.StrPtr("firstName LastName")
	_, _, err := TOSession.CreateUser(user1, client.RequestOptions{})
	if err != nil {
		t.Fatalf("could not create test user with username: %s", *user1.Username)
	}
	defer ForceDeleteTestUsersByUsernames(t, []string{"lock_user1"})

	// Establish a session with the newly created non admin level user
	userSession, _, err := client.LoginWithAgent(Config.TrafficOps.URL, *user1.Username, *user1.LocalPassword, true, "to-api-v4-client-tests", false, toReqTimeout)
	if err != nil {
		t.Fatalf("could not login with user lock_user1: %v", err)
	}

	cdn := createBlankCDN("locksCDN", t)

	// Create a lock for this user
	_, _, err = userSession.CreateCDNLock(tc.CDNLock{
		CDN:     cdn.Name,
		Message: util.StrPtr("test lock"),
		Soft:    util.BoolPtr(false),
	}, client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't create cdn lock: %v", err)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", cdn.Name)
	cdns, _, err := userSession.GetCDNs(opts)
	if err != nil {
		t.Fatalf("couldn't get cdn: %v", err)
	}
	if len(cdns.Response) != 1 {
		t.Fatal("couldn't get exactly one cdn in the response, quitting")
	}
	cdnID := cdns.Response[0].ID
	// Try to update a CDN that another user has a hard lock on -> this should fail
	cdns.Response[0].DomainName = "changed_domain_name"
	_, reqInf, err := TOSession.UpdateCDN(cdnID, cdns.Response[0], client.RequestOptions{})
	if err == nil {
		t.Error("expected an error while updating a CDN for which a hard lock is held by another user, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 forbidden status while updating a CDN for which a hard lock is held by another user, but got %d", reqInf.StatusCode)
	}

	// Try to update a CDN that the same user has a hard lock on -> this should succeed
	_, reqInf, err = userSession.UpdateCDN(cdnID, cdns.Response[0], client.RequestOptions{})
	if err != nil {
		t.Errorf("expected no error while updating a CDN for which a hard lock is held by the same user, but got %v", err)
	}

	// Try to delete a CDN that another user has a hard lock on -> this should fail
	_, reqInf, err = TOSession.DeleteCDN(cdnID, client.RequestOptions{})
	if err == nil {
		t.Error("expected an error while deleting a CDN for which a hard lock is held by another user, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 forbidden status while deleting a CDN for which a hard lock is held by another user, but got %d", reqInf.StatusCode)
	}

	// Try to delete a CDN that the same user has a hard lock on -> this should succeed
	// This should also delete the lock associated with this CDN
	_, reqInf, err = userSession.DeleteCDN(cdnID, client.RequestOptions{})
	if err != nil {
		t.Errorf("expected no error while deleting a CDN for which a hard lock is held by the same user, but got %v", err)
	}

	locks, _, _ := userSession.GetCDNLocks(client.RequestOptions{})
	if len(locks.Response) != 0 {
		t.Errorf("expected deletion of CDN to delete it's associated lock, and no locks in the response, but got %d locks instead", len(locks.Response))
	}
}

func TestCDNsDNSSEC(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Users, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, ServerCapabilities, DeliveryServices}, func() {
		if includeSystemTests {
			GenerateDNSSECKeys(t)
			RefreshDNSSECKeys(t) // NOTE: testing refresh last (while no keys exist) because it's asynchronous and might affect other tests
		}
	})
}

func RefreshDNSSECKeys(t *testing.T) {
	resp, _, err := TOSession.RefreshDNSSECKeys(client.RequestOptions{})
	if err != nil {
		t.Errorf("unable to refresh DNSSEC keys: %v - alerts: %+v", err, resp.Alerts)
	}
}

func GenerateDNSSECKeys(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test updating CDNs")
	}
	firstCDN := testData.CDNs[0]
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", firstCDN.Name)
	cdns, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("Unexpected error getting CDNs filtered by name '%s': %v - alerts: %+v", firstCDN.Name, err, cdns.Alerts)
	}
	if len(cdns.Response) != 1 {
		t.Fatalf("Expected exactly one CDN named '%s' to exist, found: %d", firstCDN.Name, len(cdns.Response))
	}
	cdn := cdns.Response[0]

	ttl := util.JSONIntStr(60)
	req := tc.CDNDNSSECGenerateReq{
		Key:               util.StrPtr(firstCDN.Name),
		TTL:               &ttl,
		KSKExpirationDays: &ttl,
		ZSKExpirationDays: &ttl,
	}
	resp, _, err := TOSession.GenerateCDNDNSSECKeys(req, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error generating CDN DNSSEC keys: %v - alerts: %+v", err, resp.Alerts)
	}

	res, _, err := TOSession.GetCDNDNSSECKeys(firstCDN.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error getting CDN DNSSEC keys: %v - alerts: %+v", err, res.Alerts)
	}
	if _, ok := res.Response[firstCDN.Name]; !ok {
		t.Errorf("getting CDN DNSSEC keys - expected: key %s, actual: missing", firstCDN.Name)
	}
	originalKeys := res.Response

	resp, _, err = TOSession.GenerateCDNDNSSECKeys(req, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error generating CDN DNSSEC keys: %v - alerts: %+v", err, resp.Alerts)
	}
	res, _, err = TOSession.GetCDNDNSSECKeys(firstCDN.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error getting CDN DNSSEC keys: %v - alerts: %+v", err, res.Alerts)
	}
	newKeys := res.Response

	if reflect.DeepEqual(originalKeys, newKeys) {
		t.Errorf("generating CDN DNSSEC keys - expected: original keys to differ from new keys, actual: they are the same")
	}

	kskReq := tc.CDNGenerateKSKReq{
		ExpirationDays: util.Uint64Ptr(30),
	}
	originalKSK := newKeys
	resp, _, err = TOSession.GenerateCDNDNSSECKSK(firstCDN.Name, kskReq, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error generating DNSSEC KSK: %v - alerts: %+v", err, resp.Alerts)
	}
	res, _, err = TOSession.GetCDNDNSSECKeys(firstCDN.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error getting CDN DNSSEC keys: %v - alerts: %+v", err, res.Alerts)
	}
	newKSK := res.Response
	if reflect.DeepEqual(originalKSK[firstCDN.Name].KSK, newKSK[firstCDN.Name].KSK) {
		t.Error("generating CDN DNSSEC KSK - expected: KSK to be different, actual: KSK is the same")
	}
	if !reflect.DeepEqual(originalKSK[firstCDN.Name].ZSK, newKSK[firstCDN.Name].ZSK) {
		t.Error("generating CDN DNSSEC KSK - expected: ZSK to be equal, actual: ZSK is different")
	}

	// ensure that when DNSSEC is enabled on a CDN, creating a new DS will generate DNSSEC keys for that DS:
	if !cdn.DNSSECEnabled {
		cdn.DNSSECEnabled = true
		resp, _, err := TOSession.UpdateCDN(cdn.ID, cdn, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error updating CDN: %v - alerts: %+v", err, resp.Alerts)
		}
		defer func() {
			cdn.DNSSECEnabled = false
			resp, _, err := TOSession.UpdateCDN(cdn.ID, cdn, client.RequestOptions{})
			if err != nil {
				t.Errorf("Unexpected error updating CDN: %v - alerts: %+v", err, resp.Alerts)
			}
		}()
	}

	opts.QueryParameters.Set("name", "HTTP")
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("Unexpected error getting Types filteed by name 'HTTP': %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) != 1 {
		t.Fatalf("Expected exactly one Type to exist with name 'HTTP', found: %d", len(types.Response))
	}
	dsXMLID := "testdnssecgen"
	customDS := getCustomDS(cdn.ID, types.Response[0].ID, dsXMLID, "cdn", "https://testdnssecgen.example.com", dsXMLID)
	ds, _, err := TOSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating Delivery Service: %v - alerts: %+v", err, ds.Alerts)
	}
	if len(ds.Response) != 1 {
		t.Fatalf("Expected creating a Delivery Service to create exactly one Delivery Service, Traffic Ops returned: %d", len(ds.Response))
	}
	if ds.Response[0].ID == nil {
		t.Fatal("Traffic Ops returned a representation for a created Delivery Service with null or undefined ID")
	}
	res, _, err = TOSession.GetCDNDNSSECKeys(firstCDN.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error getting CDN DNSSEC keys: %v - alerts: %+v", err, res.Alerts)
	}
	if _, ok := res.Response[dsXMLID]; !ok {
		t.Error("after creating a new delivery service for a DNSSEC-enabled CDN - expected: DNSSEC keys to be found for the delivery service, actual: no DNSSEC keys found for the delivery service")
	}
	alerts, _, err := TOSession.DeleteDeliveryService(*ds.Response[0].ID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deleting Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
	}

	delResp, _, err := TOSession.DeleteCDNDNSSECKeys(firstCDN.Name, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deleting CDN DNSSEC keys: %v - alerts: %+v", err, delResp.Alerts)
	}
}

func UpdateTestCDNsWithHeaders(t *testing.T, header http.Header) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test updating CDNs")
	}
	firstCDN := testData.CDNs[0]
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", firstCDN.Name)
	opts.Header = header
	// Retrieve the CDN by name so we can get the id for the Update
	resp, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("cannot get CDN '%s': %v - alerts: %+v", firstCDN.Name, err, resp.Alerts)
	}
	if len(resp.Response) > 0 {
		remoteCDN := resp.Response[0]
		remoteCDN.DomainName = "domain2"
		opts.QueryParameters.Del("name")
		_, reqInf, err := TOSession.UpdateCDN(remoteCDN.ID, remoteCDN, opts)
		if err == nil {
			t.Errorf("Expected error about Precondition Failed, got none")
		}
		if reqInf.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
		}
	}
}

func GetTestCDNsIMSAfterChange(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header
	for _, cdn := range testData.CDNs {
		opts.QueryParameters.Set("name", cdn.Name)
		_, reqInf, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if reqInf.StatusCode != http.StatusOK {
			t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
		}
	}

	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)

	opts.Header.Set(rfc.IfModifiedSince, timeStr)

	for _, cdn := range testData.CDNs {
		opts.QueryParameters.Set("name", cdn.Name)
		_, reqInf, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if reqInf.StatusCode != http.StatusNotModified {
			t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
		}
	}
}

func GetTestCDNsIMS(t *testing.T) {
	opts := client.NewRequestOptions()

	for _, cdn := range testData.CDNs {
		futureTime := time.Now().AddDate(0, 0, 1)
		time := futureTime.Format(time.RFC1123)
		opts.Header.Set(rfc.IfModifiedSince, time)
		opts.QueryParameters.Set("name", cdn.Name)
		_, reqInf, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if reqInf.StatusCode != http.StatusNotModified {
			t.Fatalf("Expected 304 status code, got %d", reqInf.StatusCode)
		}
	}
}

func CreateTestCDNs(t *testing.T) {
	for _, cdn := range testData.CDNs {
		resp, _, err := TOSession.CreateCDN(cdn, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create CDN: %v - alerts: %+v", err, resp.Alerts)
		}
	}

}

func CreateTestCDNsAlreadyExist(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatal("Need at least one CDN to test duplicate CDNs")
	}

	cdn := testData.CDNs[0]
	resp, reqInf, err := TOSession.CreateCDN(cdn, client.RequestOptions{})
	if err == nil {
		t.Errorf("cdn domain_name 'mycdn.ciab.test' already exists.  but got - alerts: %+v", resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 status code, got %v", reqInf.StatusCode)
	}
}

func SortTestCDNs(t *testing.T) {
	var sortedList []string
	resp, _, err := TOSession.GetCDNs(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got %v - alerts: %+v", err, resp.Alerts)
	}
	for _, cdn := range resp.Response {
		sortedList = append(sortedList, cdn.Name)
	}

	res := sort.SliceIsSorted(sortedList, func(p, q int) bool {
		return sortedList[p] < sortedList[q]
	})
	if res != true {
		t.Errorf("list is not sorted by their names: %v", sortedList)
	}
}

func UpdateTestCDNs(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatal("Need at least one CDN to test updating CDNs")
	}

	firstCDN := testData.CDNs[0]
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", firstCDN.Name)
	// Retrieve the CDN by name so we can get the id for the Update
	resp, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("cannot get CDN '%s': %v - alert: %+v", firstCDN.Name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one CDN to be named '%s', found: %d", firstCDN.Name, len(resp.Response))
	}
	remoteCDN := resp.Response[0]
	originalDomain := remoteCDN.DomainName
	expectedCDNDomain := "domain2"
	remoteCDN.DomainName = expectedCDNDomain
	var alert tc.Alerts
	alert, _, err = TOSession.UpdateCDN(remoteCDN.ID, remoteCDN, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update CDN: %v - alerts: %+v", err, alert)
	} else {
		defer func(id int, cdn tc.CDN, domain string) {
			cdn.DomainName = domain
			alerts, _, err := TOSession.UpdateCDN(id, cdn, client.RequestOptions{})
			if err != nil {
				t.Errorf("Unexpected error restoring CDN domain name: %v - alerts: %+v", err, alerts.Alerts)
			}
		}(remoteCDN.ID, remoteCDN, originalDomain)
	}

	// Retrieve the CDN to check CDN name got updated
	opts.QueryParameters.Del("name")
	opts.QueryParameters.Set("id", strconv.Itoa(remoteCDN.ID))
	resp, _, err = TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("cannot get CDN '%s': %v - alerts: %+v", firstCDN.Name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one CDN to exist with ID %d, found: %d", remoteCDN.ID, len(resp.Response))
	}
	respCDN := resp.Response[0]
	if respCDN.DomainName != expectedCDNDomain {
		t.Errorf("results do not match actual: %s, expected: %s", respCDN.DomainName, expectedCDNDomain)
	}
}

func GetTestCDNs(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, cdn := range testData.CDNs {
		opts.QueryParameters.Set("name", cdn.Name)
		resp, _, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Errorf("cannot get CDN '%s': %v - alerts: %+v", cdn.Name, err, resp.Alerts)
		}
	}
}

func GetTestCDNsbyDomainName(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test get CDNs")
	}

	opts := client.NewRequestOptions()
	cdn := testData.CDNs[0]
	opts.QueryParameters.Set("domainName", cdn.DomainName)
	cdns, reqInf, err := TOSession.GetCDNs(opts)
	if len(cdns.Response) != 1 {
		t.Errorf("Expected exactly one CDN to exist with domain name '%s', found: %d", cdn.DomainName, len(cdns.Response))
	}
	if err != nil {
		t.Errorf("cannot get CDN by '%s': %v - alerts: %+v", cdn.DomainName, err, cdns.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestCDNsbyDnssec(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test get CDNs")
	}

	opts := client.NewRequestOptions()
	cdn := testData.CDNs[0]
	opts.QueryParameters.Set("dnssecEnabled", strconv.FormatBool(cdn.DNSSECEnabled))
	cdns, reqInf, err := TOSession.GetCDNs(opts)
	if len(cdns.Response) < 1 {
		t.Fatalf("Expected atleast one cdn response %v", cdns)
	}
	if err != nil {
		t.Errorf("cannot get CDN by '%s': %v - alerts: %+v", cdn.DomainName, err, cdns.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 status code, got %v", reqInf.StatusCode)
	}
}

func DeleteTestCDNs(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, cdn := range testData.CDNs {
		// Retrieve the CDN by name so we can get the id for the Update
		opts.QueryParameters.Set("name", cdn.Name)
		resp, _, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Errorf("cannot get CDN '%s': %v - alerts: %+v", cdn.Name, err, resp.Alerts)
		}
		if len(resp.Response) > 0 {
			respCDN := resp.Response[0]

			delResp, _, err := TOSession.DeleteCDN(respCDN.ID, client.RequestOptions{})
			if err != nil {
				t.Errorf("cannot delete CDN '%s' (#%d): %v - alerts: %+v", respCDN.Name, respCDN.ID, err, delResp.Alerts)
			}

			// Retrieve the CDN to see if it got deleted
			cdns, _, err := TOSession.GetCDNs(opts)
			if err != nil {
				t.Errorf("error deleting CDN '%s': %v - alerts: %+v", cdn.Name, err, cdns.Alerts)
			}
			if len(cdns.Response) > 0 {
				t.Errorf("expected CDN '%s' to be deleted", cdn.Name)
			}
		}
	}
}

func DeleteTestCDNsInvalidId(t *testing.T) {

	delResp, reqInf, err := TOSession.DeleteCDN(100000, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected, no cdn with that key found  but got - alerts: %+v", delResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected response status code %d, got %d", http.StatusNotFound, reqInf.StatusCode)
	}
}

func CreateTestCDNEmptyName(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test creating CDNs")
	}

	firstData := testData.CDNs[0]
	firstData.Name = ""
	firstData.DomainName = "EmptyCDNName"
	resp, reqInf, err := TOSession.CreateCDN(firstData, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected 'name' cannot be blank  but got - alerts: %+v", resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 status code, got %v", reqInf.StatusCode)
	}
}

func CreateTestCDNEmptyDomainName(t *testing.T) {
	if len(testData.CDNs) < 1 {
		t.Fatalf("need at least one CDN to test creating CDNs")
	}

	firstData := testData.CDNs[0]
	firstData.Name = "EmptyDomainName"
	firstData.DomainName = ""
	resp, reqInf, err := TOSession.CreateCDN(firstData, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected 'domainName' cannot be blank  but got - alerts: %+v", resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestPaginationSupportCdns(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("orderby", "id")
	resp, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("Unexpected error getting CDNs: %v - alerts: %+v", err, resp.Alerts)
	}
	cdns := resp.Response

	if len(cdns) > 0 {
		opts.QueryParameters = url.Values{}
		opts.QueryParameters.Set("orderby", "id")
		opts.QueryParameters.Set("limit", "1")
		cdnsWithLimit, _, err := TOSession.GetCDNs(opts)
		if err == nil {
			if !reflect.DeepEqual(cdns[:1], cdnsWithLimit.Response) {
				t.Error("expected GET CDN with limit = 1 to return first result")
			}
		} else {
			t.Errorf("Unexpected error getting CDN with a limit: %v - alerts: %+v", err, cdnsWithLimit.Alerts)
		}
		if len(cdns) > 1 {
			opts.QueryParameters = url.Values{}
			opts.QueryParameters.Set("orderby", "id")
			opts.QueryParameters.Set("limit", "1")
			opts.QueryParameters.Set("offset", "1")
			cdnsWithOffset, _, err := TOSession.GetCDNs(opts)
			if err == nil {
				if !reflect.DeepEqual(cdns[1:2], cdnsWithOffset.Response) {
					t.Error("expected GET CDN with limit = 1, offset = 1 to return second result")
				}
			} else {
				t.Errorf("Unexpected error getting CDN with a limit and an offset: %v - alerts: %+v", err, cdnsWithOffset.Alerts)
			}

			opts.QueryParameters = url.Values{}
			opts.QueryParameters.Set("orderby", "id")
			opts.QueryParameters.Set("limit", "1")
			opts.QueryParameters.Set("page", "2")
			cdnsWithPage, _, err := TOSession.GetCDNs(opts)
			if err == nil {
				if !reflect.DeepEqual(cdns[1:2], cdnsWithPage.Response) {
					t.Error("expected GET CDN with limit = 1, page = 2 to return second result")
				}
			} else {
				t.Errorf("Unexpected error getting CDN with a limit and a page: %v - alerts: %+v", err, cdnsWithPage.Alerts)
			}
		} else {
			t.Errorf("only one CDN found, so offset functionality can't test")
		}
	} else {
		t.Errorf("No CDN found to check pagination")
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "-2")
	resp, _, err = TOSession.GetCDNs(opts)
	if err == nil {
		t.Error("expected GET CDN to return an error when limit is not bigger than -1")
	} else if !strings.Contains(err.Error(), "must be bigger than -1") {
		t.Errorf("expected GET CDN to return an error for limit is not bigger than -1, actual error: %v - alerts: %+v", err, resp.Alerts)
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("offset", "0")
	resp, _, err = TOSession.GetCDNs(opts)
	if err == nil {
		t.Error("expected GET CDN to return an error when offset is not a positive integer")
	} else if !strings.Contains(err.Error(), "must be a positive integer") {
		t.Errorf("expected GET CDN to return an error for offset is not a positive integer, actual error: %v - alerts: %+v", err, resp.Alerts)
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("page", "0")
	resp, _, err = TOSession.GetCDNs(opts)
	if err == nil {
		t.Error("expected GET CDN to return an error when page is not a positive integer")
	} else if !strings.Contains(err.Error(), "must be a positive integer") {
		t.Errorf("expected GET CDN to return an error for page is not a positive integer, actual error: %v - alerts: %+v", err, resp.Alerts)
	}
}

func SortTestCdnDesc(t *testing.T) {
	resp, _, err := TOSession.GetCDNs(client.RequestOptions{})
	if err != nil {
		t.Errorf("Expected no error, but got error in CDN with default ordering: %v - alerts: %+v", err, resp.Alerts)
	}
	respAsc := resp.Response
	if len(respAsc) < 1 {
		t.Fatal("Need at least one CDN in Traffic Ops to test CDN sort ordering")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("sortOrder", "desc")
	resp, _, err = TOSession.GetCDNs(opts)
	if err != nil {
		t.Errorf("Expected no error, but got error in CDN with Descending ordering: %v - alerts: %+v", err, resp.Alerts)
	}
	respDesc := resp.Response
	if len(respDesc) < 1 {
		t.Fatal("Need at least one CDN in Traffic Ops to test CDN sort ordering")
	}

	if len(respAsc) != len(respDesc) {
		t.Fatalf("Traffic Ops returned %d CDN using default sort order, but %d CDN when sort order was explicitly set to descending", len(respAsc), len(respDesc))
	}

	// reverse the descending-sorted response and compare it to the ascending-sorted one
	// TODO ensure at least two in each slice? A list of length one is
	// trivially sorted both ascending and descending.
	for start, end := 0, len(respDesc)-1; start < end; start, end = start+1, end-1 {
		respDesc[start], respDesc[end] = respDesc[end], respDesc[start]
	}
	if respDesc[0].Name != respAsc[0].Name {
		t.Errorf("CDN responses are not equal after reversal: Asc: %s - Desc: %s", respDesc[0].Name, respAsc[0].Name)
	}
}
