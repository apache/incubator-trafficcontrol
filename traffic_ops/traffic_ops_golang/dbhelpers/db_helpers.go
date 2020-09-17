package dbhelpers

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

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/lib/pq"
)

type WhereColumnInfo struct {
	Column  string
	Checker func(string) error
}

const BaseWhere = "\nWHERE"
const BaseOrderBy = "\nORDER BY"
const BaseLimit = "\nLIMIT"
const BaseOffset = "\nOFFSET"

const getDSTenantIDFromXMLIDQuery = `
SELECT deliveryservice.tenant_id
FROM deliveryservice
WHERE deliveryservice.xml_id = $1
`

const getFederationIDForUserIDByXMLIDQuery = `
SELECT federation_deliveryservice.federation
FROM federation_deliveryservice
WHERE federation_deliveryservice.deliveryservice IN (
	SELECT deliveryservice.id
	FROM deliveryservice
	WHERE deliveryservice.xml_id = $1
) AND federation_deliveryservice.federation IN (
	SELECT federation_tmuser.federation
	FROM federation_tmuser
	WHERE federation_tmuser.tm_user = $2
)
`

const getUserBaseQuery = `
SELECT tm_user.address_line1,
       tm_user.address_line2,
       tm_user.city,
       tm_user.company,
       tm_user.country,
       tm_user.email,
       tm_user.full_name,
       tm_user.gid,
       tm_user.id,
       tm_user.last_updated,
       tm_user.new_user,
       tm_user.phone_number,
       tm_user.postal_code,
       tm_user.public_ssh_key,
       tm_user.registration_sent,
       tm_user.role,
       role.name AS role_name,
       tm_user.state_or_province,
       tenant.name AS tenant,
       tm_user.tenant_id,
       tm_user.token,
       tm_user.uid,
       tm_user.username
FROM tm_user
LEFT OUTER JOIN role ON role.id = tm_user.role
LEFT OUTER JOIN tenant ON tenant.id = tm_user.tenant_id
`
const getUserByIDQuery = getUserBaseQuery + `
WHERE tm_user.id = $1
`

const getUserByEmailQuery = getUserBaseQuery + `
WHERE tm_user.email = $1
`

func BuildWhereAndOrderByAndPagination(parameters map[string]string, queryParamsToSQLCols map[string]WhereColumnInfo) (string, string, string, map[string]interface{}, []error) {
	whereClause := BaseWhere
	orderBy := BaseOrderBy
	paginationClause := BaseLimit
	var criteria string
	var queryValues map[string]interface{}
	var errs []error
	criteria, queryValues, errs = parseCriteriaAndQueryValues(queryParamsToSQLCols, parameters)

	if len(queryValues) > 0 {
		whereClause += " " + criteria
	}
	if len(errs) > 0 {
		return "", "", "", queryValues, errs
	}

	if orderby, ok := parameters["orderby"]; ok {
		log.Debugln("orderby: ", orderby)
		if colInfo, ok := queryParamsToSQLCols[orderby]; ok {
			log.Debugln("orderby column ", colInfo)
			orderBy += " " + colInfo.Column

			// if orderby is specified and valid, also check for sortOrder
			if sortOrder, exists := parameters["sortOrder"]; exists {
				log.Debugln("sortOrder: ", sortOrder)
				if sortOrder == "desc" {
					orderBy += " DESC"
				} else if sortOrder != "asc" {
					log.Debugln("sortOrder value must be desc or asc. Invalid value provided: ", sortOrder)
				}
			}
		} else {
			log.Debugln("This column is not configured to support orderby: ", orderby)
		}
	}

	if limit, exists := parameters["limit"]; exists {
		// try to convert to int, if it fails the limit parameter is invalid, so return an error
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt < -1 {
			errs = append(errs, errors.New("limit parameter must be bigger than -1"))
			return "", "", "", queryValues, errs
		}
		log.Debugln("limit: ", limit)
		if limitInt == -1 {
			paginationClause = ""
		} else {
			paginationClause += " " + limit
		}
		if offset, exists := parameters["offset"]; exists {
			// check that offset is valid
			offsetInt, err := strconv.Atoi(offset)
			if err != nil || offsetInt < 1 {
				errs = append(errs, errors.New("offset parameter must be a positive integer"))
				return "", "", "", queryValues, errs
			}
			paginationClause += BaseOffset + " " + offset
		} else if page, exists := parameters["page"]; exists {
			// check that offset is valid
			page, err := strconv.Atoi(page)
			if err != nil || page < 1 {
				errs = append(errs, errors.New("page parameter must be a positive integer"))
				return "", "", "", queryValues, errs
			}
			paginationClause += BaseOffset + " " + strconv.Itoa((page-1)*limitInt)
		}
	}

	if whereClause == BaseWhere {
		whereClause = ""
	}
	if orderBy == BaseOrderBy {
		orderBy = ""
	}
	if paginationClause == BaseLimit {
		paginationClause = ""
	}
	log.Debugf("\n--\n Where: %s \n Order By: %s \n Limit+Offset: %s", whereClause, orderBy, paginationClause)
	return whereClause, orderBy, paginationClause, queryValues, errs
}

func parseCriteriaAndQueryValues(queryParamsToSQLCols map[string]WhereColumnInfo, parameters map[string]string) (string, map[string]interface{}, []error) {
	var criteria string

	var criteriaArgs []string
	errs := []error{}
	queryValues := make(map[string]interface{})
	for key, colInfo := range queryParamsToSQLCols {
		if urlValue, ok := parameters[key]; ok {
			var err error
			if colInfo.Checker != nil {
				err = colInfo.Checker(urlValue)
			}
			if err != nil {
				errs = append(errs, errors.New(key+" "+err.Error()))
			} else {
				criteria = colInfo.Column + "=:" + key
				criteriaArgs = append(criteriaArgs, criteria)
				queryValues[key] = urlValue
			}
		}
	}
	criteria = strings.Join(criteriaArgs, " AND ")

	return criteria, queryValues, errs
}

// AddTenancyCheck takes a WHERE clause (can be ""), the associated queryValues
// (can be empty), a tenantColumnName that should provide a bigint corresponding
// to the tenantID of the object being checked (this may require a CAST), and an
// array of the tenantIDs the user has access to; it returns a where clause and
// associated queryValues including filtering based on tenancy.
func AddTenancyCheck(where string, queryValues map[string]interface{}, tenantColumnName string, tenantIDs []int) (string, map[string]interface{}) {
	if where == "" {
		where = BaseWhere + " " + tenantColumnName + " = ANY(CAST(:accessibleTenants AS bigint[]))"
	} else {
		where += " AND " + tenantColumnName + " = ANY(CAST(:accessibleTenants AS bigint[]))"
	}
	queryValues["accessibleTenants"] = pq.Array(tenantIDs)

	return where, queryValues
}

// CommitIf commits if doCommit is true at the time of execution.
// This is designed as a defer helper.
//
// Example:
//
//  tx, err := db.Begin()
//  txCommit := false
//  defer dbhelpers.CommitIf(tx, &txCommit)
//  if err := tx.Exec("select ..."); err != nil {
//    return errors.New("executing: " + err.Error())
//  }
//  txCommit = true
//  return nil
//
func CommitIf(tx *sql.Tx, doCommit *bool) {
	if *doCommit {
		tx.Commit()
	} else {
		tx.Rollback()
	}
}

// GetPrivLevelFromRoleID returns the priv_level associated with a role, whether it exists, and any error.
// This method exists on a temporary basis. After priv_level is fully deprecated and capabilities take over,
// this method will not only no longer be needed, but the corresponding new privilege check should be done
// via the primary database query for the users endpoint. The users json response will contain a list of
// capabilities in the future, whereas now the users json response currently does not contain privLevel.
// See the wiki page on the roles/capabilities as a system:
// https://cwiki.apache.org/confluence/pages/viewpage.action?pageId=68715910
func GetPrivLevelFromRoleID(tx *sql.Tx, id int) (int, bool, error) {
	var privLevel int
	err := tx.QueryRow(`SELECT priv_level FROM role WHERE role.id = $1`, id).Scan(&privLevel)

	if err == sql.ErrNoRows {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, fmt.Errorf("getting priv_level from role: %v", err)
	}
	return privLevel, true, nil
}

// GetDSNameFromID loads the DeliveryService's xml_id from the database, from the ID. Returns whether the delivery service was found, and any error.
func GetDSNameFromID(tx *sql.Tx, id int) (tc.DeliveryServiceName, bool, error) {
	name := tc.DeliveryServiceName("")
	if err := tx.QueryRow(`SELECT xml_id FROM deliveryservice WHERE id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return tc.DeliveryServiceName(""), false, nil
		}
		return tc.DeliveryServiceName(""), false, fmt.Errorf("querying xml_id for delivery service ID '%v': %v", id, err)
	}
	return name, true, nil
}

// GetDSCDNIdFromID loads the DeliveryService's cdn ID from the database, from the delivery service ID. Returns whether the delivery service was found, and any error.
func GetDSCDNIdFromID(tx *sql.Tx, dsID int) (int, bool, error) {
	var cdnID int
	if err := tx.QueryRow(`SELECT cdn_id FROM deliveryservice WHERE id = $1`, dsID).Scan(&cdnID); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("querying cdn_id for delivery service ID '%v': %v", dsID, err)
	}
	return cdnID, true, nil
}

// GetDSTenantIDFromXMLID fetches the ID of the Tenant to whom the Delivery Service identified by the
// the provided XMLID belongs. It returns, in order, the requested ID (if one could be found), a
// boolean indicating whether or not a Delivery Service with the provided xmlid could be found, and
// an error for logging in case something unexpected goes wrong.
func GetDSTenantIDFromXMLID(tx *sql.Tx, xmlid string) (int, bool, error) {
	var id int
	if err := tx.QueryRow(getDSTenantIDFromXMLIDQuery, xmlid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return -1, false, nil
		}
		return -1, false, fmt.Errorf("Fetching Tenant ID for DS %s: %v", xmlid, err)
	}
	return id, true, nil
}

// returns returns the delivery service name and cdn, whether it existed, and any error.
func GetDSNameAndCDNFromID(tx *sql.Tx, id int) (tc.DeliveryServiceName, tc.CDNName, bool, error) {
	name := tc.DeliveryServiceName("")
	cdn := tc.CDNName("")
	if err := tx.QueryRow(`
SELECT ds.xml_id, cdn.name
FROM deliveryservice as ds
JOIN cdn on cdn.id = ds.cdn_id
WHERE ds.id = $1
`, id).Scan(&name, &cdn); err != nil {
		if err == sql.ErrNoRows {
			return tc.DeliveryServiceName(""), tc.CDNName(""), false, nil
		}
		return tc.DeliveryServiceName(""), tc.CDNName(""), false, errors.New("querying delivery service name: " + err.Error())
	}
	return name, cdn, true, nil
}

// GetDSIDAndCDNFromName returns the delivery service ID and cdn name given from the delivery service name, whether a result existed, and any error.
func GetDSIDAndCDNFromName(tx *sql.Tx, xmlID string) (int, tc.CDNName, bool, error) {
	dsId := 0
	cdn := tc.CDNName("")
	if err := tx.QueryRow(`
SELECT ds.id, cdn.name
FROM deliveryservice as ds
JOIN cdn on cdn.id = ds.cdn_id
WHERE ds.xml_id = $1
`, xmlID).Scan(&dsId, &cdn); err != nil {
		if err == sql.ErrNoRows {
			return dsId, tc.CDNName(""), false, nil
		}
		return dsId, tc.CDNName(""), false, errors.New("querying delivery service name: " + err.Error())
	}
	return dsId, cdn, true, nil
}

// GetFederationResolversByFederationID fetches all of the federation resolvers currently assigned to a federation.
// In the event of an error, it will return an empty slice and the error.
func GetFederationResolversByFederationID(tx *sql.Tx, fedID int) ([]tc.FederationResolver, error) {
	qry := `
		SELECT
		  fr.ip_address,
		  frt.name as resolver_type,
		  ffr.federation_resolver
		FROM
		  federation_federation_resolver ffr
		  JOIN federation_resolver fr ON ffr.federation_resolver = fr.id
		  JOIN type frt on fr.type = frt.id
		WHERE
		  ffr.federation = $1
		ORDER BY fr.ip_address
	`
	rows, err := tx.Query(qry, fedID)
	if err != nil {
		return nil, fmt.Errorf(
			"error querying federation_resolvers by federation ID [%d]: %s", fedID, err.Error(),
		)
	}
	defer rows.Close()

	resolvers := []tc.FederationResolver{}
	for rows.Next() {
		fr := tc.FederationResolver{}
		err := rows.Scan(
			&fr.IPAddress,
			&fr.Type,
			&fr.ID,
		)
		if err != nil {
			return resolvers, fmt.Errorf(
				"error scanning federation_resolvers rows for federation ID [%d]: %s", fedID, err.Error(),
			)
		}
		resolvers = append(resolvers, fr)
	}
	return resolvers, nil
}

// GetFederationNameFromID returns the federation's name, whether a federation with ID exists, or any error.
func GetFederationNameFromID(id int, tx *sql.Tx) (string, bool, error) {
	var name string
	if err := tx.QueryRow(`SELECT cname from federation where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return name, false, fmt.Errorf(
			"error querying federation name from id [%d]: %s", id, err.Error(),
		)
	}
	return name, true, nil
}

// GetProfileNameFromID returns the profile's name, whether a profile with ID exists, or any error.
func GetProfileNameFromID(id int, tx *sql.Tx) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name from profile where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying profile name from id: " + err.Error())
	}
	return name, true, nil
}

// GetProfileIDFromName returns the profile's ID, whether a profile with name exists, or any error.
func GetProfileIDFromName(name string, tx *sql.Tx) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id from profile where name = $1`, name).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, errors.New("querying profile id from name: " + err.Error())
	}
	return id, true, nil
}

// GetServerCapabilitiesFromName returns the server's capabilities.
func GetServerCapabilitiesFromName(name string, tx *sql.Tx) ([]string, error) {
	var caps []string
	q := `SELECT ARRAY(SELECT ssc.server_capability FROM server s JOIN server_server_capability ssc ON s.id = ssc.server WHERE s.host_name = $1 ORDER BY ssc.server_capability);`
	rows, err := tx.Query(q, name)
	if err != nil {
		return nil, errors.New("querying server capabilities from name: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(pq.Array(&caps)); err != nil {
			return nil, errors.New("scanning capability: " + err.Error())
		}
	}
	return caps, nil
}

const dsrExistsQuery = `
SELECT EXISTS(
	SELECT id
	FROM deliveryservice_request
	WHERE status <> 'complete' AND
		status <> 'rejected' AND
		(
			(change_type = 'delete' AND original->>'xmlId' = $1)
			OR
			(change_type <> 'delete' AND deliveryservice->>'xmlId' = $1)
		)
)
`

// DSRExistsWithXMLID returns whether or not an **open** Delivery Service
// Request with the given xmlid exists, and any error that occurred.
func DSRExistsWithXMLID(xmlid string, tx *sql.Tx) (bool, error) {
	if tx == nil {
		return false, errors.New("checking for DSR with nil transaction")
	}

	var exists bool
	if err := tx.QueryRow(dsrExistsQuery, xmlid).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// GetDSRequiredCapabilitiesFromID returns the server's capabilities.
func GetDSRequiredCapabilitiesFromID(id int, tx *sql.Tx) ([]string, error) {
	var caps []string
	q := `SELECT ARRAY(SELECT drc.required_capability FROM deliveryservices_required_capability drc WHERE drc.deliveryservice_id = $1 ORDER BY drc.required_capability);`
	rows, err := tx.Query(q, id)
	if err != nil {
		return nil, errors.New("querying deliveryservice required capabilities from id: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(pq.Array(&caps)); err != nil {
			return nil, errors.New("scanning capability: " + err.Error())
		}
	}
	return caps, nil
}

// Returns true if the cdn exists
func CDNExists(cdnName string, tx *sql.Tx) (bool, error) {
	var id int
	if err := tx.QueryRow(`SELECT id FROM cdn WHERE name = $1`, cdnName).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.New("Error querying CDN name: " + err.Error())
	}
	return true, nil
}

func GetCDNNameFromID(tx *sql.Tx, id int64) (tc.CDNName, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name FROM cdn WHERE id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying CDN ID: " + err.Error())
	}
	return tc.CDNName(name), true, nil
}

// GetCDNIDFromName returns the ID of the CDN if a CDN with the name exists
func GetCDNIDFromName(tx *sql.Tx, name tc.CDNName) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM cdn WHERE name = $1`, name).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		return id, false, errors.New("querying CDN ID: " + err.Error())
	}
	return id, true, nil
}

// GetCDNDomainFromName returns the domain, whether the cdn exists, and any error.
func GetCDNDomainFromName(tx *sql.Tx, cdnName tc.CDNName) (string, bool, error) {
	domain := ""
	if err := tx.QueryRow(`SELECT domain_name FROM cdn WHERE name = $1`, cdnName).Scan(&domain); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("Error querying CDN name: " + err.Error())
	}
	return domain, true, nil
}

// GetServerInfo returns a ServerInfo struct, whether the server exists, and an error (if one occurs).
func GetServerInfo(serverID int, tx *sql.Tx) (tc.ServerInfo, bool, error) {
	q := `
SELECT
  s.cachegroup as cachegroup_id,
  s.cdn_id as cdn_id,
  s.domain_name as domain_name,
  s.host_name as host_name,
  t.name as server_type
FROM
  server s
JOIN type t ON s.type = t.id
WHERE s.id = $1
`
	row := tc.ServerInfo{}
	if err := tx.QueryRow(q, serverID).Scan(
		&row.CachegroupID,
		&row.CDNID,
		&row.DomainName,
		&row.HostName,
		&row.Type,
	); err != nil {
		if err == sql.ErrNoRows {
			return row, false, nil
		}
		return row, false, fmt.Errorf("querying server id %d: %v", serverID, err.Error())
	}
	return row, true, nil
}

// GetServerInterfaces, given the IDs of one or more servers, returns all of their network
// interfaces mapped by their ids, or an error if one occurs during retrieval.
func GetServersInterfaces(ids []int, tx *sql.Tx) (map[int]map[string]tc.ServerInterfaceInfo, error) {
	q := `
	SELECT max_bandwidth,
	       monitor,
	       mtu,
	       name,
	       server
	FROM interface
	WHERE interface.server = ANY ($1)
	`
	ifaceRows, err := tx.Query(q, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer ifaceRows.Close()

	infs := map[int]map[string]tc.ServerInterfaceInfo{}
	for ifaceRows.Next() {
		var inf tc.ServerInterfaceInfo
		var server int
		if err := ifaceRows.Scan(&inf.MaxBandwidth, &inf.Monitor, &inf.MTU, &inf.Name, &server); err != nil {
			return nil, err
		}

		if _, ok := infs[server]; !ok {
			infs[server] = make(map[string]tc.ServerInterfaceInfo)
		}

		infs[server][inf.Name] = inf
	}

	q = `
	SELECT address,
	       gateway,
	       service_address,
	       interface,
	       server
	FROM ip_address
	WHERE ip_address.server = ANY ($1)
	`
	ipRows, err := tx.Query(q, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer ipRows.Close()

	for ipRows.Next() {
		var ip tc.ServerIPAddress
		var inf string
		var server int
		if err = ipRows.Scan(&ip.Address, &ip.Gateway, &ip.ServiceAddress, &inf, &server); err != nil {
			return nil, err
		}

		ifaces, ok := infs[server]
		if !ok {
			return nil, fmt.Errorf("retrieved ip_address with server not previously found: %d", server)
		}

		iface, ok := ifaces[inf]
		if !ok {
			return nil, fmt.Errorf("retrieved ip_address with interface not previously found: %s", inf)
		}
		iface.IPAddresses = append(iface.IPAddresses, ip)
		infs[server][inf] = iface
	}

	return infs, nil
}

// GetStatusByID returns a Status struct, a bool for whether or not a status of the given ID exists, and an error (if one occurs).
func GetStatusByID(id int, tx *sql.Tx) (tc.StatusNullable, bool, error) {
	q := `
SELECT
  description,
  id,
  last_updated,
  name
FROM
  status s
WHERE
  id = $1
`
	row := tc.StatusNullable{}
	if err := tx.QueryRow(q, id).Scan(
		&row.Description,
		&row.ID,
		&row.LastUpdated,
		&row.Name,
	); err != nil {
		if err == sql.ErrNoRows {
			return row, false, nil
		}
		return row, false, fmt.Errorf("querying status id %d: %v", id, err.Error())
	}
	return row, true, nil
}

// GetStatusByName returns a Status struct, a bool for whether or not a status of the given name exists, and an error (if one occurs).
func GetStatusByName(name string, tx *sql.Tx) (tc.StatusNullable, bool, error) {
	q := `
SELECT
  description,
  id,
  last_updated,
  name
FROM
  status s
WHERE
  name = $1
`
	row := tc.StatusNullable{}
	if err := tx.QueryRow(q, name).Scan(
		&row.Description,
		&row.ID,
		&row.LastUpdated,
		&row.Name,
	); err != nil {
		if err == sql.ErrNoRows {
			return row, false, nil
		}
		return row, false, fmt.Errorf("querying status name %s: %v", name, err.Error())
	}
	return row, true, nil
}

// GetServerIDFromName gets server id from a given name
func GetServerIDFromName(serverName string, tx *sql.Tx) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM server WHERE host_name = $1`, serverName).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		return id, false, errors.New("querying server name: " + err.Error())
	}
	return id, true, nil
}

func GetServerNameFromID(tx *sql.Tx, id int) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT host_name FROM server WHERE id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying server name: " + err.Error())
	}
	return name, true, nil
}

type ServerHostNameCDNIDAndType struct {
	HostName string
	CDNID    int
	Type     string
}

// GetServerHostNamesAndTypesFromIDs returns the server's hostname, cdn ID and associated type name
func GetServerHostNamesAndTypesFromIDs(tx *sql.Tx, ids []int) ([]ServerHostNameCDNIDAndType, error) {
	qry := `
SELECT
  s.host_name,
  s.cdn_id,
  t.name
FROM
  server s JOIN type t ON s.type = t.id
WHERE
  s.id = ANY($1)
`
	rows, err := tx.Query(qry, pq.Array(ids))
	if err != nil {
		return nil, errors.New("querying server host names and types: " + err.Error())
	}
	defer log.Close(rows, "error closing rows")

	servers := []ServerHostNameCDNIDAndType{}
	for rows.Next() {
		s := ServerHostNameCDNIDAndType{}
		if err := rows.Scan(&s.HostName, &s.CDNID, &s.Type); err != nil {
			return nil, errors.New("scanning server host name and type: " + err.Error())
		}
		servers = append(servers, s)
	}
	return servers, nil
}

// GetServerTypesCdnIdFromHostNames returns the host names, server cdn and types of the given server host names or an error if any occur.
func GetServerTypesCdnIdFromHostNames(tx *sql.Tx, hostNames []string) ([]ServerHostNameCDNIDAndType, error) {
	qry := `
SELECT
  s.host_name,
  s.cdn_id,
  t.name
FROM
  server s JOIN type t ON s.type = t.id
WHERE
  s.host_name = ANY($1)
`
	rows, err := tx.Query(qry, pq.Array(hostNames))
	if err != nil {
		return nil, errors.New("querying server host names and types: " + err.Error())
	}
	defer log.Close(rows, "error closing rows")

	servers := []ServerHostNameCDNIDAndType{}
	for rows.Next() {
		s := ServerHostNameCDNIDAndType{}
		if err := rows.Scan(&s.HostName, &s.CDNID, &s.Type); err != nil {
			return nil, errors.New("scanning server host name and type: " + err.Error())
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func GetCDNDSes(tx *sql.Tx, cdn tc.CDNName) (map[tc.DeliveryServiceName]struct{}, error) {
	dses := map[tc.DeliveryServiceName]struct{}{}
	qry := `SELECT xml_id from deliveryservice where cdn_id = (select id from cdn where name = $1)`
	rows, err := tx.Query(qry, cdn)
	if err != nil {
		return nil, errors.New("querying: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		ds := tc.DeliveryServiceName("")
		if err := rows.Scan(&ds); err != nil {
			return nil, errors.New("scanning: " + err.Error())
		}
		dses[ds] = struct{}{}
	}
	return dses, nil
}

func GetCDNs(tx *sql.Tx) (map[tc.CDNName]struct{}, error) {
	cdns := map[tc.CDNName]struct{}{}
	qry := `SELECT name from cdn;`
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, errors.New("querying: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		cdn := tc.CDNName("")
		if err := rows.Scan(&cdn); err != nil {
			return nil, errors.New("scanning: " + err.Error())
		}
		cdns[cdn] = struct{}{}
	}
	return cdns, nil
}

// GetGlobalParams returns the value of the global param, whether it existed, or any error
func GetGlobalParam(tx *sql.Tx, name string) (string, bool, error) {
	return GetParam(tx, name, "global")
}

// GetParam returns the value of the param, whether it existed, or any error.
func GetParam(tx *sql.Tx, name string, configFile string) (string, bool, error) {
	val := ""
	if err := tx.QueryRow(`select value from parameter where name = $1 and config_file = $2`, name, configFile).Scan(&val); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("Error querying global paramter '" + name + "': " + err.Error())
	}
	return val, true, nil
}

// GetParamNameByID returns the name of the param, whether it existed, or any error.
func GetParamNameByID(tx *sql.Tx, id int) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`select name from parameter where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("Error querying global paramter %v: %v", id, err.Error())
	}
	return name, true, nil
}

// GetCacheGroupNameFromID Get Cache Group name from a given ID
func GetCacheGroupNameFromID(tx *sql.Tx, id int) (tc.CacheGroupName, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name FROM cachegroup WHERE id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying cachegroup ID: " + err.Error())
	}
	return tc.CacheGroupName(name), true, nil
}

// TopologyExists checks if a Topology with the given name exists.
// Returns whether or not the Topology exists, along with any encountered error.
func TopologyExists(tx *sql.Tx, name string) (bool, error) {
	q := `
	SELECT COUNT("name")
	FROM topology
	WHERE name = $1
	`
	var count int
	var err error
	if err = tx.QueryRow(q, name).Scan(&count); err != nil {
		err = fmt.Errorf("querying topologies: %s", err)
	}
	return count > 0, err
}

// GetDeliveryServicesWithTopologies returns a list containing the delivery services in the given dsIDs
// list that have a topology assigned. An error indicates unexpected errors that occurred when querying.
func GetDeliveryServicesWithTopologies(tx *sql.Tx, dsIDs []int) ([]int, error) {
	q := `
SELECT
  id
FROM
  deliveryservice
WHERE
  id = ANY($1::bigint[])
  AND topology IS NOT NULL
`
	rows, err := tx.Query(q, pq.Array(dsIDs))
	if err != nil {
		return nil, errors.New("querying deliveryservice topologies: " + err.Error())
	}
	defer log.Close(rows, "error closing rows")
	dses := make([]int, 0)
	for rows.Next() {
		id := 0
		if err := rows.Scan(&id); err != nil {
			return nil, errors.New("scanning deliveryservice id: " + err.Error())
		}
		dses = append(dses, id)
	}
	return dses, nil
}

// GetFederationIDForUserIDByXMLID retrieves the ID of the Federation assigned to the user defined by
// userID on the Delivery Service identified by xmlid. If no such federation exists, the boolean
// returned will be 'false', while the error indicates unexpected errors that occurred when querying.
func GetFederationIDForUserIDByXMLID(tx *sql.Tx, userID int, xmlid string) (uint, bool, error) {
	var id uint
	if err := tx.QueryRow(getFederationIDForUserIDByXMLIDQuery, xmlid, userID).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("Getting Federation ID for user #%d by DS XMLID '%s': %v", userID, xmlid, err)
	}
	return id, true, nil
}

// UsernameExists reports whether or not the the given username exists as a user in the database to
// which the passed transaction refers. If anything goes wrong when checking the existence of said
// user, the error is directly returned to the caller. Note that in that case, no real meaning
// should be assigned to the returned boolean value.
func UsernameExists(uname string, tx *sql.Tx) (bool, error) {
	row := tx.QueryRow(`SELECT EXISTS(SELECT * FROM tm_user WHERE tm_user.username=$1)`, uname)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

// GetTypeIDByName reports the id of the type and whether or not a type exists with the given name.
func GetTypeIDByName(t string, tx *sql.Tx) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM type WHERE name = $1`, t).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		return id, false, errors.New("querying type id: " + err.Error())
	}
	return id, true, nil
}

// GetUserByEmail retrieves the user with the given email. If no such user exists, the boolean
// returned will be 'false', while the error indicates unexpected errors that occurred when querying.
func GetUserByEmail(email string, tx *sql.Tx) (tc.User, bool, error) {
	row := tx.QueryRow(getUserByEmailQuery, email)
	return scanUserRow(row)
}

// GetUserByID returns the user with the requested ID if one exists. The second return value is a
// boolean indicating whether said user actually did exist, and the third contains any error
// encountered along the way.
func GetUserByID(id int, tx *sql.Tx) (tc.User, bool, error) {
	row := tx.QueryRow(getUserByIDQuery, id)
	return scanUserRow(row)
}

func scanUserRow(row *sql.Row) (tc.User, bool, error) {
	var u tc.User
	err := row.Scan(&u.AddressLine1,
		&u.AddressLine2,
		&u.City,
		&u.Company,
		&u.Country,
		&u.Email,
		&u.FullName,
		&u.GID,
		&u.ID,
		&u.LastUpdated,
		&u.NewUser,
		&u.PhoneNumber,
		&u.PostalCode,
		&u.PublicSSHKey,
		&u.RegistrationSent,
		&u.Role,
		&u.RoleName,
		&u.StateOrProvince,
		&u.Tenant,
		&u.TenantID,
		&u.Token,
		&u.UID,
		&u.Username)
	if err == sql.ErrNoRows {
		return u, false, nil
	}
	return u, true, err
}

// CachegroupParameterAssociationExists returns whether a cachegroup parameter association with the given parameter id exists, and any error.
func CachegroupParameterAssociationExists(id int, cachegroup int, tx *sql.Tx) (bool, error) {
	count := 0
	if err := tx.QueryRow(`SELECT count(*) from cachegroup_parameter where parameter = $1 and cachegroup = $2`, id, cachegroup).Scan(&count); err != nil {
		return false, errors.New("querying cachegroup parameter existence: " + err.Error())
	}
	return count > 0, nil
}

// GetDeliveryServiceType returns the type of the deliveryservice.
func GetDeliveryServiceType(dsID int, tx *sql.Tx) (tc.DSType, bool, error) {
	var dsType tc.DSType
	if err := tx.QueryRow(`SELECT t.name FROM deliveryservice as ds JOIN type t ON ds.type = t.id WHERE ds.id=$1`, dsID).Scan(&dsType); err != nil {
		if err == sql.ErrNoRows {
			return tc.DSTypeInvalid, false, nil
		}
		return tc.DSTypeInvalid, false, errors.New("querying type from delivery service: " + err.Error())
	}
	return dsType, true, nil
}

// GetDeliveryServiceTypeAndTopology returns the type of the deliveryservice and the name of its topology.
func GetDeliveryServiceTypeRequiredCapabilitiesAndTopology(dsID int, tx *sql.Tx) (tc.DSType, []string, *string, bool, error) {
	var dsType tc.DSType
	var reqCap []string
	var topology *string
	q := `
SELECT
  t.name,
  ARRAY_REMOVE(ARRAY_AGG(dsrc.required_capability ORDER BY dsrc.required_capability), NULL) AS required_capabilities,
  ds.topology
FROM deliveryservice AS ds
LEFT JOIN deliveryservices_required_capability AS dsrc ON dsrc.deliveryservice_id = ds.id
JOIN type t ON ds.type = t.id
WHERE ds.id = $1
GROUP BY t.name, ds.topology
`
	if err := tx.QueryRow(q, dsID).Scan(&dsType, pq.Array(&reqCap), &topology); err != nil {
		if err == sql.ErrNoRows {
			return tc.DSTypeInvalid, nil, nil, false, nil
		}
		return tc.DSTypeInvalid, nil, nil, false, errors.New("querying type from delivery service: " + err.Error())
	}
	return dsType, reqCap, topology, true, nil
}
