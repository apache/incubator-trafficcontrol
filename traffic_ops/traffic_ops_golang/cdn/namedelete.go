package cdn

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

func DeleteName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cdnName := tc.CDNName(inf.Params["name"])
	if ok, err := cdnExists(inf.Tx, cdnName); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking CDN existence: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx, http.StatusNotFound, nil, nil)
		return
	}
	if ok, err := cdnUnused(inf.Tx, cdnName); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking CDN usage: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx, http.StatusBadRequest, errors.New("Failed to delete cdn name = "+string(cdnName)+" has delivery services or servers"), nil)
		return
	}
	if err := deleteCDNByName(inf.Tx, tc.CDNName(cdnName)); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("deleting CDN: "+err.Error()))
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, "cdn was deleted.")
}

func deleteCDNByName(tx *sql.Tx, name tc.CDNName) error {
	if _, err := tx.Exec(`DELETE FROM cdn WHERE name = $1`, name); err != nil {
		return errors.New("deleting cdns: " + err.Error())
	}
	return nil
}

func cdnExists(tx *sql.Tx, name tc.CDNName) (bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM cdn WHERE name = $1`, name).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.New("querying cdn existence: " + err.Error())
	}
	return true, nil
}

func cdnUnused(tx *sql.Tx, name tc.CDNName) (bool, error) {
	useCount := 0
	if err := tx.QueryRow(`
WITH cdn_id as (
  SELECT id as v FROM cdn WHERE name = $1
)
SELECT
  (SELECT COUNT(*) FROM server WHERE server.cdn_id = (select v from cdn_id)) +
	(SELECT COUNT(*) FROM deliveryservice WHERE deliveryservice.cdn_id = (select v from cdn_id))
`, name).Scan(&useCount); err != nil {
		return false, errors.New("querying cdn use count: " + err.Error())
	}
	if useCount > 0 {
		return false, nil
	}
	return true, nil
}
