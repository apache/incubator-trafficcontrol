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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
)

type DsKey struct {
	XmlId   string
	Version sql.NullInt64
}

type DsExpirationInfo struct {
	XmlId      string
	Version    util.JSONIntStr
	Expiration time.Time
	AuthType   string
	Error      error
}

type ExpirationSummary struct {
	LetsEncryptExpirations []DsExpirationInfo
	SelfSignedExpirations  []DsExpirationInfo
	AcmeExpirations        []DsExpirationInfo
	OtherExpirations       []DsExpirationInfo
}

const emailTemplateFile = "/opt/traffic_ops/app/templates/send_mail/autorenewcerts_mail.html"
const API_ACME_AUTORENEW = "acme_autorenew"

// RenewCertificatesDeprecated renews all SSL certificates that are expiring within a certain time limit with a deprecation alert.
//// This will renew Let's Encrypt and ACME certificates.
func RenewCertificatesDeprecated(w http.ResponseWriter, r *http.Request) {
	renewCertificates(w, r, true)
}

// RenewCertificates renews all SSL certificates that are expiring within a certain time limit.
// This will renew Let's Encrypt and ACME certificates.
func RenewCertificates(w http.ResponseWriter, r *http.Request) {
	renewCertificates(w, r, false)
}

func renewCertificates(w http.ResponseWriter, r *http.Request, deprecated bool) {
	deprecation := util.StrPtr(API_ACME_AUTORENEW)
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, errCode, userErr, sysErr, deprecated, deprecation)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, errors.New("the Traffic Vault service is unavailable"), errors.New("getting SSL keys from Traffic Vault by xml id: Traffic Vault is not configured"), deprecated, deprecation)
		return
	}

	rows, err := inf.Tx.Tx.Query(`SELECT xml_id, ssl_key_version FROM deliveryservice WHERE ssl_key_version != 0`)
	if err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err, deprecated, deprecation)
		return
	}
	defer rows.Close()

	existingCerts := []ExistingCerts{}
	for rows.Next() {
		ds := DsKey{}
		err := rows.Scan(&ds.XmlId, &ds.Version)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
			return
		}
		existingCerts = append(existingCerts, ExistingCerts{Version: ds.Version, XmlId: ds.XmlId})
	}

	ctx, cancelTx := context.WithTimeout(r.Context(), AcmeTimeout*time.Duration(len(existingCerts)))

	asyncStatusId, errCode, userErr, sysErr := api.InsertAsyncStatus(inf.Tx.Tx, "ACME async job has started.")
	if userErr != nil || sysErr != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	go RunAutorenewal(existingCerts, inf.Config, ctx, cancelTx, inf.User, asyncStatusId, inf.Vault)

	var alerts tc.Alerts
	if deprecated {
		alerts.AddAlerts(api.CreateDeprecationAlerts(deprecation))
	}

	alerts.AddAlert(tc.Alert{
		Text:  "Beginning async call to renew certificates. This may take a few minutes. Status updates can be found here: " + api.CurrentAsyncEndpoint + strconv.Itoa(asyncStatusId),
		Level: tc.SuccessLevel.String(),
	})

	w.Header().Add("Location", api.CurrentAsyncEndpoint+strconv.Itoa(asyncStatusId))
	api.WriteAlerts(w, r, http.StatusAccepted, alerts)

}
func RunAutorenewal(existingCerts []ExistingCerts, cfg *config.Config, ctx context.Context, cancelTx context.CancelFunc, currentUser *auth.CurrentUser, asyncStatusId int, tv trafficvault.TrafficVault) {
	defer cancelTx()
	db, err := api.GetDB(ctx)
	if err != nil {
		log.Errorf("Error getting db: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return
	}
	tx, err := db.Begin()
	if err != nil {
		log.Errorf("Error getting tx: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return
	}

	logTx, err := db.Begin()
	if err != nil {
		log.Errorf("Error getting logTx: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return
	}
	defer logTx.Commit()

	keysFound := ExpirationSummary{}

	renewedCount := 0
	errorCount := 0

	for _, ds := range existingCerts {
		if !ds.Version.Valid || ds.Version.Int64 == 0 {
			continue
		}

		dsExpInfo := DsExpirationInfo{}
		keyObj, ok, err := tv.GetDeliveryServiceSSLKeys(ds.XmlId, strconv.Itoa(int(ds.Version.Int64)), tx, ctx)
		if err != nil {
			log.Errorf("getting ssl keys for xmlId: %s and version: %d : %s", ds.XmlId, ds.Version.Int64, err.Error())
			dsExpInfo.XmlId = ds.XmlId
			dsExpInfo.Version = util.JSONIntStr(int(ds.Version.Int64))
			dsExpInfo.Error = errors.New("getting ssl keys for xmlId: " + ds.XmlId + " and version: " + strconv.Itoa(int(ds.Version.Int64)) + " :" + err.Error())
			keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
			continue
		}
		if !ok {
			log.Errorf("no object found for the specified key with xmlId: %s and version: %d", ds.XmlId, ds.Version.Int64)
			dsExpInfo.XmlId = ds.XmlId
			dsExpInfo.Version = util.JSONIntStr(int(ds.Version.Int64))
			dsExpInfo.Error = errors.New("no object found for the specified key with xmlId: " + ds.XmlId + " and version: " + strconv.Itoa(int(ds.Version.Int64)))
			keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
			continue
		}

		err = Base64DecodeCertificate(&keyObj.Certificate)
		if err != nil {
			log.Errorf("cert autorenewal: error getting SSL keys for XMLID '%s': %s", ds.XmlId, err.Error())
			dsExpInfo.XmlId = ds.XmlId
			dsExpInfo.Version = util.JSONIntStr(int(ds.Version.Int64))
			dsExpInfo.Error = errors.New("decoding the certificate for xmlId: " + ds.XmlId + " and version: " + strconv.Itoa(int(ds.Version.Int64)))
			keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
			continue
		}

		expiration, err := parseExpirationFromCert([]byte(keyObj.Certificate.Crt))
		if err != nil {
			log.Errorf("cert autorenewal: %s: %s", ds.XmlId, err.Error())
			dsExpInfo.XmlId = ds.XmlId
			dsExpInfo.Version = util.JSONIntStr(int(ds.Version.Int64))
			dsExpInfo.Error = errors.New("parsing the expiration for xmlId: " + ds.XmlId + " and version: " + strconv.Itoa(int(ds.Version.Int64)))
			keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
			continue
		}

		// Renew only certificates within configured limit. Default is 30 days.
		if cfg.ConfigAcmeRenewal.RenewDaysBeforeExpiration == 0 {
			cfg.ConfigAcmeRenewal.RenewDaysBeforeExpiration = 30
		}
		if expiration.After(time.Now().Add(time.Hour * 24 * time.Duration(cfg.ConfigAcmeRenewal.RenewDaysBeforeExpiration))) {
			continue
		}

		log.Debugf("renewing certificate for xmlId = %s, version = %d, and auth type = %s ", ds.XmlId, ds.Version.Int64, keyObj.AuthType)

		newVersion := util.JSONIntStr(keyObj.Version.ToInt64() + 1)

		dsExpInfo.XmlId = keyObj.DeliveryService
		dsExpInfo.Version = keyObj.Version
		dsExpInfo.Expiration = expiration
		dsExpInfo.AuthType = keyObj.AuthType

		if keyObj.AuthType == tc.LetsEncryptAuthType || (keyObj.AuthType == tc.SelfSignedCertAuthType && cfg.ConfigLetsEncrypt.ConvertSelfSigned) {
			req := tc.DeliveryServiceAcmeSSLKeysReq{
				DeliveryServiceSSLKeysReq: tc.DeliveryServiceSSLKeysReq{
					HostName:        &keyObj.Hostname,
					DeliveryService: &keyObj.DeliveryService,
					CDN:             &keyObj.CDN,
					Version:         &newVersion,
					AuthType:        &keyObj.AuthType,
				},
			}

			if err := GetAcmeCertificates(cfg, req, ctx, nil, false, currentUser, 0, tv); err != nil {
				dsExpInfo.Error = err
				errorCount++
			} else {
				renewedCount++
			}
			keysFound.LetsEncryptExpirations = append(keysFound.LetsEncryptExpirations, dsExpInfo)

		} else if keyObj.AuthType == tc.SelfSignedCertAuthType {
			keysFound.SelfSignedExpirations = append(keysFound.SelfSignedExpirations, dsExpInfo)
		} else {
			acmeAccount := GetAcmeAccountConfig(cfg, keyObj.AuthType)
			if acmeAccount == nil {
				keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
			} else {
				// background httpCtx since this is run in a goroutine spawned off the original http request
				// so the context isn't cancelled when the http connection is closed
				userErr, sysErr, statusCode := renewAcmeCerts(cfg, keyObj.DeliveryService, ctx, context.Background(), currentUser, tv)
				if userErr != nil {
					errorCount++
					dsExpInfo.Error = userErr
				} else if sysErr != nil {
					errorCount++
					dsExpInfo.Error = sysErr
				} else if statusCode != http.StatusOK {
					errorCount++
					dsExpInfo.Error = errors.New("Status code not 200: " + strconv.Itoa(statusCode))
				} else {
					renewedCount++
				}
				keysFound.AcmeExpirations = append(keysFound.AcmeExpirations, dsExpInfo)
			}

		}

		if asycErr := api.UpdateAsyncStatus(db, api.AsyncPending, "ACME renewal in progress. "+strconv.Itoa(renewedCount)+" certs renewed, "+strconv.Itoa(errorCount)+" errors.", asyncStatusId, false); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}

	}

	// put status as succeeded if any certs were successfully renewed
	asyncStatus := api.AsyncSucceeded
	if errorCount > 0 && renewedCount == 0 {
		asyncStatus = api.AsyncFailed
	}
	if asycErr := api.UpdateAsyncStatus(db, asyncStatus, "ACME renewal complete. "+strconv.Itoa(renewedCount)+" certs renewed, "+strconv.Itoa(errorCount)+" errors.", asyncStatusId, true); asycErr != nil {
		log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
	}

	if cfg.SMTP.Enabled && cfg.ConfigAcmeRenewal.SummaryEmail != "" {
		errCode, userErr, sysErr := AlertExpiringCerts(keysFound, *cfg)
		if userErr != nil || sysErr != nil {
			log.Errorf("cert autorenewal: sending email: errCode: %d userErr: %v sysErr: %v", errCode, userErr, sysErr)
			return
		}

	}
}

func AlertExpiringCerts(certsFound ExpirationSummary, config config.Config) (int, error, error) {
	header := "From: " + config.ConfigTO.EmailFrom.String() + "\r\n" +
		"To: " + config.ConfigAcmeRenewal.SummaryEmail + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
		"Subject: Certificate Expiration Summary\r\n\r\n"

	return api.SendEmailFromTemplate(config, header, certsFound, emailTemplateFile, config.ConfigAcmeRenewal.SummaryEmail)
}

type ExistingCerts struct {
	Version sql.NullInt64
	XmlId   string
}
