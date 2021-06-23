package deliveryservice

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
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

// GetURLKeysByID returns the URL sig keys for a delivery service identified by the id in the path parameter.
func GetURLKeysByID(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.GetURLKeysByID: Traffic Vault is not configured"))
		return
	}

	ds, ok, err := dbhelpers.GetDSNameFromID(inf.Tx.Tx, inf.IntParams["id"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from ID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
		return
	}

	dsTenantID, ok, err := getDSTenantIDByID(inf.Tx.Tx, inf.IntParams["id"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	keys, ok, err := inf.Vault.GetURLSigKeys(string(ds), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from Traffic Vault: "+err.Error()))
		return
	}
	if !ok {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "No url sig keys found", struct{}{})
		return
	}
	api.WriteResp(w, r, keys)
}

// GetURLKeysByName returns the URL sig keys for a delivery service identified by the xmlId in the path parameter.
func GetURLKeysByName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.GetURLKeysByName: Traffic Vault is not configured"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])

	dsTenantID, ok, err := getDSTenantIDByName(inf.Tx.Tx, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	keys, ok, err := inf.Vault.GetURLSigKeys(string(ds), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from Traffic Vault: "+err.Error()))
		return
	}
	if !ok {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "No url sig keys found", struct{}{})
		return
	}
	api.WriteResp(w, r, keys)
}

// CopyURLKeys copies the URL sig keys from a delivery service in the 'copy-name' path parameter
// to the delivery service in the 'name' path parameter.
func CopyURLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name", "copy-name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.CopyURLKeys: Traffic Vault is not configured"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])
	copyDS := tc.DeliveryServiceName(inf.Params["copy-name"])

	dsTenantID, ok, err := getDSTenantIDByName(inf.Tx.Tx, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	{
		copyDSTenantID, ok, err := getDSTenantIDByName(inf.Tx.Tx, copyDS)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
			return
		}
		if authorized, err := tenant.IsResourceAuthorizedToUserTx(*copyDSTenantID, inf.User, inf.Tx.Tx); err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		} else if !authorized {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this copy tenant"), nil)
			return
		}
	}

	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, string(copyDS))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.CopyURLKeys: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+string(copyDS)), nil)
		return
	}

	keys, ok, err := inf.Vault.GetURLSigKeys(string(copyDS), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from Traffic Vault: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("Unable to retrieve keys from Delivery Service '"+string(copyDS)+"'"), nil)
		return
	}

	if err := inf.Vault.PutURLSigKeys(string(ds), keys, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("setting URL Sig keys for '"+string(ds)+" copied from "+string(copyDS)+": "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(ds)+", ID: "+strconv.Itoa(dsID)+", ACTION: Copied URL sig keys from "+string(copyDS), inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully copied and stored keys")
}

// GenerateURLKeys generates new URL sig keys for the delivery service identified by the xmlId in the path parameter.
func GenerateURLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.GenerateURLKeys: Traffic Vault is not configured"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])

	dsTenantID, ok, err := getDSTenantIDByName(inf.Tx.Tx, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, string(ds))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateURLKeys: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+string(ds)), nil)
		return
	}

	keys, err := GenerateURLSigKeys()
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating URL sig keys: "+err.Error()))
		return
	}

	if err := inf.Vault.PutURLSigKeys(string(ds), keys, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("setting URL Sig keys for '"+string(ds)+": "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(ds)+", ID: "+strconv.Itoa(dsID)+", ACTION: Generated URL sig keys", inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully generated and stored keys")
}

// GenerateURLSigKeys generates new URL sig keys.
func GenerateURLSigKeys() (tc.URLSigKeys, error) {
	chars := `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_`
	numKeys := 16
	numChars := 32
	keys := map[string]string{}
	for i := 0; i < numKeys; i++ {
		v := ""
		for i := 0; i < numChars; i++ {
			bi, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				return nil, errors.New("generating crypto rand int: " + err.Error())
			}
			if !bi.IsInt64() {
				return nil, fmt.Errorf("crypto rand int returned non-int64")
			}
			i := bi.Int64()
			if i >= int64(len(chars)) {
				return nil, fmt.Errorf("crypto rand int returned a number larger than requested")
			}
			v += string(chars[int(i)])
		}
		key := "key" + strconv.Itoa(i)
		keys[key] = v
	}
	return keys, nil
}

// DeleteURLKeysByID deletes the URL sig keys for the delivery service identified by the id in the path parameter.
func DeleteURLKeysByID(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteURLKeysByID: Traffic Vault is not configured"))
		return
	}

	dsId := inf.IntParams["id"]
	ds, ok, err := dbhelpers.GetDSNameFromID(inf.Tx.Tx, dsId)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from ID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
		return
	}

	dsTenantID, ok, err := getDSTenantIDByID(inf.Tx.Tx, inf.IntParams["id"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	err = inf.Vault.DeleteURLSigKeys(string(ds), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting URL Sig keys from Traffic Vault: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(ds)+", ID: "+strconv.Itoa(dsId)+", ACTION: Deleted URL sig keys", inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully deleted URL Sig keys from Traffic Vault")
}

// DeleteURLKeysByName deletes the URL sig keys for the delivery service identified by the xmlId in the path parameter.
func DeleteURLKeysByName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteURLKeysByName: Traffic Vault is not configured"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])

	dsTenantID, ok, err := getDSTenantIDByName(inf.Tx.Tx, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
		return
	}
	if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
		return
	} else if !authorized {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	dsId, ok, err := getDSIDFromName(inf.Tx.Tx, string(ds))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.DeleteURLKeysByName: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+string(ds)), nil)
		return
	}

	err = inf.Vault.DeleteURLSigKeys(string(ds), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting URL Sig keys from Traffic Vault: "+err.Error()))
		return
	}

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(ds)+", ID: "+strconv.Itoa(dsId)+", ACTION: Deleted URL sig keys", inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully deleted URL Sig keys from Traffic Vault")
}
