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
)

func TestATSConfigs(t *testing.T) {
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

	GetTestATSConfigs(t)

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

func GetTestATSConfigs(t *testing.T) {
	if len(testData.Servers) < 1 {
		t.Fatalf("cannot GET Server: no test data\n")
	}
	testServer := testData.Servers[0]

	serverList, _, err := TOSession.GetServerByHostName(testServer.HostName)
	if err != nil {
		t.Fatalf("cannot GET Server: %v\n", err)
	}
	if len(serverList) < 1 {
		t.Fatalf("cannot GET Server '" + testServer.HostName + "', returned no servers\n")
	}
	server := serverList[0]

	_, _, err = TOSession.GetATSServerConfigList(server.ID)
	if err != nil {
		t.Fatalf("Getting server '" + server.HostName + "' config list: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSServerConfigListByName(server.HostName)
	if err != nil {
		t.Fatalf("Getting server by name '" + server.HostName + "' config list: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSServerConfig(server.ID, "remap.config")
	if err != nil {
		t.Fatalf("Getting server '" + server.HostName + "' config remap.config: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSServerConfigByName(server.HostName, "remap.config")
	if err != nil {
		t.Fatalf("Getting server by name '" + server.HostName + "' config remap.config: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSProfileConfig(server.ProfileID, "storage.config")
	if err != nil {
		t.Fatalf("Getting profile '" + server.Profile + "' config storage.config: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSProfileConfigByName(server.Profile, "storage.config")
	if err != nil {
		t.Fatalf("Getting profile by name '" + server.Profile + "' config storage.config: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSCDNConfig(server.CDNID, "bg_fetch.config")
	if err != nil {
		t.Fatalf("Getting cdn '" + server.CDNName + "' config bg_fetch.config: " + err.Error() + "\n")
	}

	_, _, err = TOSession.GetATSCDNConfigByName(server.CDNName, "bg_fetch.config")
	if err != nil {
		t.Fatalf("Getting cdn by name '" + server.CDNName + "' config bg_fetch.config: " + err.Error() + "\n")
	}
	log.Debugln("GetTestATSConfigs() PASSED: ")
}
