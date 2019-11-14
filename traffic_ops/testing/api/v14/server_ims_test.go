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
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestServerIMS(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Users, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, DeliveryServices}, func() {
		defer DeleteTestDeliveryServiceServersCreated(t)
		CreateTestDeliveryServiceServers(t)

		GetTestServerIMSAllServers(t)
		GetTestServerIMSSingleServer(t)
		GetTestServerIMSDeliveryServiceServers(t)
		GetTestServerIMSMids(t)
	})
}

func GetTestServerIMSAllServers(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers", nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	resp, err := TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	lastModified := resp.Header.Get(rfc.HdrLastModified)
	if lastModified == "" {
		t.Fatalf("servers request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("servers request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	{
		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers", nil)
		if err != nil {
			t.Fatalf("failed to create request: %s", err.Error())
		}
		req.Header.Add(rfc.HdrIfModifiedSince, lastModified)

		resp, err := TOSession.Client.Do(req)
		if err != nil {
			t.Fatalf("running request: %s", err.Error())
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotModified {
			t.Errorf("servers request with " + rfc.HdrIfModifiedSince + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}

	{
		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers", nil)
		if err != nil {
			t.Fatalf("failed to create request: %s", err.Error())
		}
		req.Header.Add(rfc.HdrIfNoneMatch, etag)

		resp, err := TOSession.Client.Do(req)
		if err != nil {
			t.Fatalf("running request: %s", err.Error())
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotModified {
			t.Errorf("servers request with " + rfc.HdrIfNoneMatch + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}
}

func GetTestServerIMSSingleServer(t *testing.T) {
	servers, _, err := TOSession.GetServers()
	if err != nil {
		t.Errorf("cannot GET servers: %v\n", err)
	}
	server := tc.Server{}
	for _, toserver := range servers {
		if toserver.Type != string(tc.CacheTypeEdge) {
			continue
		}
		server = toserver
		break
	}
	if server.ID == 0 {
		t.Fatalf("GET Servers returned no EDGE server, must have at least 1 to test")
	}

	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	resp, err := TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	lastModified := resp.Header.Get(rfc.HdrLastModified)
	if lastModified == "" {
		t.Fatalf("server request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("server request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	{
		// test a single server with INM works

		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %s", err.Error())
		}
		req.Header.Add(rfc.HdrIfNoneMatch, etag)

		resp, err := TOSession.Client.Do(req)
		if err != nil {
			t.Fatalf("running request: %s", err.Error())
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotModified {
			t.Errorf("servers request with " + rfc.HdrIfNoneMatch + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}

	{
		// test modifying the server with a field on the same table, and verify an IMS/INM do NOT return a 304

		server.XMPPID += "-testchange"
		time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
		if _, _, err := TOSession.UpdateServerByID(server.ID, server); err != nil {
			t.Fatalf("cannot UPDATE server by ID: %v\n", err)
		}

		{
			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %s", err.Error())
			}
			req.Header.Add(rfc.HdrIfNoneMatch, etag)

			resp, err := TOSession.Client.Do(req)
			if err != nil {
				t.Fatalf("running request: %s", err.Error())
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("servers request with " + rfc.HdrIfNoneMatch + " and modified DS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
		{
			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %s", err.Error())
			}
			req.Header.Add(rfc.HdrIfModifiedSince, lastModified)

			resp, err := TOSession.Client.Do(req)
			if err != nil {
				t.Fatalf("running request: %s", err.Error())
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("servers request with " + rfc.HdrIfModifiedSince + " and modified server expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
	}

	{
		// test modifying the server with a field on a different table, and verify an IMS/INM do NOT return a 304

		// Need to get a new LastModified and ETag, because of the above modification

		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %s", err.Error())
		}

		resp, err := TOSession.Client.Do(req)
		if err != nil {
			t.Fatalf("running request: %s", err.Error())
		}
		resp.Body.Close()

		lastModified := resp.Header.Get(rfc.HdrLastModified)
		if lastModified == "" {
			t.Fatalf("servers request expected: " + rfc.HdrLastModified + " header, actual: missing")
		}

		etag := resp.Header.Get(rfc.HdrETag)
		if etag == "" {
			t.Fatalf("server request expected: " + rfc.HdrETag + " header, actual: missing")
		}

		types, _, err := TOSession.GetTypeByID(server.TypeID)
		if err != nil {
			t.Fatalf("cannot get type by ID: %v\n", err)
		}
		if len(types) != 1 {
			t.Fatalf("get types expected 1, actual %v\n", len(types))
		}
		typ := types[0]
		typ.Description += " addsomething"

		time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
		if _, _, err := TOSession.UpdateTypeByID(typ.ID, typ); err != nil {
			t.Fatalf("cannot update type by ID: %v\n", err)
		}

		{
			// test INM after update of single server with non-server table

			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %s", err.Error())
			}
			req.Header.Add(rfc.HdrIfNoneMatch, etag)

			resp, err := TOSession.Client.Do(req)
			if err != nil {
				t.Fatalf("running request: %s", err.Error())
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("servers request with " + rfc.HdrIfNoneMatch + " and modified server expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
		{
			// test IMS after update of single server with non-server table

			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?id="+strconv.Itoa(server.ID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %s", err.Error())
			}
			req.Header.Add(rfc.HdrIfModifiedSince, lastModified)

			resp, err := TOSession.Client.Do(req)
			if err != nil {
				t.Fatalf("running request: %s", err.Error())
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("servers request with " + rfc.HdrIfModifiedSince + " and modified server expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
	}
}

func GetTestServerIMSDeliveryServiceServers(t *testing.T) {
	dsServers, _, err := TOSession.GetDeliveryServiceServers()
	if err != nil {
		t.Fatalf("GET delivery service servers: %v\n", err)
	} else if len(dsServers.Response) == 0 {
		t.Fatalf("GET delivery service servers: no servers found\n")
	} else if dsServers.Response[0].Server == nil {
		t.Fatalf("GET delivery service servers: returned nil server\n")
	} else if dsServers.Response[0].DeliveryService == nil {
		t.Fatalf("GET delivery service servers: returned nil ds\n")
	}
	serverID := *dsServers.Response[0].Server
	dsID := *dsServers.Response[0].DeliveryService

	dsServerIDs := map[int]struct{}{}
	for _, dss := range dsServers.Response {
		if dss.DeliveryService == nil || dss.Server == nil || *dss.DeliveryService != dsID {
			continue
		}
		dsServerIDs[*dss.Server] = struct{}{}
	}

	// get a different serverID not already assigned to this dsID

	servers, _, err := TOSession.GetServers()
	if err != nil {
		t.Errorf("cannot GET servers: %v\n", err)
	}
	otherServer := tc.Server{}
	for _, toserver := range servers {
		if toserver.Type != string(tc.CacheTypeEdge) || toserver.ID == serverID {
			continue
		}
		if _, ok := dsServerIDs[toserver.ID]; ok {
			continue
		}
		otherServer = toserver
		break
	}
	if otherServer.ID == 0 {
		t.Fatalf("GET Servers returned no EDGE server not assigned to dsID %v, must have at least 1 to test", dsID)
	}

	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(dsID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	resp, err := TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	lastModified := resp.Header.Get(rfc.HdrLastModified)
	if lastModified == "" {
		t.Fatalf("server request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("server request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
	_, err = TOSession.CreateDeliveryServiceServers(dsID, []int{otherServer.ID}, false)
	if err != nil {
		t.Fatalf("create dss request err: " + err.Error())
	}

	// test IMS of ?dsId after creating a DSS

	req, err = http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(dsID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}
	req.Header.Add(rfc.HdrIfModifiedSince, lastModified)

	resp, err = TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("servers request with " + rfc.HdrIfModifiedSince + " and added DSS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
	}

	// re-get lastModified+etag of servers?dsId after we created a DSS

	req, err = http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(dsID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	resp, err = TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	lastModified = resp.Header.Get(rfc.HdrLastModified)
	if lastModified == "" {
		t.Fatalf("server request expected: "+rfc.HdrLastModified+" header, actual: missing (code %v) (hdrs %++v)\n", resp.StatusCode, resp.Header)
	}

	etag = resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("server request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
	_, _, err = TOSession.DeleteDeliveryServiceServer(dsID, otherServer.ID)
	if err != nil {
		t.Fatalf("delete dss request err: " + err.Error())
	}

	// test IMS of ?dsId after deleting a DSS

	req, err = http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(dsID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}
	req.Header.Add(rfc.HdrIfModifiedSince, lastModified)

	resp, err = TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("servers request with " + rfc.HdrIfModifiedSince + " and deleted DSS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
	}
}

func GetTestServerIMSMids(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices()
	if err != nil {
		t.Errorf("cannot GET DeliveryServices: %v - %v\n", err, dses)
	}
	dsIDMap := map[int]tc.DeliveryService{}
	for _, ds := range dses {
		dsIDMap[ds.ID] = ds
	}

	dsServers, _, err := TOSession.GetDeliveryServiceServers()
	if err != nil {
		t.Fatalf("GET delivery service servers: %v\n", err)
	} else if len(dsServers.Response) == 0 {
		t.Fatalf("GET delivery service servers: no servers found\n")
	} else if dsServers.Response[0].Server == nil {
		t.Fatalf("GET delivery service servers: returned nil server\n")
	} else if dsServers.Response[0].DeliveryService == nil {
		t.Fatalf("GET delivery service servers: returned nil ds\n")
	}

	ds := tc.DeliveryService{}
	for _, dss := range dsServers.Response {
		if dss.Server == nil || dss.DeliveryService == nil {
			continue
		}
		dssDS := dsIDMap[*dss.DeliveryService]
		if dssDS.Type.UsesMidCache() {
			ds = dssDS
			break
		}
	}
	if ds.ID == 0 {
		t.Fatalf("DSS returned no UsesMidCache type DS %v, must have at least 1 to test", ds.ID)
	}

	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(ds.ID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	resp, err := TOSession.Client.Do(req)
	if err != nil {
		t.Fatalf("running request: %s", err.Error())
	}

	lastModified := resp.Header.Get(rfc.HdrLastModified)
	if lastModified == "" {
		t.Fatalf("server request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("server request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	serversResp := tc.ServersResponse{}
	err = json.NewDecoder(resp.Body).Decode(&serversResp)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("/servers?dsId failed to decode body: " + err.Error())
	}

	midServer := tc.Server{}
	for _, sv := range serversResp.Response {
		if sv.Type == string(tc.CacheTypeMid) {
			midServer = sv
			break
		}
	}
	if midServer.ID == 0 {
		t.Fatalf("/servers?dsId=%v returned no mids, must have at least 1 to test", ds.ID)
	}

	midServer.XMPPID += "-testchange"
	time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
	if _, _, err := TOSession.UpdateServerByID(midServer.ID, midServer); err != nil {
		t.Fatalf("cannot UPDATE mid server by ID: %v\n", err)
	}

	{
		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/servers?dsId="+strconv.Itoa(ds.ID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %s", err.Error())
		}
		req.Header.Add(rfc.HdrIfNoneMatch, etag)

		resp, err := TOSession.Client.Do(req)
		if err != nil {
			t.Fatalf("running request: %s", err.Error())
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("servers request with " + rfc.HdrIfNoneMatch + " and modified parent mid expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}
}
