package main

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
import "encoding/json"
import "net/http"
import "strconv"
import "time"

import "github.com/apache/trafficcontrol/lib/go-tc"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

// For "ILO*", "Mgmt*", IPv6 stuff and other things that need to be empty strings because the database doesn't model them as NULL
var EMPTY_STRING = ""

// These can be constant for all servers
var GATEWAY = "192.0.2.0"
var NETMASK = "255.255.0.0"

var TO_SERVER_DOMAIN_NAME = "cdn.test"
var TO_SERVER_HOSTNAME = "trafficops"
var TO_SERVER_HTTPS_PORT = 443
var TO_SERVER_ID = 1
var TO_SERVER_INTERFACE_NAME = "eth0"
var TO_SERVER_INTERFACE_MTU = 1500
var TO_SERVER_IP = "192.0.2.1"
var TO_SERVER_REVAL_PENDING = false
var TO_SERVER_UPD_PENDING = false
var TO_SERVER_TCP_PORT = 80
var TO_SERVER_XMPPID = "trafficops"

var TODB_SERVER_DOMAIN_NAME = "cdn.test"
var TODB_SERVER_HOSTNAME = "todb"
var TODB_SERVER_HTTPS_PORT = 5432
var TODB_SERVER_ID = 2
var TODB_SERVER_INTERFACE_NAME = "eth0"
var TODB_SERVER_INTERFACE_MTU = 1500
var TODB_SERVER_IP = "192.0.2.2"
var TODB_SERVER_REVAL_PENDING = false
var TODB_SERVER_UPD_PENDING = false
var TODB_SERVER_TCP_PORT = 5432
var TODB_SERVER_XMPPID = "todb"

var EDGE_SERVER_DOMAIN_NAME = "cdn.test"
var EDGE_SERVER_HOSTNAME = "edge"
var EDGE_SERVER_HTTPS_PORT = 443
var EDGE_SERVER_ID = 3
var EDGE_SERVER_INTERFACE_NAME = "eth0"
var EDGE_SERVER_INTERFACE_MTU = 1500
var EDGE_SERVER_IP = "192.0.2.3"
var EDGE_SERVER_REVAL_PENDING = true
var EDGE_SERVER_UPD_PENDING = true
var EDGE_SERVER_TCP_PORT = 80
var EDGE_SERVER_XMPPID = "edge"

var MID_SERVER_DOMAIN_NAME = "cdn.test"
var MID_SERVER_HOSTNAME = "mid"
var MID_SERVER_HTTPS_PORT = 443
var MID_SERVER_ID = 4
var MID_SERVER_INTERFACE_NAME = "eth0"
var MID_SERVER_INTERFACE_MTU = 1500
var MID_SERVER_IP = "192.0.2.4"
var MID_SERVER_REVAL_PENDING = true
var MID_SERVER_UPD_PENDING = true
var MID_SERVER_TCP_PORT = 80
var MID_SERVER_XMPPID = "mid"

var SERVERS = []tc.ServerNullable{
	tc.ServerNullable{
		Cachegroup:       &EDGE_CACHEGROUP,
		CachegroupID:     &EDGE_CACHEGROUP_ID,
		CDNID:            &ALL_CDN_ID,
		CDNName:          &ALL_CDN,
		DeliveryServices: nil,
		DomainName:       &TO_SERVER_DOMAIN_NAME,
		FQDN:             nil,
		FqdnTime:         time.Time{},
		GUID:             nil, // it's because of this little bastard that I have to use "Nullable" - the real API output `null`, so it's gotta be a pointer
		HostName:         &TO_SERVER_HOSTNAME,
		HTTPSPort:        &TO_SERVER_HTTPS_PORT,
		ID:               &TO_SERVER_ID,
		ILOIPAddress:     &EMPTY_STRING,
		ILOIPGateway:     &EMPTY_STRING,
		ILOIPNetmask:     &EMPTY_STRING,
		ILOPassword:      &EMPTY_STRING,
		ILOUsername:      &EMPTY_STRING,
		InterfaceMtu:     &TO_SERVER_INTERFACE_MTU,
		InterfaceName:    &TO_SERVER_INTERFACE_NAME,
		IP6Address:       nil,
		IP6Gateway:       &EMPTY_STRING,
		IPAddress:        &TO_SERVER_IP,
		IPGateway:        &GATEWAY,
		IPNetmask:        &NETMASK,
		LastUpdated:      CURRENT_TIME,
		MgmtIPAddress:    &EMPTY_STRING,
		MgmtIPGateway:    &EMPTY_STRING,
		MgmtIPNetmask:    &EMPTY_STRING,
		OfflineReason:    &EMPTY_STRING,
		PhysLocation:     &(LOCATION.Name),
		PhysLocationID:   &(LOCATION.ID),
		Profile:          &TO_PROFILE_NAME,
		ProfileDesc:      &TO_PROFILE_DESCRIPTION,
		ProfileID:        &TO_PROFILE_ID,
		Rack:             &EMPTY_STRING,
		RevalPending:     &TO_SERVER_REVAL_PENDING,
		RouterHostName:   &EMPTY_STRING,
		RouterPortName:   &EMPTY_STRING,
		Status:           &(STATUS_ONLINE.Name),
		StatusID:         &(STATUS_ONLINE.ID),
		TCPPort:          &TO_SERVER_TCP_PORT,
		Type:             TYPE_TRAFFIC_OPS.Name,
		TypeID:           &(TYPE_TRAFFIC_OPS.ID),
		UpdPending:       &TO_SERVER_UPD_PENDING,
		XMPPID:           &TO_SERVER_XMPPID,
		XMPPPasswd:       &EMPTY_STRING,
	},
	tc.ServerNullable{
		Cachegroup:       &EDGE_CACHEGROUP,
		CachegroupID:     &EDGE_CACHEGROUP_ID,
		CDNID:            &ALL_CDN_ID,
		CDNName:          &ALL_CDN,
		DeliveryServices: nil,
		DomainName:       &TODB_SERVER_DOMAIN_NAME,
		FQDN:             nil,
		FqdnTime:         time.Time{},
		GUID:             nil, // it's because of this little bastard that I have to use "Nullable" - the real API output `null`, so it's gotta be a pointer
		HostName:         &TODB_SERVER_HOSTNAME,
		HTTPSPort:        &TODB_SERVER_HTTPS_PORT,
		ID:               &TODB_SERVER_ID,
		ILOIPAddress:     &EMPTY_STRING,
		ILOIPGateway:     &EMPTY_STRING,
		ILOIPNetmask:     &EMPTY_STRING,
		ILOPassword:      &EMPTY_STRING,
		ILOUsername:      &EMPTY_STRING,
		InterfaceMtu:     &TODB_SERVER_INTERFACE_MTU,
		InterfaceName:    &TODB_SERVER_INTERFACE_NAME,
		IP6Address:       nil,
		IP6Gateway:       &EMPTY_STRING,
		IPAddress:        &TODB_SERVER_IP,
		IPGateway:        &GATEWAY,
		IPNetmask:        &NETMASK,
		LastUpdated:      CURRENT_TIME,
		MgmtIPAddress:    &EMPTY_STRING,
		MgmtIPGateway:    &EMPTY_STRING,
		MgmtIPNetmask:    &EMPTY_STRING,
		OfflineReason:    &EMPTY_STRING,
		PhysLocation:     &(LOCATION.Name),
		PhysLocationID:   &(LOCATION.ID),
		Profile:          &TO_DB_PROFILE_NAME,
		ProfileDesc:      &TO_DB_PROFILE_DESCRIPTION,
		ProfileID:        &TO_DB_PROFILE_ID,
		Rack:             &EMPTY_STRING,
		RevalPending:     &TODB_SERVER_REVAL_PENDING,
		RouterHostName:   &EMPTY_STRING,
		RouterPortName:   &EMPTY_STRING,
		Status:           &(STATUS_ONLINE.Name),
		StatusID:         &(STATUS_ONLINE.ID),
		TCPPort:          &TODB_SERVER_TCP_PORT,
		Type:             TYPE_TRAFFIC_OPS_DB.Name,
		TypeID:           &(TYPE_TRAFFIC_OPS_DB.ID),
		UpdPending:       &TODB_SERVER_UPD_PENDING,
		XMPPID:           &TODB_SERVER_XMPPID,
		XMPPPasswd:       &EMPTY_STRING,
	},
	tc.ServerNullable{
		Cachegroup:       &EDGE_CACHEGROUP,
		CachegroupID:     &EDGE_CACHEGROUP_ID,
		CDNID:            &CDN_ID,
		CDNName:          &CDN,
		DeliveryServices: nil,
		DomainName:       &EDGE_SERVER_DOMAIN_NAME,
		FQDN:             nil,
		FqdnTime:         time.Time{},
		GUID:             nil, // it's because of this little bastard that I have to use "Nullable" - the real API output `null`, so it's gotta be a pointer
		HostName:         &EDGE_SERVER_HOSTNAME,
		HTTPSPort:        &EDGE_SERVER_HTTPS_PORT,
		ID:               &EDGE_SERVER_ID,
		ILOIPAddress:     &EMPTY_STRING,
		ILOIPGateway:     &EMPTY_STRING,
		ILOIPNetmask:     &EMPTY_STRING,
		ILOPassword:      &EMPTY_STRING,
		ILOUsername:      &EMPTY_STRING,
		InterfaceMtu:     &EDGE_SERVER_INTERFACE_MTU,
		InterfaceName:    &EDGE_SERVER_INTERFACE_NAME,
		IP6Address:       nil,
		IP6Gateway:       &EMPTY_STRING,
		IPAddress:        &EDGE_SERVER_IP,
		IPGateway:        &GATEWAY,
		IPNetmask:        &NETMASK,
		LastUpdated:      CURRENT_TIME,
		MgmtIPAddress:    &EMPTY_STRING,
		MgmtIPGateway:    &EMPTY_STRING,
		MgmtIPNetmask:    &EMPTY_STRING,
		OfflineReason:    &EMPTY_STRING,
		PhysLocation:     &(LOCATION.Name),
		PhysLocationID:   &(LOCATION.ID),
		Profile:          &EDGE_PROFILE_NAME,
		ProfileDesc:      &EDGE_PROFILE_DESCRIPTION,
		ProfileID:        &EDGE_PROFILE_ID,
		Rack:             &EMPTY_STRING,
		RevalPending:     &EDGE_SERVER_REVAL_PENDING,
		RouterHostName:   &EMPTY_STRING,
		RouterPortName:   &EMPTY_STRING,
		Status:           &(STATUS_REPORTED.Name),
		StatusID:         &(STATUS_REPORTED.ID),
		TCPPort:          &EDGE_SERVER_TCP_PORT,
		Type:             TYPE_EDGE.Name,
		TypeID:           &(TYPE_EDGE.ID),
		UpdPending:       &EDGE_SERVER_UPD_PENDING,
		XMPPID:           &EDGE_SERVER_XMPPID,
		XMPPPasswd:       &EMPTY_STRING,
	},
	tc.ServerNullable{
		Cachegroup:       &MID_CACHEGROUP,
		CachegroupID:     &MID_CACHEGROUP_ID,
		CDNID:            &CDN_ID,
		CDNName:          &CDN,
		DeliveryServices: nil,
		DomainName:       &MID_SERVER_DOMAIN_NAME,
		FQDN:             nil,
		FqdnTime:         time.Time{},
		GUID:             nil, // it's because of this little bastard that I have to use "Nullable" - the real API output `null`, so it's gotta be a pointer
		HostName:         &MID_SERVER_HOSTNAME,
		HTTPSPort:        &MID_SERVER_HTTPS_PORT,
		ID:               &MID_SERVER_ID,
		ILOIPAddress:     &EMPTY_STRING,
		ILOIPGateway:     &EMPTY_STRING,
		ILOIPNetmask:     &EMPTY_STRING,
		ILOPassword:      &EMPTY_STRING,
		ILOUsername:      &EMPTY_STRING,
		InterfaceMtu:     &MID_SERVER_INTERFACE_MTU,
		InterfaceName:    &MID_SERVER_INTERFACE_NAME,
		IP6Address:       nil,
		IP6Gateway:       &EMPTY_STRING,
		IPAddress:        &MID_SERVER_IP,
		IPGateway:        &GATEWAY,
		IPNetmask:        &NETMASK,
		LastUpdated:      CURRENT_TIME,
		MgmtIPAddress:    &EMPTY_STRING,
		MgmtIPGateway:    &EMPTY_STRING,
		MgmtIPNetmask:    &EMPTY_STRING,
		OfflineReason:    &EMPTY_STRING,
		PhysLocation:     &(LOCATION.Name),
		PhysLocationID:   &(LOCATION.ID),
		Profile:          &MID_PROFILE_NAME,
		ProfileDesc:      &MID_PROFILE_DESCRIPTION,
		ProfileID:        &MID_PROFILE_ID,
		Rack:             &EMPTY_STRING,
		RevalPending:     &MID_SERVER_REVAL_PENDING,
		RouterHostName:   &EMPTY_STRING,
		RouterPortName:   &EMPTY_STRING,
		Status:           &(STATUS_REPORTED.Name),
		StatusID:         &(STATUS_REPORTED.ID),
		TCPPort:          &MID_SERVER_TCP_PORT,
		Type:             TYPE_MID.Name,
		TypeID:           &(TYPE_MID.ID),
		UpdPending:       &MID_SERVER_UPD_PENDING,
		XMPPID:           &MID_SERVER_XMPPID,
		XMPPPasswd:       &EMPTY_STRING,
	},
}

func servers(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, SERVERS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}

/**
 * There's no modeling for any part of the config files routes in Go, so we are truly on our own now.
 */
type ConfigFileInfo struct {
	CDNID         int    `json:"cdnId"`
	CDNName       string `json:"cdnName"`
	ProfileID     int    `json:"profileId"`
	ProfileName   string `json:"profileName"`
	ServerID      int    `json:"serverId"`
	ServerIPv4    string `json:"serverIpv4"`
	ServerName    string `json:"serverName"`
	ServerTCPPort int    `json:"serverTcpPort"`
	TOURL         string `json:"toUrl"`
}

type ConfigFile struct {
	FileName string `json:"fnameOnDisk"`
	Location string `json:"location"`
	APIURI   string `json:"apiUri"`
	Scope    string `json:"scope"`
}

type ConfigFilesAPIResponse struct {
	Info  ConfigFileInfo `json:"info"`
	Files []ConfigFile   `json:"configFiles"`
}

var EDGE_CONFIG_FILE_INFO = ConfigFileInfo{
	CDNID:         CDN_ID,
	CDNName:       CDN,
	ProfileID:     EDGE_PROFILE_ID,
	ProfileName:   EDGE_PROFILE_NAME,
	ServerID:      EDGE_SERVER_ID,
	ServerIPv4:    EDGE_SERVER_IP,
	ServerName:    EDGE_SERVER_HOSTNAME,
	ServerTCPPort: EDGE_SERVER_TCP_PORT,
	TOURL:         "https://localhost:443/",
}

var MID_CONFIG_FILE_INFO = ConfigFileInfo{
	CDNID:         CDN_ID,
	CDNName:       CDN,
	ProfileID:     MID_PROFILE_ID,
	ProfileName:   MID_PROFILE_NAME,
	ServerID:      MID_SERVER_ID,
	ServerIPv4:    MID_SERVER_IP,
	ServerName:    MID_SERVER_HOSTNAME,
	ServerTCPPort: MID_SERVER_TCP_PORT,
	TOURL:         "https://localhost:443/",
}

var TO_CONFIG_FILE_INFO = ConfigFileInfo{
	CDNID:         ALL_CDN_ID,
	CDNName:       ALL_CDN,
	ProfileID:     TO_PROFILE_ID,
	ProfileName:   TO_PROFILE_NAME,
	ServerID:      TO_SERVER_ID,
	ServerIPv4:    TO_SERVER_IP,
	ServerName:    TO_SERVER_HOSTNAME,
	ServerTCPPort: TO_SERVER_TCP_PORT,
	TOURL:         "https://localhost:443/",
}

var TODB_CONFIG_FILE_INFO = ConfigFileInfo{
	CDNID:         ALL_CDN_ID,
	CDNName:       ALL_CDN,
	ProfileID:     TO_DB_PROFILE_ID,
	ProfileName:   TO_DB_PROFILE_NAME,
	ServerID:      TODB_SERVER_ID,
	ServerIPv4:    TODB_SERVER_IP,
	ServerName:    TODB_SERVER_HOSTNAME,
	ServerTCPPort: TODB_SERVER_TCP_PORT,
	TOURL:         "https://localhost:443/",
}

var EDGE_CONFIG_FILES = []ConfigFile{
	ConfigFile{
		FileName: "astats.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/astats.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "cache.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/cache.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "hosting.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + EDGE_SERVER_HOSTNAME + "/configfiles/ats/hosting.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "ip_allow.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/servers/" + EDGE_SERVER_HOSTNAME + "/configfiles/ats/ip_allow.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "parent.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + EDGE_SERVER_HOSTNAME + "/configfiles/ats/parent.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "plugin.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/plugin.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "records.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/records.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "regex_revalidate.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/regex_revalidate.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "remap.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + EDGE_SERVER_HOSTNAME + "/configfiles/ats/remap.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "set_dscp_0.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_0.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_10.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_10.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_12.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_12.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_14.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_14.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_16.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_16.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_18.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_18.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_20.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_20.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_22.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_22.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_24.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_24.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_26.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_26.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_28.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_28.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_30.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_30.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_32.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_32.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_34.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_34.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_36.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_36.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_37.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_37.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_38.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_38.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_40.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_40.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_48.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_48.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_56.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_56.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_8.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_8.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "ssl_multicert.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/ssl_multicert.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "storage.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/storage.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "volume.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + EDGE_PROFILE_NAME + "/configfiles/ats/volume.config",
		Scope:    "profiles",
	},
}

var MID_CONFIG_FILES = []ConfigFile{
	ConfigFile{
		FileName: "astats.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/profiles/" + MID_PROFILE_NAME + "/configfiles/ats/astats.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "cache.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + MID_SERVER_HOSTNAME + "/configfiles/ats/cache.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "hosting.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + MID_SERVER_HOSTNAME + "/configfiles/ats/hosting.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "ip_allow.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/servers/" + MID_SERVER_HOSTNAME + "/configfiles/ats/ip_allow.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "parent.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + MID_SERVER_HOSTNAME + "/configfiles/ats/parent.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "plugin.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + MID_PROFILE_NAME + "/configfiles/ats/plugin.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "records.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + MID_PROFILE_NAME + "/configfiles/ats/records.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "regex_revalidate.config",
		Location: "/etc/trafficserver",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/regex_revalidate.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "remap.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/servers/" + MID_SERVER_HOSTNAME + "/configfiles/ats/remap.config",
		Scope:    "servers",
	},
	ConfigFile{
		FileName: "set_dscp_0.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_0.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_10.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_10.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_12.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_12.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_14.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_14.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_16.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_16.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_18.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_18.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_20.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_20.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_22.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_22.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_24.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_24.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_26.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_26.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_28.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_28.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_30.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_30.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_32.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_32.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_34.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_34.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_36.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_36.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_37.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_37.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_38.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_38.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_40.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_40.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_48.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_48.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_56.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_56.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "set_dscp_8.config",
		Location: "/etc/trafficserver/dscp",
		APIURI:   "/api/1.2/cdns/" + CDN + "/configfiles/ats/set_dscp_8.config",
		Scope:    "cdns",
	},
	ConfigFile{
		FileName: "storage.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + MID_PROFILE_NAME + "/configfiles/ats/storage.config",
		Scope:    "profiles",
	},
	ConfigFile{
		FileName: "volume.config",
		Location: "/etc/trafficserver/",
		APIURI:   "/api/1.2/profiles/" + MID_PROFILE_NAME + "/configfiles/ats/volume.config",
		Scope:    "profiles",
	},
}

func edgeConfigFiles(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error
	switch r.Method {
	case http.MethodGet:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{EDGE_CONFIG_FILE_INFO, EDGE_CONFIG_FILES}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Write(payload)
		}
	case http.MethodHead:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{EDGE_CONFIG_FILE_INFO, EDGE_CONFIG_FILES}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		}
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
		w.WriteHeader(http.StatusNoContent)
	default:
		common(w)
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func midConfigFiles(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error
	switch r.Method {
	case http.MethodGet:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{MID_CONFIG_FILE_INFO, MID_CONFIG_FILES}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Write(payload)
		}
	case http.MethodHead:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{MID_CONFIG_FILE_INFO, MID_CONFIG_FILES}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		}
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
		w.WriteHeader(http.StatusNoContent)
	default:
		common(w)
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func toConfigFiles(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error
	switch r.Method {
	case http.MethodGet:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{TO_CONFIG_FILE_INFO, []ConfigFile{}}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Write(payload)
		}
	case http.MethodHead:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{TO_CONFIG_FILE_INFO, []ConfigFile{}}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		}
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
		w.WriteHeader(http.StatusNoContent)
	default:
		common(w)
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.WriteHeader(http.StatusMethodNotAllowed)

	}
}

func toDbConfigFiles(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error
	switch r.Method {
	case http.MethodGet:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{TODB_CONFIG_FILE_INFO, []ConfigFile{}}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Write(payload)
		}
	case http.MethodHead:
		common(w)
		if payload, err = json.Marshal(ConfigFilesAPIResponse{TODB_CONFIG_FILE_INFO, []ConfigFile{}}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"alerts":[{"level":"error","text":"` + err.Error() + `"}]}`))
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		}
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
		w.WriteHeader(http.StatusNoContent)
	default:
		common(w)
		w.Header().Set("Allow", http.MethodGet+","+http.MethodHead+","+http.MethodOptions)
		w.WriteHeader(http.StatusMethodNotAllowed)

	}
}
