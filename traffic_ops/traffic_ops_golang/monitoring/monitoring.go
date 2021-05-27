// Package monitoring contains handlers and supporting logic for the
// /cdns/{{CDN Name}}/configs/monitoring Traffic Ops API endpoint.
package monitoring

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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

	"github.com/lib/pq"
)

const CacheMonitorConfigFile = "rascal.properties"

const MonitorType = "RASCAL"
const RouterType = "CCR"
const MonitorProfilePrefix = "RASCAL"
const MonitorConfigFile = "rascal-config.txt"
const KilobitsPerMegabit = 1000
const DeliveryServiceStatus = "REPORTED"

type BasicServer struct {
	CommonServerProperties
	IP  string `json:"ip"`
	IP6 string `json:"ip6"`
}

type CommonServerProperties struct {
	Profile    string `json:"profile"`
	Status     string `json:"status"`
	Port       int    `json:"port"`
	Cachegroup string `json:"cachegroup"`
	HostName   string `json:"hostname"`
	FQDN       string `json:"fqdn"`
}

type Monitor struct {
	BasicServer
}

// LegacyCache represents a Cache for ATC versions before 5.0.
type LegacyCache struct {
	BasicServer
	InterfaceName string `json:"interfacename"`
	Type          string `json:"type"`
	HashID        string `json:"hashid"`
}

type Cache struct {
	CommonServerProperties
	Interfaces []tc.ServerInterfaceInfo `json:"interfaces"`
	Type       string                   `json:"type"`
	HashID     string                   `json:"hashid"`
}

type Cachegroup struct {
	Name        string      `json:"name"`
	Coordinates Coordinates `json:"coordinates"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Profile struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// LegacyMonitoring represents Monitoring for ATC versions before 5.0.
type LegacyMonitoring struct {
	TrafficServers   []LegacyCache          `json:"trafficServers"`
	TrafficMonitors  []Monitor              `json:"trafficMonitors"`
	Cachegroups      []Cachegroup           `json:"cacheGroups"`
	Profiles         []Profile              `json:"profiles"`
	DeliveryServices []DeliveryService      `json:"deliveryServices"`
	Config           map[string]interface{} `json:"config"`
}

type Monitoring struct {
	TrafficServers   []Cache                `json:"trafficServers"`
	TrafficMonitors  []Monitor              `json:"trafficMonitors"`
	Cachegroups      []Cachegroup           `json:"cacheGroups"`
	Profiles         []Profile              `json:"profiles"`
	DeliveryServices []DeliveryService      `json:"deliveryServices"`
	Config           map[string]interface{} `json:"config"`
}

// LegacyMonitoringResponse represents MontiroingResponse for ATC versions before 5.0.
type LegacyMonitoringResponse struct {
	Response LegacyMonitoring `json:"response"`
}

type MonitoringResponse struct {
	Response Monitoring `json:"response"`
}

type Router struct {
	Type    string
	Profile string
}

type DeliveryService struct {
	XMLID              string  `json:"xmlId"`
	TotalTPSThreshold  float64 `json:"totalTpsThreshold"`
	Status             string  `json:"status"`
	TotalKBPSThreshold float64 `json:"totalKbpsThreshold"`
}

func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	api.RespWriter(w, r, inf.Tx.Tx)(GetMonitoringJSON(inf.Tx.Tx, inf.Params["cdn"]))
}

func GetMonitoringJSON(tx *sql.Tx, cdnName string) (*Monitoring, error) {
	monitors, caches, routers, err := getMonitoringServers(tx, cdnName)
	if err != nil {
		return nil, fmt.Errorf("error getting servers: %v", err)
	}

	cachegroups, err := getCachegroups(tx, cdnName)
	if err != nil {
		return nil, fmt.Errorf("error getting cachegroups: %v", err)
	}

	profiles, err := getProfiles(tx, caches, routers)
	if err != nil {
		return nil, fmt.Errorf("error getting profiles: %v", err)
	}

	deliveryServices, err := getDeliveryServices(tx)
	if err != nil {
		return nil, fmt.Errorf("error getting deliveryservices: %v", err)
	}

	config, err := getConfig(tx, cdnName)
	if err != nil {
		return nil, fmt.Errorf("error getting config: %v", err)
	}

	return &Monitoring{
		TrafficServers:   caches,
		TrafficMonitors:  monitors,
		Cachegroups:      cachegroups,
		Profiles:         profiles,
		DeliveryServices: deliveryServices,
		Config:           config,
	}, nil
}

func getMonitoringServers(tx *sql.Tx, cdn string) ([]Monitor, []Cache, []Router, error) {
	serversQuery := `
SELECT
	me.host_name as hostName,
	CONCAT(me.host_name, '.', me.domain_name) as fqdn,
	status.name as status,
	cachegroup.name as cachegroup,
	me.tcp_port as port,
	profile.name as profile,
	type.name as type,
	me.xmpp_id as hashID,
    me.id as serverID
FROM server me
JOIN type type ON type.id = me.type
JOIN status status ON status.id = me.status
JOIN cachegroup cachegroup ON cachegroup.id = me.cachegroup
JOIN profile profile ON profile.id = me.profile
JOIN cdn cdn ON cdn.id = me.cdn_id
WHERE cdn.name = $1
`

	interfacesQuery := `
SELECT
   i.name, i.max_bandwidth, i.mtu, i.monitor, i.server
FROM interface i
WHERE i.server in (
	SELECT
		s.id
	FROM "server" s
	JOIN cdn c
		on c.id = s.cdn_id
	WHERE c.name = $1
)`

	ipAddressQuery := `
SELECT
	ip.address, ip.gateway, ip.service_address, ip.server, ip.interface
FROM ip_address ip
JOIN server s
	ON s.id = ip.server
JOIN cdn cdn
	ON cdn.id = s.cdn_id
WHERE ip.server = ANY($1)
AND ip.interface = ANY($2)
AND cdn.name = $3
`

	interfaceRows, err := tx.Query(interfacesQuery, cdn)
	if err != nil {
		return nil, nil, nil, err
	}
	defer interfaceRows.Close()

	//For constant time lookup of which interface/server belongs to the ipAddress
	var interfacesByNameAndServer = make(map[int]map[string]tc.ServerInterfaceInfo)
	var serverIDs []int
	var interfaceNames []string
	var interfaceName string
	var serverID int
	for interfaceRows.Next() {
		interf := tc.ServerInterfaceInfo{}
		if err := interfaceRows.Scan(&interf.Name, &interf.MaxBandwidth, &interf.MTU, &interf.Monitor, &serverID); err != nil {
			return nil, nil, nil, err
		}
		if _, ok := interfacesByNameAndServer[serverID]; !ok {
			interfacesByNameAndServer[serverID] = make(map[string]tc.ServerInterfaceInfo)
		}
		interfacesByNameAndServer[serverID][interf.Name] = interf
		serverIDs = append(serverIDs, serverID)
		interfaceNames = append(interfaceNames, interf.Name)
	}

	ipAddressRows, err := tx.Query(ipAddressQuery, pq.Array(serverIDs), pq.Array(interfaceNames), cdn)
	if err != nil {
		return nil, nil, nil, err
	}
	defer ipAddressRows.Close()
	for ipAddressRows.Next() {
		address := tc.ServerIPAddress{}
		if err := ipAddressRows.Scan(&address.Address, &address.Gateway, &address.ServiceAddress, &serverID, &interfaceName); err != nil {
			return nil, nil, nil, err
		}
		found := false
		var addresses []tc.ServerIPAddress
		if _, ok := interfacesByNameAndServer[serverID]; ok {
			if _, ok := interfacesByNameAndServer[serverID][interfaceName]; ok {
				addresses = append(addresses, address)
				found = ok
			}
		}
		if !found {
			log.Errorf("ip_address exists without corresponding interface; server: %v, interfaceName: %v!", serverID, interfaceName)
			continue
		}
		interf := interfacesByNameAndServer[serverID][interfaceName]
		interf.IPAddresses = append(interf.IPAddresses, addresses...)
		interfacesByNameAndServer[serverID][interfaceName] = interf
	}

	rows, err := tx.Query(serversQuery, cdn)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	monitors := []Monitor{}
	caches := []Cache{}
	routers := []Router{}

	for rows.Next() {
		var hostName sql.NullString
		var fqdn sql.NullString
		var status sql.NullString
		var cachegroup sql.NullString
		var port sql.NullInt64
		var profile sql.NullString
		var ttype sql.NullString
		var hashID sql.NullString
		var serverID sql.NullInt64

		if err := rows.Scan(&hostName, &fqdn, &status, &cachegroup, &port, &profile, &ttype, &hashID, &serverID); err != nil {
			return nil, nil, nil, err
		}
		cacheStatus := tc.CacheStatusFromString(status.String)

		if ttype.String == tc.MonitorTypeName {
			monitors = append(monitors, Monitor{
				BasicServer: BasicServer{
					CommonServerProperties: CommonServerProperties{
						Profile:    profile.String,
						Status:     status.String,
						Port:       int(port.Int64),
						Cachegroup: cachegroup.String,
						HostName:   hostName.String,
						FQDN:       fqdn.String,
					},
				},
			})
		} else if (strings.HasPrefix(ttype.String, "EDGE") || strings.HasPrefix(ttype.String, "MID")) &&
			(cacheStatus == tc.CacheStatusOnline || cacheStatus == tc.CacheStatusReported || cacheStatus == tc.CacheStatusAdminDown) {
			var cacheInterfaces []tc.ServerInterfaceInfo
			if _, ok := interfacesByNameAndServer[int(serverID.Int64)]; ok {
				for _, interf := range interfacesByNameAndServer[int(serverID.Int64)] {
					cacheInterfaces = append(cacheInterfaces, interf)
				}
			}
			if len(cacheInterfaces) == 0 {
				log.Errorf("cache with hashID: %v, has no interfaces!", hashID.String)
			}
			cache := Cache{
				CommonServerProperties: CommonServerProperties{
					Profile:    profile.String,
					Status:     status.String,
					Port:       int(port.Int64),
					Cachegroup: cachegroup.String,
					HostName:   hostName.String,
					FQDN:       fqdn.String,
				},
				Interfaces: cacheInterfaces,
				Type:       ttype.String,
				HashID:     hashID.String,
			}
			caches = append(caches, cache)
		} else if ttype.String == tc.RouterTypeName {
			routers = append(routers, Router{
				Type:    ttype.String,
				Profile: profile.String,
			})
		}
	}
	return monitors, caches, routers, nil
}

func getCachegroups(tx *sql.Tx, cdn string) ([]Cachegroup, error) {
	query := `
SELECT cg.name, co.latitude, co.longitude
FROM cachegroup cg
LEFT JOIN coordinate co ON co.id = cg.coordinate
WHERE cg.id IN
  (SELECT cachegroup FROM server WHERE server.cdn_id =
    (SELECT id FROM cdn WHERE name = $1));`

	rows, err := tx.Query(query, cdn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cachegroups := []Cachegroup{}
	for rows.Next() {
		var name sql.NullString
		var lat sql.NullFloat64
		var lon sql.NullFloat64
		if err := rows.Scan(&name, &lat, &lon); err != nil {
			return nil, err
		}
		cachegroups = append(cachegroups, Cachegroup{
			Name: name.String,
			Coordinates: Coordinates{
				Latitude:  lat.Float64,
				Longitude: lon.Float64,
			},
		})
	}
	return cachegroups, nil
}

func getProfiles(tx *sql.Tx, caches []Cache, routers []Router) ([]Profile, error) {
	cacheProfileTypes := map[string]string{}
	profiles := map[string]Profile{}
	profileNames := []string{}
	for _, router := range routers {
		profiles[router.Profile] = Profile{
			Name: router.Profile,
			Type: router.Type,
		}
	}

	for _, cache := range caches {
		if _, ok := cacheProfileTypes[cache.Profile]; !ok {
			cacheProfileTypes[cache.Profile] = cache.Type
			profiles[cache.Profile] = Profile{
				Name: cache.Profile,
				Type: cache.Type,
			}
			profileNames = append(profileNames, cache.Profile)
		}
	}

	query := `
SELECT p.name as profile, pr.name, pr.value
FROM parameter pr
JOIN profile p ON p.name = ANY($1)
JOIN profile_parameter pp ON pp.profile = p.id and pp.parameter = pr.id
WHERE pr.config_file = $2;
`
	rows, err := tx.Query(query, pq.Array(profileNames), CacheMonitorConfigFile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var profileName sql.NullString
		var name sql.NullString
		var value sql.NullString
		if err := rows.Scan(&profileName, &name, &value); err != nil {
			return nil, err
		}
		if name.String == "" {
			return nil, fmt.Errorf("null name") // TODO continue and warn?
		}
		profile := profiles[profileName.String]
		if profile.Parameters == nil {
			profile.Parameters = map[string]interface{}{}
		}

		if valNum, err := strconv.Atoi(value.String); err == nil {
			profile.Parameters[name.String] = valNum
		} else {
			profile.Parameters[name.String] = value.String
		}
		profiles[profileName.String] = profile

	}

	profilesArr := []Profile{} // TODO make for efficiency?
	for _, profile := range profiles {
		profilesArr = append(profilesArr, profile)
	}
	return profilesArr, nil
}

func getDeliveryServices(tx *sql.Tx) ([]DeliveryService, error) {
	query := `
	SELECT ds.xml_id, ds.global_max_tps, ds.global_max_mbps
	FROM deliveryservice ds
	WHERE ds.active = 'ACTIVE'
	`
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dses := []DeliveryService{}

	for rows.Next() {
		var xmlid sql.NullString
		var tps sql.NullFloat64
		var mbps sql.NullFloat64
		if err := rows.Scan(&xmlid, &tps, &mbps); err != nil {
			return nil, err
		}
		dses = append(dses, DeliveryService{
			XMLID:              xmlid.String,
			TotalTPSThreshold:  tps.Float64,
			Status:             DeliveryServiceStatus,
			TotalKBPSThreshold: mbps.Float64 * KilobitsPerMegabit,
		})
	}
	return dses, nil
}

func getConfig(tx *sql.Tx, cdnName string) (map[string]interface{}, error) {
	// TODO remove 'like' in query? Slow?
	query := `
SELECT pr.name, pr.value
FROM parameter pr
JOIN profile p ON p.name LIKE $1
JOIN profile_parameter pp ON pp.profile = p.id and pp.parameter = pr.id
JOIN cdn c ON c.id=p.cdn
WHERE pr.config_file = $2
AND c.name = $3
`
	rows, err := tx.Query(query, tc.MonitorProfilePrefix+"%%", MonitorConfigFile, cdnName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cfg := map[string]interface{}{}

	for rows.Next() {
		var name sql.NullString
		var val sql.NullString
		if err := rows.Scan(&name, &val); err != nil {
			return nil, err
		}
		if valNum, err := strconv.Atoi(val.String); err == nil {
			cfg[name.String] = valNum
		} else {
			cfg[name.String] = val.String
		}
	}
	return cfg, nil
}
