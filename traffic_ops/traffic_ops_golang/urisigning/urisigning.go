package urisigning

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
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

// endpoint handler for fetching uri signing keys from riak
func GetURIsignkeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusServiceUnavailable, errors.New("the Traffic Vault service is unavailable"), errors.New("getting Traffic Vault SSL keys by host name: Traffic Vault is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]

	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	ro, _, err := inf.Vault.GetURISigningKeys(xmlID, inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting URI signing keys: "+err.Error()))
		return
	}
	if len(ro) == 0 {
		ro, err = json.Marshal(tc.URISignerKeyset{})
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("marshalling empty URISignerKeyset: "+err.Error()))
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(ro)
}

// removeDeliveryServiceURIKeysHandler is the HTTP DELETE handler used to remove urisigning keys assigned to a delivery service.
func RemoveDeliveryServiceURIKeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusServiceUnavailable, errors.New("rhe Traffic Vault service is unavailable"), errors.New("getting Traffic Vault SSL keys by host name: Traffic Vault is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, xmlID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, errors.New("error finding delivery service with xmlID: "+xmlID), errors.New("getting DS id from name failed: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}

	_, found, err := inf.Vault.GetURISigningKeys(xmlID, inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("removing URI signing keys: "+err.Error()))
		return
	}

	if !found {
		api.WriteRespAlert(w, r, tc.InfoLevel, "not deleted, no object found to delete")
		return
	}
	if err := inf.Vault.DeleteURISigningKeys(xmlID, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("removing URI signing keys: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+xmlID+", ID: "+strconv.Itoa(dsID)+", ACTION: Removed URI signing keys", inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "object deleted")
	return
}

// saveDeliveryServiceURIKeysHandler is the HTTP POST or PUT handler used to store urisigning keys to a delivery service.
func SaveDeliveryServiceURIKeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusServiceUnavailable, errors.New("the Traffic Vault service is unavailable"), errors.New("getting Traffic Vault SSL keys by host name: Traffic Vault is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, xmlID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, errors.New("error finding delivery service with xmlID: "+xmlID), errors.New("getting DS id from name failed: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, errors.New("failed to read body"), errors.New("failed to read body: "+err.Error()))
		return
	}
	keySet := map[string]tc.URISignerKeyset{}
	if err := json.Unmarshal(data, &keySet); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON"), nil)
		return
	}
	if err := validateURIKeyset(keySet); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("invalid keyset: "+err.Error()), nil)
		return
	}

	if err := inf.Vault.PutURISigningKeys(xmlID, data, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("saving URI signing keys: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+xmlID+", ID: "+strconv.Itoa(dsID)+", ACTION: Stored URI signing keys to a delivery service", inf.User, inf.Tx.Tx)
	w.Header().Set("Content-Type", rfc.ApplicationJSON)
	w.Write(data)
}

// getDSIDFromName loads the DeliveryService's ID from the database, from the xml_id. Returns whether the delivery service was found, and any error.
func getDSIDFromName(tx *sql.Tx, xmlID string) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM deliveryservice WHERE xml_id = $1`, xmlID).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		return id, false, fmt.Errorf("querying ID for delivery service ID '%v': %v", xmlID, err)
	}
	return id, true, nil
}

// validateURIKeyset validates URISigingKeyset json.
func validateURIKeyset(msg map[string]tc.URISignerKeyset) error {
	var renewalKidFound int
	var renewalKidMatched = false

	for key, value := range msg {
		issuer := key
		renewalKid := value.RenewalKid
		if issuer == "" {
			return errors.New("JSON Keyset has no issuer")
		}

		if renewalKid != nil {
			renewalKidFound++
		}

		for _, skey := range value.Keys {
			if skey.Algorithm == "" {
				return errors.New("A Key has no algorithm, alg, specified")
			}
			if skey.KeyID == "" {
				return errors.New("A Key has no key id, kid, specified")
			}
			if renewalKid != nil && strings.Compare(*renewalKid, skey.KeyID) == 0 {
				renewalKidMatched = true
			}
		}
	}

	// should only have one renewal_kid
	switch renewalKidFound {
	case 0:
		return errors.New("No renewal_kid was found in any keyset")
	case 1: // okay, this is what we want
		break
	default:
		return errors.New("More than one renewal_kid was found in the keysets")
	}

	// the renewal_kid should match the kid of one key
	if !renewalKidMatched {
		return errors.New("No key was found with a kid that matches the renewal kid")
	}

	return nil
}
