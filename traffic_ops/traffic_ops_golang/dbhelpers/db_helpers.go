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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/jmoiron/sqlx"
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
		if err != nil || limitInt < 1 {
			errs = append(errs, errors.New("limit parameter must be a positive integer"))
			return "", "", "", queryValues, errs
		}
		log.Debugln("limit: ", limit)
		paginationClause += " " + limit
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

// AddTenancyCheck takes a WHERE clause (can be ""), the associated queryValues (can be empty),
// a tenantColumnName that should provide a bigint corresponding to the tenantID of the object being checked (this may require a CAST),
// and an array of the tenantIDs the user has access to; it returns a where clause and associated queryValues including filtering based on tenancy.
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

// GetCacheGroupNameFromID Get Cache Group name from a given ID
func GetCacheGroupNameFromID(tx *sql.Tx, id int64) (tc.CacheGroupName, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name FROM cachegroup WHERE id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying cachegroup ID: " + err.Error())
	}
	return tc.CacheGroupName(name), true, nil
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

// ResourceModified is a helper to get whether the last modified time of any table used by the resource is newer than the If-Modified-Since or If-None-Match time sent by the client.
//
// The lastModified should be the If-Modified-Since or If-None-Match time sent the by the client.
//
// The query should be a query, which selects the latest modified time of any table used by the resource.
// For an example, see server/cachecontrol.go makeServerModfiiedQry.
//
// The queryParams should be named argument for the query, in the sqlx named argument format.
//
func ResourceModified(tx *sqlx.Tx, lastModified time.Time, query string, queryParams map[string]interface{}) bool {
	modTime, ok, err := resourceModifiedTime(tx, query, queryParams)
	if err != nil {
		// log, but don't let this prevent us from trying to get the DS otherwise, just consider it modified
		log.Errorln("Getting resource modified time: " + err.Error())
		return true
	} else if !ok {
		return true
	}

	// if the given lastModified is exactly on the second, it must have been from Last-Modified and not ETag, so truncate the queried time to the second too (otherwise, it'd never be unmodified)
	if lastModified.Truncate(time.Second) == lastModified {
		modTime = modTime.Truncate(time.Second)
	}

	if modTime.After(lastModified) {
		return true
	}
	return false
}

func resourceModifiedTime(tx *sqlx.Tx, query string, queryParams map[string]interface{}) (time.Time, bool, error) {
	rows, err := tx.NamedQuery(query, queryParams)
	if err != nil {
		return time.Time{}, false, errors.New("querying: " + err.Error())
	}
	defer rows.Close()

	if !rows.Next() {
		return time.Time{}, false, nil
	}

	modTime := time.Time{}
	if err := rows.Scan(&modTime); err != nil {
		return time.Time{}, false, errors.New("scanning: " + err.Error())
	}

	if rows.Next() {
		return time.Time{}, false, errors.New("scanning: expected 1 row, but got many")
	}

	return modTime, true, nil
}
