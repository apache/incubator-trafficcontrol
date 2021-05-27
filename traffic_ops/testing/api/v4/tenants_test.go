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
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestTenants(t *testing.T) {
	WithObjs(t, []TCObj{Tenants}, func() {
		SortTestTenants(t)
		GetTestTenants(t)
		UpdateTestTenants(t)
		UpdateTestRootTenant(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfUnmodifiedSince, time)
		UpdateTestTenantsWithHeaders(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestTenantsWithHeaders(t, header)
	})
}

// This test will break if the testing Tenancy tree is modified in specific ways
func UpdateTestTenantsWithHeaders(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header

	// Retrieve the Tenant by name so we can get the id for the Update
	name := "tenant2"
	opts.QueryParameters.Set("name", name)
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'tenant2', found: %d", len(resp.Response))
	}
	modTenant := resp.Response[0]

	parentName := "tenant1"
	opts.QueryParameters.Set("name", parentName)
	resp, _, err = TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", parentName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'tenant1', found: %d", len(resp.Response))
	}
	newParent := resp.Response[0]

	modTenant.ParentID = newParent.ID
	opts.QueryParameters.Del("name")
	_, reqInf, err := TOSession.UpdateTenant(modTenant.ID, modTenant, opts)
	if err == nil {
		t.Fatalf("expected a precondition failed error, got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("expected a status 412 Precondition Failed, but got %d", reqInf.StatusCode)
	}
}

func TestTenantsActive(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, CacheGroups, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, Servers, Topologies, DeliveryServices, Users}, func() {
		UpdateTestTenantsActive(t)
	})
}

func CreateTestTenants(t *testing.T) {
	for _, ten := range testData.Tenants {
		resp, _, err := TOSession.CreateTenant(ten, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create Tenant '%s': %v - alerts: %+v", ten.Name, err, resp.Alerts)
		} else if resp.Response.Name != ten.Name {
			t.Errorf("expected tenant '%s'; got '%s'", ten.Name, resp.Response.Name)
		}
	}
}

func GetTestTenants(t *testing.T) {
	resp, _, err := TOSession.GetTenants(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get all Tenants: %v - alerts: %+v", err, resp.Alerts)
		return
	}
	foundTenants := make(map[string]tc.Tenant, len(resp.Response))
	for _, ten := range resp.Response {
		foundTenants[ten.Name] = ten
	}

	// expect root and badTenant (defined in todb.go) + all defined in testData.Tenants
	if len(resp.Response) != 2+len(testData.Tenants) {
		t.Errorf("expected %d tenants,  got %d", 2+len(testData.Tenants), len(resp.Response))
	}

	for _, ten := range testData.Tenants {
		if ft, ok := foundTenants[ten.Name]; ok {
			if ft.ParentName != ten.ParentName {
				t.Errorf("Tenant '%s': expected parent '%s', got '%s'", ten.Name, ten.ParentName, ft.ParentName)
			}
		} else {
			t.Errorf("expected Tenant '%s': not found", ten.Name)
		}
	}
}

func SortTestTenants(t *testing.T) {
	resp, _, err := TOSession.GetTenants(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	sortedList := make([]string, 0, len(resp.Response))
	for _, tenant := range resp.Response {
		sortedList = append(sortedList, tenant.Name)
	}

	if !sort.StringsAreSorted(sortedList) {
		t.Errorf("list is not sorted by their names: %v", sortedList)
	}
}

func UpdateTestTenants(t *testing.T) {

	// Retrieve the Tenant by name so we can get the id for the Update
	name := "tenant2"
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", name)
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'tenant2', found: %d", len(resp.Response))
	}
	modTenant := resp.Response[0]

	parentName := "tenant1"
	opts.QueryParameters.Set("name", parentName)
	resp, _, err = TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", parentName, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'tenant1', found: %d", len(resp.Response))
	}
	newParent := resp.Response[0]
	modTenant.ParentID = newParent.ID

	response, _, err := TOSession.UpdateTenant(modTenant.ID, modTenant, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update Tenant: %v - alerts: %+v", err, response.Alerts)
	}

	// Retrieve the Tenant to check Tenant parent name got updated
	opts.QueryParameters.Del("name")
	opts.QueryParameters.Set("id", strconv.Itoa(modTenant.ID))
	resp, _, err = TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with ID %d, found: %d", modTenant.ID, len(resp.Response))
	}
	respTenant := resp.Response[0]
	if respTenant.ParentName != parentName {
		t.Errorf("results do not match actual: %s, expected: %s", respTenant.ParentName, parentName)
	}

}

func UpdateTestRootTenant(t *testing.T) {
	// Retrieve the Tenant by name so we can get the id for the Update
	name := "root"
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", name)
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'root', found: %d", len(resp.Response))
	}
	modTenant := resp.Response[0]

	modTenant.Active = false
	modTenant.ParentID = modTenant.ID
	_, reqInf, err := TOSession.UpdateTenant(modTenant.ID, modTenant, client.RequestOptions{})
	if err == nil {
		t.Fatalf("expected an error when trying to update the 'root' tenant, but got nothing")
	}
	if reqInf.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a status 400 Bad Request, but got %d", reqInf.StatusCode)
	}
}

func DeleteTestTenants(t *testing.T) {
	t1 := "tenant1"
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", t1)
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", t1, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expeected exactly one Tenant to exist with the name '%s', found: %d", t1, len(resp.Response))
	}
	tenant1 := resp.Response[0]

	expectedChildDeleteErrMsg := fmt.Sprintf("Tenant '%d' has child tenants. Please update these child tenants and retry.", tenant1.ID)
	if response, _, err := TOSession.DeleteTenant(tenant1.ID, client.RequestOptions{}); err == nil {
		t.Fatalf("%s has child tenants -- should not be able to delete", t1)
	} else if !alertsHaveError(response.Alerts, expectedChildDeleteErrMsg) {
		t.Errorf("expected error: %s; got: %v - alerts: %+v", expectedChildDeleteErrMsg, err, response.Alerts)
	}

	deletedTenants := map[string]struct{}{}
	for {
		initLenDeleted := len(deletedTenants)
		for _, tn := range testData.Tenants {
			if _, ok := deletedTenants[tn.Name]; ok {
				continue
			}

			hasParent := false
			for _, otherTenant := range testData.Tenants {
				if _, ok := deletedTenants[otherTenant.Name]; ok {
					continue
				}
				if otherTenant.ParentName == tn.Name {
					hasParent = true
					break
				}
			}
			if hasParent {
				continue
			}

			opts.QueryParameters.Set("name", tn.Name)
			resp, _, err := TOSession.GetTenants(opts)
			if err != nil {
				t.Fatalf("getting Tenants filtered by name '%s': %v - alerts: %+v", tn.Name, err, resp.Alerts)
			}
			if len(resp.Response) != 1 {
				t.Fatalf("Expected exactly one Tenant to exist with the name '%s', found: %d", tn.Name, len(resp.Response))
			}
			toTenant := resp.Response[0]

			if alerts, _, err := TOSession.DeleteTenant(toTenant.ID, client.RequestOptions{}); err != nil {
				t.Fatalf("deleting Tenant '%s': %v - alerts: %+v", toTenant.Name, err, alerts.Alerts)
			}
			deletedTenants[tn.Name] = struct{}{}

		}
		if len(deletedTenants) == len(testData.Tenants) {
			break
		}
		if len(deletedTenants) == initLenDeleted {
			t.Fatal("could not delete tenants: not tenant without an existing child found (cycle?)")
		}
	}
}

func UpdateTestTenantsActive(t *testing.T) {
	originalTenants, _, err := TOSession.GetTenants(client.RequestOptions{})
	if err != nil {
		t.Fatalf("getting Tenants error expected: nil, actual: %v - alerts: %+v", err, originalTenants.Alerts)
	}

	setTenantActive(t, "tenant1", true)
	setTenantActive(t, "tenant2", true)
	setTenantActive(t, "tenant3", false)

	// ds3 has tenant3. Even though tenant3 is inactive, we should still be able to get it, because our user is tenant1, which is active.
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", "ds3")
	resp, _, err := TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("failed to get delivery service, when the DS's tenant was inactive (even though our user's tenant was active): %v - alerts: %+v", err, resp.Alerts)
	} else if len(resp.Response) != 1 {
		t.Error("admin user getting delivery service ds3 with tenant3, expected: ds, actual: empty")
	}

	setTenantActive(t, "tenant1", true)
	setTenantActive(t, "tenant2", false)
	setTenantActive(t, "tenant3", true)

	// ds3 has tenant3. Even though tenant3's parent, tenant2, is inactive, we should still be able to get it, because our user is tenant1, which is active.
	resp, _, err = TOSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("failed to get delivery service, when a parent tenant was inactive (even though our user's tenant was active): %v - alerts: %+v", err, resp.Alerts)
	}

	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	tenant3Session, _, err := client.LoginWithAgent(TOSession.URL, "tenant3user", "pa$$word", true, "to-api-v1-client-tests/tenant3user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to get log in with tenant3user: " + err.Error())
	}

	tenant4Session, _, err := client.LoginWithAgent(TOSession.URL, "tenant4user", "pa$$word", true, "to-api-v1-client-tests/tenant4user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to get log in with tenant4user: " + err.Error())
	}

	// tenant3user with tenant3 has no access to ds3 with tenant3 when parent tenant2 is inactive
	resp, _, err = tenant3Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Services filtered by XMLID 'ds3': %v - alerts: %+v", err, resp.Alerts)
	}
	for _, ds := range resp.Response {
		t.Errorf("tenant3user got delivery service %s with tenant3 but tenant3 parent tenant2 is inactive, expected: no ds", ds.XMLID)
	}

	setTenantActive(t, "tenant1", true)
	setTenantActive(t, "tenant2", true)
	setTenantActive(t, "tenant3", false)

	// tenant3user with tenant3 has no access to ds3 with tenant3 when tenant3 is inactive
	resp, _, err = tenant3Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Services filtered by XMLID 'ds3': %v - alerts: %+v", err, resp.Alerts)
	}
	for _, ds := range resp.Response {
		t.Errorf("tenant3user got delivery service %s with tenant3 but tenant3 is inactive, expected: no ds", ds.XMLID)
	}

	setTenantActive(t, "tenant1", true)
	setTenantActive(t, "tenant2", true)
	setTenantActive(t, "tenant3", true)

	// tenant3user with tenant3 has access to ds3 with tenant3
	resp, _, err = tenant3Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("tenant3user getting delivery service ds3 error expected: nil, actual: %+v", err)
	} else if len(resp.Response) == 0 {
		t.Error("tenant3user getting delivery service ds3 with tenant3, expected: ds, actual: empty")
	}

	// 1. ds2 has tenant2.
	// 2. tenant3user has tenant3.
	// 3. tenant2 is not a child of tenant3 (tenant3 is a child of tenant2)
	// 4. Therefore, tenant3user should not have access to ds2
	opts.QueryParameters.Set("xmlId", "ds2")
	resp, _, err = tenant3Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Services filtered by XMLID 'ds2': %v - alerts: %+v", err, resp.Alerts)
	}
	for _, ds := range resp.Response {
		t.Errorf("tenant3user got delivery service %s with tenant2, expected: no ds", ds.XMLID)
	}

	// 1. ds1 has tenant1.
	// 2. tenant4user has tenant4.
	// 3. tenant1 is not a child of tenant4 (tenant4 is unrelated to tenant1)
	// 4. Therefore, tenant4user should not have access to ds1
	opts.QueryParameters.Set("xmlId", "ds1")
	resp, _, err = tenant4Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Services filtered by XMLID 'ds1': %v - alerts: %+v", err, resp.Alerts)
	}
	for _, ds := range resp.Response {
		t.Errorf("tenant4user got delivery service %s with tenant1, expected: no ds", ds.XMLID)
	}

	setTenantActive(t, "tenant3", false)
	opts.QueryParameters.Set("xmlId", "ds3")
	resp, _, err = tenant3Session.GetDeliveryServices(opts)
	if err != nil {
		t.Errorf("Unexpected error fetching Delivery Services filtered by XMLID 'ds3': %v - alerts: %+v", err, resp.Alerts)
	}
	for _, ds := range resp.Response {
		t.Errorf("tenant3user was inactive, but got delivery service %s with tenant3, expected: no ds", ds.XMLID)
	}

	for _, tn := range originalTenants.Response {
		if tn.Name == "root" {
			continue
		}
		if resp, _, err := TOSession.UpdateTenant(tn.ID, tn, client.RequestOptions{}); err != nil {
			t.Fatalf("restoring original tenants: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func setTenantActive(t *testing.T, name string, active bool) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", name)
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Fatalf("cannot get Tenants filtered by name '%s': %v - alerts: %+v", name, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name '%s', found: %d", name, len(resp.Response))
	}
	tn := resp.Response[0]

	tn.Active = active
	response, _, err := TOSession.UpdateTenant(tn.ID, tn, client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot update Tenant: %v - alerts: %+v", err, response.Alerts)
	}
}
