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
	"encoding/json"
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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestDeliveryServices(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Users, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, ServerCapabilities, DeliveryServices}, func() {
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		ti := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, ti)
		header.Set(rfc.IfUnmodifiedSince, ti)
		if includeSystemTests {
			t.Run("Update CDN for a Delivery Service with SSL keys", SSLDeliveryServiceCDNUpdateTest)
			t.Run("Create URL Signature keys for a Delivery Service", CreateTestDeliveryServicesURLSignatureKeys)
			t.Run("Retrieve URL Signature keys for a Delivery Service", GetTestDeliveryServicesURLSignatureKeys)
			t.Run("Delete URL Signature keys for a Delivery Service", DeleteTestDeliveryServicesURLSignatureKeys)
			t.Run("Create URI Signing Keys for a Delivery Service", CreateTestDeliveryServicesURISigningKeys)
			t.Run("Retrieve URI Signing keys for a Delivery Service", GetTestDeliveryServicesURISigningKeys)
			t.Run("Delete URI Signing keys for a Delivery Service", DeleteTestDeliveryServicesURISigningKeys)
			t.Run("Delete old CDN SSL keys", DeleteCDNOldSSLKeys)
			t.Run("Create and retrieve SSL keys for a Delivery Service", DeliveryServiceSSLKeys)
		}

		t.Run("Create a Delivery Service with the removed Long Description 2 and 3 fields", CreateTestDeliveryServiceWithLongDescFields)
		t.Run("Update a Delivery Service, setting its removed Long Description 2 and 3 fields", UpdateTestDeliveryServiceWithLongDescFields)
		t.Run("Getting unmodified Delivery Services using the If-Modified-Since header", GetTestDeliveryServicesIMS)
		t.Run("Getting Delivery Services accessible to a Tenant", GetAccessibleToTest)
		t.Run("Basic update of some Delivery Service fields", UpdateTestDeliveryServices)
		t.Run("Assign an Origin not in a Cache Group used by a Delivery Service's Topology to that Delivery Service", UpdateValidateORGServerCacheGroup)
		t.Run("Attempt to update a Delivery Service with If-Unmodified-Since", testUpdatingDeliveryServicesWithIUSOrIfMatch(header))
		t.Run("Basic update of some other Delivery Service fields", UpdateNullableTestDeliveryServices)
		t.Run("Update a Delivery Service giving it invalid Raw Remap text", UpdateDeliveryServiceWithInvalidRemapText)
		t.Run("Update a Delivery Service giving it invalid combinations of Range Slice Block Size and Range Request Handling", UpdateDeliveryServiceWithInvalidSliceRangeRequest)
		t.Run("Invalid Topology-to-Delivery Service assignments", UpdateDeliveryServiceWithInvalidTopology)
		t.Run("Getting modified Delivery Services using the If-Modified-Since header", testGettingDeliveryServicesWithIMSAfterModification(header))
		t.Run("Basic update of Delivery Service header rewrite fields", UpdateDeliveryServiceTopologyHeaderRewriteFields)
		t.Run("Basic GET request", GetTestDeliveryServices)
		t.Run("GET requests using the 'active' query string parameter", GetInactiveTestDeliveryServices)
		t.Run("Basic GET request for /deliveryservices/{{ID}}/capacity", GetTestDeliveryServicesCapacity)
		t.Run("Update fields added in new minor versions of the API", DeliveryServiceMinorVersionsTest)
		t.Run("Verify Tenancy-restricted Delivery Service access", DeliveryServiceTenancyTest)
		t.Run("Attempt to create invalid Delivery Services", PostDeliveryServiceTest)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		t.Run("Attempt to update a Delivery Service with If-Match", testUpdatingDeliveryServicesWithIUSOrIfMatch(header))
		t.Run("GET requests using pagination-controlling query string parameters", VerifyPaginationSupportDS)
		t.Run("GET requests using the 'cdn' query string parameter", GetDeliveryServiceByCdn)
		t.Run("Check behavior of 'cdn' query string parameter when CDN doesn't exist", GetDeliveryServiceByInvalidCdn)
		t.Run("Check behavior of 'profile' query string parameter when Profile doesn't exist", GetDeliveryServiceByInvalidProfile)
		t.Run("Check behavior of 'tenant' query string parameter when Tenant doesn't exist", GetDeliveryServiceByInvalidTenant)
		t.Run("Check behavior of 'type' query string parameter when Type doesn't exist", GetDeliveryServiceByInvalidType)
		t.Run("Check behavior of 'accessibleTo' query string parameter when the Tenant doesn't exist", GetDeliveryServiceByInvalidAccessibleTo)
		t.Run("Check behavior of 'xmlId' query string parameter when xmlId doesn't exist", GetDeliveryServiceByInvalidXmlId)
		t.Run("GET request using the 'logsEnabled' query string parameter", GetDeliveryServiceByLogsEnabled)
		t.Run("GET request using the 'profile' query string parameter", GetDeliveryServiceByValidProfile)
		t.Run("GET request using the 'tenant' query string parameter", GetDeliveryServiceByValidTenant)
		t.Run("GET request using the 'type' query string parameter", GetDeliveryServiceByValidType)
		t.Run("GET request using the 'xmlId' query string parameter", GetDeliveryServiceByValidXmlId)
		t.Run("Descending order sorted response to GET request", SortTestDeliveryServicesDesc)
		if includeSystemTests {
			t.Run("Create/ Update/ Delete delivery services with locks", CUDDeliveryServiceWithLocks)
		}
		t.Run("TLS Versions property", addTLSVersionsToDeliveryService)
	})
}

func CUDDeliveryServiceWithLocks(t *testing.T) {
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
	//util.IntPtr(resp.Response[0].ID)
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
	if len(testData.DeliveryServices) == 0 {
		t.Fatalf("no deliveryservices to run the test on, quitting")
	}

	cdn := createBlankCDN("sslkeytransfer", t)
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "HTTP")
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("unable to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) < 1 {
		t.Fatal("expected at least one type")
	}
	customDS := getCustomDS(cdn.ID, types.Response[0].ID, "cdn_locks_test_ds_name", "routingName", "https://test_cdn_locks.com", "cdn_locks_test_ds_xml_id")

	// Create a lock for this user
	_, _, err = userSession.CreateCDNLock(tc.CDNLock{
		CDN:     cdn.Name,
		Message: util.StrPtr("test lock"),
		Soft:    util.BoolPtr(false),
	}, client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't create cdn lock: %v", err)
	}
	// Try to create a new ds on a CDN that another user has a hard lock on -> this should fail
	_, reqInf, err := TOSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err == nil {
		t.Error("expected an error while creating a new ds for a CDN for which a hard lock is held by another user, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 forbidden status while creating a new ds for a CDN for which a hard lock is held by another user, but got %d", reqInf.StatusCode)
	}

	// Try to create a new ds on a CDN that the same user has a hard lock on -> this should succeed
	dsResp, reqInf, err := userSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("expected no error while creating a new ds for a CDN for which a hard lock is held by the same user, but got %v", err)
	}
	if len(dsResp.Response) != 1 {
		t.Fatalf("one response expected, but got %d", len(dsResp.Response))
	}
	opts = client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", *customDS.XMLID)
	deliveryServices, _, err := userSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("couldn't get ds: %v", err)
	}
	if len(deliveryServices.Response) != 1 {
		t.Fatal("couldn't get exactly one ds in the response, quitting")
	}
	dsID := dsResp.Response[0].ID
	// Try to update a ds on a CDN that another user has a hard lock on -> this should fail
	customDS.LongDesc = util.StrPtr("changed_long_desc")
	_, reqInf, err = TOSession.UpdateDeliveryService(*dsID, customDS, client.RequestOptions{})
	if err == nil {
		t.Error("expected an error while updating a ds for a CDN for which a hard lock is held by another user, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 forbidden status while updating a ds for a CDN for which a hard lock is held by another user, but got %d", reqInf.StatusCode)
	}
	// Try to update a ds on a CDN that the same user has a hard lock on -> this should succeed
	_, reqInf, err = userSession.UpdateDeliveryService(*dsID, customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("expected no error while updating a ds for a CDN for which a hard lock is held by the same user, but got %v", err)
	}
	// Try to delete a ds on a CDN that another user has a hard lock on -> this should fail
	_, reqInf, err = TOSession.DeleteDeliveryService(*dsID, client.RequestOptions{})
	if err == nil {
		t.Error("expected an error while deleting a ds for a CDN for which a hard lock is held by another user, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("expected a 403 forbidden status while deleting a ds for a CDN for which a hard lock is held by another user, but got %d", reqInf.StatusCode)
	}

	// Try to delete a ds on a CDN that the same user has a hard lock on -> this should succeed
	_, reqInf, err = userSession.DeleteDeliveryService(*dsID, client.RequestOptions{})
	if err != nil {
		t.Errorf("expected no error while deleting a ds for a CDN for which a hard lock is held by the same user, but got %v", err)
	}

	// Delete the lock
	_, _, err = userSession.DeleteCDNLocks(client.RequestOptions{QueryParameters: url.Values{"cdn": []string{cdn.Name}}})
	if err != nil {
		t.Errorf("expected no error while deleting other user's lock using admin endpoint, but got %v", err)
	}
}

func CreateTestDeliveryServiceWithLongDescFields(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating a Delivery Service")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in the test data with a null or undefined XMLID")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID")
			continue
		}
		if *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Errorf("GET Delivery Services missing: %v", firstDS.XMLID)
	}
	if remoteDS.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined ID") //... or it returned no DSes at all
	}
	remoteDS.XMLID = util.StrPtr("testDSCreation")
	remoteDS.DisplayName = util.StrPtr("test DS with LD1 and LD2 fields")
	remoteDS.LongDesc1 = util.StrPtr("long desc 1")
	remoteDS.LongDesc2 = util.StrPtr("Long desc 2")
	_, reqInf, err := TOSession.CreateDeliveryService(remoteDS, client.RequestOptions{})
	if err == nil {
		t.Errorf("expected an error stating that Long Desc 1 and Long Desc 2 fields are not supported in api version 4.0 onwards, but got nothing")
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 status code, but got %d", reqInf.StatusCode)
	}
}

func UpdateTestDeliveryServiceWithLongDescFields(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating a Delivery Service")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in the test data with a null or undefined XMLID")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID")
			continue
		}
		if *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Errorf("GET Delivery Services missing: %v", firstDS.XMLID)
	}
	if remoteDS.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined ID") //... or it returned no DSes at all
	}

	remoteDS.LongDesc1 = util.StrPtr("long desc 1")
	remoteDS.LongDesc2 = util.StrPtr("Long desc 2")
	_, reqInf, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{})
	if err == nil {
		t.Errorf("expected an error stating that Long Desc 1 and Long Desc 2 fields are not supported in api version 4.0 onwards, but got nothing")
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 status code, but got %d", reqInf.StatusCode)
	}
}

func testUpdatingDeliveryServicesWithIUSOrIfMatch(h http.Header) func(*testing.T) {
	return func(t *testing.T) {
		UpdateTestDeliveryServicesWithHeaders(t, h)
	}
}

func UpdateTestDeliveryServicesWithHeaders(t *testing.T, header http.Header) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating Delivery Services with HTTP Headers")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with null or undefined XMLID")
	}

	opts := client.RequestOptions{Header: header}
	dses, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID != nil && *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Fatalf("GET Delivery Services missing: %s", *firstDS.XMLID)
	}
	if remoteDS.ID == nil {
		t.Fatalf("Traffic Ops returned a representation for Delivery Service '%s' that had a null or undefined ID", *firstDS.XMLID)
	}

	updatedLongDesc := "something different"
	updatedMaxDNSAnswers := 164598
	updatedMaxOriginConnections := 100
	remoteDS.LongDesc = &updatedLongDesc
	remoteDS.MaxDNSAnswers = &updatedMaxDNSAnswers
	remoteDS.MaxOriginConnections = &updatedMaxOriginConnections
	remoteDS.MatchList = nil // verify that this field is optional in a PUT request, doesn't cause nil dereference panic

	_, reqInf, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, opts)
	if err == nil {
		t.Errorf("expected precondition failed error, got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("expected status code to be 412 Precondition Failed, but got: %d", reqInf.StatusCode)
	}
}

func createBlankCDN(cdnName string, t *testing.T) tc.CDN {
	_, _, err := TOSession.CreateCDN(tc.CDN{
		DNSSECEnabled: false,
		DomainName:    cdnName + ".ai",
		Name:          cdnName,
	}, client.RequestOptions{})
	if err != nil {
		t.Fatal("unable to create cdn: " + err.Error())
	}

	originalKeys, _, err := TOSession.GetCDNSSLKeys(cdnName, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to get keys on cdn %v: %v - alerts: %+v", cdnName, err, originalKeys.Alerts)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", cdnName)
	cdns, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Fatalf("unable to get cdn %v: %v - alerts: %+v", cdnName, err, cdns.Alerts)
	}
	if len(cdns.Response) < 1 {
		t.Fatal("expected more than 0 cdns")
	}
	keys, _, err := TOSession.GetCDNSSLKeys(cdnName, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to get keys on cdn %v: %v", cdnName, err)
	}
	if len(keys.Response) != len(originalKeys.Response) {
		t.Fatalf("expected %v ssl keys on cdn %v, got %v", len(originalKeys.Response), cdnName, len(keys.Response))
	}
	return cdns.Response[0]
}

func cleanUp(t *testing.T, ds tc.DeliveryServiceV4, oldCDNID int, newCDNID int, sslKeyVersions []string) {
	if ds.ID == nil || ds.XMLID == nil {
		t.Error("Cannot clean up Delivery Service with nil ID and/or XMLID")
		return
	}
	xmlid := *ds.XMLID
	id := *ds.ID

	opts := client.NewRequestOptions()
	for _, version := range sslKeyVersions {
		opts.QueryParameters.Set("version", version)
		resp, _, err := TOSession.DeleteDeliveryServiceSSLKeys(xmlid, opts)
		if err != nil {
			t.Errorf("Unexpected error deleting Delivery Service SSL Keys: %v - alerts: %+v", err, resp.Alerts)
		}
	}
	resp, _, err := TOSession.DeleteDeliveryService(id, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deleting Delivery Service '%s' (#%d) during cleanup: %v - alerts: %+v", xmlid, id, err, resp.Alerts)
	}
	if oldCDNID != -1 {
		resp2, _, err := TOSession.DeleteCDN(oldCDNID, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error deleting CDN (#%d) during cleanup: %v - alerts: %+v", oldCDNID, err, resp2.Alerts)
		}
	}
	if newCDNID != -1 {
		resp2, _, err := TOSession.DeleteCDN(newCDNID, client.RequestOptions{})
		if err != nil {
			t.Errorf("Unexpected error deleting CDN (#%d) during cleanup: %v - alerts: %+v", newCDNID, err, resp2.Alerts)
		}
	}
}

// getCustomDS returns a DS that is guaranteed to have non-nil:
//
//    Active
//    CDNID
//    DSCP
//    DisplayName
//    RoutingName
//    GeoLimit
//    GeoProvider
//    IPV6RoutingEnabled
//    InitialDispersion
//    LogsEnabled
//    MissLat
//    MissLong
//    MultiSiteOrigin
//    OrgServerFQDN
//    Protocol
//    QStringIgnore
//    RangeRequestHandling
//    RegionalGeoBlocking
//    TenantID
//    TypeID
//    XMLID
//
// BUT, will ALWAYS have nil MaxRequestHeaderBytes.
// Note that the Tenant is hard-coded to #1.
func getCustomDS(cdnID, typeID int, displayName, routingName, orgFQDN, dsID string) tc.DeliveryServiceV4 {
	customDS := tc.DeliveryServiceV4{}
	customDS.Active = util.BoolPtr(true)
	customDS.CDNID = util.IntPtr(cdnID)
	customDS.DSCP = util.IntPtr(0)
	customDS.DisplayName = util.StrPtr(displayName)
	customDS.RoutingName = util.StrPtr(routingName)
	customDS.GeoLimit = util.IntPtr(0)
	customDS.GeoProvider = util.IntPtr(0)
	customDS.IPV6RoutingEnabled = util.BoolPtr(false)
	customDS.InitialDispersion = util.IntPtr(1)
	customDS.LogsEnabled = util.BoolPtr(true)
	customDS.MissLat = util.FloatPtr(0)
	customDS.MissLong = util.FloatPtr(0)
	customDS.MultiSiteOrigin = util.BoolPtr(false)
	customDS.OrgServerFQDN = util.StrPtr(orgFQDN)
	customDS.Protocol = util.IntPtr(2)
	customDS.QStringIgnore = util.IntPtr(0)
	customDS.RangeRequestHandling = util.IntPtr(0)
	customDS.RegionalGeoBlocking = util.BoolPtr(false)
	customDS.TenantID = util.IntPtr(1)
	customDS.TypeID = util.IntPtr(typeID)
	customDS.XMLID = util.StrPtr(dsID)
	customDS.MaxRequestHeaderBytes = nil
	return customDS
}

func DeleteCDNOldSSLKeys(t *testing.T) {
	cdn := createBlankCDN("sslkeytransfer", t)

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "HTTP")
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("unable to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) < 1 {
		t.Fatal("expected at least one type")
	}

	// First DS creation
	customDS := getCustomDS(cdn.ID, types.Response[0].ID, "displayName", "routingName", "https://test.com", "dsID")

	resp, _, err := TOSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating a Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected Delivery Service creation to return exactly one Delivery Service, got: %d", len(resp.Response))
	}
	ds := resp.Response[0]
	if ds.XMLID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID")
	}

	ds.CDNName = &cdn.Name
	sslKeyRequestFields := tc.SSLKeyRequestFields{
		BusinessUnit: util.StrPtr("BU"),
		City:         util.StrPtr("CI"),
		Organization: util.StrPtr("OR"),
		HostName:     util.StrPtr("*.test.com"),
		Country:      util.StrPtr("CO"),
		State:        util.StrPtr("ST"),
	}
	genResp, _, err := TOSession.GenerateSSLKeysForDS(*ds.XMLID, *ds.CDNName, sslKeyRequestFields, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error generaing SSL Keys for Delivery Service '%s': %v - alerts: %+v", *ds.XMLID, err, genResp.Alerts)
	}
	defer cleanUp(t, ds, cdn.ID, -1, []string{"1"})

	// Second DS creation
	customDS2 := getCustomDS(cdn.ID, types.Response[0].ID, "displayName2", "routingName2", "https://test2.com", "dsID2")

	resp, _, err = TOSession.CreateDeliveryService(customDS2, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating a Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected Delivery Service creation to return exactly one Delivery Service, got: %d", len(resp.Response))
	}
	ds2 := resp.Response[0]
	if ds2.XMLID == nil || ds2.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID and/or ID")
	}

	ds2.CDNName = &cdn.Name
	sslKeyRequestFields.HostName = util.StrPtr("*.test2.com")
	genResp, _, err = TOSession.GenerateSSLKeysForDS(*ds2.XMLID, *ds2.CDNName, sslKeyRequestFields, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error generaing SSL Keys for Delivery Service '%s': %v - alerts: %+v", *ds2.XMLID, err, genResp.Alerts)
	}

	var cdnKeys []tc.CDNSSLKeys
	for tries := 0; tries < 5; tries++ {
		time.Sleep(time.Second)
		var sslKeysResp tc.CDNSSLKeysResponse
		sslKeysResp, _, err = TOSession.GetCDNSSLKeys(cdn.Name, client.RequestOptions{})
		if err != nil {
			continue
		}
		cdnKeys = sslKeysResp.Response
		if len(cdnKeys) != 0 {
			break
		}
	}

	if err != nil {
		t.Fatalf("unable to get CDN %v SSL keys: %v", cdn.Name, err)
	}
	if len(cdnKeys) != 2 {
		t.Errorf("expected two ssl keys for CDN %v, got %d instead", cdn.Name, len(cdnKeys))
	}

	delResp, _, err := TOSession.DeleteDeliveryService(*ds2.ID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deleting Delivery Service #%d: %v - alerts: %+v", *ds2.ID, err, delResp.Alerts)
	}

	opts = client.NewRequestOptions()
	opts.QueryParameters.Set("cdnID", strconv.Itoa(cdn.ID))
	snapResp, _, err := TOSession.SnapshotCRConfig(opts)
	if err != nil {
		t.Fatalf("Failed to take Snapshot of CDN #%d: %v - alerts: %+v", cdn.ID, err, snapResp.Alerts)
	}
	var newCdnKeys []tc.CDNSSLKeys
	for tries := 0; tries < 5; tries++ {
		time.Sleep(time.Second)
		var sslKeysResp tc.CDNSSLKeysResponse
		sslKeysResp, _, err = TOSession.GetCDNSSLKeys(cdn.Name, client.RequestOptions{})
		newCdnKeys = sslKeysResp.Response
		if err == nil && len(newCdnKeys) == 1 {
			break
		}
	}

	if err != nil {
		t.Fatalf("unable to get CDN %v SSL keys: %v", cdn.Name, err)
	}
	if len(newCdnKeys) != 1 {
		t.Errorf("expected 1 ssl keys for CDN %v, got %d instead", cdn.Name, len(newCdnKeys))
	}
}

func DeliveryServiceSSLKeys(t *testing.T) {
	cdn := createBlankCDN("sslkeytransfer", t)

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "HTTP")
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("unable to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) < 1 {
		t.Fatal("expected at least one type")
	}

	customDS := getCustomDS(cdn.ID, types.Response[0].ID, "displayName", "routingName", "https://test.com", "dsID")

	resp, _, err := TOSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating a Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected Delivery Service creation to return exactly one Delivery Service, got: %d", len(resp.Response))
	}
	ds := resp.Response[0]
	if ds.XMLID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID")
	}

	ds.CDNName = &cdn.Name
	genResp, _, err := TOSession.GenerateSSLKeysForDS(*ds.XMLID, *ds.CDNName, tc.SSLKeyRequestFields{
		BusinessUnit: util.StrPtr("BU"),
		City:         util.StrPtr("CI"),
		Organization: util.StrPtr("OR"),
		HostName:     util.StrPtr("*.test2.com"),
		Country:      util.StrPtr("CO"),
		State:        util.StrPtr("ST"),
	}, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error generating SSL Keys for Delivery Service '%s': %v - alerts: %+v", *ds.XMLID, err, genResp.Alerts)
	}
	defer cleanUp(t, ds, cdn.ID, -1, []string{"1"})

	dsSSLKey := new(tc.DeliveryServiceSSLKeys)
	for tries := 0; tries < 5; tries++ {
		time.Sleep(time.Second)
		var sslKeysResp tc.DeliveryServiceSSLKeysResponse
		sslKeysResp, _, err = TOSession.GetDeliveryServiceSSLKeys(*ds.XMLID, client.RequestOptions{})
		*dsSSLKey = sslKeysResp.Response
		if err == nil && dsSSLKey != nil {
			break
		}
	}

	if err != nil || dsSSLKey == nil {
		t.Fatalf("unable to get DS %s SSL key: %v", *ds.XMLID, err)
	}
	if dsSSLKey.Certificate.Key == "" {
		t.Errorf("expected a valid key but got nothing")
	}
	if dsSSLKey.Certificate.Crt == "" {
		t.Errorf("expected a valid certificate, but got nothing")
	}
	if dsSSLKey.Certificate.CSR == "" {
		t.Errorf("expected a valid CSR, but got nothing")
	}

	err = deliveryservice.Base64DecodeCertificate(&dsSSLKey.Certificate)
	if err != nil {
		t.Fatalf("couldn't decode certificate: %v", err)
	}

	dsSSLKeyReq := tc.DeliveryServiceSSLKeysReq{
		AuthType:        &dsSSLKey.AuthType,
		CDN:             &dsSSLKey.CDN,
		DeliveryService: &dsSSLKey.DeliveryService,
		BusinessUnit:    &dsSSLKey.BusinessUnit,
		City:            &dsSSLKey.City,
		Organization:    &dsSSLKey.Organization,
		HostName:        &dsSSLKey.Hostname,
		Country:         &dsSSLKey.Country,
		State:           &dsSSLKey.State,
		Key:             &dsSSLKey.Key,
		Version:         &dsSSLKey.Version,
		Certificate:     &dsSSLKey.Certificate,
	}
	addSSLKeysResp, _, err := TOSession.AddSSLKeysForDS(tc.DeliveryServiceAddSSLKeysReq{DeliveryServiceSSLKeysReq: dsSSLKeyReq}, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error adding SSL keys for Delivery Service '%s': %v - alerts: %+v", dsSSLKey.DeliveryService, err, addSSLKeysResp.Alerts)
	}

	dsSSLKey = new(tc.DeliveryServiceSSLKeys)
	for tries := 0; tries < 5; tries++ {
		time.Sleep(time.Second)
		var sslKeysResp tc.DeliveryServiceSSLKeysResponse
		sslKeysResp, _, err = TOSession.GetDeliveryServiceSSLKeys(*ds.XMLID, client.RequestOptions{})
		*dsSSLKey = sslKeysResp.Response
		if err == nil && dsSSLKey != nil {
			break
		}
	}

	if err != nil || dsSSLKey == nil {
		t.Fatalf("unable to get DS %s SSL key: %v", *ds.XMLID, err)
	}
	if dsSSLKey.Certificate.Key == "" {
		t.Errorf("expected a valid key but got nothing")
	}
	if dsSSLKey.Certificate.Crt == "" {
		t.Errorf("expected a valid certificate, but got nothing")
	}
	if dsSSLKey.Certificate.CSR == "" {
		t.Errorf("expected a valid CSR, but got nothing")
	}
}

func SSLDeliveryServiceCDNUpdateTest(t *testing.T) {
	cdnNameOld := "sslkeytransfer"
	oldCdn := createBlankCDN(cdnNameOld, t)
	cdnNameNew := "sslkeytransfer1"
	newCdn := createBlankCDN(cdnNameNew, t)

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "HTTP")
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("unable to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) < 1 {
		t.Fatal("expected at least one type")
	}

	customDS := getCustomDS(oldCdn.ID, types.Response[0].ID, "displayName", "routingName", "https://test.com", "dsID")

	resp, _, err := TOSession.CreateDeliveryService(customDS, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating a custom Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected Delivery Service creation to create exactly one Delivery Service, Traffic Ops indicates %d were created", len(resp.Response))
	}
	ds := resp.Response[0]
	if ds.ID == nil || ds.XMLID == nil {
		t.Fatal("Traffic Ops created a Delivery Service with null or undefined XMLID and/or ID")
	}
	ds.CDNName = &oldCdn.Name

	defer cleanUp(t, ds, oldCdn.ID, newCdn.ID, []string{"1"})

	_, _, err = TOSession.GenerateSSLKeysForDS(*ds.XMLID, *ds.CDNName, tc.SSLKeyRequestFields{
		BusinessUnit: util.StrPtr("BU"),
		City:         util.StrPtr("CI"),
		Organization: util.StrPtr("OR"),
		HostName:     util.StrPtr("*.test.com"),
		Country:      util.StrPtr("CO"),
		State:        util.StrPtr("ST"),
	}, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to generate sslkeys for cdn %v: %v", oldCdn.Name, err)
	}

	var oldCDNKeys []tc.CDNSSLKeys
	for tries := 0; tries < 5; tries++ {
		time.Sleep(time.Second)
		resp, _, err := TOSession.GetCDNSSLKeys(oldCdn.Name, client.RequestOptions{})
		oldCDNKeys = resp.Response
		if err == nil && len(oldCDNKeys) > 0 {
			break
		}
	}
	if err != nil {
		t.Fatalf("unable to get cdn %v keys: %v", oldCdn.Name, err)
	}
	if len(oldCDNKeys) < 1 {
		t.Fatal("expected at least 1 key")
	}

	newCDNKeys, _, err := TOSession.GetCDNSSLKeys(newCdn.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to get cdn %v keys: %v", newCdn.Name, err)
	}

	ds.RoutingName = util.StrPtr("anothername")
	_, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err == nil {
		t.Fatal("should not be able to update delivery service (routing name) as it has ssl keys")
	}
	ds.RoutingName = util.StrPtr("routingName")

	ds.CDNID = &newCdn.ID
	ds.CDNName = &newCdn.Name
	_, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err == nil {
		t.Fatal("should not be able to update delivery service (cdn) as it has ssl keys")
	}

	// Check new CDN still has an ssl key
	keys, _, err := TOSession.GetCDNSSLKeys(newCdn.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to get cdn %v keys: %v - alerts: %+v", newCdn.Name, err, keys.Alerts)
	}
	if len(keys.Response) != len(newCDNKeys.Response) {
		t.Fatalf("expected %v keys, got %v", len(newCDNKeys.Response), len(keys.Response))
	}

	// Check old CDN does not have ssl key
	keys, _, err = TOSession.GetCDNSSLKeys(oldCdn.Name, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to get cdn %v keys: %v - %+v", oldCdn.Name, err, keys.Alerts)
	}
	if len(keys.Response) != len(oldCDNKeys) {
		t.Fatalf("expected %v key, got %v", len(oldCDNKeys), len(keys.Response))
	}
}

func testGettingDeliveryServicesWithIMSAfterModification(h http.Header) func(*testing.T) {
	return func(t *testing.T) {
		GetTestDeliveryServicesIMSAfterChange(t, h)
	}
}

func GetTestDeliveryServicesIMSAfterChange(t *testing.T, header http.Header) {
	opts := client.RequestOptions{Header: header}
	resp, reqInf, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Services: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}

	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)

	opts.Header.Set(rfc.IfModifiedSince, timeStr)
	resp, reqInf, err = TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Services: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func PostDeliveryServiceTest(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one testing Delivery Service to test creating Delivery Services")
	}
	ds := testData.DeliveryServices[0]
	if ds.XMLID == nil {
		t.Fatal("Found Delivery Service in testing data with null or undefined XMLID")
	}
	xmlid := *ds.XMLID + "-topology-test"

	ds.XMLID = new(string)
	_, _, err := TOSession.CreateDeliveryService(ds, client.RequestOptions{})
	if err == nil {
		t.Error("Expected error with empty xmlid")
	}
	ds.XMLID = nil
	_, _, err = TOSession.CreateDeliveryService(ds, client.RequestOptions{})
	if err == nil {
		t.Error("Expected error with nil xmlid")
	}

	ds.Topology = new(string)
	ds.XMLID = &xmlid

	_, reqInf, err := TOSession.CreateDeliveryService(ds, client.RequestOptions{})
	if err == nil {
		t.Error("Expected error with non-existent Topology")
	}
	if reqInf.StatusCode < 400 || reqInf.StatusCode >= 500 {
		t.Errorf("Expected client-level error creating DS with non-existent Topology, got: %d", reqInf.StatusCode)
	}
}

func CreateTestDeliveryServices(t *testing.T) {
	pl := tc.Parameter{
		ConfigFile: "remap.config",
		Name:       "location",
		Value:      "/remap/config/location/parameter/",
	}
	alerts, _, err := TOSession.CreateParameter(pl, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot create parameter: %v - alerts: %+v", err, alerts.Alerts)
	}
	for _, ds := range testData.DeliveryServices {
		ds = ds.RemoveLD1AndLD2()
		if ds.XMLID == nil {
			t.Error("Found a Delivery Service in testing data with null or undefined XMLID")
			continue
		}
		resp, _, err := TOSession.CreateDeliveryService(ds, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create Delivery Service '%s': %v - alerts: %+v", *ds.XMLID, err, resp.Alerts)
		}
	}
}

func GetTestDeliveryServicesIMS(t *testing.T) {
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)
	resp, reqInf, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("could not get Delivery Services: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

// Note that this test will break if anyone ever adds or modifies the test data
// Delivery Services such that any of them has more than 0 but not 3 Consistent
// Hashing Query Parameters - OR such that more than (but not less than) 2
// Delivery Services has more than 0 (but not necessarily exactly 3) Consistent
// Hashing Query Parameters.
func GetTestDeliveryServices(t *testing.T) {
	actualDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, actualDSes.Alerts)
	}
	actualDSMap := make(map[string]tc.DeliveryServiceV4, len(actualDSes.Response))
	for _, ds := range actualDSes.Response {
		if ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service with null or undefined XMLID")
			continue
		}
		actualDSMap[*ds.XMLID] = ds
	}
	cnt := 0
	for _, ds := range testData.DeliveryServices {
		if ds.XMLID == nil {
			t.Error("Delivery Service found in test data with null or undefined XMLID")
			continue
		}
		if _, ok := actualDSMap[*ds.XMLID]; !ok {
			t.Errorf("GET DeliveryService missing: %s", *ds.XMLID)
		}
		// exactly one ds should have exactly 3 query params. the rest should have none
		if c := len(ds.ConsistentHashQueryParams); c > 0 {
			if c != 3 {
				t.Errorf("deliveryservice %s has %d query params; expected 3 or 0", *ds.XMLID, c)
			}
			cnt++
		}
	}
	if cnt > 2 {
		t.Errorf("exactly 2 deliveryservices should have more than one query param; found %d", cnt)
	}
}

func GetInactiveTestDeliveryServices(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("active", strconv.FormatBool(false))
	inactiveDSes, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("cannot get inactive Delivery Services: %v - alerts: %+v", err, inactiveDSes.Alerts)
	}
	for _, ds := range inactiveDSes.Response {
		if ds.Active == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined 'active'")
			continue
		}
		if *ds.Active != false {
			t.Errorf("expected all delivery services to be inactive, but got atleast one active DS")
		}
	}

	opts.QueryParameters.Set("active", strconv.FormatBool(true))
	activeDSes, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("cannot get active Delivery Services: %v - alerts: %+v", err, activeDSes.Alerts)
	}
	for _, ds := range activeDSes.Response {
		if ds.Active == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined 'active'")
			continue
		}
		if *ds.Active != true {
			t.Errorf("expected all delivery services to be active, but got atleast one inactive DS")
		}
	}
}

func GetTestDeliveryServicesCapacity(t *testing.T) {
	actualDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, actualDSes.Alerts)
	}
	actualDSMap := map[string]tc.DeliveryServiceV4{}
	for _, ds := range actualDSes.Response {
		if ds.ID == nil || ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID and/or ID")
			continue
		}
		actualDSMap[*ds.XMLID] = ds
		capDS, _, err := TOSession.GetDeliveryServiceCapacity(*ds.ID, client.RequestOptions{})
		if err != nil {
			t.Errorf(`cannot get Delivery Service "%s"'s (#%d) Capacity: %v - alerts: %+v`, *ds.XMLID, *ds.ID, err, capDS.Alerts)
		}
	}

}

func UpdateTestDeliveryServices(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating a Delivery Service")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in the test data with a null or undefined XMLID")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID")
			continue
		}
		if *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Errorf("GET Delivery Services missing: %v", firstDS.XMLID)
	}
	if remoteDS.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined ID") //... or it returned no DSes at all
	}

	updatedMaxRequestHeaderSize := 131080
	updatedLongDesc := "something different"
	updatedMaxDNSAnswers := 164598
	updatedMaxOriginConnections := 100
	remoteDS.LongDesc = &updatedLongDesc
	remoteDS.MaxDNSAnswers = &updatedMaxDNSAnswers
	remoteDS.MaxOriginConnections = &updatedMaxOriginConnections
	remoteDS.MatchList = nil // verify that this field is optional in a PUT request, doesn't cause nil dereference panic
	remoteDS.MaxRequestHeaderBytes = &updatedMaxRequestHeaderSize

	if updateResp, _, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{}); err != nil {
		t.Errorf("cannot update Delivery Service: %v - %v", err, updateResp)
	}

	// Retrieve the server to check rack and interfaceName values were updated
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("id", strconv.Itoa(*remoteDS.ID))
	apiResp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service #%d: %v - alerts: %+v", *remoteDS.ID, err, apiResp.Alerts)
	}
	if len(apiResp.Response) != 1 {
		t.Fatalf("expected exactly one Delivery Service to exist with ID %d, found: %d", *remoteDS.ID, len(apiResp.Response))
	}
	resp := apiResp.Response[0]
	if resp.LongDesc == nil {
		t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined long description")
	} else if *resp.LongDesc != updatedLongDesc {
		t.Errorf("long description do not match actual: %s, expected: %s", *resp.LongDesc, updatedLongDesc)
	}

	if resp.MaxDNSAnswers == nil {
		t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined max DNS answers")
	} else if *resp.MaxDNSAnswers != updatedMaxDNSAnswers {
		t.Errorf("max DNS answers do not match actual: %d, expected: %d", *resp.MaxDNSAnswers, updatedMaxDNSAnswers)
	}

	if resp.MaxOriginConnections == nil {
		t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined max origin connections")
	} else if *resp.MaxOriginConnections != updatedMaxOriginConnections {
		t.Errorf("max origin connections do not match actual: %d, expected: %d", resp.MaxOriginConnections, updatedMaxOriginConnections)
	}

	if resp.MaxRequestHeaderBytes == nil {
		t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined max request header bytes")
	} else if *resp.MaxRequestHeaderBytes != updatedMaxRequestHeaderSize {
		t.Errorf("max request header sizes do not match actual: %d, expected: %d", resp.MaxRequestHeaderBytes, updatedMaxRequestHeaderSize)
	}
}

func UpdateNullableTestDeliveryServices(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating nullable fields of a Delivery Service")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in the test data with a null or undefined XMLID")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil || ds.ID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID and/or ID")
			continue
		}
		if *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Fatalf("GET Delivery Services missing: %v", firstDS.XMLID)
	}

	updatedLongDesc := "something else different"
	updatedMaxDNSAnswers := 164599
	remoteDS.LongDesc = &updatedLongDesc
	remoteDS.MaxDNSAnswers = &updatedMaxDNSAnswers

	if updateResp, _, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{}); err != nil {
		t.Fatalf("cannot update Delivery Service #%d: %v - alerts: %+v", *remoteDS.ID, err, updateResp)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("id", strconv.Itoa(*remoteDS.ID))
	apiResp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service #%d: %v - alerts: %+v", *remoteDS.ID, err, apiResp.Alerts)
	}
	if len(apiResp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service to exist with ID %d, found: %d", *remoteDS.ID, len(apiResp.Response))
	}
	resp := apiResp.Response[0]

	if resp.LongDesc == nil {
		t.Errorf("results do not match actual: <nil>, expected: %s", updatedLongDesc)
	} else if *resp.LongDesc != updatedLongDesc {
		t.Errorf("results do not match actual: %s, expected: %s", *resp.LongDesc, updatedLongDesc)
	}
	if resp.MaxDNSAnswers == nil {
		t.Fatalf("results do not match actual: <nil>, expected: %d", updatedMaxDNSAnswers)
	} else if *resp.MaxDNSAnswers != updatedMaxDNSAnswers {
		t.Fatalf("results do not match actual: %d, expected: %d", *resp.MaxDNSAnswers, updatedMaxDNSAnswers)
	}

}

// UpdateDeliveryServiceWithInvalidTopology ensures that a topology cannot be:
// - assigned to (CLIENT_)STEERING delivery services
// - assigned to any delivery services which have required capabilities that the topology can't satisfy
func UpdateDeliveryServiceWithInvalidTopology(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	found := false
	var nonCSDS *tc.DeliveryServiceV4
	for _, ds := range dses.Response {
		if ds.Type == nil || ds.ID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service that had null or undefined Type and/or ID")
			continue
		}
		if *ds.Type == tc.DSTypeClientSteering {
			found = true
			ds.Topology = util.StrPtr("my-topology")
			if _, _, err := TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{}); err == nil {
				t.Errorf("assigning topology to CLIENT_STEERING delivery service - expected: error, actual: no error")
			}
		} else if nonCSDS == nil {
			nonCSDS = new(tc.DeliveryServiceV4)
			*nonCSDS = ds
		}
	}
	if !found || nonCSDS == nil {
		t.Fatal("Expected at least one non-CLIENT_STEERING Delivery Service to exist")
	}

	nonCSDS.Topology = new(string)
	_, inf, err := TOSession.UpdateDeliveryService(*nonCSDS.ID, *nonCSDS, client.RequestOptions{})
	if err == nil {
		t.Error("Expected an error assigning a non-existent topology")
	}
	if inf.StatusCode < 400 || inf.StatusCode >= 500 {
		t.Errorf("Expected client-level error assigning a non-existent topology, got: %d", inf.StatusCode)
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Add("xmlId", "ds-top-req-cap")
	dses, _, err = TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service: %v - alerts: %+v", err, dses.Alerts)
	}
	if len(dses.Response) != 1 {
		t.Fatalf("expected: 1 DS, actual: %d", len(dses.Response))
	}
	ds := dses.Response[0]
	if ds.Topology == nil || ds.ID == nil || ds.XMLID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service that had null or undefined Topology and/or XMLID and/or ID")
	}
	// unassign its topology, add a required capability that its topology
	// can't satisfy, then attempt to reassign its topology
	top := *ds.Topology
	ds.Topology = nil
	resp, _, err := TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating DS to remove topology, expected: no error, actual: %v - alerts: %+v", err, resp.Alerts)
	}
	reqCap := tc.DeliveryServicesRequiredCapability{
		DeliveryServiceID:  ds.ID,
		RequiredCapability: util.StrPtr("asdf"),
	}
	dsrcResp, _, err := TOSession.CreateDeliveryServicesRequiredCapability(reqCap, client.RequestOptions{})
	if err != nil {
		t.Fatalf("adding 'asdf' required capability to '%s', expected: no error, actual: %v - alerts: %+v", *ds.XMLID, err, dsrcResp.Alerts)
	}
	ds.Topology = &top
	_, reqInf, err := TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err == nil {
		t.Errorf("updating DS topology which doesn't meet the DS required capabilities - expected: error, actual: nil")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Errorf("updating DS topology which doesn't meet the DS required capabilities - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}
	dsrcResp, _, err = TOSession.DeleteDeliveryServicesRequiredCapability(*ds.ID, "asdf", client.RequestOptions{})
	if err != nil {
		t.Fatalf("removing 'asdf' required capability from '%s', expected: no error, actual: %v - alerts: %+v", *ds.XMLID, err, dsrcResp.Alerts)
	}
	_, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err != nil {
		t.Errorf("updating DS topology - expected: no error, actual: %v", err)
	}

	const xmlID = "top-ds-in-cdn2"
	dses, _, err = TOSession.GetDeliveryServices(client.RequestOptions{QueryParameters: url.Values{"xmlId": {xmlID}}})
	if err != nil {
		t.Fatalf("getting Delivery Services filtered by XMLID '%s': %v - alerts: %+v", xmlID, err, dses.Alerts)
	}
	const expectedSize = 1
	if len(dses.Response) != expectedSize {
		t.Fatalf("expected %d Delivery Service with xmlId '%s' but instead received %d Delivery Services", expectedSize, xmlID, len(dses.Response))
	}
	ds = dses.Response[0]
	if ds.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service that had null or undefined ID")
	}
	dsTopology := ds.Topology
	ds.Topology = nil
	resp, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating Delivery Service '%s' (#%d): %v - alerts: %+v", xmlID, *ds.ID, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service to have ID %d, found: %d", *ds.ID, len(resp.Response))
	}
	ds = resp.Response[0]
	if ds.CDNID == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service that had null or undefined CDN ID")
	}

	const cdn1Name = "cdn1"
	opts = client.NewRequestOptions()
	opts.QueryParameters.Set("name", cdn1Name)
	cdns, _, err := TOSession.GetCDNs(opts)
	if err != nil {
		t.Fatalf("getting CDN %s: %v - alerts: %+v", cdn1Name, err, cdns.Alerts)
	}
	if len(cdns.Response) != expectedSize {
		t.Fatalf("expected %d CDN with name %s but instead received %d CDNs", expectedSize, cdn1Name, len(cdns.Response))
	}
	cdn1 := cdns.Response[0]
	const cacheGroupName = "dtrc1"
	opts = client.RequestOptions{
		QueryParameters: url.Values{
			"name": {cacheGroupName},
		},
	}
	cachegroups, _, err := TOSession.GetCacheGroups(opts)
	if err != nil {
		t.Fatalf("getting Cache Group %s: %s", cacheGroupName, err.Error())
	}
	if len(cachegroups.Response) != expectedSize {
		t.Fatalf("expected %d Cache Group with name %s but instead received %d Cache Groups", expectedSize, cacheGroupName, len(cachegroups.Response))
	}
	cachegroup := cachegroups.Response[0]
	if cachegroup.ID == nil {
		t.Fatalf("Traffic Ops returned a representation for Cache Group '%s' that had null or undefined ID", cacheGroupName)
	}
	opts.QueryParameters = url.Values{"cdn": {strconv.Itoa(*ds.CDNID)}, "cachegroup": {strconv.Itoa(*cachegroup.ID)}}
	servers, _, err := TOSession.GetServers(opts)
	if err != nil {
		t.Fatalf("getting Server with params %v: %v - alerts: %+v", opts.QueryParameters, err, servers.Alerts)
	}
	if len(servers.Response) != expectedSize {
		t.Fatalf("expected %d Server returned for query params %v but instead received %d Servers", expectedSize, opts.QueryParameters, len(servers.Response))
	}
	server := servers.Response[0]
	if server.CDNID == nil {
		t.Error("Traffic Ops returned a representation for a Server that had null or undefined CDN ID")
		server.CDNID = new(int)
	}
	*server.CDNID = cdn1.ID

	if server.Profile == nil || server.ProfileID == nil || server.ProfileDesc == nil || server.ID == nil || server.HostName == nil {
		t.Fatal("Traffic Ops returned a representation for a Server that had null or undefined Profile and/or Profile ID and/or Profile Description and/or ID and/or Host Name")
	}
	// A profile specific to CDN 1 is required
	profileCopy := tc.ProfileCopy{
		Name:         *server.Profile + "_BUT_IN_CDN1",
		ExistingID:   *server.ProfileID,
		ExistingName: *server.Profile,
		Description:  *server.ProfileDesc,
	}
	copyResp, _, err := TOSession.CopyProfile(profileCopy, client.RequestOptions{})
	if err != nil {
		t.Fatalf("copying Profile %s: %v - alerts: %+v", *server.Profile, err, copyResp.Alerts)
	}

	profileOpts := client.NewRequestOptions()
	profileOpts.QueryParameters.Set("name", profileCopy.Name)
	profiles, _, err := TOSession.GetProfiles(profileOpts)
	if err != nil {
		t.Fatalf("getting Profile %s: %v - alerts: %+v", profileCopy.Name, err, profiles.Alerts)
	}
	if len(profiles.Response) != expectedSize {
		t.Fatalf("expected %d Profile with name %s but instead received %d Profiles", expectedSize, profileCopy.Name, len(profiles.Response))
	}
	profile := profiles.Response[0]
	profile.CDNID = cdn1.ID
	alerts, _, err := TOSession.UpdateProfile(profile.ID, profile, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating Profile %s: %v - alerts: %+v", profile.Name, err, alerts.Alerts)
	}
	*server.ProfileID = profile.ID

	// Empty Cache Group dtrc1 with respect to CDN 2
	alerts, _, err = TOSession.UpdateServer(*server.ID, server, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating Server '%s': %v - alerts: %+v", *server.HostName, err, alerts.Alerts)
	}
	ds.Topology = dsTopology
	_, reqInf, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err == nil {
		t.Fatalf("expected 400-level error assigning Topology %s to Delivery Service %s because Cache Group %s has no Servers in it in CDN %d, no error received", *dsTopology, xmlID, cacheGroupName, *ds.CDNID)
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("expected %d-level status code but received status code %d", http.StatusBadRequest, reqInf.StatusCode)
	}
	*server.CDNID = *ds.CDNID
	*server.ProfileID = profileCopy.ExistingID

	// Put things back the way they were
	alerts, _, err = TOSession.UpdateServer(*server.ID, server, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating Server '%s': %v - alerts: %+v", *server.HostName, err, alerts.Alerts)
	}

	alerts, _, err = TOSession.DeleteProfile(profile.ID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("deleting Profile %s: %v - alerts: %+v", profile.Name, err, alerts.Alerts)
	}

	resp, reqInf, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating Delivery Service '%s': %v - alerts: %+v", xmlID, err, resp.Alerts)
	}
}

// UpdateDeliveryServiceTopologyHeaderRewriteFields ensures that a delivery service can only use firstHeaderRewrite,
// innerHeaderRewrite, or lastHeadeRewrite if a topology is assigned.
func UpdateDeliveryServiceTopologyHeaderRewriteFields(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	foundTopology := false
	for _, ds := range dses.Response {
		if ds.ID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service that had null or undefined ID")
			continue
		}
		if ds.Topology != nil {
			foundTopology = true
		}
		ds.FirstHeaderRewrite = util.StrPtr("foo")
		ds.InnerHeaderRewrite = util.StrPtr("bar")
		ds.LastHeaderRewrite = util.StrPtr("baz")
		resp, _, err := TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
		if ds.Topology != nil && err != nil {
			t.Errorf("expected: no error updating topology-based header rewrite fields for topology-based DS, actual: %v - alerts: %+v", err, resp.Alerts)
		}
		if ds.Topology == nil && err == nil {
			t.Error("expected: error updating topology-based header rewrite fields for non-topology-based DS, actual: nil")
		}
		ds.FirstHeaderRewrite = nil
		ds.InnerHeaderRewrite = nil
		ds.LastHeaderRewrite = nil
		ds.EdgeHeaderRewrite = util.StrPtr("foo")
		ds.MidHeaderRewrite = util.StrPtr("bar")
		resp, _, err = TOSession.UpdateDeliveryService(*ds.ID, ds, client.RequestOptions{})
		if ds.Topology != nil && err == nil {
			t.Errorf("expected: error updating legacy header rewrite fields for topology-based DS, actual: nil")
		}
		if ds.Topology == nil && err != nil {
			t.Errorf("expected: no error updating legacy header rewrite fields for non-topology-based DS, actual: %v - alerts: %+v", err, resp.Alerts)
		}
	}
	if !foundTopology {
		t.Errorf("expected: at least one topology-based delivery service, actual: none found")
	}
}

// UpdateDeliveryServiceWithInvalidRemapText ensures that a delivery service can't be updated with a remap text value with a line break in it.
func UpdateDeliveryServiceWithInvalidRemapText(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test updating Delivery Service with invalid remap text")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in the test data that has null or undefined XMLID")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil || ds.ID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service that had null or undefined XMLID and/or ID")
			continue
		}
		if *ds.XMLID == *firstDS.XMLID {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Fatalf("GET Delivery Services missing: %s", *firstDS.XMLID)
	}

	updatedRemapText := "@plugin=tslua.so @pparam=/opt/trafficserver/etc/trafficserver/remapPlugin1.lua\nline2"
	remoteDS.RemapText = &updatedRemapText

	if _, _, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{}); err == nil {
		t.Errorf("Delivery Service successfully updated with invalid remap text: %v", updatedRemapText)
	}
}

// UpdateDeliveryServiceWithInvalidSliceRangeRequest ensures that a delivery service can't be updated with a invalid slice range request handler setting.
func UpdateDeliveryServiceWithInvalidSliceRangeRequest(t *testing.T) {
	// GET a HTTP / DNS type DS
	var dsXML *string
	for _, ds := range testData.DeliveryServices {
		if ds.Type == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service that had null or undefined Type")
			continue
		}
		if ds.Type.IsDNS() || ds.Type.IsHTTP() {
			dsXML = ds.XMLID
			break
		}
	}
	if dsXML == nil {
		t.Fatal("no HTTP or DNS Delivery Services to test with")
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var remoteDS tc.DeliveryServiceV4
	found := false
	for _, ds := range dses.Response {
		if ds.XMLID == nil || ds.ID == nil {
			t.Error("Traffic Ops returned a representation for a Delivery Service that had null or undefined XMLID and/or ID")
			continue
		}
		if *ds.XMLID == *dsXML {
			found = true
			remoteDS = ds
			break
		}
	}
	if !found {
		t.Fatalf("GET Delivery Services missing: %s", *dsXML)
	}

	testCases := []struct {
		description         string
		rangeRequestSetting *int
		slicePluginSize     *int
	}{
		{
			description:         "Missing slice plugin size",
			rangeRequestSetting: util.IntPtr(tc.RangeRequestHandlingSlice),
			slicePluginSize:     nil,
		},
		{
			description:         "Slice plugin size set with incorrect range request setting",
			rangeRequestSetting: util.IntPtr(tc.RangeRequestHandlingBackgroundFetch),
			slicePluginSize:     util.IntPtr(262144),
		},
		{
			description:         "Slice plugin size set to small",
			rangeRequestSetting: util.IntPtr(tc.RangeRequestHandlingSlice),
			slicePluginSize:     util.IntPtr(0),
		},
		{
			description:         "Slice plugin size set to large",
			rangeRequestSetting: util.IntPtr(tc.RangeRequestHandlingSlice),
			slicePluginSize:     util.IntPtr(40000000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			remoteDS.RangeSliceBlockSize = tc.slicePluginSize
			remoteDS.RangeRequestHandling = tc.rangeRequestSetting
			if _, _, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{}); err == nil {
				t.Error("Delivery Service successfully updated with invalid slice plugin configuration")
			}
		})
	}

}

// UpdateValidateORGServerCacheGroup validates ORG server's cachegroup are part of topology's cachegroup
func UpdateValidateORGServerCacheGroup(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "ds-top")

	//Get the correct DS
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service with the XMLID 'ds-top', found: %d", len(resp.Response))
	}
	remoteDS := resp.Response[0]
	if remoteDS.XMLID == nil || remoteDS.ID == nil || remoteDS.Topology == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service that had null or undefined XMLID and/or ID and/or Topology")
	}

	//Assign ORG server to DS
	assignServer := []string{"denver-mso-org-01"}
	alerts, _, err := TOSession.AssignServersToDeliveryService(assignServer, *remoteDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot assign server to Delivery Services: %v - alerts: %+v", err, alerts)
	}

	//Update DS's Topology to a non-ORG server's cachegroup
	origTopo := *remoteDS.Topology
	remoteDS.Topology = util.StrPtr("another-topology")
	ds, reqInf, err := TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{})
	if err == nil {
		t.Error("should not be able to update Delivery Service changing Topology when servers are assigned, but update was successful")
	} else {
		const msg = "the following ORG server cachegroups are not in the delivery service's topology"
		found := false
		for _, alert := range ds.Alerts.Alerts {
			if strings.Contains(alert.Text, msg) && alert.Level == tc.ErrorLevel.String() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected an error-level alert containing '%s' to be in the response, but it was not found", msg)
		}
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected to fail since ORG server's topology not part of DS. Expected:%v, Got: :%v", http.StatusBadRequest, reqInf.StatusCode)
	}

	// Retrieve the DS to check if topology was updated with missing ORG server
	// TODO: clear params?
	opts.QueryParameters.Set("id", strconv.Itoa(*remoteDS.ID))
	apiResp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Service #%d: %v - alerts: %+v", *remoteDS.ID, err, apiResp.Alerts)
	}
	if len(apiResp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service to exist with ID %d, found: %d", *remoteDS.ID, len(apiResp.Response))
	}

	//Set topology back to as it was for further testing
	remoteDS.Topology = &origTopo
	resp, _, err = TOSession.UpdateDeliveryService(*remoteDS.ID, remoteDS, client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't update DS Topology to '%s': %v - alerts: %+v", *remoteDS.Topology, err, resp.Alerts)
	}
}

func GetAccessibleToTest(t *testing.T) {
	//Every delivery service is associated with the root tenant
	err := getByTenants(1, len(testData.DeliveryServices))
	if err != nil {
		t.Fatal(err.Error())
	}

	tenant := tc.Tenant{
		Active:     true,
		Name:       "the strongest",
		ParentID:   1,
		ParentName: "root",
	}

	resp, _, err := TOSession.CreateTenant(tenant, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error creating a tenant: %v - alerts: %+v", err, resp.Alerts)
	}
	tenant = resp.Response

	//No delivery services are associated with this new tenant
	err = getByTenants(tenant.ID, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	//First and only child tenant, no access to root
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "tenant1")
	childTenant, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Fatalf("unable to get tenant: %v - alerts: %+v", err, childTenant.Alerts)
	}
	if len(childTenant.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'tenant1', found: %d", len(childTenant.Response))
	}

	// TODO: document that all DSes added to the fixture data need to have the
	// Tenant 'tenant1' unless you change this code
	err = getByTenants(childTenant.Response[0].ID, len(testData.DeliveryServices)-1)
	if err != nil {
		t.Fatal(err.Error())
	}

	alerts, _, err := TOSession.DeleteTenant(tenant.ID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("unable to clean up Tenant: %v - alerts: %+v", err, alerts.Alerts)
	}
}

func getByTenants(tenantID int, expectedCount int) error {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("accessibleTo", strconv.Itoa(tenantID))
	deliveryServices, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		return fmt.Errorf("Unexpected error fetching Delivery Services for Tenant #%d: %v - alerts: %+v", tenantID, err, deliveryServices.Alerts)
	}
	if len(deliveryServices.Response) != expectedCount {
		return fmt.Errorf("expected %d delivery service, got %d", expectedCount, len(deliveryServices.Response))
	}
	return nil
}

func DeleteTestDeliveryServices(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	for _, testDS := range testData.DeliveryServices {
		if testDS.XMLID == nil {
			t.Error("Found a Delivery Service in testing data with null or undefined XMLID")
			continue
		}
		var ds tc.DeliveryServiceV4
		found := false
		for _, realDS := range dses.Response {
			if realDS.XMLID == nil || realDS.ID == nil {
				t.Errorf("Traffic Ops returned a representation for a Delivery Service with null or undefined XMLID and/or ID")
				continue
			}
			if *realDS.XMLID == *testDS.XMLID {
				ds = realDS
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Delivery Service not found in Traffic Ops: %s", *testDS.XMLID)
			continue
		}

		delResp, _, err := TOSession.DeleteDeliveryService(*ds.ID, client.RequestOptions{})
		if err != nil {
			t.Errorf("cannot delete DeliveryService by ID: %v - alerts: %+v", err, delResp.Alerts)
			continue
		}

		// Retrieve the Server to see if it got deleted
		opts := client.NewRequestOptions()
		opts.QueryParameters.Set("id", strconv.Itoa(*ds.ID))
		foundDS, _, err := TOSession.GetDeliveryServices(opts)
		if err != nil {
			t.Errorf("Unexpected error deleting Delivery Service '%s': %v - alelts: %+v", *ds.XMLID, err, foundDS.Alerts)
		}
		if len(foundDS.Response) > 0 {
			t.Errorf("expected Delivery Service: %s to be deleted, but %d exist with same ID (#%d)", *ds.XMLID, len(foundDS.Response), *ds.ID)
		}
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "location")
	opts.QueryParameters.Set("configFile", "remap.config")
	params, _, err := TOSession.GetParameters(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Parameters with name 'location' and configFile 'remap.config': %v - alerts: %+v", err, params.Alerts)
	}
	for _, param := range params.Response {
		deleted, _, err := TOSession.DeleteParameter(param.ID, client.RequestOptions{})
		if err != nil {
			t.Errorf("cannot delete Parameter by ID (%d): %v - alerts: %+v", param.ID, err, deleted.Alerts)
		}
	}
}

func DeliveryServiceMinorVersionsTest(t *testing.T) {
	if len(testData.DeliveryServices) < 5 {
		t.Fatalf("Need at least 5 DSes to test minor versions; got: %d", len(testData.DeliveryServices))
	}
	testDS := testData.DeliveryServices[4]
	if testDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}
	if *testDS.XMLID != "ds-test-minor-versions" {
		t.Errorf("expected XMLID: ds-test-minor-versions, actual: %s", *testDS.XMLID)
	}

	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}

	var ds tc.DeliveryServiceV4
	found := false
	for _, d := range dses.Response {
		if d.XMLID != nil && *d.XMLID == *testDS.XMLID {
			ds = d
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Delivery Service '%s' not found in Traffic Ops", *testDS.XMLID)
	}

	// GET latest, verify expected values for 1.3 and 1.4 fields
	if ds.DeepCachingType == nil {
		t.Errorf("expected DeepCachingType: %s, actual: nil", testDS.DeepCachingType.String())
	} else if *ds.DeepCachingType != *testDS.DeepCachingType {
		t.Errorf("expected DeepCachingType: %s, actual: %s", testDS.DeepCachingType.String(), ds.DeepCachingType.String())
	}
	if ds.FQPacingRate == nil {
		t.Errorf("expected FQPacingRate: %d, actual: nil", testDS.FQPacingRate)
	} else if *ds.FQPacingRate != *testDS.FQPacingRate {
		t.Errorf("expected FQPacingRate: %d, actual: %d", testDS.FQPacingRate, *ds.FQPacingRate)
	}
	if ds.SigningAlgorithm == nil {
		t.Errorf("expected SigningAlgorithm: %s, actual: nil", *testDS.SigningAlgorithm)
	} else if *ds.SigningAlgorithm != *testDS.SigningAlgorithm {
		t.Errorf("expected SigningAlgorithm: %s, actual: %s", *testDS.SigningAlgorithm, *ds.SigningAlgorithm)
	}
	if ds.Tenant == nil {
		t.Errorf("expected Tenant: %s, actual: nil", *testDS.Tenant)
	} else if *ds.Tenant != *testDS.Tenant {
		t.Errorf("expected Tenant: %s, actual: %s", *testDS.Tenant, *ds.Tenant)
	}
	if ds.TRRequestHeaders == nil {
		t.Errorf("expected TRRequestHeaders: %s, actual: nil", *testDS.TRRequestHeaders)
	} else if *ds.TRRequestHeaders != *testDS.TRRequestHeaders {
		t.Errorf("expected TRRequestHeaders: %s, actual: %s", *testDS.TRRequestHeaders, *ds.TRRequestHeaders)
	}
	if ds.TRResponseHeaders == nil {
		t.Errorf("expected TRResponseHeaders: %s, actual: nil", *testDS.TRResponseHeaders)
	} else if *ds.TRResponseHeaders != *testDS.TRResponseHeaders {
		t.Errorf("expected TRResponseHeaders: %s, actual: %s", *testDS.TRResponseHeaders, *ds.TRResponseHeaders)
	}
	if ds.ConsistentHashRegex == nil {
		t.Errorf("expected ConsistentHashRegex: %s, actual: nil", *testDS.ConsistentHashRegex)
	} else if *ds.ConsistentHashRegex != *testDS.ConsistentHashRegex {
		t.Errorf("expected ConsistentHashRegex: %s, actual: %s", *testDS.ConsistentHashRegex, *ds.ConsistentHashRegex)
	}
	if ds.ConsistentHashQueryParams == nil {
		t.Errorf("expected ConsistentHashQueryParams: %v, actual: nil", testDS.ConsistentHashQueryParams)
	} else if !reflect.DeepEqual(ds.ConsistentHashQueryParams, testDS.ConsistentHashQueryParams) {
		t.Errorf("expected ConsistentHashQueryParams: %v, actual: %v", testDS.ConsistentHashQueryParams, ds.ConsistentHashQueryParams)
	}
	if ds.MaxOriginConnections == nil {
		t.Errorf("expected MaxOriginConnections: %d, actual: nil", testDS.MaxOriginConnections)
	} else if *ds.MaxOriginConnections != *testDS.MaxOriginConnections {
		t.Errorf("expected MaxOriginConnections: %d, actual: %d", testDS.MaxOriginConnections, *ds.MaxOriginConnections)
	}

	ds.ID = nil
	_, err = json.Marshal(ds)
	if err != nil {
		// TODO: should this actually be doing a POST?
		t.Errorf("cannot POST deliveryservice, failed to marshal JSON: %s", err.Error())
	}
}

func DeliveryServiceTenancyTest(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	var tenant3DS tc.DeliveryServiceV4
	foundTenant3DS := false
	for _, d := range dses.Response {
		if d.XMLID == nil || d.ID == nil || d.Tenant == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service that had null or undefined XMLID and/or ID and/or Tenant")
			continue
		}
		if *d.XMLID == "ds3" {
			tenant3DS = d
			foundTenant3DS = true
		}
	}
	if !foundTenant3DS || *tenant3DS.Tenant != "tenant3" {
		t.Fatal("expected to find deliveryservice 'ds3' with tenant 'tenant3'")
	}

	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	tenant4TOClient, _, err := client.LoginWithAgent(TOSession.URL, "tenant4user", "pa$$word", true, "to-api-v4-client-tests/tenant4user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to log in with tenant4user: %v", err.Error())
	}

	dsesReadableByTenant4, _, err := tenant4TOClient.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("tenant4user cannot GET deliveryservices: %v - alerts: %+v", err, dsesReadableByTenant4.Alerts)
	}

	// assert that tenant4user cannot read deliveryservices outside of its tenant
	for _, ds := range dsesReadableByTenant4.Response {
		if ds.XMLID == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service that had null or undefined XMLID")
			continue
		}
		if *ds.XMLID == "ds3" {
			t.Error("expected tenant4 to be unable to read delivery services from tenant 3")
		}
	}

	// assert that tenant4user cannot update tenant3user's deliveryservice
	if _, _, err = tenant4TOClient.UpdateDeliveryService(*tenant3DS.ID, tenant3DS, client.RequestOptions{}); err == nil {
		t.Errorf("expected tenant4user to be unable to update tenant3's deliveryservice (%s)", *tenant3DS.XMLID)
	}

	// assert that tenant4user cannot delete tenant3user's deliveryservice
	if _, _, err = tenant4TOClient.DeleteDeliveryService(*tenant3DS.ID, client.RequestOptions{}); err == nil {
		t.Errorf("expected tenant4user to be unable to delete tenant3's deliveryservice (%s)", *tenant3DS.XMLID)
	}

	// assert that tenant4user cannot create a deliveryservice outside of its tenant
	tenant3DS.XMLID = util.StrPtr("deliveryservicetenancytest")
	tenant3DS.DisplayName = util.StrPtr("deliveryservicetenancytest")
	if _, _, err = tenant4TOClient.CreateDeliveryService(tenant3DS, client.RequestOptions{}); err == nil {
		t.Error("expected tenant4user to be unable to create a deliveryservice outside of its tenant")
	}
}

func VerifyPaginationSupportDS(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("orderby", "id")
	deliveryservice, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, deliveryservice.Alerts)
	}
	if len(deliveryservice.Response) < 3 {
		t.Fatalf("Need at least three Delivery Services to test pagination, found: %d", len(deliveryservice.Response))
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("orderby", "id")
	opts.QueryParameters.Set("limit", "1")
	deliveryserviceWithLimit, _, err := TOSession.GetDeliveryServices(opts)
	if !reflect.DeepEqual(deliveryservice.Response[:1], deliveryserviceWithLimit.Response) {
		t.Error("expected GET deliveryservice with limit = 1 to return first result")
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("orderby", "id")
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("offset", "1")
	deliveryserviceWithOffset, _, err := TOSession.GetDeliveryServices(opts)
	if !reflect.DeepEqual(deliveryservice.Response[1:2], deliveryserviceWithOffset.Response) {
		t.Error("expected GET deliveryservice with limit = 1, offset = 1 to return second result")
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("orderby", "id")
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("page", "2")
	deliveryserviceWithPage, _, err := TOSession.GetDeliveryServices(opts)
	if !reflect.DeepEqual(deliveryservice.Response[1:2], deliveryserviceWithPage.Response) {
		t.Error("expected GET deliveryservice with limit = 1, page = 2 to return second result")
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "-2")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err == nil {
		t.Error("expected GET deliveryservice to return an error when limit is not bigger than -1")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be bigger than -1") {
		t.Errorf("expected GET deliveryservice to return an error for limit is not bigger than -1, actual error: %v - alerts: %+v", err, resp.Alerts)
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("offset", "0")
	resp, _, err = TOSession.GetDeliveryServices(opts)
	if err == nil {
		t.Error("expected GET deliveryservice to return an error when offset is not a positive integer")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be a positive integer") {
		t.Errorf("expected GET deliveryservice to return an error for offset is not a positive integer, actual error: %v - alerts: %+v", err, resp.Alerts)
	}

	opts.QueryParameters = url.Values{}
	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("page", "0")
	resp, _, err = TOSession.GetDeliveryServices(opts)
	if err == nil {
		t.Error("expected GET deliveryservice to return an error when page is not a positive integer")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be a positive integer") {
		t.Errorf("expected GET deliveryservice to return an error for page is not a positive integer, actual error: %v - alerts: %+v", err, resp.Alerts)
	}
}

func GetDeliveryServiceByCdn(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services by CDN")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.CDNName == nil {
		t.Fatal("CDN Name is nil in the pre-requisites")
	}

	opts := client.NewRequestOptions()
	if firstDS.CDNID == nil {
		opts.QueryParameters.Set("name", *firstDS.CDNName)
		cdns, _, err := TOSession.GetCDNs(opts)
		if err != nil {
			t.Errorf("Unexpected error getting CDN '%s' by name: %v - alerts: %+v", *firstDS.CDNName, err, cdns.Alerts)
		}
		if len(cdns.Response) != 1 {
			t.Fatalf("Expected exactly one CDN named '%s' to exist, found: %d", *firstDS.CDNName, len(cdns.Response))
		}
		firstDS.CDNID = new(int)
		*firstDS.CDNID = cdns.Response[0].ID
		opts.QueryParameters.Del("name")
	}

	opts.QueryParameters.Set("cdn", strconv.Itoa(*firstDS.CDNID))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by CDN ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Delivery Service to exist in CDN '%s' (#%d)", *firstDS.CDNName, *firstDS.CDNID)
	}
	if resp.Response[0].CDNName == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined CDN Name")
	}
	if *resp.Response[0].CDNName != *firstDS.CDNName {
		t.Errorf("CDN Name expected: '%s', actual: '%s'", *firstDS.CDNName, *resp.Response[0].CDNName)
	}
}

func GetDeliveryServiceByInvalidCdn(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("cdn", "10000")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by presumably non-existent CDN ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services in presumably non-existent CDN, found: %d", len(resp.Response))
	}
}

func GetDeliveryServiceByInvalidProfile(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("profile", "10000")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by presumably non-existent Profile ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services with presumably non-existent Profile, found: %d", len(resp.Response))
	}
}

func GetDeliveryServiceByInvalidTenant(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("tenant", "10000")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by presumably non-existent Tenant ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services with presumably non-existent Tenant, found: %d", len(resp.Response))
	}
}

func GetDeliveryServiceByInvalidType(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("type", "10000")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by presumably non-existent Type ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services with presumably non-existent Type, found: %d", len(resp.Response))
	}
}

func GetDeliveryServiceByInvalidAccessibleTo(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("accessibleTo", "10000")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by accessibility to presumably non-existent Tenant ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services accessible to presumably non-existent Tenant, found: %d", len(resp.Response))
	}
}

func GetDeliveryServiceByInvalidXmlId(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "test")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by presumably non-existentXMLID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) >= 1 {
		t.Errorf("Didn't expect to find any Delivery Services with presumably non-existent XMLID, found: %d", len(resp.Response))
	}
}

func GetTestDeliveryServicesURLSignatureKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	_, _, err := TOSession.GetDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("failed to get url sig keys: %v", err)
	}
}

func CreateTestDeliveryServicesURLSignatureKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]

	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	resp, _, err := TOSession.CreateDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creaetting URL signing keys: %v - alerts: %+v", err, resp.Alerts)
	}
	firstKeys, _, err := TOSession.GetDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting URL signing keys: %v - alerts: %+v", err, firstKeys.Alerts)
	}
	if len(firstKeys.Response) == 0 {
		t.Errorf("failed to create URL signing keys")
	}
	firstKey, ok := firstKeys.Response["key0"]
	if !ok {
		t.Fatal("Expected to find 'key0' in URL signing keys, but didn't")
	}

	// Create new keys again and check that they are different
	resp, _, err = TOSession.CreateDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating URL signing keys: %v - alerts: %+v", err, resp.Alerts)
	}
	secondKeys, _, err := TOSession.GetDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting URL signing keys: %v - alerts: %+v", err, secondKeys.Alerts)
	}
	if len(secondKeys.Response) == 0 {
		t.Errorf("failed to create url sig keys")
	}
	secondKey, ok := secondKeys.Response["key0"]
	if !ok {
		t.Fatal("Expected to find 'key0' in URL signing keys, but didn't")
	}

	if secondKey == firstKey {
		t.Errorf("second create did not generate new url sig keys")
	}
}

func DeleteTestDeliveryServicesURLSignatureKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]

	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	resp, _, err := TOSession.DeleteDeliveryServiceURLSignatureKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deletining URL signing keys: %v - alerts: %+v", err, resp.Alerts)
	}

}

func GetTestDeliveryServicesURISigningKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]

	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	_, _, err := TOSession.GetDeliveryServiceURISigningKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting URI signing keys for Delivery Service '%s': %v", *firstDS.XMLID, err)
	}
}

const (
	keySet1 = `
{
  "Kabletown URI Authority 1": {
    "renewal_kid": "First Key",
    "keys": [
      {
        "alg": "HS256",
        "kid": "First Key",
        "kty": "oct",
        "k": "Kh_RkUMj-fzbD37qBnDf_3e_RvQ3RP9PaSmVEpE24AM"
      }
    ]
  }
}`
	keySet2 = `
{
"Kabletown URI Authority 1": {
    "renewal_kid": "New First Key",
    "keys": [
      {
        "alg": "HS256",
        "kid": "New First Key",
        "kty": "oct",
        "k": "Kh_RkUMj-fzbD37qBnDf_3e_RvQ3RP9PaSmVEpE24AM"
      }
    ]
  }
}`
)

func CreateTestDeliveryServicesURISigningKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	var keyset map[string]tc.URISignerKeyset

	if err := json.Unmarshal([]byte(keySet1), &keyset); err != nil {
		t.Errorf("json.UnMarshal(): expected nil error, actual: %v", err)
	}

	_, _, err := TOSession.CreateDeliveryServiceURISigningKeys(*firstDS.XMLID, keyset, client.RequestOptions{})
	if err != nil {
		t.Error("failed to create uri sig keys: " + err.Error())
	}

	firstKeysBytes, _, err := TOSession.GetDeliveryServiceURISigningKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Error("failed to get uri sig keys: " + err.Error())
	}

	firstKeys := map[string]tc.URISignerKeyset{}
	if err := json.Unmarshal(firstKeysBytes, &firstKeys); err != nil {
		t.Errorf("failed to unmarshal uri sig keys")
	}

	kabletownFirstKeys, ok := firstKeys["Kabletown URI Authority 1"]
	if !ok {
		t.Fatal("failed to create uri sig keys: 'Kabletown URI Authority 1' not found in response after creation")
	}
	if len(kabletownFirstKeys.Keys) < 1 {
		t.Fatal("failed to create URI signing keys: 'Kabletown URI Authority 1' had zero keys after creation")
	}

	// Create new keys again and check that they are different
	var keyset2 map[string]tc.URISignerKeyset

	if err := json.Unmarshal([]byte(keySet2), &keyset2); err != nil {
		t.Errorf("json.UnMarshal(): expected nil error, actual: %v", err)
	}

	alerts, _, err := TOSession.CreateDeliveryServiceURISigningKeys(*firstDS.XMLID, keyset2, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating URI Signature Keys for Delivery Service '%s': %v - alerts: %+v", *firstDS.XMLID, err, alerts.Alerts)
	}

	secondKeysBytes, _, err := TOSession.GetDeliveryServiceURISigningKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Error("failed to get uri sig keys: " + err.Error())
	}

	secondKeys := map[string]tc.URISignerKeyset{}
	if err := json.Unmarshal(secondKeysBytes, &secondKeys); err != nil {
		t.Errorf("failed to unmarshal uri sig keys")
	}

	kabletownSecondKeys, ok := secondKeys["Kabletown URI Authority 1"]
	if !ok {
		t.Fatal("failed to create uri sig keys: 'Kabletown URI Authority 1' not found in response after creation")
	}
	if len(kabletownSecondKeys.Keys) < 1 {
		t.Fatal("failed to create URI signing keys: 'Kabletown URI Authority 1' had zero keys after creation")
	}

	if kabletownSecondKeys.Keys[0].KeyID == kabletownFirstKeys.Keys[0].KeyID {
		t.Errorf("second create did not generate new uri sig keys - key mismatch")
	}
}

func DeleteTestDeliveryServicesURISigningKeys(t *testing.T) {
	if len(testData.DeliveryServices) == 0 {
		t.Fatal("couldn't get the xml ID of test DS")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	resp, _, err := TOSession.DeleteDeliveryServiceURISigningKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error deleting URI Signing keys for Delivery Service '%s': %v - alerts: %+v", *firstDS.XMLID, err, resp.Alerts)
	}

	emptyBytes, _, err := TOSession.GetDeliveryServiceURISigningKeys(*firstDS.XMLID, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting URI signing keys for Delivery Service '%s': %v", *firstDS.XMLID, err)
	}
	emptyMap := make(map[string]interface{})
	err = json.Unmarshal(emptyBytes, &emptyMap)
	if err != nil {
		t.Errorf("unexpected error unmarshalling empty URI signing keys response: %v", err)
	}
	renewalKid, hasRenewalKid := emptyMap["renewal_kid"]
	keys, hasKeys := emptyMap["keys"]
	if !hasRenewalKid {
		t.Error("getting empty URI signing keys - expected: 'renewal_kid' key, actual: not present")
	}
	if !hasKeys {
		t.Error("getting empty URI signing keys - expected: 'keys' key, actual: not present")
	}
	if renewalKid != nil {
		t.Errorf("getting empty URI signing keys - expected: 'renewal_kid' value to be nil, actual: %+v", renewalKid)
	}
	if keys != nil {
		t.Errorf("getting empty URI signing keys - expected: 'keys' value to be nil, actual: %+v", keys)
	}
}

func GetDeliveryServiceByLogsEnabled(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services filtered by their Logs Enabled property")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.LogsEnabled == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined LogsEnabled")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("logsEnabled", strconv.FormatBool(*firstDS.LogsEnabled))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by 'logsEnabled': %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Delivery Service to exist with Logs Enabled set to %t", *firstDS.LogsEnabled)
	}
	if resp.Response[0].LogsEnabled == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined Logs Enabled property")
	}
	if *resp.Response[0].LogsEnabled != *firstDS.LogsEnabled {
		t.Errorf("Logs enabled status expected: %t, actual: %t", *firstDS.LogsEnabled, *resp.Response[0].LogsEnabled)
	}
}

// Note this test assumes that the first Delivery Service in the testing data's
// deliveryservices array has a Profile.
func GetDeliveryServiceByValidProfile(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services filtered by their Profile ID")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.ProfileName == nil {
		t.Fatal("Profile name is nil in the Pre-requisites")
	}

	opts := client.NewRequestOptions()
	if firstDS.ProfileID == nil {
		opts.QueryParameters.Set("name", *firstDS.ProfileName)
		profile, _, err := TOSession.GetProfiles(opts)
		if err != nil {
			t.Errorf("Unexpected error getting Profiles filtered by name: %v - alerts: %+v", err, profile.Alerts)
		}
		if len(profile.Response) != 1 {
			t.Fatalf("Expected exactly one Profile to exist with name '%s', found %d:", *firstDS.ProfileName, len(profile.Response))
		}
		firstDS.ProfileID = new(int)
		*firstDS.ProfileID = profile.Response[0].ID
		opts.QueryParameters.Del("name")
	}

	opts.QueryParameters.Set("profile", strconv.Itoa(*firstDS.ProfileID))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by Profile ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Delivery Service to exist with Profile '%s'", *firstDS.ProfileName)
	}
	if resp.Response[0].ProfileName == nil {
		t.Fatal("Traffic Ops returned a representation for a Delivery Service with null or undefined Profile Name")
	}
	if *resp.Response[0].ProfileName != *firstDS.ProfileName {
		t.Errorf("Profile name expected: '%s', actual: '%s'", *firstDS.ProfileName, *resp.Response[0].ProfileName)
	}
}

func GetDeliveryServiceByValidTenant(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services filtered by their Tenant IDs")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.Tenant == nil {
		t.Fatal("Tenant name is nil in the Pre-requisites")
	}

	opts := client.NewRequestOptions()
	if firstDS.TenantID == nil {
		opts.QueryParameters.Set("name", *firstDS.Tenant)
		tenants, _, err := TOSession.GetTenants(opts)
		if err != nil {
			t.Errorf("Unexpected error getting Tenants filtered by name: %v - alerts: %+v", err, tenants.Alerts)
		}
		if len(tenants.Response) != 1 {
			t.Fatalf("Expected exactly one Tenant to exist with name '%s', found %d:", *firstDS.Tenant, len(tenants.Response))
		}
		firstDS.TenantID = new(int)
		*firstDS.TenantID = tenants.Response[0].ID
		opts.QueryParameters.Del("name")
	}

	opts.QueryParameters.Set("tenant", strconv.Itoa(*firstDS.TenantID))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by Tenant ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Delivery Service to exist belonging to Tenant '%s'", *firstDS.Tenant)
	}
	if resp.Response[0].Tenant == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service with null or undefined Tenant")
	}
	if *resp.Response[0].Tenant != *firstDS.Tenant {
		t.Errorf("Tenant name expected: '%s', actual: '%s'", *firstDS.Tenant, *resp.Response[0].Tenant)
	}
}

func GetDeliveryServiceByValidType(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services filtered by Type")
	}
	firstDS := testData.DeliveryServices[0]
	if firstDS.Type == nil {
		t.Fatal("Type name is nil in the Pre-requisites")
	}

	opts := client.NewRequestOptions()
	if firstDS.TypeID == nil {
		opts.QueryParameters.Set("name", firstDS.Type.String())
		types, _, err := TOSession.GetTypes(opts)
		if err != nil {
			t.Errorf("Unexpected error getting Types filtered by name: %v - alerts: %+v", err, types.Alerts)
		}
		if len(types.Response) != 1 {
			t.Fatalf("Expected exactly one Type to exist with name '%s', found %d:", *firstDS.Type, len(types.Response))
		}
		firstDS.TypeID = new(int)
		*firstDS.TypeID = types.Response[0].ID
		opts.QueryParameters.Del("name")
	}

	opts.QueryParameters.Set("type", strconv.Itoa(*firstDS.TypeID))
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by Type ID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("Expected at least one Delivery Service to exist with Type '%s' (#%d)", *firstDS.Type, *firstDS.TypeID)
	}
	if resp.Response[0].Type == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service with null or undefined Type Name")
	}
	if *resp.Response[0].Type != *firstDS.Type {
		t.Errorf("Type expected: '%s', actual: '%s'", *firstDS.Type, *resp.Response[0].Type)
	}
}

func GetDeliveryServiceByValidXmlId(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test getting Delivery Services filtered by XMLID")
	}
	firstDS := testData.DeliveryServices[0]

	if firstDS.XMLID == nil {
		t.Fatal("Found a Delivery Service in testing data with a null or undefined XMLID")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", *firstDS.XMLID)
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services filtered by XMLID: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Delivery Service to exist with XMLID '%s', found: %d", *firstDS.XMLID, len(resp.Response))
	}
	if resp.Response[0].XMLID == nil {
		t.Fatal("Traffic Ops returned a representation of a Delivery Service with null or undefined XMLID")
	}
	if *resp.Response[0].XMLID != *firstDS.XMLID {
		t.Errorf("Delivery Service XMLID expected: %s, actual: %s", *firstDS.XMLID, *resp.Response[0].XMLID)
	}
}

func SortTestDeliveryServicesDesc(t *testing.T) {
	resp, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services with default sort order: %v - alerts: %+v", err, resp.Alerts)
	}
	respAsc := resp.Response
	if len(respAsc) == 0 {
		t.Fatal("Need at least one Delivery Service in Traffic Ops to test sort order")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("sortOrder", "desc")
	resp, _, err = TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error getting Delivery Services with explicit descending sort order: %v - alerts: %+v", err, resp.Alerts)
	}
	respDesc := resp.Response
	if len(respDesc) == 0 {
		t.Fatal("Need at least one Delivery Service in Traffic Ops to test sort order")
	}

	// TODO: test the entire array(s)?
	// TODO: check that the responses have the same length?
	// TODO: check that the responses have more than one entry, since otherwise it's trivially sorted anyway?
	// reverse the descending-sorted response and compare it to the ascending-sorted one
	for start, end := 0, len(respDesc)-1; start < end; start, end = start+1, end-1 {
		respDesc[start], respDesc[end] = respDesc[end], respDesc[start]
	}
	if respDesc[0].XMLID != nil && respAsc[0].XMLID != nil {
		if !reflect.DeepEqual(respDesc[0].XMLID, respAsc[0].XMLID) {
			t.Errorf("Delivery Service responses are not equal after reversal: %v - %v", *respDesc[0].XMLID, *respAsc[0].XMLID)
		}
	}
}

func addTLSVersionsToDeliveryService(t *testing.T) {
	me, _, err := TOSession.GetUserCurrent(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Failed to get current User: %v - alerts: %+v", err, me.Alerts)
	}
	if me.Response.Tenant == nil || me.Response.TenantID == nil {
		t.Fatal("Traffic Ops returned a representation for the current user with null or undefined tenant and/or tenantID")
	}

	var ds tc.DeliveryServiceV4
	ds.Active = new(bool)
	ds.CDNName = new(string)
	ds.DisplayName = new(string)
	ds.DSCP = new(int)
	ds.GeoLimit = new(int)
	ds.GeoProvider = new(int)
	ds.InitialDispersion = new(int)
	ds.IPV6RoutingEnabled = new(bool)
	ds.LogsEnabled = new(bool)
	ds.MissLat = new(float64)
	ds.MissLong = new(float64)
	ds.MultiSiteOrigin = new(bool)
	ds.OrgServerFQDN = new(string)
	ds.Protocol = new(int)
	ds.QStringIgnore = new(int)
	ds.RangeRequestHandling = new(int)
	ds.RegionalGeoBlocking = new(bool)
	ds.Tenant = new(string)
	ds.TenantID = me.Response.TenantID
	ds.TLSVersions = []string{
		"1.1",
	}
	ds.Type = new(tc.DSType)
	ds.XMLID = new(string)
	*ds.DSCP = 1
	*ds.InitialDispersion = 1
	*ds.Tenant = *me.Response.Tenant
	*ds.DisplayName = "ds-test-tls-versions"
	*ds.XMLID = "ds-test-tls-versions"

	cdns, _, err := TOSession.GetCDNs(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Failed to get CDNs: %v - alerts: %+v", err, cdns.Alerts)
	}
	if len(cdns.Response) < 1 {
		t.Fatalf("Need at least one CDN to exist in order to test Delivery Service TLS Versions")
	}
	ds.CDNID = &cdns.Response[0].ID
	*ds.CDNName = cdns.Response[0].Name

	*ds.Type = "STEERING"
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", string(*ds.Type))
	types, _, err := TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("Failed to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) != 1 {
		t.Fatalf("Expected exactly one Type to exist named 'STEERING', found: %d", len(types.Response))
	}
	ds.TypeID = &types.Response[0].ID

	_, _, err = TOSession.CreateDeliveryService(ds, client.RequestOptions{})
	if err == nil {
		t.Error("Expected an error creating a STEERING Delivery Service with explicit TLS Versions, but didn't")
	} else if !strings.Contains(err.Error(), "'tlsVersions' must be 'null' for STEERING-Type") {
		t.Errorf("Expected an error about non-null TLS Versions for STEERING-Type Delivery Services, got: %v", err)
	}

	*ds.Type = "HTTP"
	opts.QueryParameters.Set("name", string(*ds.Type))
	types, _, err = TOSession.GetTypes(opts)
	if err != nil {
		t.Fatalf("Failed to get Types: %v - alerts: %+v", err, types.Alerts)
	}
	if len(types.Response) != 1 {
		t.Fatalf("Expected exactly one Type to exist named 'HTTP', found: %d", len(types.Response))
	}
	ds.TypeID = &types.Response[0].ID

	*ds.OrgServerFQDN = "https://origin.test"
	resp, _, err := TOSession.CreateDeliveryService(ds, client.RequestOptions{})
	if err != nil {
		t.Errorf("Unexpected error creating a Delivery Service: %v - alerts: %+v", err, resp.Alerts)
	} else if len(resp.Response) != 1 {
		t.Errorf("Expected creating a new Delivery Service to create exactly one Delivery Service, but Traffic Ops indicated that %d were created", len(resp.Response))
	} else if resp.Response[0].ID == nil {
		t.Error("Traffic Ops returned a representation for a created Delivery Service that had null or undefined ID")
	} else {
		alerts, _, err := TOSession.DeleteDeliveryService(*resp.Response[0].ID, client.RequestOptions{})
		if err != nil {
			t.Errorf("Failed to clean up newly created Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
		}
	}
}
