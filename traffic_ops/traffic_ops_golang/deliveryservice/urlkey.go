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
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

func GetURLKeysByID(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Riak is not configured!"))
		return
	}

	ds, ok, err := GetDSNameFromID(inf.Tx, inf.IntParams["id"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from ID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
		return
	}

	// TODO create a helper function to check all this in a single line.
	ok, err = tenant.IsTenancyEnabledTx(inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenancy enabled: "+err.Error()))
		return
	}
	if ok {
		dsTenantID, ok, err := GetDSTenantIDByIDTx(inf.Tx, inf.IntParams["id"])
		if err != nil {
			api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+inf.Params["id"]+" not found"), nil)
			return
		}
		if dsTenantID != nil {
			if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx); err != nil {
				api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
				return
			} else if !authorized {
				api.HandleErr(w, r, inf.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}
	}

	keys, ok, err := riaksvc.GetURLSigKeys(inf.Tx, inf.Config.RiakAuthOptions, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from riak: "+err.Error()))
		return
	}
	if !ok {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "No url sig keys found", struct{}{})
		return
	}
	api.WriteResp(w, r, keys)
}

func GetURLKeysByName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Riak is not configured!"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])

	// TODO create a helper function to check all this in a single line.
	ok, err := tenant.IsTenancyEnabledTx(inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenancy enabled: "+err.Error()))
		return
	}
	if ok {
		dsTenantID, ok, err := GetDSTenantIDByNameTx(inf.Tx, ds)
		if err != nil {
			api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
			return
		}
		if dsTenantID != nil {
			if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx); err != nil {
				api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
				return
			} else if !authorized {
				api.HandleErr(w, r, inf.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}
	}

	keys, ok, err := riaksvc.GetURLSigKeys(inf.Tx, inf.Config.RiakAuthOptions, ds)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from riak: "+err.Error()))
		return
	}
	if !ok {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, "No url sig keys found", struct{}{})
		return
	}
	api.WriteResp(w, r, keys)
}

func CopyURLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"name", "copy-name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Riak is not configured!"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])
	copyDS := tc.DeliveryServiceName(inf.Params["copy-name"])

	// TODO create a helper function to check all this in a single line.
	ok, err := tenant.IsTenancyEnabledTx(inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenancy enabled: "+err.Error()))
		return
	}
	if ok {
		dsTenantID, ok, err := GetDSTenantIDByNameTx(inf.Tx, ds)
		if err != nil {
			api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
			return
		}
		if dsTenantID != nil {
			if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx); err != nil {
				api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
				return
			} else if !authorized {
				api.HandleErr(w, r, inf.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}

		{
			copyDSTenantID, ok, err := GetDSTenantIDByNameTx(inf.Tx, copyDS)
			if err != nil {
				api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
				return
			}
			if !ok {
				api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
				return
			}
			if copyDSTenantID != nil {
				if authorized, err := tenant.IsResourceAuthorizedToUserTx(*copyDSTenantID, inf.User, inf.Tx); err != nil {
					api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
					return
				} else if !authorized {
					api.HandleErr(w, r, inf.Tx, http.StatusForbidden, errors.New("not authorized on this copy tenant"), nil)
					return
				}
			}
		}
	}

	keys, ok, err := riaksvc.GetURLSigKeys(inf.Tx, inf.Config.RiakAuthOptions, copyDS)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("getting URL Sig keys from riak: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx, http.StatusBadRequest, errors.New("Unable to retrieve keys from Delivery Service '"+string(copyDS)+"'"), nil)
		return
	}

	if err := riaksvc.PutURLSigKeys(inf.Tx, inf.Config.RiakAuthOptions, ds, keys); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("setting URL Sig keys for '"+string(ds)+" copied from "+string(copyDS)+": "+err.Error()))
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully copied and stored keys")
}

// GetDSNameFromID loads the DeliveryService's xml_id from the database, from the ID. Returns whether the delivery service was found, and any error.
// TODO move somewhere generic
func GetDSNameFromID(tx *sql.Tx, id int) (tc.DeliveryServiceName, bool, error) {
	name := tc.DeliveryServiceName("")
	if err := tx.QueryRow(`SELECT xml_id FROM deliveryservice where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return tc.DeliveryServiceName(""), false, nil
		}
		return tc.DeliveryServiceName(""), false, fmt.Errorf("querying xml_id for delivery service ID '%v': %v", id, err)
	}
	return name, true, nil
}

func GenerateURLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Riak is not configured!"))
		return
	}

	ds := tc.DeliveryServiceName(inf.Params["name"])

	// TODO create a helper function to check all this in a single line.
	ok, err := tenant.IsTenancyEnabledTx(inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenancy enabled: "+err.Error()))
		return
	}
	if ok {
		dsTenantID, ok, err := GetDSTenantIDByNameTx(inf.Tx, ds)
		if err != nil {
			api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx, http.StatusNotFound, errors.New("delivery service "+string(ds)+" not found"), nil)
			return
		}
		if dsTenantID != nil {
			if authorized, err := tenant.IsResourceAuthorizedToUserTx(*dsTenantID, inf.User, inf.Tx); err != nil {
				api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant: "+err.Error()))
				return
			} else if !authorized {
				api.HandleErr(w, r, inf.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}
	}

	keys, err := GenerateURLSigKeys()
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("generating URL sig keys: "+err.Error()))
	}

	if err := riaksvc.PutURLSigKeys(inf.Tx, inf.Config.RiakAuthOptions, ds, keys); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("setting URL Sig keys for '"+string(ds)+": "+err.Error()))
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Successfully generated and stored keys")
}

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
