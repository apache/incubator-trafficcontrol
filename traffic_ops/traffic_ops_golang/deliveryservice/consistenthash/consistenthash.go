package consistenthash

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
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// struct for the response object from Traffic Router
type TRConsistentHashResult struct {
	ResultingPathToConsistentHash string `json:"resultingPathToConsistentHash"`
	ConsistentHashRegex           string `json:"consistentHashRegex"`
	RequestPath                   string `json:"requestPath"`
}

// struct for the incoming request object
type TRConsistentHashRequest struct {
	ConsistentHashRegex string `json:"regex"`
	RequestPath         string `json:"requestPath"`
	CdnID               int64  `json:"cdnId"`
}

// endpoint to test Traffic Router's Pattern-Based Consistent Hashing feature
// /api/1.5/consistenthash
func Post(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	req := TRConsistentHashRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}

	responseFromTR, err := getPatternBasedConsistentHash(inf.Tx.Tx, req.ConsistentHashRegex, req.RequestPath, req.CdnID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting pattern based consistent hash from Traffic Router: "+err.Error()))
		return
	}

	consistentHashResult := TRConsistentHashResult{}
	json.Unmarshal(responseFromTR, &consistentHashResult)

	api.WriteResp(w, r, consistentHashResult)
}

const RouterRequestTimeout = time.Second * 10

// queries database for active Traffic Router on the CDN specified by cdnId
// passes regex and requestPath to the Traffic Router via API request,
// and returns the response
func getPatternBasedConsistentHash(tx *sql.Tx, regex string, requestPath string, cdnId int64) ([]byte, error) {
	q := `
SELECT concat(server.host_name, '.', server.domain_name) AS fqdn,
   parameter.value AS apiport
  FROM (((server
    JOIN profile ON ((profile.id = server.profile)))
    JOIN profile_parameter ON ((profile_parameter.profile = profile.id)))
    JOIN parameter ON ((parameter.id = profile_parameter.parameter)))
    JOIN status ON ((server.status = status.id))
 WHERE ((server.type = (select id from type where name = 'CCR')) AND
 (parameter.name = 'api.port'::text) AND
 (status.name = 'ONLINE') AND
 (server.cdn_id = $1))
ORDER BY RANDOM()
LIMIT 1
`

	trafficRouter := ""
	apiPort := ""
	if err := tx.QueryRow(q, cdnId).Scan(&trafficRouter, &apiPort); err != nil {
		return nil, errors.New("querying for eligible Traffic Router to test pattern based consistent hashing: " + err.Error())
	}

	if trafficRouter == "" {
		return nil, errors.New("no eligible Traffic Router found for pattern based consistent hashing with cdn Id: " + strconv.FormatInt(cdnId, 10))
	}
	if apiPort == "" {
		return nil, errors.New("no parameter 'api.port' found for pattern based consistent hashing with cdn Id: " + strconv.FormatInt(cdnId, 10))
	}

	trafficRouterAPI := "http://" + trafficRouter + ":" + apiPort + "/crs/consistenthash/patternbased/regex?regex=" + url.QueryEscape(regex) + "&requestPath=" + url.QueryEscape(requestPath)

	trClient := &http.Client{
		Timeout: RouterRequestTimeout,
	}
	r, err := trClient.Get(trafficRouterAPI)

	if err != nil {
		return nil, errors.New("Error creating request to Traffic Router: " + err.Error())
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("failed to read body: " + err.Error())
	}

	if r.StatusCode != 200 {
		return nil, errors.New("Traffic Router returned " + strconv.Itoa(r.StatusCode))
	}
	return body, nil
}
