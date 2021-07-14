package e2e

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
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestBasicReq(t *testing.T) {
	dses := filterAssetDSes(Cfg.DSAssets, TO.DeliveryServices)

	ds := filterDS(dses, func(ds *tc.DeliveryService) bool {
		if !ds.Active {
			return false
		}
		if !ds.Type.IsHTTP() {
			return false
		}
		if len(ds.ExampleURLs) == 0 {
			return false
		}
		return true
	})

	if ds == nil {
		t.Fatalf("An active HTTP delivery service with an asset in the config is required to run this test.")
	}

	reqURI := ds.ExampleURLs[0] + Cfg.DSAssets[tc.DeliveryServiceName(ds.XMLID)]

	log.Infof("TestBasicReq using ds '%+v' uri '%+v'\n", ds.XMLID, reqURI)

	resp, err := http.Get(reqURI)
	if err != nil {
		t.Fatalf("error getting ds '%+v' URI '%+v': %+v\n", ds.XMLID, reqURI, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("request ds '%+v' URI '%+v' expected code %+v actual %+v\n", ds.XMLID, reqURI, http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("request ds '%+v' URI '%+v' reading body err expected %+v actual %+v\n", ds.XMLID, reqURI, nil, err)
	}
	if len(body) == 0 {
		t.Fatalf("request ds '%+v' URI '%+v' reading body len expected %+v actual %+v\n", ds.XMLID, reqURI, ">0", len(body))
	}
}

func TestHTTPDSReq(t *testing.T) {
	dses := filterAssetDSes(Cfg.DSAssets, TO.DeliveryServices)

	ds := filterDS(dses, func(ds *tc.DeliveryService) bool {
		if !ds.Active {
			return false
		}
		if !ds.Type.IsHTTP() {
			return false
		}
		if len(ds.ExampleURLs) == 0 {
			return false
		}
		return true
	})

	if ds == nil {
		t.Fatalf("An active HTTP delivery service with an asset in the config is required to run this test.")
	}

	reqURI := ds.ExampleURLs[0] + Cfg.DSAssets[tc.DeliveryServiceName(ds.XMLID)]

	log.Infof("TestBasicReq using ds '%+v' uri '%+v'\n", ds.XMLID, reqURI)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	} // make a client that doesn't follow redirects

	resp, err := client.Get(reqURI)
	if err != nil {
		t.Fatalf("error getting ds '%+v' URI '%+v': %+v\n", ds.XMLID, reqURI, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("request ds '%+v' URI '%+v' expected code %+v actual %+v\n", ds.XMLID, reqURI, http.StatusFound, resp.StatusCode)
	}

	locHdr := resp.Header.Get("Location")
	if locHdr == "" {
		t.Fatalf("request ds '%+v' URI '%+v' location expected:  nonempty actual: empty\n", ds.XMLID, reqURI)
	}

	locResp, err := client.Get(locHdr)
	if err != nil {
		t.Fatalf("error getting ds '%+v' location URI '%+v': %+v\n", ds.XMLID, locHdr, err)
	}
	defer locResp.Body.Close()
	body, err := ioutil.ReadAll(locResp.Body)
	if err != nil {
		t.Fatalf("request ds '%+v' location URI '%+v' reading body err expected %+v actual %+v\n", ds.XMLID, locHdr, nil, err)
	}
	if len(body) == 0 {
		t.Fatalf("request ds '%+v' location URI '%+v' reading body len expected %+v actual %+v\n", ds.XMLID, locHdr, ">0", len(body))
	}
}

func TestDNSDSReq(t *testing.T) {
	dses := filterAssetDSes(Cfg.DSAssets, TO.DeliveryServices)

	ds := filterDS(dses, func(ds *tc.DeliveryService) bool {
		if !ds.Active {
			return false
		}
		if !ds.Type.IsDNS() {
			return false
		}
		if len(ds.ExampleURLs) == 0 {
			return false
		}
		return true
	})

	if ds == nil {
		t.Fatalf("An active HTTP delivery service with an asset in the config is required to run this test.")
	}

	reqURI := ds.ExampleURLs[0] + Cfg.DSAssets[tc.DeliveryServiceName(ds.XMLID)]

	log.Infof("TestDNSDSReq using ds '%+v' uri '%+v'\n", ds.XMLID, reqURI)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	} // make a client that doesn't follow redirects - a DNS DS should have no HTTP redirects.

	remoteAddr := net.Addr(nil)

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			remoteAddr = connInfo.Conn.RemoteAddr()
		},
	}

	req, err := http.NewRequest(http.MethodGet, reqURI, nil)
	if err != nil {
		t.Fatalf("error creating http request for ds '%+v' URI '%+v': %+v\n", ds.XMLID, reqURI, err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("error getting ds '%+v' URI '%+v': %+v\n", ds.XMLID, reqURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("request ds '%+v' URI '%+v' expected code %+v actual %+v\n", ds.XMLID, reqURI, http.StatusOK, resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("request ds '%+v' URI '%+v' reading body err expected %+v actual %+v\n", ds.XMLID, reqURI, nil, err)
	}
	if len(body) == 0 {
		t.Fatalf("request ds '%+v' URI '%+v' reading body len expected %+v actual %+v\n", ds.XMLID, reqURI, ">0", len(body))
	}

	remoteAddrHost, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		t.Fatalf("request ds '%+v' URI '%+v' error getting remoteAddr SplitHostPort: %+v\n", ds.XMLID, reqURI, err)
	}

	// tc.DeliveryServiceName(ds.XMLID)
	foundServer := false
	// log.Infof("len(DeliveryServiceServers[%v]) %+v\n", ds.XMLID, len(DeliveryServiceServers[tc.DeliveryServiceName(ds.XMLID)]))
	// log.Infof("DeliveryServiceServers %+v\n\n", DeliveryServiceServers)
	for _, serverName := range TO.DeliveryServiceServers[tc.DeliveryServiceName(ds.XMLID)] {
		server := TO.Servers[serverName]
		if serverIPEqualsRemoteAddr(server.IPAddress, server.IP6Address, remoteAddrHost) {
			log.Infof("TestDNSDSReq using ds '%+v' uri '%+v' used server '%+v'\n", ds.XMLID, reqURI, server.HostName)
			foundServer = true
			break
		}
	}
	if !foundServer {
		t.Fatalf("request ds '%+v' URI '%+v' request RemoteAddr expected: %+v, actual %+v\n", ds.XMLID, reqURI, "a server assigned to this ds", remoteAddr)
	}
}

func filterAssetDSes(dsAssets map[tc.DeliveryServiceName]string, dses map[tc.DeliveryServiceName]tc.DeliveryService) map[tc.DeliveryServiceName]tc.DeliveryService {
	filtered := map[tc.DeliveryServiceName]tc.DeliveryService{}
	for dsName, _ := range dsAssets {
		if ds, ok := dses[dsName]; ok {
			filtered[dsName] = ds
		}
	}
	return filtered
}

// filterDS takes a map of delivery services, and a filter func, and returns the first delivery service which matches the filter, or nil if none match.
func filterDS(dses map[tc.DeliveryServiceName]tc.DeliveryService, filter func(ds *tc.DeliveryService) bool) *tc.DeliveryService {
	for _, ds := range dses {
		if filter(&ds) {
			return &ds
		}
	}
	return nil
}

func serverIPEqualsRemoteAddr(serverIP4 string, serverIP6 string, remoteAddrHost string) bool {
	remoteAddrIP := net.ParseIP(remoteAddrHost)
	if remoteAddrIP == nil {
		log.Infoln("serverIPEqualsRemoteAddr '" + remoteAddrHost + "' is nil, returning false")
		return false

	}
	serverIPStr := serverIP6
	if remoteAddrIP.To4() != nil {
		serverIPStr = serverIP4
	}

	if isCIDR := strings.Contains(serverIPStr, "/"); isCIDR {
		_, cidr, err := net.ParseCIDR(serverIPStr)
		if err != nil {
			log.Infoln("serverIPEqualsRemoteAddr error parsing cidr %+v %+v, returning false", serverIPStr, err)
			return false
		}
		return cidr.Contains(remoteAddrIP)
	} else {
		serverIP := net.ParseIP(serverIPStr)
		if serverIP == nil {
			log.Infoln("serverIPEqualsRemoteAddr error parsing ip %+v, returning false", serverIPStr)
			return false
		}
		return serverIP.Equal(remoteAddrIP)
	}
}

func getAddrIPStr(addr string) string {
	if addr == "" {
		return addr
	}
	if addr[len(addr)-1] == ']' {
		return addr // this is an IPv6 address with no port on the end
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return addr
	}
	return addr[:i]
}
