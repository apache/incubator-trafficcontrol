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

package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// CreateDeliveryServiceServers associates the given servers with the given delivery services. If replace is true, it deletes any existing associations for the given delivery service.
func (to *Session) CreateDeliveryServiceServers(dsID int, serverIDs []int, replace bool) (*tc.DSServerIDs, toclientlib.ReqInf, error) {
	req := tc.DSServerIDs{
		DeliveryServiceID: util.IntPtr(dsID),
		ServerIDs:         serverIDs,
		Replace:           util.BoolPtr(replace),
	}
	resp := struct {
		Response tc.DSServerIDs `json:"response"`
	}{}
	reqInf, err := to.post(APIDeliveryServiceServer, req, nil, &resp)
	if err != nil {
		return nil, reqInf, err
	}
	return &resp.Response, reqInf, nil
}

func (to *Session) DeleteDeliveryServiceServer(dsID int, serverID int) (tc.Alerts, toclientlib.ReqInf, error) {
	route := `/deliveryserviceserver/` + strconv.Itoa(dsID) + "/" + strconv.Itoa(serverID)
	resp := tc.Alerts{}
	reqInf, err := to.del(route, nil, &resp)
	return resp, reqInf, err
}

// AssignServersToDeliveryService assigns the given list of servers to the delivery service with the given xmlId.
func (to *Session) AssignServersToDeliveryService(servers []string, xmlId string) (tc.Alerts, toclientlib.ReqInf, error) {
	route := fmt.Sprintf(APIDeliveryServicesServers, url.QueryEscape(xmlId))
	dss := tc.DeliveryServiceServers{ServerNames: servers, XmlId: xmlId}
	resp := tc.Alerts{}
	reqInf, err := to.post(route, dss, nil, &resp)
	return resp, reqInf, err
}

// GetServersByDeliveryService gets the servers that are assigned to the delivery service with the given ID.
func (to *Session) GetServersByDeliveryService(id int) (tc.DSServerResponseV30, toclientlib.ReqInf, error) {
	route := fmt.Sprintf(APIDeliveryServicesServers, strconv.Itoa(id))
	resp := tc.DSServerResponseV30{}
	reqInf, err := to.get(route, nil, &resp)
	return resp, reqInf, err
}

// GetDeliveryServiceServer returns associations between Delivery Services and servers using the
// provided pagination controls.
// Deprecated: GetDeliveryServiceServer will be removed in 6.0. Use GetDeliveryServiceServerWithHdr.
func (to *Session) GetDeliveryServiceServer(page, limit string) ([]tc.DeliveryServiceServer, toclientlib.ReqInf, error) {
	return to.GetDeliveryServiceServerWithHdr(page, limit, nil)
}

func (to *Session) GetDeliveryServiceServerWithHdr(page, limit string, header http.Header) ([]tc.DeliveryServiceServer, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceServerResponse
	// TODO: page and limit should be integers not strings
	reqInf, err := to.get(APIDeliveryServiceServer+"?page="+url.QueryEscape(page)+"&limit="+url.QueryEscape(limit), header, &data)
	return data.Response, reqInf, err
}

func (to *Session) GetDeliveryServiceServersWithHdr(h http.Header) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	return to.getDeliveryServiceServers(url.Values{}, h)
}

// GetDeliveryServiceServers gets all delivery service servers, with the default API limit.
// Deprecated: GetDeliveryServiceServers will be removed in 6.0. Use GetDeliveryServiceServersWithHdr.
func (to *Session) GetDeliveryServiceServers() (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	return to.GetDeliveryServiceServersWithHdr(nil)
}

func (to *Session) GetDeliveryServiceServersNWithHdr(n int, header http.Header) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	return to.getDeliveryServiceServers(url.Values{"limit": []string{strconv.Itoa(n)}}, header)
}

// GetDeliveryServiceServersN gets all delivery service servers, with a limit of n.
// Deprecated: GetDeliveryServiceServersN will be removed in 6.0. Use GetDeliveryServiceServersNWithHdr.
func (to *Session) GetDeliveryServiceServersN(n int) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	return to.GetDeliveryServiceServersNWithHdr(n, nil)
}

func (to *Session) GetDeliveryServiceServersWithLimitsWithHdr(limit int, deliveryServiceIDs []int, serverIDs []int, header http.Header) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}

	if len(deliveryServiceIDs) != 0 {
		dsIDStrs := []string{}
		for _, dsID := range deliveryServiceIDs {
			dsIDStrs = append(dsIDStrs, strconv.Itoa(dsID))
		}
		vals.Set("deliveryserviceids", strings.Join(dsIDStrs, ","))
	}

	if len(serverIDs) != 0 {
		serverIDStrs := []string{}
		for _, serverID := range serverIDs {
			serverIDStrs = append(serverIDStrs, strconv.Itoa(serverID))
		}
		vals.Set("serverids", strings.Join(serverIDStrs, ","))
	}

	return to.getDeliveryServiceServers(vals, header)
}

// GetDeliveryServiceServersWithLimits gets all delivery service servers, allowing specifying the limit of mappings to return, the delivery services to return, and the servers to return.
// The limit may be 0, in which case the default limit will be applied. The deliveryServiceIDs and serverIDs may be nil or empty, in which case all delivery services and/or servers will be returned.
// Deprecated: GetDeliveryServiceServersWithLimits will be removed in 6.0. Use GetDeliveryServiceServersWithLimitsWithHdr.
func (to *Session) GetDeliveryServiceServersWithLimits(limit int, deliveryServiceIDs []int, serverIDs []int) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	return to.GetDeliveryServiceServersWithLimitsWithHdr(limit, deliveryServiceIDs, serverIDs, nil)
}

func (to *Session) getDeliveryServiceServers(urlQuery url.Values, h http.Header) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	route := APIDeliveryServiceServer
	if qry := urlQuery.Encode(); qry != "" {
		route += `?` + qry
	}
	resp := tc.DeliveryServiceServerResponse{}
	reqInf, err := to.get(route, h, &resp)
	return resp, reqInf, err
}
