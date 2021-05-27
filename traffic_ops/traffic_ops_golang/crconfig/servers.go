package crconfig

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
	"strconv"
	"strings"

	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

const RouterTypeName = "CCR"
const MonitorTypeName = "RASCAL"
const EdgeTypePrefix = "EDGE"
const MidTypePrefix = "MID"

func makeCRConfigServers(cdn string, tx *sql.Tx, cdnDomain string) (
	map[string]tc.CRConfigTrafficOpsServer,
	map[string]tc.CRConfigRouter,
	map[string]tc.CRConfigMonitor,
	error,
) {
	allServers, err := getAllServers(cdn, tx)
	if err != nil {
		return nil, nil, nil, err
	}

	serverDSes, err := getServerDSes(cdn, tx, cdnDomain)
	if err != nil {
		return nil, nil, nil, errors.New("getting server deliveryservices: " + err.Error())
	}

	servers := map[string]tc.CRConfigTrafficOpsServer{}
	routers := map[string]tc.CRConfigRouter{}
	monitors := map[string]tc.CRConfigMonitor{}
	for host, s := range allServers {
		switch {
		case *s.ServerType == tc.RouterTypeName:
			status := tc.CRConfigRouterStatus(*s.ServerStatus)
			routers[host] = tc.CRConfigRouter{
				APIPort:       s.APIPort,
				FQDN:          s.Fqdn,
				HTTPSPort:     s.HttpsPort,
				IP:            s.Ip,
				IP6:           s.Ip6,
				Location:      s.LocationId,
				Port:          s.Port,
				Profile:       s.Profile,
				SecureAPIPort: s.SecureAPIPort,
				ServerStatus:  &status,
			}
		case *s.ServerType == tc.MonitorTypeName:
			monitors[host] = tc.CRConfigMonitor{
				FQDN:         s.Fqdn,
				HTTPSPort:    s.HttpsPort,
				IP:           s.Ip,
				IP6:          s.Ip6,
				Location:     s.LocationId,
				Port:         s.Port,
				Profile:      s.Profile,
				ServerStatus: s.ServerStatus,
			}
		case strings.HasPrefix(*s.ServerType, tc.EdgeTypePrefix) || strings.HasPrefix(*s.ServerType, tc.MidTypePrefix):
			if s.RoutingDisabled == 0 {
				s.CRConfigTrafficOpsServer.DeliveryServices = serverDSes[tc.CacheName(host)]
			}
			servers[host] = s.CRConfigTrafficOpsServer
		}
	}
	return servers, routers, monitors, nil
}

// ServerUnion has all fields from all servers. This is used to select all server data with a single query, and then convert each to the proper type afterwards.
type ServerUnion struct {
	tc.CRConfigTrafficOpsServer
	APIPort       *string
	SecureAPIPort *string
}

type ServerAndHost struct {
	Server ServerUnion
	Host   string
}

const DefaultWeightMultiplier = 1000.0
const DefaultWeight = 0.999

func getAllServers(cdn string, tx *sql.Tx) (map[string]ServerUnion, error) {
	serverParams, err := getServerParams(cdn, tx)
	if err != nil {
		return nil, errors.New("Error getting server params: " + err.Error())
	}

	// TODO select deliveryservices as array?
	q := `
	SELECT
		s.id,
		s.host_name,
		cg.name as cachegroup,
		concat(s.host_name, '.', s.domain_name) AS fqdn,
		s.xmpp_id AS hashid,
		s.https_port,
		s.tcp_port,
		p.name AS profile_name,
		cast(p.routing_disabled AS int),
		st.name AS status,
		t.name AS type,
		(SELECT ARRAY_AGG(server_capability ORDER BY server_capability)
			FROM server_server_capability
			WHERE server = s.id) AS capabilities
	FROM server AS s
	INNER JOIN cachegroup AS cg ON cg.id = s.cachegroup
	INNER JOIN type AS t on t.id = s.type
	INNER JOIN profile AS p ON p.id = s.profile
	INNER JOIN status AS st ON st.id = s.status
	WHERE cdn_id = (SELECT id FROM cdn WHERE name = $1)
	AND (st.name = 'REPORTED' OR st.name = 'ONLINE' OR st.name = 'ADMIN_DOWN')
	`
	rows, err := tx.Query(q, cdn)
	if err != nil {
		return nil, errors.New("Error querying servers: " + err.Error())
	}
	defer rows.Close()

	servers := map[int]ServerAndHost{}
	ids := []int{}
	for rows.Next() {
		var port sql.NullInt64
		var hashId sql.NullString
		var httpsPort sql.NullInt64

		var s ServerAndHost

		var status string
		var id int
		if err := rows.Scan(&id, &s.Host, &s.Server.CacheGroup, &s.Server.Fqdn, &hashId, &httpsPort, &port, &s.Server.Profile, &s.Server.RoutingDisabled, &status, &s.Server.ServerType, pq.Array(&s.Server.Capabilities)); err != nil {
			return nil, errors.New("Error scanning server: " + err.Error())
		}

		ids = append(ids, id)

		s.Server.LocationId = s.Server.CacheGroup

		serverStatus := tc.CRConfigServerStatus(status)
		s.Server.ServerStatus = &serverStatus
		if port.Valid {
			i := int(port.Int64)
			s.Server.Port = &i
		}

		if hashId.String != "" {
			s.Server.HashId = &hashId.String
		} else {
			s.Server.HashId = &s.Host
		}

		if httpsPort.Valid {
			i := int(httpsPort.Int64)
			s.Server.HttpsPort = &i
		}

		params, hasParams := serverParams[s.Host]
		if hasParams && params.APIPort != nil {
			s.Server.APIPort = params.APIPort
		}

		if hasParams && params.SecureAPIPort != nil {
			s.Server.SecureAPIPort = params.SecureAPIPort
		}

		weightMultiplier := DefaultWeightMultiplier
		if hasParams && params.WeightMultiplier != nil {
			weightMultiplier = *params.WeightMultiplier
		}
		weight := DefaultWeight
		if hasParams && params.Weight != nil {
			weight = *params.Weight
		}
		hashCount := int(weight * weightMultiplier)
		s.Server.HashCount = &hashCount

		servers[id] = s
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("Error iterating router param rows: " + err.Error())
	}

	interfaces, err := dbhelpers.GetServersInterfaces(ids, tx)
	if err != nil {
		return nil, fmt.Errorf("getting interfaces for servers: %v", err)
	}

	hostToServerMap := make(map[string]ServerUnion, len(servers))
	for id, server := range servers {
		ifaces, ok := interfaces[id]
		if !ok {
			log.Warnf("server '%s' (#%d) has no interfaces", server.Host, id)
			server.Server.InterfaceName = new(string)
			server.Server.Ip = new(string)
			server.Server.Ip6 = new(string)
			hostToServerMap[server.Host] = server.Server
			continue
		}

		infs := make([]tc.ServerInterfaceInfoV40, 0, len(ifaces))
		for _, inf := range ifaces {
			infs = append(infs, inf)
		}

		legacyNet, err := tc.V4InterfaceInfoToLegacyInterfaces(infs)
		if err != nil {
			return nil, fmt.Errorf("Error converting interfaces to legacy data for server '%s' (#%d): %v", server.Host, id, err)
		}

		server.Server.Ip = legacyNet.IPAddress
		server.Server.Ip6 = legacyNet.IP6Address

		if server.Server.Ip == nil {
			server.Server.Ip = new(string)
		}
		if server.Server.Ip6 == nil {
			server.Server.Ip6 = new(string)
		}

		server.Server.InterfaceName = legacyNet.InterfaceName
		if server.Server.InterfaceName == nil {
			server.Server.InterfaceName = new(string)
			log.Warnf("Server %s (#%d) had no service-address-containing interfaces", server.Host, id)
		}

		hostToServerMap[server.Host] = server.Server
	}

	return hostToServerMap, nil
}

func getServerDSNames(cdn string, tx *sql.Tx) (map[tc.CacheName][]tc.DeliveryServiceName, error) {
	q := `
select s.host_name, ds.xml_id
from deliveryservice_server as dss
inner join server as s on dss.server = s.id
inner join deliveryservice as ds on ds.id = dss.deliveryservice
inner join type as dt on dt.id = ds.type
inner join profile as p on p.id = s.profile
inner join status as st ON st.id = s.status
where ds.cdn_id = (select id from cdn where name = $1)
and ds.active = 'ACTIVE'` +
		fmt.Sprintf(" and dt.name != '%s' ", tc.DSTypeAnyMap) + `
and p.routing_disabled = false
and (st.name = 'REPORTED' or st.name = 'ONLINE' or st.name = 'ADMIN_DOWN')
`
	rows, err := tx.Query(q, cdn)
	if err != nil {
		return nil, errors.New("Error querying server deliveryservice names: " + err.Error())
	}
	defer rows.Close()

	serverDSes := map[tc.CacheName][]tc.DeliveryServiceName{}
	for rows.Next() {
		ds := ""
		server := ""
		if err := rows.Scan(&server, &ds); err != nil {
			return nil, errors.New("Error scanning server deliveryservice names: " + err.Error())
		}
		serverDSes[tc.CacheName(server)] = append(serverDSes[tc.CacheName(server)], tc.DeliveryServiceName(ds))
	}
	return serverDSes, nil
}

type DSRouteInfo struct {
	IsDNS bool
	IsRaw bool
	Remap string
}

func getServerDSes(cdn string, tx *sql.Tx, domain string) (map[tc.CacheName]map[string][]string, error) {
	serverDSNames, err := getServerDSNames(cdn, tx)
	if err != nil {
		return nil, errors.New("Error getting server deliveryservices: " + err.Error())
	}

	q := `
select ds.xml_id as ds, dt.name as ds_type, ds.routing_name, r.pattern as pattern,
ds.topology IS NOT NULL as has_topology
from regex as r
inner join type as rt on r.type = rt.id
inner join deliveryservice_regex as dsr on dsr.regex = r.id
inner join deliveryservice as ds on ds.id = dsr.deliveryservice
inner join type as dt on dt.id = ds.type
where ds.cdn_id = (select id from cdn where name = $1)
and ds.active = 'ACTIVE'` +
		fmt.Sprintf(" and dt.name != '%s' ", tc.DSTypeAnyMap) + `
and rt.name = 'HOST_REGEXP'
order by dsr.set_number asc
`
	rows, err := tx.Query(q, cdn)
	if err != nil {
		return nil, errors.New("Error server deliveryservices: " + err.Error())
	}
	defer rows.Close()

	hostReplacer := strings.NewReplacer(`\`, ``, `.*`, ``)

	dsInfs := map[string][]DSRouteInfo{}
	dsHasTopology := make(map[string]bool)
	for rows.Next() {
		hasTopology := false
		ds := ""
		dsType := ""
		dsPattern := ""
		dsRoutingName := ""
		inf := DSRouteInfo{}
		if err := rows.Scan(&ds, &dsType, &dsRoutingName, &dsPattern, &hasTopology); err != nil {
			return nil, errors.New("Error scanning server deliveryservices: " + err.Error())
		}
		dsHasTopology[ds] = hasTopology
		// Topology-based delivery services do not use the contentServers.deliveryServices field
		if hasTopology {
			continue
		}
		inf.IsDNS = strings.HasPrefix(dsType, "DNS")
		inf.IsRaw = !strings.Contains(dsPattern, `.*`)
		if !inf.IsRaw {
			host := hostReplacer.Replace(dsPattern)
			if inf.IsDNS {
				inf.Remap = dsRoutingName + host + domain
			} else {
				inf.Remap = host + domain
			}
		} else {
			inf.Remap = dsPattern
		}
		dsInfs[ds] = append(dsInfs[ds], inf)
	}

	serverDSPatterns := map[tc.CacheName]map[string][]string{}
	for server, dses := range serverDSNames {
		for _, dsName := range dses {
			dsInfList, ok := dsInfs[string(dsName)]
			if !ok {
				if !dsHasTopology[string(dsName)] {
					log.Warnln("Creating CRConfig: deliveryservice " + string(dsName) + " has no regexes, skipping")
				}
				continue
			}
			for _, dsInf := range dsInfList {
				if !dsInf.IsRaw && !dsInf.IsDNS {
					dsInf.Remap = string(server) + dsInf.Remap
				}
				if _, ok := serverDSPatterns[server]; !ok {
					serverDSPatterns[server] = map[string][]string{}
				}
				serverDSPatterns[server][string(dsName)] = append(serverDSPatterns[server][string(dsName)], dsInf.Remap)
			}
		}
	}
	return serverDSPatterns, nil
}

// ServerParams contains parameter data filled in the CRConfig Servers objects. If a given param doesn't exist on the given server, it will be nil.
type ServerParams struct {
	APIPort          *string
	SecureAPIPort    *string
	Weight           *float64
	WeightMultiplier *float64
}

func getServerParams(cdn string, tx *sql.Tx) (map[string]ServerParams, error) {
	params := map[string]ServerParams{}

	q := `
select s.host_name, p.name, p.value
from server as s
left join profile_parameter as pp on pp.profile = s.profile
left join parameter as p on p.id = pp.parameter
inner join status as st ON st.id = s.status
where s.cdn_id = (select id from cdn where name = $1)
and ((p.config_file = 'CRConfig.json' and (p.name = 'weight' or p.name = 'weightMultiplier')) or (p.name = 'api.port') or (p.name = 'secure.api.port'))
and (st.name = 'REPORTED' or st.name = 'ONLINE' or st.name = 'ADMIN_DOWN')
`
	rows, err := tx.Query(q, cdn)
	if err != nil {
		return nil, errors.New("Error querying server parameters: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		server := ""
		name := ""
		val := ""
		if err := rows.Scan(&server, &name, &val); err != nil {
			return nil, errors.New("Error scanning server parameters: " + err.Error())
		}

		param := params[server]
		switch name {
		case "api.port":
			param.APIPort = &val
		case "secure.api.port":
			param.SecureAPIPort = &val
		case "weight":
			i, err := strconv.ParseFloat(val, 64)
			if err != nil {
				log.Warnln("Creating CRConfig: server " + server + " weight param " + val + " not a number, ignoring")
				continue
			}
			param.Weight = &i
		case "weightMultiplier":
			i, err := strconv.ParseFloat(val, 64)
			if err != nil {
				log.Warnln("Creating CRConfig: server " + server + " weightMultiplier param " + val + " not a number, ignoring")
				continue
			}
			param.WeightMultiplier = &i
		}
		params[server] = param
	}
	return params, nil
}

// getCDNInfo returns the CDN domain, and whether DNSSec is enabled
func getCDNInfo(cdn string, tx *sql.Tx) (string, bool, error) {
	domain := ""
	dnssec := false
	if err := tx.QueryRow(`select domain_name, dnssec_enabled from cdn where name = $1`, cdn).Scan(&domain, &dnssec); err != nil {
		return "", false, errors.New("Error querying CDN domain name: " + err.Error())
	}
	return domain, dnssec, nil
}

// getCDNNameFromID returns the CDN name given the ID, false if the no CDN with the given ID exists, and an error if the database query fails.
func getCDNNameFromID(id int, tx *sql.Tx) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`select name from cdn where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("Error querying CDN name: " + err.Error())
	}
	return name, true, nil
}

// getGlobalParam returns the global parameter with the requested name, whether it existed, and any error
func getGlobalParam(tx *sql.Tx, name string) (string, bool, error) {
	val := ""
	if err := tx.QueryRow(`SELECT value FROM parameter WHERE config_file = $1 and name = $2`, tc.GlobalConfigFileName, name).Scan(&val); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying global parameter '" + name + "': " + err.Error())
	}
	return val, true, nil
}
