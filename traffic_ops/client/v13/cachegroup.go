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

package v13

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc/common"
)

const (
	API_v13_CacheGroups = "/api/1.3/cachegroups"
)

// Create a CacheGroup
func (to *Session) CreateCacheGroup(cachegroup tc.CacheGroup) (common.Alerts, ReqInf, error) {

	var remoteAddr net.Addr
	reqBody, err := json.Marshal(cachegroup)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return common.Alerts{}, reqInf, err
	}
	resp, remoteAddr, err := to.request(http.MethodPost, API_v13_CacheGroups, reqBody)
	if err != nil {
		return common.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	var alerts common.Alerts
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	return alerts, reqInf, nil
}

// Update a CacheGroup by ID
func (to *Session) UpdateCacheGroupByID(id int, cachegroup tc.CacheGroup) (common.Alerts, ReqInf, error) {

	var remoteAddr net.Addr
	reqBody, err := json.Marshal(cachegroup)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return common.Alerts{}, reqInf, err
	}
	route := fmt.Sprintf("%s/%d", API_v13_CacheGroups, id)
	resp, remoteAddr, err := to.request(http.MethodPut, route, reqBody)
	if err != nil {
		return common.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	var alerts common.Alerts
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	return alerts, reqInf, nil
}

// Returns a list of CacheGroups
func (to *Session) GetCacheGroups() ([]tc.CacheGroup, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, API_v13_CacheGroups, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()

	var data tc.CacheGroupsResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data.Response, reqInf, nil
}

// GET a CacheGroup by the CacheGroup id
func (to *Session) GetCacheGroupByID(id int) ([]tc.CacheGroup, ReqInf, error) {
	route := fmt.Sprintf("%s/%d", API_v13_CacheGroups, id)
	resp, remoteAddr, err := to.request(http.MethodGet, route, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()

	var data tc.CacheGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, reqInf, err
	}

	return data.Response, reqInf, nil
}

// GET a CacheGroup by the CacheGroup name
func (to *Session) GetCacheGroupByName(name string) ([]tc.CacheGroup, ReqInf, error) {
	url := fmt.Sprintf("%s?name=%s", API_v13_CacheGroups, name)
	resp, remoteAddr, err := to.request(http.MethodGet, url, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()

	var data tc.CacheGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, reqInf, err
	}

	return data.Response, reqInf, nil
}

// DELETE a CacheGroup by ID
func (to *Session) DeleteCacheGroupByID(id int) (common.Alerts, ReqInf, error) {
	route := fmt.Sprintf("%s/%d", API_v13_CacheGroups, id)
	resp, remoteAddr, err := to.request(http.MethodDelete, route, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return common.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	var alerts common.Alerts
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	return alerts, reqInf, nil
}
