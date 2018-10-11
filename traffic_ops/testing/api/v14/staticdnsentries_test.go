package v14

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
	"testing"

	"github.com/apache/trafficcontrol/lib/go-log"
	tc "github.com/apache/trafficcontrol/lib/go-tc"
	"reflect"
)

func TestStaticDNSEntries(t *testing.T) {

	CreateTestCDNs(t)
	CreateTestTypes(t)
	CreateTestProfiles(t)
	CreateTestStatuses(t)
	CreateTestDivisions(t)
	CreateTestRegions(t)
	CreateTestPhysLocations(t)
	CreateTestCacheGroups(t)
	CreateTestServers(t)
	CreateTestDeliveryServices(t)
	CreateTestStaticDNSEntries(t)
	GetTestStaticDNSEntries(t)
	UpdateTestStaticDNSEntries(t)
	UpdateTestStaticDNSEntriesInvalidAddress(t)
	DeleteTestStaticDNSEntries(t)
	DeleteTestDeliveryServices(t)
	DeleteTestServers(t)
	DeleteTestCacheGroups(t)
	DeleteTestPhysLocations(t)
	DeleteTestRegions(t)
	DeleteTestDivisions(t)
	DeleteTestStatuses(t)
	DeleteTestProfiles(t)
	DeleteTestTypes(t)
	DeleteTestCDNs(t)

}

func CreateTestStaticDNSEntries(t *testing.T) {
	for _, staticDNSEntry := range testData.StaticDNSEntries {
		resp, _, err := TOSession.CreateStaticDNSEntry(staticDNSEntry)
		log.Debugln("Response: ", resp)
		if err != nil {
			t.Errorf("could not CREATE staticDNSEntry: %v\n", err)
		}
	}

}

func UpdateTestStaticDNSEntries(t *testing.T) {

	firstStaticDNSEntry := testData.StaticDNSEntries[0]
	// Retrieve the StaticDNSEntries by name so we can get the id for the Update
	resp, _, err := TOSession.GetStaticDNSEntriesByHost(firstStaticDNSEntry.Host)
	if err != nil {
		t.Errorf("cannot GET StaticDNSEntries by name: '%s', %v\n", firstStaticDNSEntry.Host, err)
	}
	remoteStaticDNSEntry := resp[0]
	expectedAddress := "192.168.0.2"
	remoteStaticDNSEntry.Address = expectedAddress
	var alert tc.Alerts
	var status int
	alert, _, status, err = TOSession.UpdateStaticDNSEntryByID(remoteStaticDNSEntry.ID, remoteStaticDNSEntry)
	log.Debugln("Status Code: ", status)
	if err != nil {
		t.Errorf("cannot UPDATE StaticDNSEntries using url: %v - %v\n", err, alert)
	}

	// Retrieve the StaticDNSEntries to check StaticDNSEntries name got updated
	resp, _, err = TOSession.GetStaticDNSEntryByID(remoteStaticDNSEntry.ID)
	if err != nil {
		t.Errorf("cannot GET StaticDNSEntries by name: '$%s', %v\n", firstStaticDNSEntry.Host, err)
	}
	respStaticDNSEntry := resp[0]
	if respStaticDNSEntry.Address != expectedAddress {
		t.Errorf("results do not match actual: %s, expected: %s\n", respStaticDNSEntry.Address, expectedAddress)
	}

}

func UpdateTestStaticDNSEntriesInvalidAddress(t *testing.T) {

	expectedAlerts := []tc.Alerts{tc.Alerts{[]tc.Alert{tc.Alert{"'address' must be a valid IPv4 address", "error"}}}, tc.Alerts{[]tc.Alert{tc.Alert{"'address' must be a valid DNS name", "error"}}}, tc.Alerts{[]tc.Alert{tc.Alert{"'address' must be a valid IPv6 address", "error"}}}}

	// A_RECORD
	firstStaticDNSEntry := testData.StaticDNSEntries[0]
	// Retrieve the StaticDNSEntries by name so we can get the id for the Update
	resp, _, err := TOSession.GetStaticDNSEntriesByHost(firstStaticDNSEntry.Host)
	if err != nil {
		t.Errorf("cannot GET StaticDNSEntries by name: '%s', %v\n", firstStaticDNSEntry.Host, err)
	}
	remoteStaticDNSEntry := resp[0]
	expectedAddress := "test.testdomain.net."
	remoteStaticDNSEntry.Address = expectedAddress
	var alert tc.Alerts
	var status int
	alert, _, status, err = TOSession.UpdateStaticDNSEntryByID(remoteStaticDNSEntry.ID, remoteStaticDNSEntry)
	log.Debugln("Status Code [expect 400]: ", status)
	if err != nil {
		log.Debugf("cannot UPDATE StaticDNSEntries using url: %v - %v\n", err, alert)
	}
	if !reflect.DeepEqual(alert, expectedAlerts[0]) {
		t.Errorf("got alerts: %v but expected alerts: %v", alert, expectedAlerts[0])
	}

	// CNAME_RECORD
	secondStaticDNSEntry := testData.StaticDNSEntries[1]
	// Retrieve the StaticDNSEntries by name so we can get the id for the Update
	resp, _, err = TOSession.GetStaticDNSEntriesByHost(secondStaticDNSEntry.Host)
	if err != nil {
		t.Errorf("cannot GET StaticDNSEntries by name: '%s', %v\n", secondStaticDNSEntry.Host, err)
	}
	remoteStaticDNSEntry = resp[0]
	expectedAddress = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	remoteStaticDNSEntry.Address = expectedAddress
	alert, _, status, err = TOSession.UpdateStaticDNSEntryByID(remoteStaticDNSEntry.ID, remoteStaticDNSEntry)
	log.Debugln("Status Code [expect 400]: ", status)
	if err != nil {
		log.Debugf("cannot UPDATE StaticDNSEntries using url: %v - %v\n", err, alert)
	}
	if !reflect.DeepEqual(alert, expectedAlerts[1]) {
		t.Errorf("got alerts: %v but expected alerts: %v", alert, expectedAlerts[1])
	}

	// AAAA_RECORD
	thirdStaticDNSEntry := testData.StaticDNSEntries[2]
	// Retrieve the StaticDNSEntries by name so we can get the id for the Update
	resp, _, err = TOSession.GetStaticDNSEntriesByHost(thirdStaticDNSEntry.Host)
	if err != nil {
		t.Errorf("cannot GET StaticDNSEntries by name: '%s', %v\n", thirdStaticDNSEntry.Host, err)
	}
	remoteStaticDNSEntry = resp[0]
	expectedAddress = "192.168.0.1"
	remoteStaticDNSEntry.Address = expectedAddress
	alert, _, status, err = TOSession.UpdateStaticDNSEntryByID(remoteStaticDNSEntry.ID, remoteStaticDNSEntry)
	log.Debugln("Status Code [expect 400]: ", status)
	if err != nil {
		log.Debugf("cannot UPDATE StaticDNSEntries using url: %v - %v\n", err, alert)
	}
	if !reflect.DeepEqual(alert, expectedAlerts[2]) {
		t.Errorf("got alerts: %v but expected alerts: %v", alert, expectedAlerts[2])
	}
}

func GetTestStaticDNSEntries(t *testing.T) {

	for _, staticDNSEntry := range testData.StaticDNSEntries {
		resp, _, err := TOSession.GetStaticDNSEntriesByHost(staticDNSEntry.Host)
		if err != nil {
			t.Errorf("cannot GET StaticDNSEntries by name: %v - %v\n", err, resp)
		}
	}
}

func DeleteTestStaticDNSEntries(t *testing.T) {

	for _, staticDNSEntry := range testData.StaticDNSEntries {
		// Retrieve the StaticDNSEntries by name so we can get the id for the Update
		resp, _, err := TOSession.GetStaticDNSEntriesByHost(staticDNSEntry.Host)
		if err != nil {
			t.Errorf("cannot GET StaticDNSEntries by name: %v - %v\n", staticDNSEntry.Host, err)
		}
		if len(resp) > 0 {
			respStaticDNSEntry := resp[0]

			_, _, err := TOSession.DeleteStaticDNSEntryByID(respStaticDNSEntry.ID)
			if err != nil {
				t.Errorf("cannot DELETE StaticDNSEntry by name: '%s' %v\n", respStaticDNSEntry.Host, err)
			}

			// Retrieve the StaticDNSEntry to see if it got deleted
			staticDNSEntries, _, err := TOSession.GetStaticDNSEntriesByHost(staticDNSEntry.Host)
			if err != nil {
				t.Errorf("error deleting StaticDNSEntrie name: %s\n", err.Error())
			}
			if len(staticDNSEntries) > 0 {
				t.Errorf("expected StaticDNSEntry name: %s to be deleted\n", staticDNSEntry.Host)
			}
		}
	}
}
