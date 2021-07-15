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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// apiCDNs is the API version-relative path for the /cdns API endpoint.
const apiCDNs = "/cdns"

// CreateCDN creates a CDN.
func (to *Session) CreateCDN(cdn tc.CDN, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqInf, err := to.post(apiCDNs, opts, cdn, &alerts)
	return alerts, reqInf, err
}

// UpdateCDN replaces the identified CDN with the provided CDN.
func (to *Session) UpdateCDN(id int, cdn tc.CDN, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	route := fmt.Sprintf("%s/%d", apiCDNs, id)
	var alerts tc.Alerts
	reqInf, err := to.put(route, opts, cdn, &alerts)
	return alerts, reqInf, err
}

// GetCDNs retrieves CDNs from Traffic Ops.
func (to *Session) GetCDNs(opts RequestOptions) (tc.CDNsResponse, toclientlib.ReqInf, error) {
	var data tc.CDNsResponse
	reqInf, err := to.get(apiCDNs, opts, &data)
	return data, reqInf, err
}

// DeleteCDN deletes the CDN with the given ID.
func (to *Session) DeleteCDN(id int, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	route := fmt.Sprintf("%s/%d", apiCDNs, id)
	var alerts tc.Alerts
	reqInf, err := to.del(route, opts, &alerts)
	return alerts, reqInf, err
}

// GetCDNSSLKeys retrieves the SSL keys for the CDN with the given name.
func (to *Session) GetCDNSSLKeys(name string, opts RequestOptions) (tc.CDNSSLKeysResponse, toclientlib.ReqInf, error) {
	route := fmt.Sprintf("%s/name/%s/sslkeys", apiCDNs, url.PathEscape(name))
	var data tc.CDNSSLKeysResponse
	reqInf, err := to.get(route, opts, &data)
	return data, reqInf, err
}

// QueueUpdatesForCDN set the "updPending" field of a list of servers identified by
// 'cdnID' and any other query params (type or profile) to the value of 'queueUpdate'
func (to *Session) QueueUpdatesForCDN(cdnID int, queueUpdate bool, opts RequestOptions) (tc.CDNQueueUpdateResponse, toclientlib.ReqInf, error) {
	req := tc.CDNQueueUpdateRequest{Action: queueUpdateActions[queueUpdate]}
	var resp tc.CDNQueueUpdateResponse
	if opts.QueryParameters == nil {
		opts.QueryParameters = url.Values{}
	}
	path := fmt.Sprintf("/cdns/%d/queue_update", cdnID)
	reqInf, err := to.post(path, opts, req, &resp)
	return resp, reqInf, err
}
