// Package server provides tools for manipulating the server database table and
// corresponding http handlers.
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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/routing/middleware"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/topology/topology_validation"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const serversFromAndJoin = `
FROM server AS s
JOIN cachegroup cg ON s.cachegroup = cg.id
JOIN cdn cdn ON s.cdn_id = cdn.id
JOIN phys_location pl ON s.phys_location = pl.id
JOIN profile p ON s.profile = p.id
JOIN status st ON s.status = st.id
JOIN type t ON s.type = t.id
`

/* language=SQL */
const dssTopologiesJoinSubquery = `
(SELECT
	ARRAY_AGG(CAST(ROW(td.id, s.id, NULL) AS deliveryservice_server))
FROM "server" s
JOIN cachegroup c on s.cachegroup = c.id
JOIN topology_cachegroup tc ON c.name = tc.cachegroup
JOIN deliveryservice td ON td.topology = tc.topology
JOIN type t ON s.type = t.id
LEFT JOIN deliveryservice_server dss
	ON s.id = dss."server"
	AND dss.deliveryservice = td.id
WHERE td.id = :dsId
AND (
	t.name != '` + tc.OriginTypeName + `'
	OR dss.deliveryservice IS NOT NULL
)),
`

/* language=SQL */
const deliveryServiceServersJoin = `
FULL OUTER JOIN (
SELECT (dss.dss_record).deliveryservice, (dss.dss_record).server FROM (
	SELECT UNNEST(COALESCE(
		%s
		(SELECT
			ARRAY_AGG(CAST(ROW(dss.deliveryservice, dss."server", NULL) AS deliveryservice_server))
		FROM deliveryservice_server dss)
	)) AS dss_record) AS dss
) dss ON dss.server = s.id
JOIN deliveryservice d ON cdn.id = d.cdn_id AND dss.deliveryservice = d.id
`

/* language=SQL */
const requiredCapabilitiesCondition = `
AND (
	SELECT ARRAY_AGG(ssc.server_capability)
	FROM server_server_capability ssc
	WHERE ssc."server" = s.id
) @> (
	SELECT ARRAY_AGG(drc.required_capability)
	FROM deliveryservices_required_capability drc
	WHERE drc.deliveryservice_id = d.id
)
`

const serverCountQuery = `
SELECT COUNT(s.id)
` + serversFromAndJoin

const selectQuery = `
SELECT
	cg.name AS cachegroup,
	s.cachegroup AS cachegroup_id,
	s.cdn_id,
	cdn.name AS cdn_name,
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
	s.last_updated,
	s.mgmt_ip_address,
	s.mgmt_ip_gateway,
	s.mgmt_ip_netmask,
	s.offline_reason,
	pl.name AS phys_location,
	s.phys_location AS phys_location_id,
	p.name AS profile,
	p.description AS profile_desc,
	s.profile AS profile_id,
	s.rack,
	s.reval_pending,
	st.name AS status,
	s.status AS status_id,
	s.tcp_port,
	t.name AS server_type,
	s.type AS server_type_id,
	s.upd_pending AS upd_pending,
	s.xmpp_id,
	s.xmpp_passwd,
	s.status_last_updated
` + serversFromAndJoin

const selectIDQuery = `
SELECT
	s.id
` + serversFromAndJoin

const midWhereClause = `
WHERE t.name = :cache_type_mid AND s.cachegroup IN (
	SELECT cg.parent_cachegroup_id FROM cachegroup AS cg
	WHERE cg.id IN (
	SELECT s.cachegroup FROM server AS s
	WHERE s.id = ANY(:edge_ids)))
	AND (SELECT d.topology
		FROM deliveryservice d
		WHERE d.id = :ds_id) IS NULL
`

const insertQueryV3 = `
INSERT INTO server (
	cachegroup,
	cdn_id,
	domain_name,
	host_name,
	https_port,
	ilo_ip_address,
	ilo_ip_netmask,
	ilo_ip_gateway,
	ilo_username,
	ilo_password,
	mgmt_ip_address,
	mgmt_ip_netmask,
	mgmt_ip_gateway,
	offline_reason,
	phys_location,
	profile,
	rack,
	status,
	tcp_port,
	type,
	upd_pending,
	xmpp_id,
	xmpp_passwd,
	status_last_updated
) VALUES (
	:cachegroup_id,
	:cdn_id,
	:domain_name,
	:host_name,
	:https_port,
	:ilo_ip_address,
	:ilo_ip_netmask,
	:ilo_ip_gateway,
	:ilo_username,
	:ilo_password,
	:mgmt_ip_address,
	:mgmt_ip_netmask,
	:mgmt_ip_gateway,
	:offline_reason,
	:phys_location_id,
	:profile_id,
	:rack,
	:status_id,
	:tcp_port,
	:server_type_id,
	:upd_pending,
	:xmpp_id,
	:xmpp_passwd,
	:status_last_updated
) RETURNING
	(SELECT name FROM cachegroup WHERE cachegroup.id=server.cachegroup) AS cachegroup,
	cachegroup AS cachegroup_id,
	cdn_id,
	(SELECT name FROM cdn WHERE cdn.id=server.cdn_id) AS cdn_name,
	domain_name,
	guid,
	host_name,
	https_port,
	id,
	ilo_ip_address,
	ilo_ip_gateway,
	ilo_ip_netmask,
	ilo_password,
	ilo_username,
	last_updated,
	mgmt_ip_address,
	mgmt_ip_gateway,
	mgmt_ip_netmask,
	offline_reason,
	(SELECT name FROM phys_location WHERE phys_location.id=server.phys_location) AS phys_location,
	phys_location AS phys_location_id,
	profile AS profile_id,
	(SELECT description FROM profile WHERE profile.id=server.profile) AS profile_desc,
	(SELECT name FROM profile WHERE profile.id=server.profile) AS profile,
	rack,
	reval_pending,
	(SELECT name FROM status WHERE status.id=server.status) AS status,
	status AS status_id,
	tcp_port,
	(SELECT name FROM type WHERE type.id=server.type) AS server_type,
	type AS server_type_id,
	upd_pending
`
const insertQueryV4 = `
INSERT INTO server (
	cachegroup,
	cdn_id,
	domain_name,
	host_name,
	https_port,
	ilo_ip_address,
	ilo_ip_netmask,
	ilo_ip_gateway,
	ilo_username,
	ilo_password,
	mgmt_ip_address,
	mgmt_ip_netmask,
	mgmt_ip_gateway,
	offline_reason,
	phys_location,
	profile,
	rack,
	status,
	tcp_port,
	type,
	upd_pending,
	xmpp_id,
	xmpp_passwd
) VALUES (
	:cachegroup_id,
	:cdn_id,
	:domain_name,
	:host_name,
	:https_port,
	:ilo_ip_address,
	:ilo_ip_netmask,
	:ilo_ip_gateway,
	:ilo_username,
	:ilo_password,
	:mgmt_ip_address,
	:mgmt_ip_netmask,
	:mgmt_ip_gateway,
	:offline_reason,
	:phys_location_id,
	:profile_id,
	:rack,
	:status_id,
	:tcp_port,
	:server_type_id,
	:upd_pending,
	:xmpp_id,
	:xmpp_passwd
) RETURNING
	(SELECT name FROM cachegroup WHERE cachegroup.id=server.cachegroup) AS cachegroup,
	cachegroup AS cachegroup_id,
	cdn_id,
	(SELECT name FROM cdn WHERE cdn.id=server.cdn_id) AS cdn_name,
	domain_name,
	guid,
	host_name,
	https_port,
	id,
	ilo_ip_address,
	ilo_ip_gateway,
	ilo_ip_netmask,
	ilo_password,
	ilo_username,
	last_updated,
	mgmt_ip_address,
	mgmt_ip_gateway,
	mgmt_ip_netmask,
	offline_reason,
	(SELECT name FROM phys_location WHERE phys_location.id=server.phys_location) AS phys_location,
	phys_location AS phys_location_id,
	profile AS profile_id,
	(SELECT description FROM profile WHERE profile.id=server.profile) AS profile_desc,
	(SELECT name FROM profile WHERE profile.id=server.profile) AS profile,
	rack,
	reval_pending,
	(SELECT name FROM status WHERE status.id=server.status) AS status,
	status AS status_id,
	tcp_port,
	(SELECT name FROM type WHERE type.id=server.type) AS server_type,
	type AS server_type_id,
	upd_pending
`

const insertQuery = `
INSERT INTO server (
	cachegroup,
	cdn_id,
	domain_name,
	host_name,
	https_port,
	ilo_ip_address,
	ilo_ip_netmask,
	ilo_ip_gateway,
	ilo_username,
	ilo_password,
	mgmt_ip_address,
	mgmt_ip_netmask,
	mgmt_ip_gateway,
	offline_reason,
	phys_location,
	profile,
	rack,
	status,
	tcp_port,
	type,
	upd_pending,
	xmpp_id,
	xmpp_passwd
) VALUES (
	:cachegroup_id,
	:cdn_id,
	:domain_name,
	:host_name,
	:https_port,
	:ilo_ip_address,
	:ilo_ip_netmask,
	:ilo_ip_gateway,
	:ilo_username,
	:ilo_password,
	:mgmt_ip_address,
	:mgmt_ip_netmask,
	:mgmt_ip_gateway,
	:offline_reason,
	:phys_location_id,
	:profile_id,
	:rack,
	:status_id,
	:tcp_port,
	:server_type_id,
	:upd_pending,
	:xmpp_id,
	:xmpp_passwd
) RETURNING
	(SELECT name FROM cachegroup WHERE cachegroup.id=server.cachegroup) AS cachegroup,
	cachegroup AS cachegroup_id,
	cdn_id,
	(SELECT name FROM cdn WHERE cdn.id=server.cdn_id) AS cdn_name,
	domain_name,
	guid,
	host_name,
	https_port,
	id,
	ilo_ip_address,
	ilo_ip_gateway,
	ilo_ip_netmask,
	ilo_password,
	ilo_username,
	last_updated,
	mgmt_ip_address,
	mgmt_ip_gateway,
	mgmt_ip_netmask,
	offline_reason,
	(SELECT name FROM phys_location WHERE phys_location.id=server.phys_location) AS phys_location,
	phys_location AS phys_location_id,
	profile AS profile_id,
	(SELECT description FROM profile WHERE profile.id=server.profile) AS profile_desc,
	(SELECT name FROM profile WHERE profile.id=server.profile) AS profile,
	rack,
	reval_pending,
	(SELECT name FROM status WHERE status.id=server.status) AS status,
	status AS status_id,
	tcp_port,
	(SELECT name FROM type WHERE type.id=server.type) AS server_type,
	type AS server_type_id,
	upd_pending
`

const updateQuery = `
UPDATE server SET
	cachegroup=:cachegroup_id,
	cdn_id=:cdn_id,
	domain_name=:domain_name,
	host_name=:host_name,
	https_port=:https_port,
	ilo_ip_address=:ilo_ip_address,
	ilo_ip_netmask=:ilo_ip_netmask,
	ilo_ip_gateway=:ilo_ip_gateway,
	ilo_username=:ilo_username,
	ilo_password=:ilo_password,
	mgmt_ip_address=:mgmt_ip_address,
	mgmt_ip_netmask=:mgmt_ip_netmask,
	mgmt_ip_gateway=:mgmt_ip_gateway,
	offline_reason=:offline_reason,
	phys_location=:phys_location_id,
	profile=:profile_id,
	rack=:rack,
	status=:status_id,
	tcp_port=:tcp_port,
	type=:server_type_id,
	upd_pending=:upd_pending,
	xmpp_passwd=:xmpp_passwd,
	status_last_updated=:status_last_updated
WHERE id=:id
RETURNING
	(SELECT name FROM cachegroup WHERE cachegroup.id=server.cachegroup) AS cachegroup,
	cachegroup AS cachegroup_id,
	cdn_id,
	(SELECT name FROM cdn WHERE cdn.id=server.cdn_id) AS cdn_name,
	domain_name,
	guid,
	host_name,
	https_port,
	id,
	ilo_ip_address,
	ilo_ip_gateway,
	ilo_ip_netmask,
	ilo_password,
	ilo_username,
	last_updated,
	mgmt_ip_address,
	mgmt_ip_gateway,
	mgmt_ip_netmask,
	offline_reason,
	(SELECT name FROM phys_location WHERE phys_location.id=server.phys_location) AS phys_location,
	phys_location AS phys_location_id,
	profile AS profile_id,
	(SELECT description FROM profile WHERE profile.id=server.profile) AS profile_desc,
	(SELECT name FROM profile WHERE profile.id=server.profile) AS profile,
	rack,
	reval_pending,
	(SELECT name FROM status WHERE status.id=server.status) AS status,
	status AS status_id,
	tcp_port,
	(SELECT name FROM type WHERE type.id=server.type) AS server_type,
	type AS server_type_id,
	upd_pending,
	status_last_updated
`

const originServerQuery = `
JOIN deliveryservice_server dsorg
ON dsorg.server = s.id
WHERE t.name = '` + tc.OriginTypeName + `'
AND dsorg.deliveryservice=:dsId
`
const deleteServerQuery = `DELETE FROM server WHERE id=$1`
const deleteInterfacesQuery = `DELETE FROM interface WHERE server=$1`
const deleteIPsQuery = `DELETE FROM ip_address WHERE server = $1`

func newUUID() *string {
	uuidReference := uuid.New().String()
	return &uuidReference
}

func validateCommon(s *tc.CommonServerProperties, tx *sql.Tx) []error {

	noSpaces := validation.NewStringRule(tovalidate.NoSpaces, "cannot contain spaces")

	errs := tovalidate.ToErrors(validation.Errors{
		"cachegroupId":   validation.Validate(s.CachegroupID, validation.NotNil),
		"cdnId":          validation.Validate(s.CDNID, validation.NotNil),
		"domainName":     validation.Validate(s.DomainName, validation.Required, noSpaces),
		"hostName":       validation.Validate(s.HostName, validation.Required, noSpaces),
		"physLocationId": validation.Validate(s.PhysLocationID, validation.NotNil),
		"profileId":      validation.Validate(s.ProfileID, validation.NotNil),
		"statusId":       validation.Validate(s.StatusID, validation.NotNil),
		"typeId":         validation.Validate(s.TypeID, validation.NotNil),
		"updPending":     validation.Validate(s.UpdPending, validation.NotNil),
		"httpsPort":      validation.Validate(s.HTTPSPort, validation.By(tovalidate.IsValidPortNumber)),
		"tcpPort":        validation.Validate(s.TCPPort, validation.By(tovalidate.IsValidPortNumber)),
	})

	if len(errs) > 0 {
		return errs
	}

	if _, err := tc.ValidateTypeID(tx, s.TypeID, "server"); err != nil {
		errs = append(errs, err)
	}

	var cdnID int
	if err := tx.QueryRow("SELECT cdn from profile WHERE id=$1", s.ProfileID).Scan(&cdnID); err != nil {
		log.Errorf("could not execute select cdnID from profile: %s\n", err)
		if err == sql.ErrNoRows {
			errs = append(errs, fmt.Errorf("no such profileId: '%d'", *s.ProfileID))
		} else {
			errs = append(errs, tc.DBError)
		}
		return errs
	}

	log.Infof("got cdn id: %d from profile and cdn id: %d from server", cdnID, *s.CDNID)
	if cdnID != *s.CDNID {
		errs = append(errs, fmt.Errorf("CDN id '%d' for profile '%d' does not match Server CDN '%d'", cdnID, *s.ProfileID, *s.CDNID))
	}

	return errs
}

func validateV1(s *tc.ServerNullableV11, tx *sql.Tx) error {
	if s.IP6Address != nil && len(strings.TrimSpace(*s.IP6Address)) == 0 {
		s.IP6Address = nil
	}

	errs := []error{}
	if (s.IPAddress == nil || *s.IPAddress == "") && s.IP6Address == nil {
		errs = append(errs, tc.NeedsAtLeastOneIPError)
	}

	validateErrs := validation.Errors{
		"interfaceMtu":  validation.Validate(s.InterfaceMtu, validation.NotNil),
		"interfaceName": validation.Validate(s.InterfaceName, validation.NotNil),
	}

	if s.IPAddress != nil && *s.IPAddress != "" {
		validateErrs["ipAddress"] = validation.Validate(s.IPAddress, is.IPv4)
		validateErrs["ipNetmask"] = validation.Validate(s.IPNetmask, validation.NotNil)
		validateErrs["ipGateway"] = validation.Validate(s.IPGateway, validation.NotNil)
	}
	if s.IP6Address != nil && *s.IP6Address != "" {
		validateErrs["ip6Address"] = validation.Validate(s.IP6Address, validation.By(tovalidate.IsValidIPv6CIDROrAddress))
	}
	errs = append(errs, tovalidate.ToErrors(validateErrs)...)
	errs = append(errs, validateCommon(&s.CommonServerProperties, tx)...)

	return util.JoinErrs(errs)
}

func validateV2(s *tc.ServerNullableV2, tx *sql.Tx) error {
	var errs []error

	if err := validateV1(&s.ServerNullableV11, tx); err != nil {
		return err
	}

	// default boolean value is false
	if s.IPIsService == nil {
		s.IPIsService = new(bool)
	}
	if s.IP6IsService == nil {
		s.IP6IsService = new(bool)
	}

	if !*s.IPIsService && !*s.IP6IsService {
		errs = append(errs, tc.NeedsAtLeastOneServiceAddressError)
	}

	if *s.IPIsService && (s.IPAddress == nil || *s.IPAddress == "") {
		errs = append(errs, tc.EmptyAddressCannotBeAServiceAddressError)
	}

	if *s.IP6IsService && (s.IP6Address == nil || *s.IP6Address == "") {
		errs = append(errs, tc.EmptyAddressCannotBeAServiceAddressError)
	}
	return util.JoinErrs(errs)
}

func validateMTU(mtu interface{}) error {
	m, ok := mtu.(*uint64)
	if !ok {
		return errors.New("must be an unsigned integer with 64-bit precision")
	}
	if m == nil {
		return nil
	}

	if *m < 1280 {
		return errors.New("must be at least 1280")
	}
	return nil
}

func validateV4(s *tc.ServerV40, tx *sql.Tx) (string, error) {
	if len(s.Interfaces) == 0 {
		return "", errors.New("a server must have at least one interface")
	}
	var errs []error
	var serviceAddrV4Found bool
	var ipv4 string
	var serviceAddrV6Found bool
	var ipv6 string
	var serviceInterface string
	for _, iface := range s.Interfaces {

		ruleName := fmt.Sprintf("interface '%s' ", iface.Name)
		errs = append(errs, tovalidate.ToErrors(validation.Errors{
			ruleName + "name":        validation.Validate(iface.Name, validation.Required),
			ruleName + "mtu":         validation.Validate(iface.MTU, validation.By(validateMTU)),
			ruleName + "ipAddresses": validation.Validate(iface.IPAddresses, validation.Required),
		})...)

		for _, addr := range iface.IPAddresses {
			ruleName += fmt.Sprintf("address '%s'", addr.Address)

			var parsedIP net.IP
			var err error
			if parsedIP, _, err = net.ParseCIDR(addr.Address); err != nil {
				if parsedIP = net.ParseIP(addr.Address); parsedIP == nil {
					errs = append(errs, fmt.Errorf("%s: address: %v", ruleName, err))
					continue
				}
			}

			if addr.Gateway != nil {
				if gateway := net.ParseIP(*addr.Gateway); gateway == nil {
					errs = append(errs, fmt.Errorf("%s: gateway: could not parse '%s' as a network gateway", ruleName, *addr.Gateway))
				} else if (gateway.To4() == nil && parsedIP.To4() != nil) || (gateway.To4() != nil && parsedIP.To4() == nil) {
					errs = append(errs, errors.New(ruleName+": address family mismatch between address and gateway"))
				}
			}

			if addr.ServiceAddress {
				if serviceInterface != "" && serviceInterface != iface.Name {
					errs = append(errs, fmt.Errorf("interfaces: both %s and %s interfaces contain service addresses - only one service-address-containing-interface is allowed", serviceInterface, iface.Name))
				}
				serviceInterface = iface.Name
				if parsedIP.To4() != nil {
					if serviceAddrV4Found {
						errs = append(errs, fmt.Errorf("interfaces: address '%s' of interface '%s' is marked as a service address, but an IPv4 service address appears earlier in the list", addr.Address, iface.Name))
					}
					serviceAddrV4Found = true
					ipv4 = addr.Address
				} else {
					if serviceAddrV6Found {
						errs = append(errs, fmt.Errorf("interfaces: address '%s' of interface '%s' is marked as a service address, but an IPv6 service address appears earlier in the list", addr.Address, iface.Name))
					}
					serviceAddrV6Found = true
					ipv6 = addr.Address
				}
			}
		}
	}

	if !serviceAddrV6Found && !serviceAddrV4Found {
		errs = append(errs, errors.New("a server must have at least one service address"))
	}
	if errs = append(errs, validateCommon(&s.CommonServerProperties, tx)...); errs != nil {
		return serviceInterface, util.JoinErrs(errs)
	}
	query := `
SELECT s.ID, ip.address FROM server s
JOIN profile p on p.Id = s.Profile
JOIN interface i on i.server = s.ID
JOIN ip_address ip on ip.Server = s.ID and ip.interface = i.name
WHERE ip.service_address = true
and p.id = $1
`
	var rows *sql.Rows
	var err error
	//ProfileID already validated
	if s.ID != nil {
		rows, err = tx.Query(query+" and s.id != $2", *s.ProfileID, *s.ID)
	} else {
		rows, err = tx.Query(query, *s.ProfileID)
	}
	if err != nil {
		errs = append(errs, errors.New("unable to determine service address uniqueness"))
	} else if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var ipaddress string
			err = rows.Scan(&id, &ipaddress)
			if err != nil {
				errs = append(errs, errors.New("unable to determine service address uniqueness"))
			} else if (ipaddress == ipv4 || ipaddress == ipv6) && (s.ID == nil || *s.ID != id) {
				errs = append(errs, fmt.Errorf("there exists a server with id %v on the same profile that has the same service address %s", id, ipaddress))
			}
		}
	}

	return serviceInterface, util.JoinErrs(errs)
}

func validateV3(s *tc.ServerV30, tx *sql.Tx) (string, error) {

	if len(s.Interfaces) == 0 {
		return "", errors.New("a server must have at least one interface")
	}
	var errs []error
	var serviceAddrV4Found bool
	var ipv4 string
	var serviceAddrV6Found bool
	var ipv6 string
	var serviceInterface string
	for _, iface := range s.Interfaces {

		ruleName := fmt.Sprintf("interface '%s' ", iface.Name)
		errs = append(errs, tovalidate.ToErrors(validation.Errors{
			ruleName + "name":        validation.Validate(iface.Name, validation.Required),
			ruleName + "mtu":         validation.Validate(iface.MTU, validation.By(validateMTU)),
			ruleName + "ipAddresses": validation.Validate(iface.IPAddresses, validation.Required),
		})...)

		for _, addr := range iface.IPAddresses {
			ruleName += fmt.Sprintf("address '%s'", addr.Address)

			var parsedIP net.IP
			var err error
			if parsedIP, _, err = net.ParseCIDR(addr.Address); err != nil {
				if parsedIP = net.ParseIP(addr.Address); parsedIP == nil {
					errs = append(errs, fmt.Errorf("%s: address: %v", ruleName, err))
					continue
				}
			}

			if addr.Gateway != nil {
				if gateway := net.ParseIP(*addr.Gateway); gateway == nil {
					errs = append(errs, fmt.Errorf("%s: gateway: could not parse '%s' as a network gateway", ruleName, *addr.Gateway))
				} else if (gateway.To4() == nil && parsedIP.To4() != nil) || (gateway.To4() != nil && parsedIP.To4() == nil) {
					errs = append(errs, errors.New(ruleName+": address family mismatch between address and gateway"))
				}
			}

			if addr.ServiceAddress {
				if serviceInterface != "" && serviceInterface != iface.Name {
					errs = append(errs, fmt.Errorf("interfaces: both %s and %s interfaces contain service addresses - only one service-address-containing-interface is allowed", serviceInterface, iface.Name))
				}
				serviceInterface = iface.Name
				if parsedIP.To4() != nil {
					if serviceAddrV4Found {
						errs = append(errs, fmt.Errorf("interfaces: address '%s' of interface '%s' is marked as a service address, but an IPv4 service address appears earlier in the list", addr.Address, iface.Name))
					}
					serviceAddrV4Found = true
					ipv4 = addr.Address
				} else {
					if serviceAddrV6Found {
						errs = append(errs, fmt.Errorf("interfaces: address '%s' of interface '%s' is marked as a service address, but an IPv6 service address appears earlier in the list", addr.Address, iface.Name))
					}
					serviceAddrV6Found = true
					ipv6 = addr.Address
				}
			}
		}
	}

	if !serviceAddrV6Found && !serviceAddrV4Found {
		errs = append(errs, errors.New("a server must have at least one service address"))
	}
	if errs = append(errs, validateCommon(&s.CommonServerProperties, tx)...); errs != nil {
		return serviceInterface, util.JoinErrs(errs)
	}
	query := `
SELECT s.ID, ip.address FROM server s
JOIN profile p on p.Id = s.Profile
JOIN interface i on i.server = s.ID
JOIN ip_address ip on ip.Server = s.ID and ip.interface = i.name
WHERE ip.service_address = true
and p.id = $1
`
	var rows *sql.Rows
	var err error
	//ProfileID already validated
	if s.ID != nil {
		rows, err = tx.Query(query+" and s.id != $2", *s.ProfileID, *s.ID)
	} else {
		rows, err = tx.Query(query, *s.ProfileID)
	}
	if err != nil {
		errs = append(errs, errors.New("unable to determine service address uniqueness"))
	} else if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var ipaddress string
			err = rows.Scan(&id, &ipaddress)
			if err != nil {
				errs = append(errs, errors.New("unable to determine service address uniqueness"))
			} else if (ipaddress == ipv4 || ipaddress == ipv6) && (s.ID == nil || *s.ID != id) {
				errs = append(errs, fmt.Errorf("there exists a server with id %v on the same profile that has the same service address %s", id, ipaddress))
			}
		}
	}

	return serviceInterface, util.JoinErrs(errs)
}

// Read is the handler for GET requests to /servers.
func Read(w http.ResponseWriter, r *http.Request) {
	var maxTime *time.Time
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	// Middleware should've already handled this, so idk why this is a pointer at all tbh
	version := inf.Version
	if version == nil {
		middleware.NotImplementedHandler().ServeHTTP(w, r)
		return
	}

	servers := []tc.ServerV40{}
	var serverCount uint64
	cfg, e := api.GetConfig(r.Context())
	useIMS := false
	if e == nil && cfg != nil {
		useIMS = cfg.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}

	servers, serverCount, userErr, sysErr, errCode, maxTime = getServers(r.Header, inf.Params, inf.Tx, inf.User, useIMS, *version)
	if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
		api.AddLastModifiedHdr(w, *maxTime)
	}
	if errCode == http.StatusNotModified {
		w.WriteHeader(errCode)
		api.WriteResp(w, r, []tc.ServerV40{})
		return
	}
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if version.Major >= 4 {
		api.WriteRespWithSummary(w, r, servers, serverCount)
		return
	}
	if version.Major >= 3 {
		v3Servers := make([]tc.ServerV30, 0)
		for _, server := range servers {
			v3Server, err := server.ToServerV3FromV4()
			if err != nil {
				api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("failed to convert servers to V3 format: %v", err))
				return
			}
			v3Servers = append(v3Servers, v3Server)
		}
		api.WriteRespWithSummary(w, r, v3Servers, serverCount)
		return
	}

	if version.Major <= 1 {
		legacyServers := make([]tc.ServerNullableV11, 0, len(servers))
		for _, server := range servers {
			legacyServer, err := server.ToServerV2FromV4()
			if err != nil {
				api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("failed to convert servers to legacy format: %v", err))
				return
			}
			legacyServers = append(legacyServers, legacyServer.ServerNullableV11)
		}
		api.WriteResp(w, r, legacyServers)
		return
	}

	legacyServers := make([]tc.ServerNullableV2, 0, len(servers))
	for _, server := range servers {
		legacyServer, err := server.ToServerV2FromV4()
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("failed to convert servers to legacy format: %v", err))
			return
		}
		legacyServers = append(legacyServers, legacyServer)
	}
	api.WriteResp(w, r, legacyServers)
}

// ReadID is the handler for GET requests to /servers/{{ID}}.
func ReadID(w http.ResponseWriter, r *http.Request) {
	alternative := "GET /servers with query parameter id"
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, tx, errCode, userErr, sysErr, &alternative)
		return
	}
	defer inf.Close()

	// Middleware should've already handled this, so idk why this is a pointer at all tbh
	version := inf.Version
	if version == nil {
		middleware.NotImplementedHandler().ServeHTTP(w, r)
		return
	}

	servers := []tc.ServerV40{}
	cfg, e := api.GetConfig(r.Context())
	useIMS := false
	if e == nil && cfg != nil {
		useIMS = cfg.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}
	servers, _, userErr, sysErr, errCode, _ = getServers(r.Header, inf.Params, inf.Tx, inf.User, useIMS, *version)
	if len(servers) > 1 {
		api.HandleDeprecatedErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("ID '%d' matched more than one server (%d total)", inf.IntParams["id"], len(servers)), &alternative)
		return
	}
	deprecationAlerts := api.CreateDeprecationAlerts(&alternative)

	// No need to bother converting if there's no data
	if len(servers) < 1 {
		api.WriteAlertsObj(w, r, http.StatusOK, deprecationAlerts, servers)
		return
	}
	legacyServers := make([]tc.ServerNullableV11, 0, len(servers))
	for _, server := range servers {
		legacyServer, err := server.ToServerV2FromV4()
		if err != nil {
			api.HandleDeprecatedErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("failed to convert servers to legacy format: %v", err), &alternative)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("failed to convert servers to legacy format: %v", err))
			return
		}
		legacyServers = append(legacyServers, legacyServer.ServerNullableV11)
	}
	api.WriteAlertsObj(w, r, http.StatusOK, deprecationAlerts, legacyServers)
	return
}

func selectMaxLastUpdatedQuery(queryAddition string, where string) string {
	return `SELECT max(t) from (
		SELECT max(s.last_updated) as t from server s JOIN cachegroup cg ON s.cachegroup = cg.id
JOIN cdn cdn ON s.cdn_id = cdn.id
JOIN phys_location pl ON s.phys_location = pl.id
JOIN profile p ON s.profile = p.id
JOIN status st ON s.status = st.id
JOIN type t ON s.type = t.id ` +
		queryAddition + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='server') as res`
}

func getServerCount(tx *sqlx.Tx, query string, queryValues map[string]interface{}) (uint64, error) {
	var serverCount uint64
	ns, err := tx.PrepareNamed(query)
	if err != nil {
		return 0, fmt.Errorf("couldn't prepare the query to get server count : %v", err)
	}
	err = tx.NamedStmt(ns).QueryRow(queryValues).Scan(&serverCount)
	if err != nil {
		return 0, fmt.Errorf("failed to get servers count: %v", err)
	}
	return serverCount, nil
}

func getServers(h http.Header, params map[string]string, tx *sqlx.Tx, user *auth.CurrentUser, useIMS bool, version api.Version) ([]tc.ServerV40, uint64, error, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup":       {Column: "s.cachegroup", Checker: api.IsInt},
		"parentCachegroup": {Column: "cg.parent_cachegroup_id", Checker: api.IsInt},
		"cdn":              {Column: "s.cdn_id", Checker: api.IsInt},
		"id":               {Column: "s.id", Checker: api.IsInt},
		"hostName":         {Column: "s.host_name", Checker: nil},
		"physLocation":     {Column: "s.phys_location", Checker: api.IsInt},
		"profileId":        {Column: "s.profile", Checker: api.IsInt},
		"status":           {Column: "st.name", Checker: nil},
		"topology":         {Column: "tc.topology", Checker: nil},
		"type":             {Column: "t.name", Checker: nil},
		"dsId":             {Column: "dss.deliveryservice", Checker: nil},
	}

	if version.Major >= 3 {
		queryParamsToSQLCols["cachegroupName"] = dbhelpers.WhereColumnInfo{
			Column:  "cg.name",
			Checker: nil,
		}
	}

	usesMids := false
	queryAddition := ""
	dsHasRequiredCapabilities := false
	var dsID int
	var cdnID int
	var serverCount uint64
	var err error

	if dsIDStr, ok := params[`dsId`]; ok {
		// don't allow query on ds outside user's tenant
		dsID, err = strconv.Atoi(dsIDStr)
		if err != nil {
			return nil, 0, errors.New("dsId must be an integer"), nil, http.StatusNotFound, nil
		}
		cdnID, _, err = dbhelpers.GetDSCDNIdFromID(tx.Tx, dsID)
		if err != nil {
			return nil, 0, nil, err, http.StatusInternalServerError, nil
		}

		userErr, sysErr, _ := tenant.CheckID(tx.Tx, user, dsID)
		if userErr != nil || sysErr != nil {
			return nil, 0, errors.New("Forbidden"), sysErr, http.StatusForbidden, nil
		}

		var joinSubQuery string
		if version.Major >= 3 {
			if err = tx.QueryRow(deliveryservice.HasRequiredCapabilitiesQuery, dsID).Scan(&dsHasRequiredCapabilities); err != nil {
				err = fmt.Errorf("unable to get required capabilities for deliveryservice %d: %s", dsID, err)
				return nil, 0, nil, err, http.StatusInternalServerError, nil
			}
			joinSubQuery = dssTopologiesJoinSubquery
		} else {
			joinSubQuery = ""
		}
		// only if dsId is part of params: add join on deliveryservice_server table
		queryAddition = fmt.Sprintf(deliveryServiceServersJoin, joinSubQuery)

		// depending on ds type, also need to add mids
		dsType, _, err := dbhelpers.GetDeliveryServiceType(dsID, tx.Tx)
		if err != nil {
			return nil, 0, nil, err, http.StatusInternalServerError, nil
		}
		usesMids = dsType.UsesMidCache()
		log.Debugf("Servers for ds %d; uses mids? %v\n", dsID, usesMids)
	}

	if _, ok := params[`topology`]; ok {
		/* language=SQL */
		queryAddition += `
			JOIN topology_cachegroup tc ON cg."name" = tc.cachegroup
`
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if dsHasRequiredCapabilities {
		where += requiredCapabilitiesCondition
	}
	if len(errs) > 0 {
		return nil, 0, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}

	countQuery := serverCountQuery + queryAddition + where
	// If we are querying for a DS that has reqd capabilities, we need to make sure that we also include all the ORG servers directly assigned to this DS
	if _, ok := params["dsId"]; ok && dsHasRequiredCapabilities {
		countQuery = `SELECT (` + countQuery + `) + (` + serverCountQuery + originServerQuery + `) AS total`
	}
	serverCount, err = getServerCount(tx, countQuery, queryValues)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get servers count: %v", err), http.StatusInternalServerError, nil
	}

	serversList := []tc.ServerV40{}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, h, queryValues, selectMaxLastUpdatedQuery(queryAddition, where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return serversList, 0, nil, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	query := selectQuery + queryAddition + where + orderBy + pagination
	// If you're looking to get the servers for a particular delivery service, make sure you're also querying the ORG servers from the deliveryservice_server table
	if _, ok := params[`dsId`]; ok {
		query = `(` + selectQuery + queryAddition + where + orderBy + pagination + `) UNION ` + selectQuery + originServerQuery
	}

	log.Debugln("Query is ", query)
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, serverCount, nil, errors.New("querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	HiddenField := "********"

	servers := make(map[int]tc.ServerV40)
	ids := []int{}
	for rows.Next() {
		var s tc.ServerV40
		if err = rows.StructScan(&s); err != nil {
			return nil, serverCount, nil, errors.New("getting servers: " + err.Error()), http.StatusInternalServerError, nil
		}
		if user.PrivLevel < auth.PrivLevelOperations {
			s.ILOPassword = &HiddenField
			s.XMPPPasswd = &HiddenField
		}

		if s.ID == nil {
			return nil, serverCount, nil, errors.New("found server with nil ID"), http.StatusInternalServerError, nil
		}
		if _, ok := servers[*s.ID]; ok {
			return nil, serverCount, nil, fmt.Errorf("found more than one server with ID #%d", *s.ID), http.StatusInternalServerError, nil
		}
		servers[*s.ID] = s
		ids = append(ids, *s.ID)
	}

	// if ds requested uses mid-tier caches, add those to the list as well
	if usesMids {
		midIDs, userErr, sysErr, errCode := getMidServers(ids, servers, dsID, cdnID, tx, dsHasRequiredCapabilities)

		log.Debugf("getting mids: %v, %v, %s\n", userErr, sysErr, http.StatusText(errCode))

		serverCount = serverCount + uint64(len(midIDs))
		if userErr != nil || sysErr != nil {
			return nil, serverCount, userErr, sysErr, errCode, nil
		}
		ids = append(ids, midIDs...)
	}

	if len(ids) < 1 {
		return []tc.ServerV40{}, serverCount, nil, nil, http.StatusOK, nil
	}

	query, args, err := sqlx.In(`SELECT max_bandwidth, monitor, mtu, name, server, router_host_name, router_port_name FROM interface WHERE server IN (?)`, ids)
	if err != nil {
		return nil, serverCount, nil, fmt.Errorf("building interfaces query: %v", err), http.StatusInternalServerError, nil
	}
	query = tx.Rebind(query)
	interfaces := map[int]map[string]tc.ServerInterfaceInfoV40{}
	interfaceRows, err := tx.Queryx(query, args...)
	if err != nil {
		return nil, serverCount, nil, fmt.Errorf("querying for interfaces: %v", err), http.StatusInternalServerError, nil
	}
	defer interfaceRows.Close()

	for interfaceRows.Next() {
		iface := tc.ServerInterfaceInfoV40{
			ServerInterfaceInfo: tc.ServerInterfaceInfo{
				IPAddresses: []tc.ServerIPAddress{},
			},
		}
		var server int
		var routerHostName string
		var routerPort string
		if err = interfaceRows.Scan(&iface.MaxBandwidth, &iface.Monitor, &iface.MTU, &iface.Name, &server, &routerHostName, &routerPort); err != nil {
			return nil, serverCount, nil, fmt.Errorf("getting server interfaces: %v", err), http.StatusInternalServerError, nil
		}

		if _, ok := servers[server]; !ok {
			continue
		}

		if _, ok := interfaces[server]; !ok {
			interfaces[server] = map[string]tc.ServerInterfaceInfoV40{}
		}
		iface.RouterHostName = routerHostName
		iface.RouterPortName = routerPort
		interfaces[server][iface.Name] = iface
	}

	query, args, err = sqlx.In(`SELECT address, gateway, service_address, server, interface FROM ip_address WHERE server IN (?)`, ids)
	if err != nil {
		return nil, serverCount, nil, fmt.Errorf("building IP addresses query: %v", err), http.StatusInternalServerError, nil
	}
	query = tx.Rebind(query)
	ipRows, err := tx.Tx.Query(query, args...)
	if err != nil {
		return nil, serverCount, nil, fmt.Errorf("querying for IP addresses: %v", err), http.StatusInternalServerError, nil
	}
	defer ipRows.Close()

	for ipRows.Next() {
		var ip tc.ServerIPAddress
		var server int
		var iface string

		if err = ipRows.Scan(&ip.Address, &ip.Gateway, &ip.ServiceAddress, &server, &iface); err != nil {
			return nil, serverCount, nil, fmt.Errorf("getting server IP addresses: %v", err), http.StatusInternalServerError, nil
		}

		if _, ok := interfaces[server]; !ok {
			continue
		}
		if i, ok := interfaces[server][iface]; !ok {
			log.Warnf("IP addresses query returned addresses for an interface that was not found in interfaces query: %s", iface)
		} else {
			i.IPAddresses = append(i.IPAddresses, ip)
			interfaces[server][iface] = i
		}
	}

	returnable := make([]tc.ServerV40, 0, len(ids))

	for _, id := range ids {
		server := servers[id]
		for _, iface := range interfaces[id] {
			server.Interfaces = append(server.Interfaces, iface)
		}
		returnable = append(returnable, server)
	}

	return returnable, serverCount, nil, nil, http.StatusOK, &maxTime
}

// getMidServers gets the mids used by the edges provided with an option to filter for a given cdn
func getMidServers(edgeIDs []int, servers map[int]tc.ServerV40, dsID int, cdnID int, tx *sqlx.Tx, includeCapabilities bool) ([]int, error, error, int) {
	if len(edgeIDs) == 0 {
		return nil, nil, nil, http.StatusOK
	}

	filters := map[string]interface{}{
		"cache_type_mid": tc.CacheTypeMid,
		"edge_ids":       pq.Array(edgeIDs),
		"ds_id":          dsID,
	}

	midIDs := []int{}
	query := ""
	if includeCapabilities {
		// Query to select the associated mids for this DS
		q := selectIDQuery + midWhereClause
		rows, err := tx.NamedQuery(q, filters)
		if err != nil {
			return nil, err, nil, http.StatusBadRequest
		}
		defer rows.Close()

		for rows.Next() {
			var midID int
			if err := rows.Scan(&midID); err != nil {
				log.Errorf("could not scan mid server id: %s\n", err)
				return nil, nil, err, http.StatusInternalServerError
			}
			midIDs = append(midIDs, midID)
		}
		filters["mid_ids"] = pq.Array(midIDs)

		// Query to select only those mids that match the required capabilities of the DS
		query = selectQuery + midWhereClause + `
		AND s.id IN (
		WITH capabilities AS (
		SELECT ARRAY_AGG(ssc.server_capability), server
		FROM server_server_capability ssc
		WHERE ssc.server = ANY(:mid_ids)
		GROUP BY server)
		SELECT server
		FROM capabilities WHERE
		capabilities.array_agg
		@>
		(
		SELECT ARRAY_AGG(drc.required_capability)
		FROM deliveryservices_required_capability drc
		WHERE drc.deliveryservice_id=:ds_id)
		)`
	} else {
		// TODO: include secondary parent?
		query = selectQuery + midWhereClause
	}

	if cdnID > 0 {
		query += ` AND s.cdn_id = :cdn_id`
		filters["cdn_id"] = cdnID
	}

	rows, err := tx.NamedQuery(query, filters)
	if err != nil {
		return nil, err, nil, http.StatusBadRequest
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var s tc.ServerV40
		if err := rows.StructScan(&s); err != nil {
			log.Errorf("could not scan mid servers: %s\n", err)
			return nil, nil, err, http.StatusInternalServerError
		}
		if s.ID == nil {
			return nil, nil, errors.New("found server with nil ID"), http.StatusInternalServerError
		}

		// This may mean that the server was caught by other query parameters,
		// so not technically an error, unlike earlier in 'getServers'.
		if _, ok := servers[*s.ID]; ok {
			continue
		}

		servers[*s.ID] = s
		ids = append(ids, *s.ID)
	}

	return ids, nil, nil, http.StatusOK
}

func checkTypeChangeSafety(server tc.CommonServerProperties, tx *sqlx.Tx) (error, error, int) {
	// see if cdn or type changed
	var cdnID int
	var typeID int
	if err := tx.QueryRow("SELECT type, cdn_id FROM server WHERE id = $1", *server.ID).Scan(&typeID, &cdnID); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("no server found with this ID"), nil, http.StatusNotFound
		}
		return nil, fmt.Errorf("getting current server type: %v", err), http.StatusInternalServerError
	}

	var dsIDs []int64
	if err := tx.QueryRowx("SELECT ARRAY(SELECT deliveryservice FROM deliveryservice_server WHERE server = $1)", server.ID).Scan(pq.Array(&dsIDs)); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("getting server assigned delivery services: %v", err), http.StatusInternalServerError
	}
	// If type is changing ensure it isn't assigned to any DSes.
	if typeID != *server.TypeID {
		if len(dsIDs) != 0 {
			return errors.New("server type can not be updated when it is currently assigned to Delivery Services"), nil, http.StatusConflict
		}
	}
	// Check to see if the user is trying to change the CDN of a server, which is already linked with a DS
	if cdnID != *server.CDNID && len(dsIDs) != 0 {
		return errors.New("server cdn can not be updated when it is currently assigned to delivery services"), nil, http.StatusConflict
	}

	return nil, nil, http.StatusOK
}

func updateStatusLastUpdatedTime(id int, statusLastUpdatedTime *time.Time, tx *sql.Tx) (error, error, int) {
	query := `UPDATE server SET
	status_last_updated=$1
WHERE id=$2 `
	if _, err := tx.Exec(query, statusLastUpdatedTime, id); err != nil {
		return errors.New("updating status last updated: " + err.Error()), nil, http.StatusInternalServerError
	}
	return nil, nil, http.StatusOK
}

func createInterfaces(id int, interfaces []tc.ServerInterfaceInfoV40, tx *sql.Tx) (error, error, int) {
	ifaceQry := `
	INSERT INTO interface (
		max_bandwidth,
		monitor,
		mtu,
		name,
		server,
		router_host_name,
		router_port_name
	) VALUES
	`
	ipQry := `
	INSERT INTO ip_address (
		address,
		gateway,
		interface,
		server,
		service_address
	) VALUES
	`

	ifaceQueryParts := make([]string, 0, len(interfaces))
	ipQueryParts := make([]string, 0, len(interfaces))
	ifaceArgs := make([]interface{}, 0, len(interfaces))
	ipArgs := make([]interface{}, 0, len(interfaces))
	for i, iface := range interfaces {
		argStart := i * 7
		ifaceQueryParts = append(ifaceQueryParts, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", argStart+1, argStart+2, argStart+3, argStart+4, argStart+5, argStart+6, argStart+7))
		ifaceArgs = append(ifaceArgs, iface.MaxBandwidth, iface.Monitor, iface.MTU, iface.Name, id, iface.RouterHostName, iface.RouterPortName)
		for _, ip := range iface.IPAddresses {
			argStart = len(ipArgs)
			ipQueryParts = append(ipQueryParts, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", argStart+1, argStart+2, argStart+3, argStart+4, argStart+5))
			ipArgs = append(ipArgs, ip.Address, ip.Gateway, iface.Name, id, ip.ServiceAddress)
		}
	}

	ifaceQry += strings.Join(ifaceQueryParts, ",")
	log.Debugf("Inserting interfaces for new server, query is: %s", ifaceQry)

	_, err := tx.Exec(ifaceQry, ifaceArgs...)
	if err != nil {
		return api.ParseDBError(err)
	}

	ipQry += strings.Join(ipQueryParts, ",")
	log.Debugf("Inserting IP addresses for new server, query is: %s", ipQry)

	_, err = tx.Exec(ipQry, ipArgs...)
	if err != nil {
		return api.ParseDBError(err)
	}
	return nil, nil, http.StatusOK
}

func deleteInterfaces(id int, tx *sql.Tx) (error, error, int) {
	if _, err := tx.Exec(deleteIPsQuery, id); err != nil && err != sql.ErrNoRows {
		return api.ParseDBError(err)
	}

	if _, err := tx.Exec(deleteInterfacesQuery, id); err != nil && err != sql.ErrNoRows {
		return api.ParseDBError(err)
	}

	return nil, nil, http.StatusOK
}

// Update is the handler for PUT requests to /servers.
func Update(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	// Middleware should've already handled this, so idk why this is a pointer at all tbh
	version := inf.Version
	if version == nil {
		middleware.NotImplementedHandler().ServeHTTP(w, r)
		return
	}

	id := inf.IntParams["id"]

	// Get original server
	originals, _, userErr, sysErr, errCode, _ := getServers(r.Header, inf.Params, inf.Tx, inf.User, false, *version)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	if len(originals) < 1 {
		api.HandleErr(w, r, tx, http.StatusNotFound, errors.New("the server doesn't exist, cannot update"), nil)
		return
	}
	if len(originals) > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("too many servers by ID %d: %d", id, len(originals)))
		return
	}

	original := originals[0]
	if original.XMPPID == nil || *original.XMPPID == "" {
		log.Warnf("original server %s had no XMPPID\n", *original.HostName)
	}
	if original.StatusID == nil {
		sysErr = errors.New("original server had no status ID")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	if original.Status == nil {
		sysErr = errors.New("original server had no status name")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	if original.CachegroupID == nil {
		sysErr = errors.New("original server had no Cache Group ID")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	if original.StatusLastUpdated == nil {
		log.Warnln("original server had no Status Last Updated time")
		if original.LastUpdated == nil {
			sysErr = errors.New("original server had no Last Updated time")
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		original.StatusLastUpdated = &original.LastUpdated.Time
	}

	var originalXMPPID string
	if original.XMPPID != nil {
		originalXMPPID = *original.XMPPID
	}
	originalStatusID := *original.StatusID

	var server tc.ServerV40
	var serverV3 tc.ServerV30
	var statusLastUpdatedTime time.Time
	if inf.Version.Major >= 4 {
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}
		if server.StatusID != nil && *server.StatusID != originalStatusID {
			currentTime := time.Now()
			server.StatusLastUpdated = &currentTime
			statusLastUpdatedTime = currentTime
		} else {
			server.StatusLastUpdated = original.StatusLastUpdated
			statusLastUpdatedTime = *original.StatusLastUpdated
		}
		_, err := validateV4(&server, tx)
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}
	} else if inf.Version.Major >= 3 {
		if err := json.NewDecoder(r.Body).Decode(&serverV3); err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}
		if serverV3.StatusID != nil && *serverV3.StatusID != originalStatusID {
			currentTime := time.Now()
			serverV3.StatusLastUpdated = &currentTime
			statusLastUpdatedTime = currentTime
		} else {
			serverV3.StatusLastUpdated = original.StatusLastUpdated
			statusLastUpdatedTime = *original.StatusLastUpdated
		}
		_, err := validateV3(&serverV3, tx)
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}
		server, err = serverV3.UpgradeToV40()
		if err != nil {
			sysErr = fmt.Errorf("error upgrading valid V3 server to V4 structure: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
	} else if inf.Version.Major == 2 {
		var legacyServer tc.ServerNullableV2
		if err := json.NewDecoder(r.Body).Decode(&legacyServer); err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}
		err := validateV2(&legacyServer, tx)
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}

		server, err = legacyServer.UpgradeToV40()
		if err != nil {
			sysErr = fmt.Errorf("error upgrading valid V2 server to V3 structure: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
	} else {
		var legacyServer tc.ServerNullableV11
		if err := json.NewDecoder(r.Body).Decode(&legacyServer); err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}

		err := validateV1(&legacyServer, tx)
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
			return
		}

		v2Server := tc.ServerNullableV2{
			ServerNullableV11: legacyServer,
			IPIsService:       util.BoolPtr(true),
			IP6IsService:      util.BoolPtr(true),
		}

		server, err = v2Server.UpgradeToV40()
		if err != nil {
			sysErr = fmt.Errorf("error upgrading valid V1 server to V3 structure: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
	}

	if *original.CachegroupID != *server.CachegroupID || *original.CDNID != *server.CDNID {
		hasDSOnCDN, err := dbhelpers.CachegroupHasTopologyBasedDeliveryServicesOnCDN(tx, *original.CachegroupID, *original.CDNID)
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
			return
		}
		CDNIDs := []int{}
		if hasDSOnCDN {
			CDNIDs = append(CDNIDs, *original.CDNID)
		}
		cacheGroupIds := []int{*original.CachegroupID}
		serverIds := []int{*original.ID}
		if err = topology_validation.CheckForEmptyCacheGroups(inf.Tx, cacheGroupIds, CDNIDs, true, serverIds); err != nil {
			api.HandleErr(w, r, tx, http.StatusBadRequest, errors.New("server is the last one in its cachegroup, which is used by a topology, so it cannot be moved to another cachegroup: "+err.Error()), nil)
			return
		}
	}

	server.ID = new(int)
	*server.ID = inf.IntParams["id"]
	status, ok, err := dbhelpers.GetStatusByID(*server.StatusID, tx)
	if err != nil {
		sysErr = fmt.Errorf("getting server #%d status (#%d): %v", id, *server.StatusID, err)
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	if !ok {
		log.Warnf("previously existent status #%d not found when fetching later", *server.StatusID)
		userErr = fmt.Errorf("no such Status: #%d", *server.StatusID)
		api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, nil)
		return
	}
	if status.Name == nil {
		sysErr = fmt.Errorf("status #%d had no name", *server.StatusID)
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	if *status.Name != string(tc.CacheStatusOnline) && *status.Name != string(tc.CacheStatusReported) {
		dsIDs, err := getActiveDeliveryServicesThatOnlyHaveThisServerAssigned(id, tx)
		if err != nil {
			sysErr = fmt.Errorf("getting Delivery Services to which server #%d is assigned that have no other servers: %v", id, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if len(dsIDs) > 0 {
			alertText := InvalidStatusForDeliveryServicesAlertText(*status.Name, dsIDs)
			api.WriteAlerts(w, r, http.StatusConflict, tc.CreateAlerts(tc.ErrorLevel, alertText))
			return
		}
	}

	if userErr, sysErr, errCode = checkTypeChangeSafety(server.CommonServerProperties, inf.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if server.XMPPID != nil && *server.XMPPID != "" && originalXMPPID != "" && *server.XMPPID != originalXMPPID {
		api.WriteAlerts(w, r, http.StatusBadRequest, tc.CreateAlerts(tc.ErrorLevel, fmt.Sprintf("server cannot be updated due to requested XMPPID change. XMPIDD is immutable")))
		return
	}

	userErr, sysErr, statusCode := api.CheckIfUnModified(r.Header, inf.Tx, *server.ID, "server")
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
		return
	}

	rows, err := inf.Tx.NamedQuery(updateQuery, server)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer rows.Close()

	rowsAffected := 0
	for rows.Next() {
		if err := rows.StructScan(&server); err != nil {
			api.HandleErr(w, r, tx, http.StatusNotFound, nil, fmt.Errorf("scanning lastUpdated from server insert: %v", err))
			return
		}
		rowsAffected++
	}

	if rowsAffected < 1 {
		api.HandleErr(w, r, tx, http.StatusNotFound, errors.New("no server found with this id"), nil)
		return
	}
	if rowsAffected > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("update for server #%d affected too many rows (%d)", *server.ID, rowsAffected))
		return
	}

	if userErr, sysErr, errCode = deleteInterfaces(id, tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if userErr, sysErr, errCode = createInterfaces(id, server.Interfaces, tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if inf.Version.Major >= 3 {
		if userErr, sysErr, errCode = updateStatusLastUpdatedTime(id, &statusLastUpdatedTime, tx); userErr != nil || sysErr != nil {
			api.HandleErr(w, r, tx, errCode, userErr, sysErr)
			return
		}
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Server updated", server)
	} else {
		v2Server, err := server.ToServerV2FromV4()
		if err != nil {
			sysErr = fmt.Errorf("converting valid v3 server to a v2 structure: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if inf.Version.Major <= 1 {
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Server updated", v2Server.ServerNullableV11)
		} else {
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Server updated", v2Server)
		}
	}

	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: updated", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

func createV1(inf *api.APIInfo, w http.ResponseWriter, r *http.Request) {
	var server tc.ServerNullableV11

	tx := inf.Tx.Tx

	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	if server.ID != nil {
		var prevID int
		err := tx.QueryRow("SELECT id from server where id = $1", server.ID).Scan(&prevID)
		if err != nil && err != sql.ErrNoRows {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("checking if server with id %d exists", *server.ID))
			return
		}
		if prevID != 0 {
			api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("server with id %d already exists. Please do not provide an id", *server.ID), nil)
			return
		}
	}

	if err := validateV1(&server, tx); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	resultRows, err := inf.Tx.NamedQuery(insertQuery, server)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.StructScan(&server.CommonServerProperties); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("server create scanning: %v", err))
			return
		}
	}
	if rowsAffected == 0 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("server create: no server was inserted, no id was returned"))
		return
	} else if rowsAffected > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("too many ids returned from server insert"))
	}

	ifaces, err := server.LegacyInterfaceDetails.ToInterfacesV4(true, true, server.RouterHostName, server.RouterPortName)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	}

	if userErr, sysErr, errCode := createInterfaces(*server.ID, ifaces, tx); userErr != nil || sysErr != nil || errCode != http.StatusOK {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	alerts := tc.CreateAlerts(tc.SuccessLevel, "server was created.")
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, server)

	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: created", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

func createV2(inf *api.APIInfo, w http.ResponseWriter, r *http.Request) {
	var server tc.ServerNullableV2

	tx := inf.Tx.Tx

	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	if server.ID != nil {
		var prevID int
		err := tx.QueryRow("SELECT id from server where id = $1", server.ID).Scan(&prevID)
		if err != nil && err != sql.ErrNoRows {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("checking if server with id %d exists", *server.ID))
			return
		}
		if prevID != 0 {
			api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("server with id %d already exists. Please do not provide an id", *server.ID), nil)
			return
		}
	}

	server.XMPPID = newUUID()

	if err := validateV2(&server, tx); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	resultRows, err := inf.Tx.NamedQuery(insertQuery, server)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.StructScan(&server.CommonServerProperties); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("server create scanning: %v", err))
			return
		}
	}
	if rowsAffected == 0 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("server create: no server was inserted, no id was returned"))
		return
	} else if rowsAffected > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("too many ids returned from server insert"))
	}

	ifaces, err := server.LegacyInterfaceDetails.ToInterfacesV4(*server.IPIsService, *server.IP6IsService, server.RouterHostName, server.RouterPortName)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
	}

	if userErr, sysErr, errCode := createInterfaces(*server.ID, ifaces, tx); err != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	alerts := tc.CreateAlerts(tc.SuccessLevel, "server was created.")
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, server)

	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: created", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

func createV3(inf *api.APIInfo, w http.ResponseWriter, r *http.Request) {
	var server tc.ServerV30

	tx := inf.Tx.Tx

	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	if server.ID != nil {
		var prevID int
		err := tx.QueryRow("SELECT id from server where id = $1", server.ID).Scan(&prevID)
		if err != nil && err != sql.ErrNoRows {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("checking if server with id %d exists", *server.ID))
			return
		}
		if prevID != 0 {
			api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("server with id %d already exists. Please do not provide an id", *server.ID), nil)
			return
		}
	}

	server.XMPPID = newUUID()
	_, err := validateV3(&server, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	currentTime := time.Now()
	server.StatusLastUpdated = &currentTime

	resultRows, err := inf.Tx.NamedQuery(insertQueryV3, server)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.StructScan(&server.CommonServerProperties); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("server create scanning: %v", err))
			return
		}
	}
	if rowsAffected == 0 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("server create: no server was inserted, no id was returned"))
		return
	} else if rowsAffected > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("too many ids returned from server insert"))
		return
	}

	interfaces, err := tc.ToInterfacesV4(server.Interfaces, server.RouterHostName, server.RouterPortName)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
	}
	userErr, sysErr, errCode := createInterfaces(*server.ID, interfaces, tx)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	alerts := tc.CreateAlerts(tc.SuccessLevel, "Server created")
	api.WriteAlertsObj(w, r, http.StatusCreated, alerts, server)

	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: created", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

func createV4(inf *api.APIInfo, w http.ResponseWriter, r *http.Request) {
	var server tc.ServerV40

	tx := inf.Tx.Tx

	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}
	if server.ID != nil {
		var prevID int
		err := tx.QueryRow("SELECT id from server where id = $1", server.ID).Scan(&prevID)
		if err != nil && err != sql.ErrNoRows {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("checking if server with id %d exists", *server.ID))
			return
		}
		if prevID != 0 {
			api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("server with id %d already exists. Please do not provide an id", *server.ID), nil)
			return
		}
	}

	str := uuid.New().String()
	server.XMPPID = &str
	_, err := validateV4(&server, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	currentTime := time.Now()
	server.StatusLastUpdated = &currentTime

	resultRows, err := inf.Tx.NamedQuery(insertQueryV4, server)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.StructScan(&server.CommonServerProperties); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("server create scanning: %v", err))
			return
		}
	}
	if rowsAffected == 0 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("server create: no server was inserted, no id was returned"))
		return
	} else if rowsAffected > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("too many ids returned from server insert"))
		return
	}

	userErr, sysErr, errCode := createInterfaces(*server.ID, server.Interfaces, tx)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	alerts := tc.CreateAlerts(tc.SuccessLevel, "Server created")
	api.WriteAlertsObj(w, r, http.StatusCreated, alerts, server)

	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: created", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

// Create is the handler for POST requests to /servers.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	switch {
	case inf.Version.Major <= 1:
		createV1(inf, w, r)
	case inf.Version.Major == 2:
		createV2(inf, w, r)
	case inf.Version.Major == 3:
		createV3(inf, w, r)
	default:
		createV4(inf, w, r)
	}
}

const lastServerOfDSesQuery = `
SELECT "deliveryservice"
FROM deliveryservice_server
WHERE "deliveryservice" IN (
	SELECT deliveryservice_server.deliveryservice
	FROM deliveryservice_server
	INNER JOIN deliveryservice
	ON deliveryservice_server.deliveryservice = deliveryservice.id
	WHERE deliveryservice_server."server" = $1
	AND deliveryservice.active = 'ACTIVE'
)
GROUP BY "deliveryservice"
HAVING COUNT("server") = 1;
`

func getActiveDeliveryServicesThatOnlyHaveThisServerAssigned(id int, tx *sql.Tx) ([]int, error) {
	var ids []int
	if tx == nil {
		return ids, errors.New("nil transaction")
	}

	rows, err := tx.Query(lastServerOfDSesQuery, id)
	if err != nil {
		return ids, fmt.Errorf("querying: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dsID int
		err = rows.Scan(&dsID)
		if err != nil {
			return ids, fmt.Errorf("scanning: %v", err)
		}
		ids = append(ids, dsID)
	}

	return ids, nil
}

// Delete is the handler for DELETE requests to the /servers API endpoint.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	// Middleware should've already handled this, so idk why this is a pointer at all tbh
	version := inf.Version
	if version == nil {
		middleware.NotImplementedHandler().ServeHTTP(w, r)
		return
	}

	id := inf.IntParams["id"]

	if dsIDs, err := getActiveDeliveryServicesThatOnlyHaveThisServerAssigned(id, tx); err != nil {
		sysErr = fmt.Errorf("checking if server #%d is the last server assigned to any Delivery Services: %v", id, err)
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	} else if len(dsIDs) > 0 {
		alertText := fmt.Sprintf("deleting server #%d would leave Active Delivery Service", id)
		if len(dsIDs) == 1 {
			alertText += fmt.Sprintf(" #%d", dsIDs[0])
		} else if len(dsIDs) == 2 {
			alertText += fmt.Sprintf("s #%d and #%d", dsIDs[0], dsIDs[1])
		} else {
			dsNums := make([]string, 0, len(dsIDs)-1)
			for _, dsID := range dsIDs[:len(dsIDs)-1] {
				dsNums = append(dsNums, "#"+strconv.Itoa(dsID))
			}
			alertText += fmt.Sprintf("s %s, and #%d", strings.Join(dsNums, ", "), dsIDs[len(dsIDs)-1])
		}
		alertText += fmt.Sprintf("  with no '%s' or '%s' servers", tc.CacheStatusOnline, tc.CacheStatusReported)

		api.WriteAlerts(w, r, http.StatusConflict, tc.CreateAlerts(tc.ErrorLevel, alertText))
		return
	}

	var servers []tc.ServerV40
	servers, _, userErr, sysErr, errCode, _ = getServers(r.Header, map[string]string{"id": inf.Params["id"]}, inf.Tx, inf.User, false, *version)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if len(servers) < 1 {
		api.HandleErr(w, r, tx, http.StatusNotFound, fmt.Errorf("no server exists by id #%d", id), nil)
		return
	}
	if len(servers) > 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("there are somehow two servers with id %d - cannot delete", id))
		return
	}
	server := servers[0]
	cacheGroupIds := []int{*server.CachegroupID}
	serverIds := []int{*server.ID}
	hasDSOnCDN, err := dbhelpers.CachegroupHasTopologyBasedDeliveryServicesOnCDN(inf.Tx.Tx, *server.CachegroupID, *server.CDNID)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	}
	CDNIDs := []int{}
	if hasDSOnCDN {
		CDNIDs = append(CDNIDs, *server.CDNID)
	}
	if err := topology_validation.CheckForEmptyCacheGroups(inf.Tx, cacheGroupIds, CDNIDs, true, serverIds); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, errors.New("server is the last one in its cachegroup, which is used by a topology: "+err.Error()), nil)
		return
	}

	if result, err := tx.Exec(deleteServerQuery, id); err != nil {
		log.Errorf("Raw error: %v", err)
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	} else if rowsAffected, err := result.RowsAffected(); err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("getting rows affected by server delete: %v", err))
		return
	} else if rowsAffected != 1 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("incorrect number of rows affected: %d", rowsAffected))
		return
	}

	if inf.Version.Major >= 3 {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Server deleted", server)
	} else {

		serverV2, err := server.ToServerV2FromV4()
		if err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
			return
		}

		if inf.Version.Major <= 1 {
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, "server was deleted.", serverV2.ServerNullableV11)
		} else {
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, "server was deleted.", serverV2)
		}
	}
	changeLogMsg := fmt.Sprintf("SERVER: %s.%s, ID: %d, ACTION: deleted", *server.HostName, *server.DomainName, *server.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}
