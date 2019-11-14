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
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestDeliveryServiceIMS(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Users, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, DeliveryServices}, func() {
		GetTestDeliveryServiceIMSAll(t)
		GetTestDeliveryServiceIMSSingle(t)
	})
}

func GetTestDeliveryServiceIMSAll(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices", nil)
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
		t.Fatalf("deliveryservices request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("deliveryservices request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	{
		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices", nil)
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
			t.Errorf("deliveryservices request with " + rfc.HdrIfModifiedSince + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}

	{
		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices", nil)
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
			t.Errorf("deliveryservices request with " + rfc.HdrIfNoneMatch + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}
}

func GetTestDeliveryServiceIMSSingle(t *testing.T) {
	dses, _, err := TOSession.GetDeliveryServices()
	if err != nil {
		t.Errorf("cannot GET DeliveryServices: %v\n", err)
	}
	ds := tc.DeliveryService{}
	for _, tods := range dses {
		if !tods.Type.IsHTTP() && !tods.Type.IsDNS() {
			continue
		}
		ds = tods
		break
	}
	if ds.ID == 0 {
		t.Fatalf("GET DeliveryServices returned no DNS or HTTP dses, must have at least 1 to test")
	}

	req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
		t.Fatalf("deliveryservices request expected: " + rfc.HdrLastModified + " header, actual: missing")
	}

	etag := resp.Header.Get(rfc.HdrETag)
	if etag == "" {
		t.Fatalf("deliveryservices request expected: " + rfc.HdrETag + " header, actual: missing")
	}

	{
		// test a single DS with INM works

		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
			t.Errorf("deliveryservices request with " + rfc.HdrIfNoneMatch + " expected: 304, actual: " + strconv.Itoa(resp.StatusCode))
		}
	}

	{
		// test modifying the DS with a field on the same table, and verify an IMS/INM do NOT return a 304

		ds.MaxOriginConnections += 50
		time.Sleep(time.Second) // sleep for 1s, because IMS is 1-second resolution. Otherwise, it may not be modified.
		if _, err := TOSession.UpdateDeliveryService(strconv.Itoa(ds.ID), &ds); err != nil {
			t.Fatalf("cannot UPDATE DeliveryService by ID: %v\n", err)
		}

		{
			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
				t.Errorf("deliveryservices request with " + rfc.HdrIfNoneMatch + " and modified DS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
		{
			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
				t.Errorf("deliveryservices request with " + rfc.HdrIfModifiedSince + " and modified DS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
	}

	{
		// test modifying the DS with a field on a different table, and verify an IMS/INM do NOT return a 304

		// Need to get a new LastModified and ETag, because of the above modification

		req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
			t.Fatalf("deliveryservices request expected: " + rfc.HdrLastModified + " header, actual: missing")
		}

		etag := resp.Header.Get(rfc.HdrETag)
		if etag == "" {
			t.Fatalf("deliveryservices request expected: " + rfc.HdrETag + " header, actual: missing")
		}

		types, _, err := TOSession.GetTypeByID(ds.TypeID)
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
			// test INM after update of single DS with non-ds table

			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
				t.Errorf("deliveryservices request with " + rfc.HdrIfNoneMatch + " and modified DS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
		{
			// test IMS after update of single DS with non-ds table

			req, err := http.NewRequest(http.MethodGet, TOSession.URL+"/api/1.4/deliveryservices?id="+strconv.Itoa(ds.ID), nil)
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
				t.Errorf("deliveryservices request with " + rfc.HdrIfModifiedSince + " and modified DS expected: 200, actual: " + strconv.Itoa(resp.StatusCode))
			}
		}
	}
}
