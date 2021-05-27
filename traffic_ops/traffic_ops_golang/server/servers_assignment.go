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
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/lib/pq"
)

type needsCheck struct {
	CDN     uint
	CDNName string
	DSID    uint
	DSXMLID string
	Tenant  int
}

const needsCheckInfoQuery = `
SELECT deliveryservice.id,
       deliveryservice.cdn_id,
       deliveryservice.tenant_id,
       deliveryservice.xml_id,
       cdn.name
FROM deliveryservice
LEFT OUTER JOIN cdn ON cdn.id=deliveryservice.cdn_id
WHERE deliveryservice.id = ANY($1)
`

func getConfigFile(prefix string, xmlId string) string {
	const configSuffix = `.config`
	return prefix + xmlId + configSuffix
}

const lastServerInActiveDeliveryServicesQuery = `
SELECT deliveryservice_server.deliveryservice
FROM deliveryservice_server
INNER JOIN server ON server.id = deliveryservice_server.server
INNER JOIN status ON status.id = server.status
WHERE deliveryservice IN (
	SELECT deliveryservice_server.deliveryservice
	FROM deliveryservice_server
	INNER JOIN deliveryservice ON deliveryservice.id = deliveryservice_server.deliveryservice
	WHERE deliveryservice_server.server=$1
	AND deliveryservice.active = 'ACTIVE'
)
AND NOT (deliveryservice_server.deliveryservice = ANY($2::BIGINT[]))
AND (status.name = '` + string(tc.CacheStatusOnline) + `' OR status.name = '` + string(tc.CacheStatusReported) + `')
GROUP BY deliveryservice_server.deliveryservice
HAVING COUNT(deliveryservice_server.server) = 1
`

func checkForLastServerInActiveDeliveryServices(serverID int, dsIDs []int, tx *sql.Tx) ([]int, error) {
	violations := []int{}
	rows, err := tx.Query(lastServerInActiveDeliveryServicesQuery, serverID, pq.Array(dsIDs))
	if err != nil {
		return violations, fmt.Errorf("querying: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var violation int
		if err = rows.Scan(&violation); err != nil {
			return violations, fmt.Errorf("scanning: %v", err)
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// AssignDeliveryServicesToServerHandler is the handler for POST requests to /servers/{{ID}}/deliveryservices.
func AssignDeliveryServicesToServerHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	dsList := []int{}
	if err := json.NewDecoder(r.Body).Decode(&dsList); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, errors.New("payload must be a list of integers representing delivery service ids"), nil)
		return
	}

	replaceQueryParameter := inf.Params["replace"]
	replace, err := strconv.ParseBool(replaceQueryParameter) //accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False. for replace url parameter documentation
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	server := inf.IntParams["id"]

	serverInfo, ok, err := dbhelpers.GetServerInfo(server, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("getting server name from ID: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, tx, http.StatusNotFound, errors.New("no server with that ID found"), nil)
		return
	}

	if !strings.HasPrefix(serverInfo.Type, tc.OriginTypeName) {
		usrErr, sysErr, status := ValidateDSCapabilities(dsList, serverInfo.HostName, tx)
		if usrErr != nil || sysErr != nil {
			api.HandleErr(w, r, tx, status, usrErr, sysErr)
			return
		}
	}

	// We already know the CDN exists because that's part of the serverInfo query above
	serverCDN, _, err := dbhelpers.GetCDNNameFromID(tx, int64(serverInfo.CDNID))
	if err != nil {
		sysErr = fmt.Errorf("Failed to get CDN name from ID: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	if len(dsList) > 0 {
		if errCode, userErr, sysErr = checkTenancyAndCDN(tx, string(serverCDN), server, serverInfo, dsList, inf.User); userErr != nil || sysErr != nil {
			api.HandleErr(w, r, tx, errCode, userErr, sysErr)
			return
		}
		if strings.HasPrefix(serverInfo.Type, tc.OriginTypeName) {
			if userErr, sysErr, status := checkOriginInTopologies(tx, serverInfo.Cachegroup, dsList); userErr != nil || sysErr != nil {
				api.HandleErr(w, r, tx, status, userErr, sysErr)
				return
			}
		}
	}

	if replace && (serverInfo.Status == tc.CacheStatusOnline.String() || serverInfo.Status == tc.CacheStatusReported.String()) {
		currentDSIDs, err := checkForLastServerInActiveDeliveryServices(server, dsList, tx)
		if err != nil {
			sysErr = fmt.Errorf("checking for deliveryservices to which server #%d is the last assigned: %v", server, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if len(currentDSIDs) > 0 {
			alertText := "Delivery Service assignment would leave Active Delivery Service"
			if len(currentDSIDs) == 1 {
				alertText += fmt.Sprintf(" #%d", currentDSIDs[0])
			} else if len(currentDSIDs) == 2 {
				alertText += fmt.Sprintf("s #%d and #%d", currentDSIDs[0], currentDSIDs[1])
			} else {
				dsNums := make([]string, 0, len(currentDSIDs)-1)
				for _, dsID := range currentDSIDs[:len(currentDSIDs)-1] {
					dsNums = append(dsNums, "#"+strconv.Itoa(dsID))
				}
				alertText += fmt.Sprintf("s %s, and #%d", strings.Join(dsNums, ", "), currentDSIDs[len(currentDSIDs)-1])
			}
			alertText += fmt.Sprintf("  with no '%s' or '%s' servers", tc.CacheStatusOnline, tc.CacheStatusReported)
			api.WriteAlerts(w, r, http.StatusConflict, tc.CreateAlerts(tc.ErrorLevel, alertText))
			return
		}
	}

	assignedDSes, err := assignDeliveryServicesToServer(server, dsList, replace, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("getting server name from ID: "+err.Error()))
		return
		// TODO: Where does this 'ok' come from?
	} else if !ok {
		api.HandleErr(w, r, tx, http.StatusNotFound, errors.New("no server with that ID found"), nil)
	}

	api.CreateChangeLogRawTx(api.ApiChange, "SERVER: "+serverInfo.HostName+", ID: "+strconv.Itoa(server)+", ACTION: Assigned "+strconv.Itoa(len(assignedDSes))+" DSes to server", inf.User, tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "successfully assigned dses to server", tc.AssignedDsResponse{ServerID: server, DSIds: assignedDSes, Replace: replace})
}

// checkOriginInTopologies checks to make sure the given ORG server's cachegroup belongs
// to the topologies of the given delivery services.
func checkOriginInTopologies(tx *sql.Tx, originCachegroup string, dsList []int) (error, error, int) {
	// get the delivery services that don't have originCachegroup in their topology
	q := `
SELECT
  ds.xml_id,
  tc.topology,
  ARRAY_AGG(tc.cachegroup)
FROM
  deliveryservice ds
  JOIN topology_cachegroup tc ON tc.topology = ds.topology
WHERE
  ds.id = ANY($1::BIGINT[])
GROUP BY ds.xml_id, tc.topology
HAVING NOT ($2 = ANY(ARRAY_AGG(tc.cachegroup)))
`
	rows, err := tx.Query(q, pq.Array(dsList), originCachegroup)
	if err != nil {
		return nil, errors.New("querying deliveryservice topologies: " + err.Error()), http.StatusInternalServerError
	}
	defer log.Close(rows, "error closing rows")

	invalid := []string{}
	for rows.Next() {
		xmlID := ""
		topology := ""
		cachegroups := []string{}
		if err := rows.Scan(&xmlID, &topology, pq.Array(&cachegroups)); err != nil {
			return nil, errors.New("scanning deliveryservice topologies: " + err.Error()), http.StatusInternalServerError
		}
		invalid = append(invalid, fmt.Sprintf("%s (%s)", topology, xmlID))
	}
	if len(invalid) > 0 {
		return fmt.Errorf("%s server cachegroup (%s) not found in the following topologies: %s", tc.OriginTypeName, originCachegroup, strings.Join(invalid, ", ")), nil, http.StatusBadRequest
	}
	return nil, nil, http.StatusOK
}

func checkTenancyAndCDN(tx *sql.Tx, serverCDN string, server int, serverInfo tc.ServerInfo, dsList []int, user *auth.CurrentUser) (int, error, error) {
	rows, err := tx.Query(needsCheckInfoQuery, pq.Array(dsList))
	if err != nil {
		if err == sql.ErrNoRows {
			return http.StatusBadRequest, errors.New("Either at least one Delivery Service ID doesn't exist, or is outside your tenancy!"), nil
		}
		return http.StatusInternalServerError, nil, err
	}
	defer rows.Close()

	tenantsToCheck := make([]needsCheck, 0, len(dsList))
	for rows.Next() {
		var n needsCheck
		if err = rows.Scan(&n.DSID, &n.CDN, &n.Tenant, &n.DSXMLID, &n.CDNName); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("Scanning cdn_id for ds: %v", err)
		}

		tenantsToCheck = append(tenantsToCheck, n)
	}

	if len(tenantsToCheck) != len(dsList) {
		return http.StatusNotFound, errors.New("Either no Delivery Service ids given, or at least one id doesn't exist!"), nil
	}

	for _, t := range tenantsToCheck {
		if ok, err := tenant.IsResourceAuthorizedToUserTx(t.Tenant, user, tx); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("Checking availability of ds %d (tenant_id: %d) to tenant_id %d: %v", t.DSID, t.Tenant, user.TenantID, err)
		} else if !ok {
			// In keeping with the behavior of /deliveryservices, we don't disclose the existences
			// of Delivery Services to which the user is forbidden access
			return http.StatusNotFound, errors.New("Either no Delivery Service ids given, or at least one id doesn't exist!"), fmt.Errorf("User %s denied access to inaccessible DS %d (owned by tenant_id %d)", user.UserName, t.DSID, t.Tenant)
		}

		if int(t.CDN) != serverInfo.CDNID {
			return http.StatusConflict, fmt.Errorf("Delivery Service %s (#%d) is not in the same CDN as server %s (#%d) (server is in %s (#%d), DS is in %s (#%d))!", t.DSXMLID, t.DSID, serverInfo.HostName, server, serverCDN, serverInfo.CDNID, t.CDNName, t.CDN), nil
		}
	}

	return http.StatusOK, nil, nil
}

// ValidateDSCapabilities checks that the server meets the requirements of each delivery service to be assigned.
func ValidateDSCapabilities(dsIDs []int, serverName string, tx *sql.Tx) (error, error, int) {
	var dsCaps []string
	sCaps, err := dbhelpers.GetServerCapabilitiesFromName(serverName, tx)

	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	for _, id := range dsIDs {
		dsCaps, err = dbhelpers.GetDSRequiredCapabilitiesFromID(id, tx)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
		for _, dsc := range dsCaps {
			if !util.ContainsStr(sCaps, dsc) {
				return errors.New(fmt.Sprintf("Caching server cannot assign this delivery service without having the required delivery service capabilities: [%v] for server %s", dsCaps, serverName)), nil, http.StatusBadRequest
			}
		}
	}

	return nil, nil, 0
}

func assignDeliveryServicesToServer(server int, dses []int, replace bool, tx *sql.Tx) ([]int, error) {
	if replace {
		//delete currently assigned dses from server
		if _, err := tx.Exec(`DELETE FROM deliveryservice_server WHERE server = $1`, server); err != nil {
			return nil, errors.New("could not delete old deliveryservice_server associations for server: " + err.Error())
		}
	}

	//assign new dses
	dsPqArray := pq.Array(dses)
	// The common table expressions (CTEs) used below allow for inserting every deliveryService in dses with the server without a loop
	// the result of the select (for server = 100, dses = [1,2,3]) used by the insert essentially looks like:
	//          | server
	//    ---------------
	//      1   |   100
	//      2   |   100
	//      3   |   100
	// UNNEST is used to turn the array of ds values into, essentially, * FROM ( VALUES (1),(2),(3) )
	// this allows for a single insert query instead of a loop over the dses.
	// This pattern is used for the other bulk inserts as well.
	q := `
INSERT INTO deliveryservice_server (deliveryservice, server)
	WITH
	q1 AS (SELECT UNNEST($1::bigint[])),
	q2 AS ( SELECT * FROM (VALUES ($2::bigint)) AS server )
	SELECT * FROM q1,q2 ON CONFLICT DO NOTHING
`
	if _, err := tx.Exec(q, dsPqArray, server); err != nil {
		return nil, errors.New("inserting deliveryservice_server: " + err.Error())
	}

	//need remap config location
	var atsConfigLocation string
	const remapFile = `remap.config`
	if err := tx.QueryRow(
		`SELECT value FROM parameter
		WHERE name = 'location'
		AND config_file = '` + remapFile + `'`).Scan(&atsConfigLocation); err != nil {
		return nil, errors.New("scanning location parameter: " + err.Error())
	}
	if strings.HasSuffix(atsConfigLocation, "/") {
		atsConfigLocation = atsConfigLocation[:len(atsConfigLocation)-1]
	}

	//we need dses: xmlids and edge_header_rewrite, regex_remap, and cache_url
	rows, err := tx.Query(`SELECT xml_id, edge_header_rewrite, regex_remap, cacheurl FROM deliveryservice WHERE id = ANY($1::bigint[])`, dsPqArray)
	if err != nil {
		return nil, errors.New("querying deliveryservice: " + err.Error())
	}
	defer rows.Close()

	//create new parameters here as necessary:
	//loop over ds results and build file parameters we need to insert / select
	//for all of: header rewrite, regex_remap, cache_url
	// if ds has it add parameter to insert list
	// other wise add to delete list
	//TODO: DylanVolz this may need to be extended refactored, there are potentially other parameters that need this like urlSigKeys...
	insert := []string{}
	delete := []string{}
	for rows.Next() {
		var xmlID sql.NullString
		var edgeHeaderRewrite sql.NullString
		var regexRemap sql.NullString
		var cacheURL sql.NullString
		if err := rows.Scan(&xmlID, &edgeHeaderRewrite, &regexRemap, &cacheURL); err != nil {
			return nil, errors.New("scanning deliveryservice: " + err.Error())
		}
		const headerRewritePrefix = `hdr_rw_`
		const regexRemapPrefix = `regex_remap_`
		const cacheURLPrefix = `cacheurl_`
		if xmlID.Valid && len(xmlID.String) > 0 {
			//param := "hdr_rw_" + xmlID.String + ".config"
			param := getConfigFile(headerRewritePrefix, xmlID.String)
			if edgeHeaderRewrite.Valid && len(edgeHeaderRewrite.String) > 0 {
				insert = append(insert, param)
			} else {
				delete = append(delete, param)
			}
			param = getConfigFile(regexRemapPrefix, xmlID.String)
			if regexRemap.Valid && len(regexRemap.String) > 0 {
				insert = append(insert, param)
			} else {
				delete = append(delete, param)
			}
			param = getConfigFile(cacheURLPrefix, xmlID.String)
			if cacheURL.Valid && len(cacheURL.String) > 0 {
				insert = append(insert, param)
			} else {
				delete = append(delete, param)
			}
		}

	}

	//insert the parameters we selected above:
	q = `
INSERT INTO parameter (config_file, name, value)
	WITH
	q1 AS (SELECT UNNEST($1::text[]) AS config_file),
	q2 AS (SELECT * FROM (VALUES ($2) ) AS name),
	q3 AS (SELECT * FROM (VALUES ($3) ) AS value)
	 SELECT * FROM q1,q2,q3 ON CONFLICT DO NOTHING
`
	fileNamePqArray := pq.Array(insert)
	if _, err = tx.Exec(q, fileNamePqArray, "location", atsConfigLocation); err != nil {
		return nil, errors.New("inserting parameters: " + err.Error())
	}

	//select the ids associated with the parameters we created above (may be able to get them from insert above to optimize)
	rows, err = tx.Query(`SELECT id FROM parameter WHERE name = 'location' AND config_file IN ($1)`, fileNamePqArray)
	if err != nil {
		return nil, errors.New("selecting location parameter after insert: " + err.Error())
	}
	defer rows.Close()

	parameterIds := []int64{}
	for rows.Next() {
		var ID int64
		if err := rows.Scan(&ID); err != nil {
			log.Errorf("could not scan parameter ID: %s\n", err)
			return nil, tc.DBError
		}
		parameterIds = append(parameterIds, ID)
	}

	//associate all parameter ids with the profiles associated with all servers associated with assigned dses.
	q = `
INSERT INTO profile_parameter (profile, parameter)
	WITH
	q1 AS ( SELECT DISTINCT profile FROM server LEFT JOIN deliveryservice_server ON server.id = deliveryservice_server.server WHERE deliveryservice_server.deliveryservice = ANY($1::bigint[]) ),
	q2 AS (SELECT UNNEST($2::bigint[]) AS parameter)
	SELECT * FROM q1,q2
	ON CONFLICT DO NOTHING
`
	if _, err = tx.Exec(q, dsPqArray, pq.Array(parameterIds)); err != nil {
		return nil, errors.New("inserting profile_parameter: " + err.Error())
	}

	//process delete list
	if _, err = tx.Exec(`DELETE FROM parameter WHERE name = 'location' AND config_file = ANY($1)`, pq.Array(delete)); err != nil {
		return nil, errors.New("deleting parameters: " + err.Error())
	}

	return dses, nil
}
