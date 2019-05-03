package tc

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"strings"
)

type ATSConfigMetaData struct {
	Info        ATSConfigMetaDataInfo         `json:"info"`
	ConfigFiles []ATSConfigMetaDataConfigFile `json:"configFiles"`
}

type ATSConfigMetaDataInfo struct {
	CDNID             int    `json:"cdnId"`
	CDNName           string `json:"cdnName"`
	ServerID          int    `json:"serverId"`
	ServerIPv4        string `json:"serverIpv4"`
	ServerName        string `json:"serverName"`
	ServerPort        int    `json:"serverTcpPort"`
	ProfileID         int    `json:"profileId"`
	ProfileName       string `json:"profileName"`
	TOReverseProxyURL string `json:"toRevProxyUrl"`
	TOURL             string `json:"toUrl"`
}

type ATSConfigMetaDataConfigFileScope string

const ATSConfigMetaDataConfigFileScopeProfiles = ATSConfigMetaDataConfigFileScope("profiles")
const ATSConfigMetaDataConfigFileScopeServers = ATSConfigMetaDataConfigFileScope("servers")
const ATSConfigMetaDataConfigFileScopeCDNs = ATSConfigMetaDataConfigFileScope("cdns")
const ATSConfigMetaDataConfigFileScopeInvalid = ATSConfigMetaDataConfigFileScope("")

type ATSConfigMetaDataConfigFile struct {
	FileNameOnDisk string `json:"fnameOnDisk"`
	Location       string `json:"location"`
	APIURI         string `json:"apiUri, omitempty"`
	URL            string `json:"url, omitempty"`
	Scope          string `json:"scope"`
}

func (t ATSConfigMetaDataConfigFileScope) String() string {
	switch t {
	case ATSConfigMetaDataConfigFileScopeProfiles:
		fallthrough
	case ATSConfigMetaDataConfigFileScopeServers:
		fallthrough
	case ATSConfigMetaDataConfigFileScopeCDNs:
		return string(t)
	default:
		return "invalid"
	}
}

func ATSConfigMetaDataConfigFileScopeFromString(s string) ATSConfigMetaDataConfigFileScope {
	s = strings.ToLower(s)
	switch s {
	case "profiles":
		return ATSConfigMetaDataConfigFileScopeProfiles
	case "servers":
		return ATSConfigMetaDataConfigFileScopeServers
	case "cdns":
		return ATSConfigMetaDataConfigFileScopeCDNs
	default:
		return ATSConfigMetaDataConfigFileScopeInvalid
	}
}
