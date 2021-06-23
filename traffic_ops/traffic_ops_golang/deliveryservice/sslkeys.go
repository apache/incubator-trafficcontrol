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
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
)

// GenerateSSLKeys generates a new private key, certificate signing request and
// certificate based on the values submitted. It then stores these values in
// TrafficVault and updates the SSL key version.
func GenerateSSLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateSSLKeys: Traffic Vault is not configured"))
		return
	}

	req := tc.DeliveryServiceGenSSLKeysReq{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parsing request: "+err.Error()), nil)
		return
	}
	if userErr, sysErr, errCode := tenant.Check(inf.User, *req.DeliveryService, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, *req.DeliveryService)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateSSLKeys: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+*req.DeliveryService), nil)
		return
	}
	if err := generatePutRiakKeys(req, inf.Tx.Tx, inf.Vault, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating and putting SSL keys: "+err.Error()))
		return
	}
	if err := updateSSLKeyVersion(*req.DeliveryService, req.Version.ToInt64(), inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating SSL keys for delivery service '"+*req.DeliveryService+"': "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: Generated SSL keys", inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, "Successfully created ssl keys for "+*req.DeliveryService)
}

// generatePutRiakKeys generates a certificate, csr, and key from the given request, and insert it into the Riak key database.
// The req MUST be validated, ensuring required fields exist.
func generatePutRiakKeys(req tc.DeliveryServiceGenSSLKeysReq, tx *sql.Tx, tv trafficvault.TrafficVault, ctx context.Context) error {
	dsSSLKeys := tc.DeliveryServiceSSLKeys{
		CDN:             *req.CDN,
		DeliveryService: *req.DeliveryService,
		BusinessUnit:    *req.BusinessUnit,
		City:            *req.City,
		Organization:    *req.Organization,
		Hostname:        *req.HostName,
		Country:         *req.Country,
		State:           *req.State,
		Key:             *req.Key,
		Version:         *req.Version,
	}
	csr, crt, key, err := GenerateCert(*req.HostName, *req.Country, *req.City, *req.State, *req.Organization, *req.BusinessUnit)
	if err != nil {
		return errors.New("generating certificate: " + err.Error())
	}
	dsSSLKeys.Certificate = tc.DeliveryServiceSSLKeysCertificate{Crt: string(crt), Key: string(key), CSR: string(csr)}

	dsSSLKeys.AuthType = tc.SelfSignedCertAuthType

	if err := tv.PutDeliveryServiceSSLKeys(dsSSLKeys, tx, ctx); err != nil {
		return errors.New("putting keys in Traffic Vault: " + err.Error())
	}
	return nil
}
