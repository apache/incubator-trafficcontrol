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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func (to *Session) GetATSServerConfigList(serverID int) (tc.ATSConfigMetaData, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+"/servers/"+strconv.Itoa(serverID)+"/configfiles/ats", nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.ATSConfigMetaData{}, reqInf, err
	}
	defer resp.Body.Close()

	data := tc.ATSConfigMetaData{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return tc.ATSConfigMetaData{}, reqInf, err
	}
	return data, reqInf, nil
}

func (to *Session) GetATSServerConfigListByName(serverHostName string) (tc.ATSConfigMetaData, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+"/servers/"+serverHostName+"/configfiles/ats", nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.ATSConfigMetaData{}, reqInf, err
	}
	defer resp.Body.Close()

	data := tc.ATSConfigMetaData{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return tc.ATSConfigMetaData{}, reqInf, err
	}
	return data, reqInf, nil
}

func (to *Session) GetATSServerConfig(serverID int, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/servers/" + strconv.Itoa(serverID) + "/configfiles/ats/" + fileName)
}

func (to *Session) GetATSServerConfigByName(serverHostName string, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/servers/" + serverHostName + "/configfiles/ats/" + fileName)
}

func (to *Session) GetATSProfileConfig(profileID int, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/profiles/" + strconv.Itoa(profileID) + "/configfiles/ats/" + fileName)
}

func (to *Session) GetATSProfileConfigByName(profileName string, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/profiles/" + profileName + "/configfiles/ats/" + fileName)
}

func (to *Session) GetATSCDNConfig(cdnID int, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/cdns/" + strconv.Itoa(cdnID) + "/configfiles/ats/" + fileName)
}

func (to *Session) GetATSCDNConfigByName(cdnName string, fileName string) (string, ReqInf, error) {
	return to.getConfigFile(apiBase + "/cdns/" + cdnName + "/configfiles/ats/" + fileName)
}

func (to *Session) getConfigFile(uri string) (string, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, uri, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return "", reqInf, err
	}
	defer resp.Body.Close()

	bts, err := ioutil.ReadAll(resp.Body)
	return string(bts), reqInf, err
}
