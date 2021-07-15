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
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestOrigins(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Users, Topologies, DeliveryServices, Coordinates, Origins}, func() {
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfUnmodifiedSince, time)
		CreateTestOriginDuplicateData(t)
		GetTestOriginsByParams(t)
		GetTestOriginsByInvalidParams(t)
		UpdateTestOrigins(t)
		UpdateTestOriginsWithHeaders(t, header)
		GetTestOrigins(t)
		NotFoundDeleteTest(t)
		OriginTenancyTest(t)
		GetTestPaginationSupportOrigins(t)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestOriginsWithHeaders(t, header)
		CreateTestOriginInvalidData(t)
		updateTestOriginsWithInvalidData(t)
	})
}

func UpdateTestOriginsWithHeaders(t *testing.T, header http.Header) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test updating Origins with an HTTP header")
	}
	firstOrigin := testData.Origins[0]
	if firstOrigin.Name == nil {
		t.Fatalf("couldn't get the name of test origin server")
	}

	// Retrieve the origin by name so we can get the id for the Update
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", *firstOrigin.Name)
	resp, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin '%s'': %v - alerts: %+v", *firstOrigin.Name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Origin to exist with name '%s', found: %d", *firstOrigin.Name, len(resp.Response))
	}

	remoteOrigin := resp.Response[0]
	if remoteOrigin.ID == nil {
		t.Fatal("couldn't get the ID of the response origin server")
	}
	updatedPort := 4321
	updatedFQDN := "updated.example.com"

	// update port and FQDN values on origin
	remoteOrigin.Port = &updatedPort
	remoteOrigin.FQDN = &updatedFQDN
	opts.QueryParameters.Del("name")
	opts.Header = header
	_, reqInf, err := TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, opts)
	if err == nil {
		t.Errorf("Expected error about precondition failed, but got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
	}
}

func CreateTestOrigins(t *testing.T) {
	// loop through origins, assign FKs and create
	for _, origin := range testData.Origins {
		resp, _, err := TOSession.CreateOrigin(origin, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create Origins: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func CreateTestOriginDuplicateData(t *testing.T) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test duplicate scenario")
	}
	firstOrigin := testData.Origins[0]
	if firstOrigin.Name == nil {
		t.Fatalf("couldn't get the name of test origin server")
	}
	resp, reqInf, err := TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected, origin name %s already exists, but no error found - Alerts %v", *firstOrigin.Name, resp.Alerts)
	}
}

func NotFoundDeleteTest(t *testing.T) {
	resp, _, err := TOSession.DeleteOrigin(2020, client.RequestOptions{})
	if err == nil {
		t.Fatal("deleting origin with what should be a non-existent id - expected: error, actual: nil error")
	}

	found := false
	for _, alert := range resp.Alerts {
		if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "not found") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("deleted origin with what should be a non-existent id - expected: 'not found' error-level alert, actual: %v - alerts: %+v", err, resp.Alerts)
	}
}

func GetTestOrigins(t *testing.T) {
	resp, _, err := TOSession.GetOrigins(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Origins: %v - alerts: %+v", err, resp.Alerts)
	}

	opts := client.NewRequestOptions()
	for _, origin := range testData.Origins {
		opts.QueryParameters.Set("name", *origin.Name)
		resp, _, err := TOSession.GetOrigins(opts)
		if err != nil {
			t.Errorf("cannot get Origin by name: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func UpdateTestOrigins(t *testing.T) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test updating Origins")
	}
	firstOrigin := testData.Origins[0]
	if firstOrigin.Name == nil {
		t.Fatal("Found an Origin in the testing data with null or undefined name")
	}
	foName := *firstOrigin.Name

	// Retrieve the origin by name so we can get the id for the Update
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", foName)
	resp, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin '%s': %v - alerts: %+v", foName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Origin to exist with name '%s', found: %d", foName, len(resp.Response))
	}

	remoteOrigin := resp.Response[0]
	if remoteOrigin.ID == nil {
		t.Fatalf("Traffic Ops responded with a representation of Origin '%s' that had null or undefined ID", foName)
	}
	updatedPort := 4321
	updatedFQDN := "updated.example.com"
	updatedIpAddress := "5.6.7.8"
	updatedIpv6Address := "dead:beef:cafe::455"
	updatedIsPrimary := false
	updatedProfile := "EDGEInCDN2"
	updateDeliveryService := "ds3"
	updateCachegroup := "multiOriginCachegroup"
	updateCoordinate := "coordinate2"
	updateProtocol := "https"
	updateTenant := "tenant2"

	// update Cachegroup/Coordinate/Name/Delivery Service/Port/FQDN/IPAddress/IPV6Address/Profile/IsPrimary/Protocol/Tenant values on origin
	originRequest := tc.Origin{
		Cachegroup:      &updateCachegroup,
		Coordinate:      &updateCoordinate,
		Name:            remoteOrigin.Name,
		DeliveryService: &updateDeliveryService,
		FQDN:            &updatedFQDN,
		IP6Address:      &updatedIpv6Address,
		IPAddress:       &updatedIpAddress,
		IsPrimary:       &updatedIsPrimary,
		Port:            &updatedPort,
		Profile:         &updatedProfile,
		Protocol:        &updateProtocol,
		Tenant:          &updateTenant,
	}

	updResp, reqInf, err := TOSession.UpdateOrigin(*remoteOrigin.ID, originRequest, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Origin '%s' (#%d): %v - %v", foName, *remoteOrigin.ID, err, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 Status Code, but found %d", reqInf.StatusCode)
	}
	// Retrieve the origin to check cachegroup, coordinate, deliveryservice, port, FQDN, IPAddress, IPV6Address, Profile, IsPrimary, Protocol, Tenant values were updated
	opts.QueryParameters.Del("name")
	opts.QueryParameters.Set("id", strconv.Itoa(*remoteOrigin.ID))
	resp, _, err = TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin #%d ('%s'): %v - alerts: %+v", *remoteOrigin.ID, foName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Origin to exist with ID %d, found: %d", *remoteOrigin.ID, len(resp.Response))
	}
	respOrigin := resp.Response[0]

	if respOrigin.Cachegroup == nil {
		t.Errorf("results do not match, actual: null or undefined - expected: %s", updateCachegroup)
	} else if *respOrigin.Cachegroup != updateCachegroup {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.Cachegroup, updateCachegroup)
	}

	if respOrigin.Coordinate == nil {
		t.Errorf("results do not match, actual: null or undefined - expected: %s", updateCoordinate)
	} else if *respOrigin.Coordinate != updateCoordinate {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.Coordinate, updateCoordinate)
	}

	if respOrigin.DeliveryService == nil {
		t.Errorf("results do not match, actual: null or undefined - expected: %s", updateDeliveryService)
	} else if *respOrigin.DeliveryService != updateDeliveryService {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.DeliveryService, updateDeliveryService)
	}

	if respOrigin.Port == nil {
		t.Errorf("results do not match, actual: null or undefined - expected: %d", updatedPort)
	} else if *respOrigin.Port != updatedPort {
		t.Errorf("results do not match actual: %d, expected: %d", *respOrigin.Port, updatedPort)
	}
	if respOrigin.FQDN == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updatedFQDN)
	} else if *respOrigin.FQDN != updatedFQDN {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.FQDN, updatedFQDN)
	}
	if respOrigin.IPAddress == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updatedIpAddress)
	} else if *respOrigin.IPAddress != updatedIpAddress {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.IPAddress, updatedIpAddress)
	}

	if respOrigin.IP6Address == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updatedIpv6Address)
	} else if *respOrigin.IP6Address != updatedIpv6Address {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.IP6Address, updatedIpv6Address)
	}

	if respOrigin.IsPrimary == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%t'", updatedIsPrimary)
	} else if *respOrigin.IsPrimary != updatedIsPrimary {
		t.Errorf("results do not match actual: %t, expected: %t", *respOrigin.IsPrimary, updatedIsPrimary)
	}

	if respOrigin.Profile == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updatedProfile)
	} else if *respOrigin.Profile != updatedProfile {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.Profile, updatedProfile)
	}

	if respOrigin.Protocol == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updateProtocol)
	} else if *respOrigin.Protocol != updateProtocol {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.Protocol, updateProtocol)
	}

	if respOrigin.Tenant == nil {
		t.Errorf("results do not match, actual: null or undefined, expected: '%s'", updateTenant)
	} else if *respOrigin.Tenant != updateTenant {
		t.Errorf("results do not match actual: %s, expected: %s", *respOrigin.Tenant, updateTenant)
	}
}

func OriginTenancyTest(t *testing.T) {
	origins, _, err := TOSession.GetOrigins(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get Origins: %v - alerts: %+v", err, origins.Alerts)
	}
	if len(origins.Response) < 1 {
		t.Fatal("Need at least one Origin to exist in Traffic Ops to test Tenancy for Origins")
	}
	// This ID check specifically needs to be a fatal condition, despite also being an error below,
	// because we explicitly dereference the ID of the 0th Origin in this slice later on.
	if origins.Response[0].ID == nil || origins.Response[0].Name == nil {
		t.Fatal("Traffic Ops returned a representation for an Origin with null or undefined ID and/or Name")
	}

	var tenant3Origin tc.Origin
	foundTenant3Origin := false
	for _, o := range origins.Response {
		if o.FQDN == nil || o.ID == nil {
			t.Error("Traffic Ops responded with a representation of an Origin with null or undefined FQDN and/or ID")
			continue
		}
		if *o.FQDN == "origin.ds3.example.net" {
			tenant3Origin = o
			foundTenant3Origin = true
		}
	}
	if !foundTenant3Origin {
		t.Error("expected to find origin with tenant 'tenant3' and fqdn 'origin.ds3.example.net'")
	}

	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	tenant4TOClient, _, err := client.LoginWithAgent(TOSession.URL, "tenant4user", "pa$$word", true, "to-api-v3-client-tests/tenant4user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to log in with tenant4user: %v", err)
	}

	originsReadableByTenant4, _, err := tenant4TOClient.GetOrigins(client.RequestOptions{})
	if err != nil {
		t.Errorf("tenant4user cannot get Origins: %v - alerts: %+v", err, originsReadableByTenant4.Alerts)
	}

	// assert that tenant4user cannot read origins outside of its tenant
	for _, origin := range originsReadableByTenant4.Response {
		if origin.FQDN == nil {
			t.Error("Traffic Ops returned a representation of an Origin with null or undefined FQDN")
		} else if *origin.FQDN == "origin.ds3.example.net" {
			t.Error("expected tenant4 to be unable to read origins from tenant 3")
		}
	}

	// assert that tenant4user cannot update tenant3user's origin
	if _, _, err = tenant4TOClient.UpdateOrigin(*tenant3Origin.ID, tenant3Origin, client.RequestOptions{}); err == nil {
		t.Error("expected tenant4user to be unable to update tenant3's origin")
	}

	// assert that tenant4user cannot delete an origin outside of its tenant
	if _, _, err = tenant4TOClient.DeleteOrigin(*origins.Response[0].ID, client.RequestOptions{}); err == nil {
		t.Errorf("expected tenant4user to be unable to delete an origin outside of its tenant (origin %s)", *origins.Response[0].Name)
	}

	// assert that tenant4user cannot create origins outside of its tenant
	tenant3Origin.FQDN = util.StrPtr("origin.tenancy.test.example.com")
	if _, _, err = tenant4TOClient.CreateOrigin(tenant3Origin, client.RequestOptions{}); err == nil {
		t.Error("expected tenant4user to be unable to create an origin outside of its tenant")
	}
}

func alertsHaveError(alerts []tc.Alert, err string) bool {
	for _, alert := range alerts {
		if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, err) {
			return true
		}
	}
	return false
}

func GetTestPaginationSupportOrigins(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("orderby", "id")
	resp, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Fatalf("cannot get Origins: %v - alerts: %+v", err, resp.Alerts)
	}
	origins := resp.Response
	if len(origins) < 3 {
		t.Fatalf("Need at least 3 Origins in Traffic Ops to test pagination, found: %d", len(resp.Response))
	}

	opts.QueryParameters.Set("limit", "1")
	originsWithLimit, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Fatalf("cannot get Origins with limit: %v - alerts: %+v", err, originsWithLimit.Alerts)
	}
	if !reflect.DeepEqual(origins[:1], originsWithLimit.Response) {
		t.Error("expected GET origins with limit = 1 to return first result")
	}

	opts.QueryParameters.Set("offset", "1")
	originsWithOffset, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Fatalf("cannot get Origins with offset: %v - alerts: %+v", err, originsWithOffset.Alerts)
	}
	if !reflect.DeepEqual(origins[1:2], originsWithOffset.Response) {
		t.Error("expected GET origins with limit = 1, offset = 1 to return second result")
	}

	opts.QueryParameters.Del("offset")
	opts.QueryParameters.Set("page", "2")
	originsWithPage, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Fatalf("cannot get Origins with page: %v - alerts: %+v", err, originsWithPage.Alerts)
	}
	if !reflect.DeepEqual(origins[1:2], originsWithPage.Response) {
		t.Error("expected GET origins with limit = 1, page = 2 to return second result")
	}

	opts.QueryParameters.Del("page")
	opts.QueryParameters.Del("orderby")
	opts.QueryParameters.Set("limit", "-2")
	resp, _, err = TOSession.GetOrigins(opts)
	if err == nil {
		t.Error("expected GET origins to return an error when limit is not bigger than -1")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be bigger than -1") {
		t.Errorf("expected GET origins to return an error for limit is not bigger than -1, actual error: " + err.Error())
	}

	opts.QueryParameters.Set("limit", "1")
	opts.QueryParameters.Set("offset", "0")
	resp, _, err = TOSession.GetOrigins(opts)
	if err == nil {
		t.Error("expected GET origins to return an error when offset is not a positive integer")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be a positive integer") {
		t.Errorf("expected GET origins to return an error for offset is not a positive integer, actual error: " + err.Error())
	}

	opts.QueryParameters.Del("offset")
	opts.QueryParameters.Set("page", "0")
	resp, _, err = TOSession.GetOrigins(opts)
	if err == nil {
		t.Error("expected GET origins to return an error when page is not a positive integer")
	} else if !alertsHaveError(resp.Alerts.Alerts, "must be a positive integer") {
		t.Errorf("expected GET origins to return an error for page is not a positive integer, actual error: " + err.Error())
	}
}

func DeleteTestOrigins(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, origin := range testData.Origins {
		if origin.Name == nil {
			t.Error("Found an Origin in the testing data with null or undefined name")
			continue
		}

		opts.QueryParameters.Set("name", *origin.Name)
		resp, _, err := TOSession.GetOrigins(opts)
		if err != nil {
			t.Errorf("cannot get Origin '%s': %v - alerts: %+v", *origin.Name, err, resp.Alerts)
		}
		if len(resp.Response) > 0 {
			respOrigin := resp.Response[0]
			if respOrigin.ID == nil {
				t.Error("Traffic Ops returned a representation for an Origin that has null or undefined ID")
				continue
			}
			delResp, _, err := TOSession.DeleteOrigin(*respOrigin.ID, client.RequestOptions{})
			if err != nil {
				t.Errorf("cannot DELETE Origin by ID: %v - %v", err, delResp)
			}

			// Retrieve the Origin to see if it got deleted
			org, _, err := TOSession.GetOrigins(opts)
			if err != nil {
				t.Errorf("error fetching Origin '%s' after deletion: %v - alerts: %+v", *origin.Name, err, org.Alerts)
			}
			if len(org.Response) > 0 {
				t.Errorf("expected Origin '%s' to be deleted, but it was found in Traffic Ops", *origin.Name)
			}
		}
	}
}

func GetTestOriginsByParams(t *testing.T) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test Get Origins by params")
	}
	origins := testData.Origins[0]
	if origins.Name == nil || len(*origins.Name) == 0 {
		t.Fatal("Found nil value in Origin name")
	}
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", *origins.Name)
	originByName, _, _ := TOSession.GetOrigins(opts)
	if len(originByName.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Delivery Service, but found %d", len(originByName.Response))
	}
	if originByName.Response[0].DeliveryServiceID == nil {
		t.Fatal("Found nil value in delivery service")
	}
	if originByName.Response[0].CachegroupID == nil {
		t.Fatal("Found nil value in Cachegroup")
	}
	if originByName.Response[0].CoordinateID == nil {
		t.Fatal("Found nil value in Coordinate")
	}
	if originByName.Response[0].ProfileID == nil {
		t.Fatal("Found nil value in Profile")
	}
	if originByName.Response[0].IsPrimary == nil {
		t.Fatal("Found nil value in IsPrimary field")
	}

	//Get Origins by DSID
	dsId := *originByName.Response[0].DeliveryServiceID
	opts.QueryParameters.Del("name")
	opts.QueryParameters.Set("deliveryservice", strconv.Itoa(dsId))
	originByDs, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by DeliveryService ID: %v - alerts: %+v", err, originByDs.Alerts)
	}
	if len(originByDs.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Delivery Service, but found %d", len(originByDs.Response))
	}

	//Get Origins by Cachegroup
	cachegroupID := *originByName.Response[0].CachegroupID
	opts.QueryParameters.Del("deliveryservice")
	opts.QueryParameters.Set("cachegroup", strconv.Itoa(cachegroupID))
	originByCg, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by Cachegroup ID: %v - alerts: %+v", err, originByCg.Alerts)
	}
	if len(originByCg.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Cachegroups, but found %d", len(originByCg.Response))
	}

	//Get Origins by Coordinate
	CoordinateID := *originByName.Response[0].CoordinateID
	opts.QueryParameters.Del("cachegroup")
	opts.QueryParameters.Set("coordinate", strconv.Itoa(CoordinateID))
	originByCoordinate, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by Coordinate ID: %v - alerts: %+v", err, originByCoordinate.Alerts)
	}
	if len(originByCoordinate.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Coordinate, but found %d", len(originByCoordinate.Response))
	}

	//Get Origins by Profile
	profileId := *originByName.Response[0].ProfileID
	opts.QueryParameters.Del("coordinate")
	opts.QueryParameters.Set("profileId", strconv.Itoa(profileId))
	originByProfile, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by Profile ID: %v - alerts: %+v", err, originByProfile.Alerts)
	}
	if len(originByProfile.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Profile, but found %d", len(originByProfile.Response))
	}

	//Get Origins by Primary
	isPrimary := *originByName.Response[0].IsPrimary
	opts.QueryParameters.Del("profileId")
	opts.QueryParameters.Set("isPrimary", strconv.FormatBool(isPrimary))
	originByPrimary, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by Primary ID: %v - alerts: %+v", err, originByPrimary.Alerts)
	}
	if len(originByPrimary.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Primary, but found %d", len(originByPrimary.Response))
	}

	//Get Origins by Tenant
	tenant := *originByName.Response[0].TenantID
	opts.QueryParameters.Del("isPrimary")
	opts.QueryParameters.Set("tenant", strconv.Itoa(tenant))
	originByTenant, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin by Tenant ID: %v - alerts: %+v", err, originByTenant.Alerts)
	}
	if len(originByTenant.Response) < 1 {
		t.Fatalf("Expected atleast one Origin for GET Origin by Tenant, but found %d", len(originByTenant.Response))
	}
}

func GetTestOriginsByInvalidParams(t *testing.T) {

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("deliveryservice", "12345")
	originByDs, _, _ := TOSession.GetOrigins(opts)
	if len(originByDs.Response) > 0 {
		t.Fatalf("Expected empty response for GET Origin by invalid Delivery Service, but found %d", len(originByDs.Response))
	}

	//Get Origins by Cachegroup
	opts.QueryParameters.Del("deliveryservice")
	opts.QueryParameters.Set("cachegroup", "12345")
	originByCg, _, _ := TOSession.GetOrigins(opts)
	if len(originByCg.Response) > 0 {
		t.Fatalf("Expected empty response for GET Origin by invalid Cachegroups, but found %d", len(originByCg.Response))
	}

	//Get Origins by Coordinate
	opts.QueryParameters.Del("cachegroup")
	opts.QueryParameters.Set("coordinate", "12345")
	originByCoordinate, _, _ := TOSession.GetOrigins(opts)
	if len(originByCoordinate.Response) > 0 {
		t.Fatalf("Expected empty response for GET Origin by invalid Coordinate, but found %d", len(originByCoordinate.Response))
	}

	//Get Origins by Profile
	opts.QueryParameters.Del("coordinate")
	opts.QueryParameters.Set("profileId", "12345")
	originByProfile, _, _ := TOSession.GetOrigins(opts)
	if len(originByProfile.Response) > 0 {
		t.Fatalf("Expected empty response for GET Origin by invalid Profile, but found %d", len(originByProfile.Response))
	}

	//Get Origins by Tenant
	opts.QueryParameters.Del("profileId")
	opts.QueryParameters.Set("tenant", "12345")
	originByTenant, _, _ := TOSession.GetOrigins(opts)
	if len(originByTenant.Response) > 0 {
		t.Fatalf("Expected empty response for GET Origin by invalid Tenant, but found %d", len(originByTenant.Response))
	}
}

func CreateTestOriginInvalidData(t *testing.T) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test duplicate Origins")
	}
	firstOrigin := testData.Origins[0]
	if firstOrigin.Name == nil {
		t.Fatalf("couldn't get the name of test origin server")
	}
	oldCachegroupId := firstOrigin.CachegroupID
	oldProfileId := firstOrigin.ProfileID
	oldTenantId := firstOrigin.TenantID
	oldProtocol := firstOrigin.Protocol
	oldCoordinateId := firstOrigin.CoordinateID
	oldIpv4 := firstOrigin.IPAddress

	//invalid cg id
	cachegroupID := new(int)
	*cachegroupID = 12345
	name := new(string)
	*name = "invalid"
	firstOrigin.CachegroupID = cachegroupID
	firstOrigin.Name = name
	resp, reqInf, err := TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected cachegroup not found - Alerts %v", resp.Alerts)
	}

	//invalid profile id
	firstOrigin.CachegroupID = oldCachegroupId
	profileId := new(int)
	*profileId = 12345
	firstOrigin.ProfileID = profileId
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected profile not found - Alerts %v", resp.Alerts)
	}

	//invalid tenant id
	firstOrigin.ProfileID = oldProfileId
	tenantId := new(int)
	*tenantId = 12345
	firstOrigin.TenantID = tenantId
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected user not authorized for requested tenant - Alerts %v", resp.Alerts)
	}

	//invalid protocol id
	firstOrigin.TenantID = oldTenantId
	protocol := new(string)
	*protocol = "abcd"
	firstOrigin.Protocol = protocol
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected 'protocol' must be http or https - Alerts %v", resp.Alerts)
	}

	//invalid coordinate id
	firstOrigin.Protocol = oldProtocol
	coordinateId := new(int)
	*coordinateId = 12345
	firstOrigin.CoordinateID = coordinateId
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected coordinate not found - Alerts %v", resp.Alerts)
	}

	//invalid IPV4
	firstOrigin.CoordinateID = oldCoordinateId
	ipv4 := new(string)
	*ipv4 = "1.11"
	firstOrigin.IPAddress = ipv4
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected 'ipAddress' must be a valid IPv4 address - Alerts %v", resp.Alerts)
	}

	//invalid IPV6
	firstOrigin.IPAddress = oldIpv4
	ipv6 := new(string)
	*ipv6 = "1:1:1:1:1"
	firstOrigin.IP6Address = ipv6
	resp, reqInf, err = TOSession.CreateOrigin(firstOrigin, client.RequestOptions{})
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status code, but found %d", reqInf.StatusCode)
	}
	if err == nil {
		t.Errorf("Expected 'ip6Address' must be a valid IPv6 address - Alerts %v", resp.Alerts)
	}
}

func updateTestOriginsWithInvalidData(t *testing.T) {
	if len(testData.Origins) < 1 {
		t.Fatal("Need at least one Origin to test updating Origins")
	}
	firstOrigin := testData.Origins[0]
	if firstOrigin.Name == nil {
		t.Fatal("Found an Origin in the testing data with null or undefined name")
	}
	foName := *firstOrigin.Name
	// Retrieve the origin by name so we can get the id for the Update
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", foName)
	resp, _, err := TOSession.GetOrigins(opts)
	if err != nil {
		t.Errorf("cannot get Origin '%s': %v - alerts: %+v", foName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Origin to exist with name '%s', found: %d", foName, len(resp.Response))
	}
	remoteOrigin := resp.Response[0]
	if remoteOrigin.ID == nil {
		t.Fatalf("Traffic Ops responded with a representation of Origin '%s' that had null or undefined ID", foName)
	}

	oldCachegroupId := remoteOrigin.CachegroupID
	oldProfileId := remoteOrigin.ProfileID
	oldTenantId := remoteOrigin.TenantID
	oldProtocol := remoteOrigin.Protocol
	oldCoordinateId := remoteOrigin.CoordinateID
	oldIpv4 := remoteOrigin.IPAddress
	oldIpv6 := remoteOrigin.IP6Address
	oldPort := remoteOrigin.Port
	oldDeliveryServiceId := remoteOrigin.DeliveryServiceID

	//update invalid port
	updatedPort := 123456
	remoteOrigin.Port = &updatedPort
	updResp, reqInf, err := TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Port must be a valid integer between 1 and 65535. Port - %d, Alerts %v", updatedPort, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status Code, but found %d", reqInf.StatusCode)
	}

	//update cachegroup id
	remoteOrigin.Port = oldPort
	updatedCachegroupId := 123456
	remoteOrigin.CachegroupID = &updatedCachegroupId
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Cachegroup not found. Cachegroup - %d, Alerts %v", updatedCachegroupId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status Code, but found %d", reqInf.StatusCode)
	}

	//update coordinate id
	remoteOrigin.CachegroupID = oldCachegroupId
	updatedCoordinateId := 123456
	remoteOrigin.CoordinateID = &updatedCoordinateId
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Coordinate not found, Coordinate - %d, Alerts %v", updatedCoordinateId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid ds id
	remoteOrigin.CoordinateID = oldCoordinateId
	updatedDeliveryServiceId := 123456
	remoteOrigin.DeliveryServiceID = &updatedDeliveryServiceId
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - checking tenancy: requested delivery service does not exist, Delivery Service - %d, Alerts %v", updatedDeliveryServiceId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid protocol
	remoteOrigin.DeliveryServiceID = oldDeliveryServiceId
	updatedProtocol := "httpsss"
	remoteOrigin.Protocol = &updatedProtocol
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Protocol must be http or https. Protocol - %s, Alerts %v", updatedProtocol, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid ipv6
	remoteOrigin.Protocol = oldProtocol
	updatedIpv6 := "1.1"
	remoteOrigin.IP6Address = &updatedIpv6
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Ip6Address must be a valid IPv6 address. IPV6 Address - %s, Alerts %v", updatedIpv6, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid ipv4
	remoteOrigin.IP6Address = oldIpv6
	updatedIpv4 := "1.1"
	remoteOrigin.IPAddress = &updatedIpv4
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - IpAddress must be a valid IPv4 address. IPV4 - %s, Alerts %v", updatedIpv4, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid tenant
	remoteOrigin.IPAddress = oldIpv4
	updatedTenantId := 11111
	remoteOrigin.TenantID = &updatedTenantId
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - user not authorized for requested tenant. Tenant - %d, Alerts %v", updatedTenantId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid profile
	remoteOrigin.TenantID = oldTenantId
	updatedProfileId := 12345
	remoteOrigin.ProfileID = &updatedProfileId
	updResp, reqInf, err = TOSession.UpdateOrigin(*remoteOrigin.ID, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - profile not found %d - Alerts %v", updatedProfileId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status Code, but found %d", reqInf.StatusCode)
	}

	//update invalid id
	remoteOrigin.ProfileID = oldProfileId
	invalidId := new(int)
	*invalidId = 12345
	updResp, reqInf, err = TOSession.UpdateOrigin(*invalidId, remoteOrigin, client.RequestOptions{})
	if err == nil {
		t.Errorf("Expected - Origin not found. Origin ID - %d, Alerts %v", *invalidId, updResp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Status Code, but found %d", reqInf.StatusCode)
	}
}
