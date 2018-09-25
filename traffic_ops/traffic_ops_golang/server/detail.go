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
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"

	"github.com/lib/pq"
)

func GetDetailHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	servers, err := getDetailServers(inf.Tx.Tx, inf.User, inf.Params["hostName"], -1, "", 0)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting detail servers: "+err.Error()))
		return
	}
	if len(servers) == 0 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}
	server := servers[0]
	api.WriteResp(w, r, server)
}

func GetDetailParamHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
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
	servers, err := getDetailServers(inf.Tx.Tx, inf.User, hostName, physLocationID, util.CamelToSnakeCase(orderBy), limit)
	respVals := map[string]interface{}{
		"orderby": orderBy,
		"limit":   limit,
		"size":    len(servers),
	}
	api.RespWriterVals(w, r, inf.Tx.Tx, respVals)(servers, err)
}

func getDetailServers(tx *sql.Tx, user *auth.CurrentUser, hostName string, physLocationID int, orderBy string, limit int) ([]tc.ServerDetail, error) {
	allowedOrderByCols := map[string]string{
		"":                 "",
		"cachegroup":       "s.cachegroup",
		"cdn_name":         "cdn.name",
		"domain_name":      "s.domain_name",
		"guid":             "s.guid",
		"host_name":        "s.host_name",
		"https_port":       "s.https_port",
		"id":               "s.id",
		"ilo_ip_address":   "s.ilo_ip_address",
		"ilo_ip_gateway":   "s.ilo_ip_gateway",
		"ilo_ip_netmask":   "s.ilo_ip_netmask",
		"ilo_password":     "s.ilo_password",
		"ilo_username":     "s.ilo_username",
		"interface_mtu":    "interface_mtu",
		"interface_name":   "s.interface_name",
		"ip6_address":      "s.ip6_address",
		"ip6_gateway":      "s.ip6_gateway",
		"ip_address":       "s.ip_address",
		"ip_gateway":       "s.ip_gateway",
		"ip_netmask":       "s.ip_netmask",
		"mgmt_ip_address":  "s.mgmt_ip_address",
		"mgmt_ip_gateway":  "s.mgmt_ip_gateway",
		"mgmt_ip_netmask":  "s.mgmt_ip_netmask",
		"offline_reason":   "s.offline_reason",
		"phys_location":    "pl.name",
		"profile":          "p.name",
		"profile_desc":     "p.description",
		"rack":             "s.rack",
		"router_host_name": "s.router_host_name",
		"router_port_name": "s.router_port_name",
		"status":           "st.name",
		"tcp_port":         "s.tcp_port",
		"server_type":      "t.name",
		"xmpp_id":          "s.xmpp_id",
		"xmpp_passwd":      "s.xmpp_passwd",
	}
	orderBy, ok := allowedOrderByCols[orderBy]
	if !ok {
		return nil, errors.New("orderBy '" + orderBy + "' not permitted")
	}
	const JumboFrameBPS = 9000
	q := `
SELECT
cg.name as cachegroup,
cdn.name as cdn_name,
ARRAY(select deliveryservice from deliveryservice_server where server = s.id),
s.domain_name,
s.guid,
s.host_name,
s.https_port,
s.id,
s.ilo_ip_address,
s.ilo_ip_gateway,
s.ilo_ip_netmask,
s.ilo_password,
s.ilo_username,
COALESCE(s.interface_mtu, ` + strconv.Itoa(JumboFrameBPS) + `) as interface_mtu,
s.interface_name,
s.ip6_address,
s.ip6_gateway,
s.ip_address,
s.ip_gateway,
s.ip_netmask,
s.mgmt_ip_address,
s.mgmt_ip_gateway,
s.mgmt_ip_netmask,
s.offline_reason,
pl.name as phys_location,
p.name as profile,
p.description as profile_desc,
s.rack,
s.router_host_name,
s.router_port_name,
st.name as status,
s.tcp_port,
t.name as server_type,
s.xmpp_id,
s.xmpp_passwd
FROM server as s
JOIN cachegroup cg ON s.cachegroup = cg.id
JOIN cdn ON s.cdn_id = cdn.id
JOIN phys_location pl ON s.phys_location = pl.id
JOIN profile p ON s.profile = p.id
JOIN status st ON s.status = st.id
JOIN type t ON s.type = t.id
`
	limitStr := ""
	if limit != 0 {
		limitStr = " LIMIT " + strconv.Itoa(limit)
	}
	orderByStr := ""
	if orderBy != "" {
		orderByStr = " ORDER BY " + orderBy
	}
	rows := (*sql.Rows)(nil)
	err := error(nil)
	if hostName != "" && physLocationID != -1 {
		q += ` WHERE s.host_name = $1::text AND s.phys_location = $2::bigint` + orderByStr + limitStr
		rows, err = tx.Query(q, hostName, physLocationID)
	} else if hostName != "" {
		q += ` WHERE s.host_name = $1::text` + orderByStr + limitStr
		rows, err = tx.Query(q, hostName)
	} else if physLocationID != -1 {
		q += ` WHERE s.phys_location = $1::int` + orderByStr + limitStr
		rows, err = tx.Query(q, physLocationID)
	} else {
		q += orderByStr + limitStr
		rows, err = tx.Query(q) // Should never happen for API <1.3, which don't allow querying without hostName or physLocation
	}
	if err != nil {
		return nil, errors.New("Error querying detail servers: " + err.Error())
	}
	defer rows.Close()
	sIDs := []int{}
	servers := []tc.ServerDetail{}
	for rows.Next() {
		s := tc.ServerDetail{}
		if err := rows.Scan(&s.CacheGroup, &s.CDNName, pq.Array(&s.DeliveryServiceIDs), &s.DomainName, &s.GUID, &s.HostName, &s.HTTPSPort, &s.ID, &s.ILOIPAddress, &s.ILOIPGateway, &s.ILOIPNetmask, &s.ILOPassword, &s.ILOUsername, &s.InterfaceMTU, &s.InterfaceName, &s.IP6Address, &s.IP6Gateway, &s.IPAddress, &s.IPGateway, &s.IPNetmask, &s.MgmtIPAddress, &s.MgmtIPGateway, &s.MgmtIPNetmask, &s.OfflineReason, &s.PhysLocation, &s.Profile, &s.ProfileDesc, &s.Rack, &s.RouterHostName, &s.RouterPortName, &s.Status, &s.TCPPort, &s.Type, &s.XMPPID, &s.XMPPPasswd); err != nil {
			return nil, errors.New("Error scanning detail server: " + err.Error())
		}

		hiddenField := "********"
		if !user.HasCapability(tc.ServerSecureCapability) {
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
