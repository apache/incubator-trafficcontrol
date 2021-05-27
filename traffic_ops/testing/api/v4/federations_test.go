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
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestFederations(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, DeliveryServices, CDNFederations}, func() {
		PostDeleteTestFederationsDeliveryServices(t)
		GetTestFederations(t)
		GetTestFederationsIMS(t)
		AddFederationResolversForCurrentUserTest(t)
		RemoveFederationResolversForCurrentUserTest(t)
	})
}

func GetTestFederationsIMS(t *testing.T) {
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)
	if len(testData.Federations) == 0 {
		t.Error("no federations test data")
	}

	resp, reqInf, err := TOSession.AllFederations(opts)
	if err != nil {
		t.Fatalf("No error expected, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestFederations(t *testing.T) {
	if len(testData.Federations) == 0 {
		t.Error("no federations test data")
	}

	feds, _, err := TOSession.AllFederations(client.RequestOptions{})
	if err != nil {
		t.Errorf("getting federations: %v - alerts: %+v", err, feds.Alerts)
	}

	if len(feds.Response) != 1 {
		t.Fatalf("federations expected 1, actual: %d", len(feds.Response))
	}
	fed := feds.Response[0]

	if len(fed.Mappings) < 1 {
		t.Fatal("federation mappings expected > 1, actual: 0")
	}

	mapping := fed.Mappings[0]
	if mapping.CName == nil {
		t.Fatal("federation mapping expected cname, actual: nil")
	}
	if mapping.TTL == nil {
		t.Fatal("federation mapping expected ttl, actual: nil")
	}

	matched := false
	for _, testFed := range testData.Federations {
		if testFed.CName == nil {
			t.Error("test federation missing cname!")
			continue
		}
		if testFed.TTL == nil {
			t.Error("test federation missing ttl!")
			continue
		}

		if *mapping.CName != *testFed.CName {
			continue
		}
		matched = true

		if *mapping.TTL != *testFed.TTL {
			t.Errorf("federation mapping ttl expected: %v, actual: %v", *testFed.TTL, *mapping.TTL)
		}
	}
	if !matched {
		t.Errorf("federation mapping expected to match test data, actual: cname %v not in test data", *mapping.CName)
	}
}

func createFederationToDeliveryServiceAssociation() (int, tc.DeliveryServiceV4, tc.DeliveryServiceV4, error) {
	var ds tc.DeliveryServiceV4
	var ds1 tc.DeliveryServiceV4
	dses, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		return -1, ds, ds1, fmt.Errorf("cannot get Delivery Services: %v - alerts: %+v", err, dses.Alerts)
	}
	if len(dses.Response) < 2 {
		return -1, ds, ds1, errors.New("no delivery services, must have at least 2 Delivery Services to test federations deliveryservices")
	}
	ds = dses.Response[0]
	ds1 = dses.Response[1]
	if ds.ID == nil || ds1.ID == nil {
		return -1, ds, ds1, errors.New("Traffic Ops returned at least one representation of a Delivery Service that had a null or undefined ID")
	}

	if len(fedIDs) == 0 {
		return -1, ds, ds1, errors.New("no federations, must have at least 1 federation to test federations deliveryservices")
	}
	fedID := fedIDs[0]

	alerts, _, err := TOSession.CreateFederationDeliveryServices(fedID, []int{*ds.ID, *ds1.ID}, true, client.RequestOptions{})
	if err != nil {
		err = fmt.Errorf("creating federations delivery services: %v - alerts: %+v", err, alerts.Alerts)
	}

	return fedID, ds, ds1, err

}

func PostDeleteTestFederationsDeliveryServices(t *testing.T) {
	fedID, ds, ds1, err := createFederationToDeliveryServiceAssociation()
	if err != nil {
		t.Fatal(err.Error())
	}

	// Test get created Federation Delivery Services
	fedDSes, _, err := TOSession.GetFederationDeliveryServices(fedID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Federation #%d Delivery Services: %v - alerts: %+v", fedID, err, fedDSes.Alerts)
	}
	if len(fedDSes.Response) != 2 {
		t.Fatalf("two Federation DeliveryService expected for Federation %d, %d was returned", fedID, len(fedDSes.Response))
	}

	// Delete one of the Delivery Services from the Federation
	alerts, _, err := TOSession.DeleteFederationDeliveryService(fedID, *ds.ID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot Delete Federation #%d Delivery Service #%d: %v - alerts: %+v", fedID, ds.ID, err, alerts.Alerts)
	}

	// Make sure it is deleted

	// Test get created Federation Delivery Services
	fedDSes, _, err = TOSession.GetFederationDeliveryServices(fedID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Federation Delivery Services: %v - alerts: %+v", err, fedDSes.Alerts)
	}
	if len(fedDSes.Response) != 1 {
		t.Fatalf("one Federation Delivery Service expected for Federation #%d, %d was returned", fedID, len(fedDSes.Response))
	}

	// Attempt to delete the last one which should fail as you cannot remove the last
	_, _, err = TOSession.DeleteFederationDeliveryService(fedID, *ds1.ID, client.RequestOptions{})
	if err == nil {
		t.Fatal("expected to receive error from attempting to delete last Delivery Service from a Federation")
	}
}

func RemoveFederationResolversForCurrentUserTest(t *testing.T) {
	if len(testData.Federations) < 1 {
		t.Fatal("No test Federations, deleting resolvers cannot be tested!")
	}

	alerts, _, err := TOSession.DeleteFederationResolverMappingsForCurrentUser(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error deleting Federation Resolvers for current user: %v - alerts: %+v", err, alerts)
	}
	for _, a := range alerts.Alerts {
		if a.Level == tc.ErrorLevel.String() {
			t.Errorf("Unexpected error-level alert from deleting Federation Resolvers for current user: %s", a.Text)
		}
	}

	// Now try deleting Federation Resolvers when there are none.
	_, _, err = TOSession.DeleteFederationResolverMappingsForCurrentUser(client.RequestOptions{})
	if err != nil {
		t.Logf("Received expected error deleting Federation Resolvers for current user: %v", err)
	} else {
		t.Error("Expected an error deleting zero Federation Resolvers, but didn't get one.")
	}
}

func AddFederationResolversForCurrentUserTest(t *testing.T) {
	fedID, ds, ds1, err := createFederationToDeliveryServiceAssociation()
	if err != nil {
		t.Fatal(err.Error())
	}

	// need to assign myself the federation to set its mappings
	me, _, err := TOSession.GetUserCurrent(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Couldn't figure out who I am: %v - alerts: %+v", err, me.Alerts)
	}
	if me.Response.ID == nil {
		t.Fatal("Current user has no ID, cannot continue.")
	}

	alerts, _, err := TOSession.CreateFederationUsers(fedID, []int{*me.Response.ID}, false, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Failed to assign Federation to current user: %v - alerts: %+v", err, alerts.Alerts)
	}

	mappings := tc.DeliveryServiceFederationResolverMappingRequest{
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: ds.XMLID,
			Mappings: tc.ResolverMapping{
				Resolve4: []string{"0.0.0.0"},
				Resolve6: []string{"::1"},
			},
		},
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: ds1.XMLID,
			Mappings: tc.ResolverMapping{
				Resolve4: []string{"1.2.3.4/28"},
				Resolve6: []string{"1234::/110"},
			},
		},
	}

	alerts, _, err = TOSession.AddFederationResolverMappingsForCurrentUser(mappings, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error adding Federation Resolver mappings for the current user: %v - alerts: %+v", err, alerts.Alerts)
	}
	for _, a := range alerts.Alerts {
		if a.Level == tc.ErrorLevel.String() {
			t.Errorf("Unexpected error-level alert from adding Federation Resolver mappings for the current user: %s", a.Text)
		}
	}

	mappings = tc.DeliveryServiceFederationResolverMappingRequest{
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: "aoeuhtns",
			Mappings: tc.ResolverMapping{
				Resolve4: []string{},
				Resolve6: []string{"dead::beef", "f1d0::f00d/82"},
			},
		},
	}

	alerts, _, err = TOSession.AddFederationResolverMappingsForCurrentUser(mappings, client.RequestOptions{})
	if err == nil {
		t.Fatal("Expected error adding Federation Resolver mappings for the current user, but didn't get one")
	}
	for _, a := range alerts.Alerts {
		if a.Level == tc.SuccessLevel.String() {
			t.Errorf("Unexpected success from adding Federation Resolver mappings for the current user: %s", a.Text)
		}
	}
}
