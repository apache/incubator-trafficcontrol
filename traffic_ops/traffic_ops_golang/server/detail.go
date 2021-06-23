package server

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
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"

	"github.com/lib/pq"
)

func GetDetailHandler(w http.ResponseWriter, r *http.Request) {
	alt := "GET servers/details with query parameters hostName"
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, &alt)
		return
	}
	defer inf.Close()

	servers, err := getDetailServers(inf.Tx.Tx, inf.User, inf.Params["hostName"], -1, "", 0, inf.Version)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting detail servers: "+err.Error()), &alt)
		return
	}
	if len(servers) == 0 {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil, &alt)
		return
	}
	server := servers[0]
	if inf.Version.Major <= 2 {
		interfaces := server.ServerInterfaces
		routerHostName := ""
		routerPortName := ""
		// All interfaces should have the same router name/port when they were upgraded from v1/2/3 to v4, so we can just choose any of them
		if len(interfaces) != 0 {
			routerHostName = interfaces[0].RouterHostName
			routerPortName = interfaces[0].RouterPortName
		}
		legacyInterface, err := tc.V4InterfaceInfoToLegacyInterfaces(interfaces)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("converting to server detail v11: "+err.Error()))
			return
		}
		v11server := tc.ServerDetailV11{}
		v11server.ServerDetail = server.ServerDetail
		v11server.RouterHostName = &routerHostName
		v11server.RouterPortName = &routerPortName
		v11server.LegacyInterfaceDetails = legacyInterface
		server := v11server
		alerts := api.CreateDeprecationAlerts(&alt)
		api.WriteAlertsObj(w, r, http.StatusOK, alerts, server)
		return
	} else if inf.Version.Major <= 3 {
		interfaces := server.ServerInterfaces
		routerHostName := ""
		routerPortName := ""
		// All interfaces should have the same router name/port when they were upgraded from v1/2/3 to v4, so we can just choose any of them
		if len(interfaces) != 0 {
			routerHostName = interfaces[0].RouterHostName
			routerPortName = interfaces[0].RouterPortName
		}
		v3server := tc.ServerDetailV30{}
		v3server.ServerDetail = server.ServerDetail
		v3server.RouterHostName = &routerHostName
		v3server.RouterPortName = &routerPortName
		v3Interfaces, err := tc.V4InterfaceInfoToV3Interfaces(interfaces)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("converting to server detail v3: "+err.Error()))
			return
		}
		v3server.ServerInterfaces = &v3Interfaces
		server := v3server
		alerts := api.CreateDeprecationAlerts(&alt)
		api.WriteAlertsObj(w, r, http.StatusOK, alerts, server)
		return
	}
	alerts := api.CreateDeprecationAlerts(&alt)
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, server)
}

func GetDetailParamHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	hostName := inf.Params["hostName"]
	physLocationIDStr := inf.Params["physLocationID"]
	physLocationID := -1
	if physLocationIDStr != "" {
		err := error(nil)
		physLocationID, err = strconv.Atoi(physLocationIDStr)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("physLocationID parameter is not an integer"), nil)
			return
		}
	}
	if hostName == "" && physLocationIDStr == "" {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("Missing required fields: 'hostname' or 'physLocationID'"), nil)
		return
	}
	orderBy := "hostName"
	if _, ok := inf.Params["orderby"]; ok {
		orderBy = inf.Params["orderby"]
	}
	limit := 1000
	if limitStr, ok := inf.Params["limit"]; ok {
		err := error(nil)
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("limit parameter is not an integer"), nil)
			return
		}
	}
	servers, err := getDetailServers(inf.Tx.Tx, inf.User, hostName, physLocationID, util.CamelToSnakeCase(orderBy), limit, inf.Version)
	respVals := map[string]interface{}{
		"orderby": orderBy,
		"limit":   limit,
		"size":    len(servers),
	}

	if inf.Version.Major <= 2 {
		v11ServerList := []tc.ServerDetailV11{}
		for _, server := range servers {
			interfaces := server.ServerInterfaces
			routerHostName := ""
			routerPortName := ""
			// All interfaces should have the same router name/port when they were upgraded from v1/2/3 to v4, so we can just choose any of them
			if len(interfaces) != 0 {
				routerHostName = interfaces[0].RouterHostName
				routerPortName = interfaces[0].RouterPortName
			}
			v11server := tc.ServerDetailV11{}
			v11server.ServerDetail = server.ServerDetail
			v11server.RouterHostName = &routerHostName
			v11server.RouterPortName = &routerPortName
			legacyInterface, err := tc.V4InterfaceInfoToLegacyInterfaces(interfaces)
			if err != nil {
				api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("converting to server detail v11: "+err.Error()))
				return
			}
			v11server.LegacyInterfaceDetails = legacyInterface

			v11ServerList = append(v11ServerList, v11server)
		}
		api.RespWriterVals(w, r, inf.Tx.Tx, respVals)(v11ServerList, err)
		return
	} else if inf.Version.Major <= 3 {
		v3ServerList := []tc.ServerDetailV30{}
		for _, server := range servers {
			v3Server := tc.ServerDetailV30{}
			interfaces := server.ServerInterfaces
			routerHostName := ""
			routerPortName := ""
			// All interfaces should have the same router name/port when they were upgraded from v1/2/3 to v4, so we can just choose any of them
			if len(interfaces) != 0 {
				routerHostName = interfaces[0].RouterHostName
				routerPortName = interfaces[0].RouterPortName
			}
			v3Server.ServerDetail = server.ServerDetail
			v3Server.RouterHostName = &routerHostName
			v3Server.RouterPortName = &routerPortName
			v3Interfaces, err := tc.V4InterfaceInfoToV3Interfaces(interfaces)
			if err != nil {
				api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("converting to server detail v3: "+err.Error()))
				return
			}
			v3Server.ServerInterfaces = &v3Interfaces
			v3ServerList = append(v3ServerList, v3Server)
		}
		api.RespWriterVals(w, r, inf.Tx.Tx, respVals)(v3ServerList, err)
		return
	}
	api.RespWriterVals(w, r, inf.Tx.Tx, respVals)(servers, err)
}

func AddWhereClauseAndQuery(tx *sql.Tx, q string, hostName string, physLocationID int, orderByStr string, limitStr string) (*sql.Rows, error) {
	if hostName != "" && physLocationID != -1 {
		q += ` WHERE server.host_name = $1::text AND server.phys_location = $2::bigint` + orderByStr + limitStr
		return tx.Query(q, hostName, physLocationID)
	} else if hostName != "" {
		q += ` WHERE server.host_name = $1::text` + orderByStr + limitStr
		return tx.Query(q, hostName)
	} else if physLocationID != -1 {
		q += ` WHERE server.phys_location = $1::int` + orderByStr + limitStr
		return tx.Query(q, physLocationID)
	} else {
		q += orderByStr + limitStr
		return tx.Query(q) // Should never happen for API <1.3, which don't allow querying without hostName or physLocation
	}
}

func getDetailServers(tx *sql.Tx, user *auth.CurrentUser, hostName string, physLocationID int, orderBy string, limit int, reqVersion api.Version) ([]tc.ServerDetailV40, error) {
	allowedOrderByCols := map[string]string{
		"":                "",
		"cachegroup":      "server.cachegroup",
		"cdn_name":        "cdn.name",
		"domain_name":     "server.domain_name",
		"guid":            "server.guid",
		"host_name":       "server.host_name",
		"https_port":      "server.https_port",
		"id":              "server.id",
		"ilo_ip_address":  "server.ilo_ip_address",
		"ilo_ip_gateway":  "server.ilo_ip_gateway",
		"ilo_ip_netmask":  "server.ilo_ip_netmask",
		"ilo_password":    "server.ilo_password",
		"ilo_username":    "server.ilo_username",
		"mgmt_ip_address": "server.mgmt_ip_address",
		"mgmt_ip_gateway": "server.mgmt_ip_gateway",
		"mgmt_ip_netmask": "server.mgmt_ip_netmask",
		"offline_reason":  "server.offline_reason",
		"phys_location":   "pl.name",
		"profile":         "p.name",
		"profile_desc":    "p.description",
		"rack":            "server.rack",
		"status":          "st.name",
		"tcp_port":        "server.tcp_port",
		"server_type":     "t.name",
		"xmpp_id":         "server.xmpp_id",
		"xmpp_passwd":     "server.xmpp_passwd",
	}
	orderBy, ok := allowedOrderByCols[orderBy]
	if !ok {
		return nil, errors.New("orderBy '" + orderBy + "' not permitted")
	}

	dataFetchQuery := `,
cg.name AS cachegroup,
cdn.name AS cdn_name,
ARRAY(select deliveryservice from deliveryservice_server where server = server.id),
server.domain_name,
server.guid,
server.host_name,
server.https_port,
server.ilo_ip_address,
server.ilo_ip_gateway,
server.ilo_ip_netmask,
server.ilo_password,
server.ilo_username,
(SELECT address FROM ip_address WHERE service_address = true AND family(address) = 4 AND server = server.id) AS service_ip,
(SELECT address FROM ip_address WHERE service_address = true AND family(address) = 6 AND server = server.id) AS service_ip6,
(SELECT gateway FROM ip_address WHERE service_address = true AND family(address) = 4 AND server = server.id) AS service_gateway,
(SELECT gateway FROM ip_address WHERE service_address = true AND family(address) = 6 AND server = server.id) AS service_gateway6,
(SELECT host(netmask(ip_address.address)) FROM ip_address WHERE service_address = true AND family(address) = 4 AND server = server.id) AS service_netmask,
(SELECT interface FROM ip_address WHERE service_address = true AND family(address) = 4 AND server = server.id) AS interface_name,
(SELECT mtu FROM interface WHERE server.id = interface.server AND interface.name = (SELECT interface FROM ip_address WHERE service_address = true AND family(address) = 4 AND server = server.id)) AS interface_mtu,
server.mgmt_ip_address,
server.mgmt_ip_gateway,
server.mgmt_ip_netmask,
server.offline_reason,
pl.name as phys_location,
p.name as profile,
p.description as profile_desc,
server.rack,
st.name as status,
server.tcp_port,
t.name as server_type,
server.xmpp_id,
server.xmpp_passwd
`
	queryFormatString := `
SELECT
	server.id
	%v
FROM server
JOIN cachegroup cg ON server.cachegroup = cg.id
JOIN cdn ON server.cdn_id = cdn.id
JOIN phys_location pl ON server.phys_location = pl.id
JOIN profile p ON server.profile = p.id
JOIN status st ON server.status = st.id
JOIN type t ON server.type = t.id
`
	limitStr := ""
	if limit != 0 {
		limitStr = " LIMIT " + strconv.Itoa(limit)
	}
	orderByStr := ""
	if orderBy != "" {
		orderByStr = " ORDER BY " + orderBy
	}
	idRows, err := AddWhereClauseAndQuery(tx, fmt.Sprintf(queryFormatString, ""), hostName, physLocationID, orderByStr, limitStr)
	if err != nil {
		return nil, errors.New("querying delivery service eligible servers: " + err.Error())
	}
	defer idRows.Close()
	var serverIDs []int
	for idRows.Next() {
		var serverID *int
		err := idRows.Scan(&serverID)
		if err != nil {
			return nil, errors.New("querying delivery service eligible server ids: " + err.Error())
		}
		serverIDs = append(serverIDs, *serverID)
	}
	serversMap, err := dbhelpers.GetServersInterfaces(serverIDs, tx)
	if err != nil {
		return nil, errors.New("unable to get server interfaces: " + err.Error())
	}
	rows, err := AddWhereClauseAndQuery(tx, fmt.Sprintf(queryFormatString, dataFetchQuery), hostName, physLocationID, orderByStr, limitStr)
	if err != nil {
		return nil, errors.New("Error querying detail servers: " + err.Error())
	}

	defer rows.Close()
	sIDs := []int{}
	servers := []tc.ServerDetailV40{}

	serviceAddress := util.StrPtr("")
	service6Address := util.StrPtr("")
	serviceGateway := util.StrPtr("")
	service6Gateway := util.StrPtr("")
	serviceNetmask := util.StrPtr("")
	serviceInterface := util.StrPtr("")
	serviceMtu := util.StrPtr("")

	for rows.Next() {
		s := tc.ServerDetailV40{}
		if err := rows.Scan(&s.ID, &s.CacheGroup, &s.CDNName, pq.Array(&s.DeliveryServiceIDs), &s.DomainName, &s.GUID, &s.HostName, &s.HTTPSPort, &s.ILOIPAddress, &s.ILOIPGateway, &s.ILOIPNetmask, &s.ILOPassword, &s.ILOUsername, &serviceAddress, &service6Address, &serviceGateway, &service6Gateway, &serviceNetmask, &serviceInterface, &serviceMtu, &s.MgmtIPAddress, &s.MgmtIPGateway, &s.MgmtIPNetmask, &s.OfflineReason, &s.PhysLocation, &s.Profile, &s.ProfileDesc, &s.Rack, &s.Status, &s.TCPPort, &s.Type, &s.XMPPID, &s.XMPPPasswd); err != nil {
			return nil, errors.New("Error scanning detail server: " + err.Error())
		}
		s.ServerInterfaces = []tc.ServerInterfaceInfoV40{}
		if interfacesMap, ok := serversMap[*s.ID]; ok {
			for _, interfaceInfo := range interfacesMap {
				s.ServerInterfaces = append(s.ServerInterfaces, interfaceInfo)
			}
		}

		hiddenField := "********"
		if user.PrivLevel < auth.PrivLevelOperations {
			s.ILOPassword = &hiddenField
			s.XMPPPasswd = &hiddenField
		}

		servers = append(servers, s)
		sIDs = append(sIDs, *s.ID)
	}

	rows, err = tx.Query(`SELECT serverid, description, val from hwinfo where serverid = ANY($1);`, pq.Array(sIDs))
	if err != nil {
		return nil, errors.New("Error querying detail servers hardware info: " + err.Error())
	}
	defer rows.Close()
	hwInfos := map[int]map[string]string{}
	for rows.Next() {
		serverID := 0
		desc := ""
		val := ""
		if err := rows.Scan(&serverID, &desc, &val); err != nil {
			return nil, errors.New("Error scanning detail server hardware info: " + err.Error())
		}

		hwInfo, ok := hwInfos[serverID]
		if !ok {
			hwInfo = map[string]string{}
		}
		hwInfo[desc] = val
		hwInfos[serverID] = hwInfo
	}
	for i, server := range servers {
		hw, ok := hwInfos[*server.ID]
		if !ok {
			continue
		}
		server.HardwareInfo = hw
		servers[i] = server
	}
	return servers, nil
}
