package client

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
	"net/url"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// CreateDeliveryServiceServers associates the given servers with the given
// Delivery Services. If replace is true, it deletes any existing associations
// for the given Delivery Service.
func (to *Session) CreateDeliveryServiceServers(dsID int, serverIDs []int, replace bool, opts RequestOptions) (tc.DeliveryserviceserverResponse, toclientlib.ReqInf, error) {
	req := tc.DSServerIDs{
		DeliveryServiceID: util.IntPtr(dsID),
		ServerIDs:         serverIDs,
		Replace:           util.BoolPtr(replace),
	}
	var resp tc.DeliveryserviceserverResponse
	reqInf, err := to.post(apiDeliveryServiceServer, opts, req, &resp)
	return resp, reqInf, err
}

// DeleteDeliveryServiceServer removes the association between the Delivery
// Service identified by dsID and the server identified by serverID.
func (to *Session) DeleteDeliveryServiceServer(dsID int, serverID int, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	route := fmt.Sprintf("%s/%d/%d", apiDeliveryServiceServer, dsID, serverID)
	var resp tc.Alerts
	reqInf, err := to.del(route, opts, &resp)
	return resp, reqInf, err
}

// AssignServersToDeliveryService assigns the given list of servers to the
// Delivery Service with the given xmlID.
func (to *Session) AssignServersToDeliveryService(servers []string, xmlID string, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	route := fmt.Sprintf(apiDeliveryServicesServers, url.PathEscape(xmlID))
	dss := tc.DeliveryServiceServers{ServerNames: servers, XmlId: xmlID}
	var resp tc.Alerts
	reqInf, err := to.post(route, opts, dss, &resp)
	return resp, reqInf, err
}

// GetServersByDeliveryService gets the servers that are assigned to the delivery service with the given ID.
func (to *Session) GetServersByDeliveryService(id int, opts RequestOptions) (tc.DSServerResponseV4, toclientlib.ReqInf, error) {
	route := fmt.Sprintf(apiDeliveryServicesServers, strconv.Itoa(id))
	resp := tc.DSServerResponseV4{}
	reqInf, err := to.get(route, opts, &resp)
	return resp, reqInf, err
}

// GetDeliveryServiceServers returns associations between Delivery Services and
// servers.
func (to *Session) GetDeliveryServiceServers(opts RequestOptions) (tc.DeliveryServiceServerResponse, toclientlib.ReqInf, error) {
	var resp tc.DeliveryServiceServerResponse
	reqInf, err := to.get(apiDeliveryServiceServer, opts, &resp)
	return resp, reqInf, err
}
