package federations

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
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/lib/pq"
)

const insertResolverQuery = `
INSERT INTO federation_resolver (ip_address, type)
VALUES ($1, (
	SELECT type.id
	FROM type
	WHERE type.name = $2
))
ON CONFLICT DO NOTHING
RETURNING federation_resolver.ip_address, federation_resolver.id
`

const associateFederationWithResolverQuery = `
INSERT INTO federation_federation_resolver (federation, federation_resolver)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

const deleteCurrentUserFederationResolversQuery = `
DELETE FROM federation_resolver
WHERE federation_resolver.id IN (
	SELECT federation_federation_resolver.federation_resolver
	FROM federation_federation_resolver
	WHERE federation_federation_resolver.federation IN (
		SELECT federation_tmuser.federation
		FROM federation_tmuser
		WHERE federation_tmuser.tm_user = $1
	)
)
RETURNING federation_resolver.ip_address
`

func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	code := http.StatusOK
	useIMS := false
	config, e := api.GetConfig(r.Context())
	if e == nil && config != nil {
		useIMS = config.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}
	var maxTime *time.Time
	var err error
	var feds []FedInfo

	feds, err, code, maxTime = getUserFederations(inf.Tx.Tx, inf.User.UserName, useIMS, r.Header)
	if code == http.StatusNotModified {
		w.WriteHeader(code)
		api.WriteResp(w, r, []tc.IAllFederation{})
		return
	}
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("federations.Get getting federations: "+err.Error()))
		return
	}
	fedsResolvers, err, code, maxTime := getFederationResolvers(inf.Tx.Tx, fedInfoIDs(feds), useIMS, r.Header)
	if code == http.StatusNotModified {
		w.WriteHeader(code)
		api.WriteResp(w, r, []tc.IAllFederation{})
		return
	}
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("federations.Get getting federations resolvers: "+err.Error()))
		return
	}
	allFederations := addResolvers([]tc.IAllFederation{}, feds, fedsResolvers)
	if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
		api.AddLastModifiedHdr(w, *maxTime)
	}
	api.WriteResp(w, r, allFederations)
}

func addResolvers(allFederations []tc.IAllFederation, feds []FedInfo, fedsResolvers map[int][]FedResolverInfo) []tc.IAllFederation {
	dsFeds := map[tc.DeliveryServiceName][]tc.FederationResolverMapping{}
	for _, fed := range feds {
		mapping := tc.FederationResolverMapping{}
		mapping.TTL = util.IntPtr(fed.TTL)
		mapping.CName = util.StrPtr(fed.CName)
		for _, resolver := range fedsResolvers[fed.ID] {
			switch resolver.Type {
			case tc.FederationResolverType4:
				mapping.Resolve4 = append(mapping.Resolve4, resolver.IP)
			case tc.FederationResolverType6:
				mapping.Resolve6 = append(mapping.Resolve6, resolver.IP)
			default:
				log.Warnf("federations addResolvers got invalid resolver type for federation '%v', skipping\n", fed.ID)
			}
		}
		dsFeds[fed.DS] = append(dsFeds[fed.DS], mapping)
	}

	for ds, mappings := range dsFeds {
		allFederations = append(allFederations, tc.AllDeliveryServiceFederationsMapping{DeliveryService: ds, Mappings: mappings})
	}
	return allFederations
}

func fedInfoIDs(feds []FedInfo) []int {
	ids := []int{}
	for _, fed := range feds {
		ids = append(ids, fed.ID)
	}
	return ids
}

type FedInfo struct {
	ID    int
	TTL   int
	CName string
	DS    tc.DeliveryServiceName
}

type FedResolverInfo struct {
	Type tc.FederationResolverType
	IP   string
}

// getFederationResolvers takes a slice of federation IDs, and returns a map[federationID]info.
func getFederationResolvers(tx *sql.Tx, fedIDs []int, useIMS bool, header http.Header) (map[int][]FedResolverInfo, error, int, *time.Time) {
	var runSecond bool
	var maxTime time.Time
	feds := map[int][]FedResolverInfo{}
	qry := `
SELECT
  ffr.federation,
  frt.name as resolver_type,
  fr.ip_address
FROM
  federation_federation_resolver ffr
  JOIN federation_resolver fr ON ffr.federation_resolver = fr.id
  JOIN type frt on fr.type = frt.id
WHERE
  ffr.federation = ANY($1)
`
	imsQuery := `SELECT max(t) from (
		SELECT max(ffr.last_updated) as t FROM
  federation_federation_resolver ffr
  JOIN federation_resolver fr ON ffr.federation_resolver = fr.id
  JOIN type frt on fr.type = frt.id
WHERE ffr.federation = ANY($1)
	UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation_federation_resolver') as res`

	if useIMS {
		runSecond, maxTime = tryIfModifiedSinceQueryFederations(header, tx, pq.Array(fedIDs), imsQuery)
		if !runSecond {
			log.Debugln("IMS HIT")
			return feds, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := tx.Query(qry, pq.Array(fedIDs))
	if err != nil {
		return nil, errors.New("all federations resolvers querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	for rows.Next() {
		fedID := 0
		f := FedResolverInfo{}
		fType := ""
		if err := rows.Scan(&fedID, &fType, &f.IP); err != nil {
			return nil, errors.New("all federations resolvers scanning: " + err.Error()), http.StatusInternalServerError, nil
		}
		f.Type = tc.FederationResolverTypeFromString(fType)
		feds[fedID] = append(feds[fedID], f)
	}
	return feds, nil, http.StatusOK, &maxTime
}

func tryIfModifiedSinceQueryFederations(header http.Header, tx *sql.Tx, fedID interface{}, query string) (bool, time.Time) {
	var max time.Time
	var imsDate time.Time
	var ok bool
	imsDateHeader := []string{}
	runSecond := true
	dontRunSecond := false
	if header == nil {
		return runSecond, max
	}
	imsDateHeader = header[rfc.IfModifiedSince]
	if len(imsDateHeader) == 0 {
		return runSecond, max
	}
	if imsDate, ok = rfc.ParseHTTPDate(imsDateHeader[0]); !ok {
		log.Warnf("IMS request header date '%s' not parsable", imsDateHeader[0])
		return runSecond, max
	}
	rows, err := tx.Query(query, fedID)
	if err != nil {
		log.Warnf("Couldn't get the max last updated time: %v", err)
		return runSecond, max
	}
	if err == sql.ErrNoRows {
		return dontRunSecond, max
	}
	defer rows.Close()
	// This should only ever contain one row
	if rows.Next() {
		v := tc.TimeNoMod{}
		if err = rows.Scan(&v); err != nil {
			log.Warnf("Failed to parse the max time stamp into a struct %v", err)
			return runSecond, max
		}
		max = v.Time
		// The request IMS time is later than the max of (lastUpdated, deleted_time)
		if imsDate.After(v.Time) {
			return dontRunSecond, max
		}
	}
	return runSecond, max
}

func getUserFederations(tx *sql.Tx, userName string, useIMS bool, header http.Header) ([]FedInfo, error, int, *time.Time) {
	var runSecond bool
	var maxTime time.Time
	feds := []FedInfo{}
	qry := `
SELECT
  fds.federation,
  fd.ttl,
  fd.cname,
  ds.xml_id
FROM
  federation_deliveryservice fds
  JOIN deliveryservice ds ON ds.id = fds.deliveryservice
  JOIN federation fd ON fd.id = fds.federation
  JOIN federation_tmuser fu on fu.federation = fd.id
  JOIN tm_user u on u.id = fu.tm_user
WHERE
  u.username = $1
ORDER BY
  ds.xml_id
`
	imsQuery := `SELECT max(t) from (
		SELECT max(fds.last_updated) as t FROM
  federation_deliveryservice fds
  JOIN deliveryservice ds ON ds.id = fds.deliveryservice
  JOIN federation fd ON fd.id = fds.federation
  JOIN federation_tmuser fu on fu.federation = fd.id
  JOIN tm_user u on u.id = fu.tm_user
WHERE
  u.username = $1
	UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation_deliveryservice') as res`

	if useIMS {
		runSecond, maxTime = tryIfModifiedSinceQuery(header, tx, userName, imsQuery)
		if !runSecond {
			log.Debugln("IMS HIT")
			return feds, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := tx.Query(qry, userName)
	if err != nil {
		return nil, errors.New("user federations querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	for rows.Next() {
		f := FedInfo{}
		if err := rows.Scan(&f.ID, &f.TTL, &f.CName, &f.DS); err != nil {
			return nil, errors.New("user federations scanning: " + err.Error()), http.StatusInternalServerError, nil
		}
		feds = append(feds, f)
	}
	return feds, nil, http.StatusOK, &maxTime
}

// AddFederationResorverMappingsForCurrentUser is the handler for a POST request to /federations.
// Confusingly, it does not create a federation, but is instead used to manipulate the resolvers
// used by one or more particular Delivery Services for one or more particular Federations.
func AddFederationResolverMappingsForCurrentUser(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	mappings, userErr, sysErr := getMappingsFromRequestBody(inf.Version, r.Body)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, sysErr)
		return
	}

	if err := mappings.Validate(tx); err != nil {
		errCode = http.StatusBadRequest
		userErr = fmt.Errorf("validating request: %v", err)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	userErr, sysErr, errCode = addFederationResolverMappingsForCurrentUser(inf.User, tx, mappings)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	msg := fmt.Sprintf("%s successfully created federation resolvers.", inf.User.UserName)
	if inf.Version.Major <= 1 && inf.Version.Minor <= 3 {
		api.WriteResp(w, r, msg)
	} else {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, msg, msg)
	}
}

// handles the main logic of the POST handler, separated out for convenience
func addFederationResolverMappingsForCurrentUser(u *auth.CurrentUser, tx *sql.Tx, mappings []tc.DeliveryServiceFederationResolverMapping) (error, error, int) {
	for _, fed := range mappings {
		dsTenant, ok, err := dbhelpers.GetDSTenantIDFromXMLID(tx, fed.DeliveryService)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		} else if !ok {
			return fmt.Errorf("'%s' - no such Delivery Service", fed.DeliveryService), nil, http.StatusConflict
		}

		if ok, err = tenant.IsResourceAuthorizedToUserTx(dsTenant, u, tx); err != nil {
			err = fmt.Errorf("Checking user #%d tenancy permissions on DS '%s' (tenant #%d)", u.ID, fed.DeliveryService, dsTenant)
			return nil, err, http.StatusInternalServerError
		} else if !ok {
			userErr := fmt.Errorf("'%s' - no such Delivery Service", fed.DeliveryService)
			sysErr := fmt.Errorf("User '%s' requested unauthorized federation resolver mapping modification on the '%s' Delivery Service", u.UserName, fed.DeliveryService)
			return userErr, sysErr, http.StatusConflict
		}

		fedID, ok, err := dbhelpers.GetFederationIDForUserIDByXMLID(tx, u.ID, fed.DeliveryService)
		if err != nil {
			return nil, fmt.Errorf("Getting Federation ID: %v", err), http.StatusInternalServerError
		} else if !ok {
			err = fmt.Errorf("No federation(s) found for user %s on delivery service '%s'.", u.UserName, fed.DeliveryService)
			return err, nil, http.StatusConflict
		}

		inserted, err := addFederationResolverMappingsToFederation(fed.Mappings, fed.DeliveryService, fedID, tx)
		if err != nil {
			err = fmt.Errorf("Adding federation resolver mapping(s) to federation: %v", err)
			return nil, err, http.StatusInternalServerError
		}

		changelogMsg := "FEDERATION DELIVERY SERVICE: %s, ID: %d, ACTION: User %s successfully added federation resolvers [ %s ]"
		changelogMsg = fmt.Sprintf(changelogMsg, fed.DeliveryService, fedID, u.UserName, inserted)
		api.CreateChangeLogRawTx(api.ApiChange, changelogMsg, u, tx)
	}
	return nil, nil, http.StatusOK
}

// adds federation resolver mappings for a particular delivery service to a given federation, creating said resolvers if
// they don't already exist.
func addFederationResolverMappingsToFederation(res tc.ResolverMapping, xmlid string, fed uint, tx *sql.Tx) (string, error) {
	var resp string
	if len(res.Resolve4) > 0 {
		inserted, err := addFederationResolver(res.Resolve4, tc.FederationResolverType4, fed, tx)
		if err != nil {
			return "", err
		}
		resp = strings.Join(inserted, ", ")
	}
	if len(res.Resolve6) > 0 {
		inserted, err := addFederationResolver(res.Resolve6, tc.FederationResolverType6, fed, tx)
		if err != nil {
			return "", err
		}
		resp += strings.Join(inserted, ", ")
	}
	return resp, nil
}

// adds federation resolvers of a specific type to the given federation
func addFederationResolver(res []string, t tc.FederationResolverType, fedID uint, tx *sql.Tx) ([]string, error) {
	inserted := []string{}
	for _, r := range res {
		var ip string
		var id uint
		if err := tx.QueryRow(insertResolverQuery, r, t).Scan(&ip, &id); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if ip != "" && id > 0 {
			inserted = append(inserted, ip)
			if _, err := tx.Exec(associateFederationWithResolverQuery, fedID, id); err != nil {
				return nil, err
			}
		}

	}

	return inserted, nil
}

// RemoveFederationResolverMappingsForCurrentUser is the handler for a DELETE request to /federations
// Confusingly, it does not delete a federation, but is instead used to remove an association
// between all federation resolvers and all federations assigned to the authenticated user.
func RemoveFederationResolverMappingsForCurrentUser(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ips, userErr, sysErr, errCode := removeFederationResolverMappingsForCurrentUser(tx, inf.User)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	// I'm not sure if I necessarily agree with treating this as a client error, but it's what Perl did.
	if len(ips) < 1 {
		errCode = http.StatusConflict
		userErr = fmt.Errorf("No federation resolvers to delete for user %s", inf.User.UserName)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	ipList := fmt.Sprintf("[ %s ]", strings.Join(ips, ", "))
	msg := fmt.Sprintf("%s successfully deleted all federation resolvers: %s", inf.User.UserName, ipList)
	changelogMsg := fmt.Sprintf("USER: %s, ID: %d, ACTION: %s", inf.User.UserName, inf.User.ID, msg)
	api.CreateChangeLogRawTx(api.ApiChange, changelogMsg, inf.User, tx)

	if inf.Version.Major <= 1 && inf.Version.Minor <= 3 {
		api.WriteResp(w, r, msg)
	} else {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, msg, msg)
	}
}

// handles the main logic of the DELETE handler, separated out for convenience
func removeFederationResolverMappingsForCurrentUser(tx *sql.Tx, u *auth.CurrentUser) ([]string, error, error, int) {
	rows, err := tx.Query(deleteCurrentUserFederationResolversQuery, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("No federation resolvers to delete for user %s", u.UserName), nil, http.StatusConflict
		} else {
			return nil, nil, fmt.Errorf("Deleting federation resolvers for user %s: %v", u.UserName, err), http.StatusInternalServerError
		}
	}
	defer rows.Close()

	ips := []string{}
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, nil, fmt.Errorf("Error scanning deleted resolver IP: %v", err), http.StatusInternalServerError
		}
		ips = append(ips, ip)
	}
	return ips, nil, nil, http.StatusOK
}

func ReplaceFederationResolverMappingsForCurrentUser(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ips, userErr, sysErr, errCode := removeFederationResolverMappingsForCurrentUser(tx, inf.User)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	ipList := fmt.Sprintf("[ %s ]", strings.Join(ips, ", "))
	deletedMsg := fmt.Sprintf("%s successfully deleted all federation resolvers: %s", inf.User.UserName, ipList)
	changelogMsg := fmt.Sprintf("USER: %s, ID: %d, ACTION: %s", inf.User.UserName, inf.User.ID, deletedMsg)
	api.CreateChangeLogRawTx(api.ApiChange, changelogMsg, inf.User, tx)

	mappings, userErr, sysErr := getMappingsFromRequestBody(inf.Version, r.Body)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, sysErr)
		return
	}

	if err := mappings.Validate(tx); err != nil {
		errCode = http.StatusBadRequest
		userErr = fmt.Errorf("validating request: %v", err)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	userErr, sysErr, errCode = addFederationResolverMappingsForCurrentUser(inf.User, tx, mappings)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	createdMsg := fmt.Sprintf("%s successfully created federation resolvers.", inf.User.UserName)
	if inf.Version.Major <= 1 && inf.Version.Minor <= 3 {
		api.WriteResp(w, r, createdMsg)
		return
	}

	alerts := tc.Alerts{
		Alerts: []tc.Alert{
			tc.Alert{
				Level: tc.SuccessLevel.String(),
				Text:  deletedMsg,
			},
			tc.Alert{
				Level: tc.SuccessLevel.String(),
				Text:  createdMsg,
			},
		},
	}
	resp := struct {
		tc.Alerts
		Response string `json:"response"`
	}{
		alerts,
		createdMsg,
	}

	respBts, err := json.Marshal(resp)
	if err != nil {
		sysErr = fmt.Errorf("Marshalling response: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.Write(append(respBts, '\n'))
}

// retrieves mappings from the given request body using the rules of the given api Version
func getMappingsFromRequestBody(v api.Version, body io.ReadCloser) (tc.DeliveryServiceFederationResolverMappingRequest, error, error) {
	defer body.Close()
	var mappings tc.DeliveryServiceFederationResolverMappingRequest

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return mappings, errors.New("Couldn't read request"), fmt.Errorf("Reading request body: %v", err)
	}

	if v.Major <= 1 && v.Minor <= 3 {
		var req tc.LegacyDeliveryServiceFederationResolverMappingRequest
		if err := json.Unmarshal(b, &req); err != nil {
			return mappings, fmt.Errorf("parsing request: %v", err), nil
		}
		mappings = req.Federations
	} else {
		var req tc.DeliveryServiceFederationResolverMappingRequest

		// fall back on legacy behavior
		if err := json.Unmarshal(b, &req); err != nil {
			var request tc.LegacyDeliveryServiceFederationResolverMappingRequest
			if err = json.Unmarshal(b, &request); err != nil {
				return mappings, fmt.Errorf("parsing request: %v", err), nil
			}
			req = request.Federations
		}
		mappings = req
	}

	return mappings, nil, nil

}
