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
	"errors"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"

	"github.com/go-acme/lego/certificate"
	"github.com/jmoiron/sqlx"
)

// RenewAcmeCertificate renews the SSL certificate for a delivery service if possible through ACME protocol.
func RenewAcmeCertificate(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"xmlid"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Traffic Vault is not configured"))
		return
	}
	xmlID := inf.Params["xmlid"]

	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	ctx, cancelTx := context.WithTimeout(r.Context(), AcmeTimeout)
	defer cancelTx()

	userErr, sysErr, statusCode := renewAcmeCerts(inf.Config, xmlID, ctx, r.Context(), inf.User, inf.Vault)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, statusCode, userErr, sysErr)
		return
	}

	api.WriteRespAlert(w, r, tc.SuccessLevel, "Certificate for "+xmlID+" successfully renewed.")

}

func renewAcmeCerts(cfg *config.Config, dsName string, ctx context.Context, httpCtx context.Context, currentUser *auth.CurrentUser, tv trafficvault.TrafficVault) (error, error, int) {
	db, err := api.GetDB(ctx)
	if err != nil {
		log.Errorf(dsName+": Error getting db: %s", err.Error())
		return nil, err, http.StatusInternalServerError
	}

	tx, err := db.Begin()
	if err != nil {
		log.Errorf(dsName+": Error getting tx: %s", err.Error())
		return nil, err, http.StatusInternalServerError
	}

	userTx, err := db.Begin()
	if err != nil {
		log.Errorf(dsName+": Error getting userTx: %s", err.Error())
		return nil, err, http.StatusInternalServerError
	}
	defer userTx.Commit()

	logTx, err := db.Begin()
	if err != nil {
		log.Errorf(dsName+": Error getting logTx: %s", err.Error())
		return nil, err, http.StatusInternalServerError
	}
	defer logTx.Commit()

	dsID, certVersion, err := getDSIdAndVersionFromName(db, dsName)
	if err != nil {
		return nil, errors.New("querying DS info: " + err.Error()), http.StatusInternalServerError
	}
	if dsID == nil || *dsID == 0 {
		return errors.New("DS id for " + dsName + " was nil or 0"), nil, http.StatusBadRequest
	}
	if certVersion == nil || *certVersion == 0 {
		return errors.New("certificate for " + dsName + " could not be renewed because version was nil or 0"), nil, http.StatusBadRequest
	}

	if cfg == nil {
		return nil, errors.New("acme: config was nil"), http.StatusInternalServerError
	}
	keyObj, ok, err := tv.GetDeliveryServiceSSLKeys(dsName, strconv.Itoa(int(*certVersion)), tx, httpCtx)
	if err != nil {
		return nil, errors.New("getting ssl keys for xmlId: " + dsName + " and version: " + strconv.Itoa(int(*certVersion)) + " : " + err.Error()), http.StatusInternalServerError
	}
	if !ok {
		return nil, errors.New("no object found for the specified key with xmlId: " + dsName + " and version: " + strconv.Itoa(int(*certVersion))), http.StatusInternalServerError
	}

	err = Base64DecodeCertificate(&keyObj.Certificate)
	if err != nil {
		return nil, errors.New("decoding cert for XMLID " + dsName + " : " + err.Error()), http.StatusInternalServerError
	}

	acmeAccount := GetAcmeAccountConfig(cfg, keyObj.AuthType)
	if acmeAccount == nil {
		return nil, errors.New("No acme account information in cdn.conf for " + keyObj.AuthType), http.StatusInternalServerError
	}

	client, err := GetAcmeClient(acmeAccount, userTx, db)
	if err != nil {
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+dsName+", ID: "+strconv.Itoa(*dsID)+", ACTION: FAILED to add SSL keys with "+acmeAccount.AcmeProvider, currentUser, logTx)
		return nil, errors.New("getting acme client: " + err.Error()), http.StatusInternalServerError
	}

	renewRequest := certificate.Resource{
		Certificate: []byte(keyObj.Certificate.Crt),
	}

	cert, err := client.Certificate.Renew(renewRequest, true, false)
	if err != nil {
		log.Errorf("Error obtaining acme certificate: %s", err.Error())
		return nil, err, http.StatusInternalServerError
	}
	if cert == nil {
		log.Errorf("Error obtaining acme certificate: certificate was nil")
		return nil, errors.New("certificate was nil"), http.StatusInternalServerError
	}

	newCertObj := tc.DeliveryServiceSSLKeys{
		AuthType:        keyObj.AuthType,
		CDN:             keyObj.CDN,
		DeliveryService: keyObj.DeliveryService,
		Key:             keyObj.DeliveryService,
		Hostname:        keyObj.Hostname,
		Version:         keyObj.Version + 1,
	}

	newCertObj.Certificate = tc.DeliveryServiceSSLKeysCertificate{
		Crt: string(EncodePEMToLegacyPerlRiakFormat(cert.Certificate)),
		Key: string(EncodePEMToLegacyPerlRiakFormat(cert.PrivateKey)),
		CSR: string(EncodePEMToLegacyPerlRiakFormat([]byte("ACME Generated"))),
	}

	if err := tv.PutDeliveryServiceSSLKeys(newCertObj, tx, httpCtx); err != nil {
		log.Errorf("Error posting acme certificate to Traffic Vault: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+dsName+", ID: "+strconv.Itoa(*dsID)+", ACTION: FAILED to add SSL keys with "+acmeAccount.AcmeProvider, currentUser, logTx)
		return nil, errors.New(dsName + ": putting keys in Traffic Vault: " + err.Error()), http.StatusInternalServerError
	}

	tx2, err := db.Begin()
	if err != nil {
		log.Errorf("starting sql transaction for delivery service " + dsName + ": " + err.Error())
		return nil, errors.New("starting sql transaction for delivery service " + dsName + ": " + err.Error()), http.StatusInternalServerError
	}

	if err := updateSSLKeyVersion(dsName, *certVersion+1, tx2); err != nil {
		log.Errorf("updating SSL key version for delivery service '" + dsName + "': " + err.Error())
		return nil, errors.New("updating SSL key version for delivery service '" + dsName + "': " + err.Error()), http.StatusInternalServerError
	}
	tx2.Commit()

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+dsName+", ID: "+strconv.Itoa(*dsID)+", ACTION: Added SSL keys with "+acmeAccount.AcmeProvider, currentUser, logTx)

	return nil, nil, http.StatusOK
}

func getDSIdAndVersionFromName(db *sqlx.DB, xmlId string) (*int, *int64, error) {
	var dsID int
	var certVersion int64

	if err := db.QueryRow(`SELECT id, ssl_key_version FROM deliveryservice WHERE xml_id = $1`, xmlId).Scan(&dsID, &certVersion); err != nil {
		return nil, nil, err
	}

	return &dsID, &certVersion, nil
}
