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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
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
	OtherExpirations       []DsExpirationInfo
}

const emailTemplateFile = "/opt/traffic_ops/app/templates/send_mail/autorenewcerts_mail.html"

func RenewCertificates(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, errors.New("the Riak service is unavailable"), errors.New("getting SSL keys from Riak by xml id: Riak is not configured"))
		return
	}

	rows, err := inf.Tx.Tx.Query(`SELECT xml_id, ssl_key_version FROM deliveryservice WHERE ssl_key_version != 0`)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	defer rows.Close()

	existingCerts := []ExistingCerts{}
	for rows.Next() {
		ds := DsKey{}
		err := rows.Scan(&ds.XmlId, &ds.Version)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		}
		existingCerts = append(existingCerts, ExistingCerts{Version: ds.Version, XmlId: ds.XmlId})
	}

	ctx, _ := context.WithTimeout(r.Context(), LetsEncryptTimeout*time.Duration(len(existingCerts)))

	go RunAutorenewal(existingCerts, inf.Config, ctx, inf.User)

	api.WriteRespAlert(w, r, tc.InfoLevel, "Beginning async call to renew Let's Encrypt certificates.  This may take a few minutes.")

}
func RunAutorenewal(existingCerts []ExistingCerts, cfg *config.Config, ctx context.Context, currentUser *auth.CurrentUser) {
	db, err := api.GetDB(ctx)
	if err != nil {
		log.Errorf("Error getting db: %s", err.Error())
		return
	}
	tx, err := db.Begin()
	if err != nil {
		log.Errorf("Error getting tx: %s", err.Error())
		return
	}

	logTx, err := db.Begin()
	if err != nil {
		log.Errorf("Error getting logTx: %s", err.Error())
		return
	}
	defer logTx.Commit()

	keysFound := ExpirationSummary{}

	for _, ds := range existingCerts {
		if !ds.Version.Valid || ds.Version.Int64 == 0 {
			continue
		}

		dsExpInfo := DsExpirationInfo{}
		keyObj, ok, err := riaksvc.GetDeliveryServiceSSLKeysObjV15(ds.XmlId, strconv.Itoa(int(ds.Version.Int64)), tx, cfg.RiakAuthOptions, cfg.RiakPort)
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

		err = base64DecodeCertificate(&keyObj.Certificate)
		if err != nil {
			log.Errorf("cert autorenewal: error getting SSL keys for XMLID '%s': %s", ds.XmlId, err.Error())
			return
		}

		expiration, err := parseExpirationFromCert([]byte(keyObj.Certificate.Crt))
		if err != nil {
			log.Errorf("cert autorenewal: %s: %s", ds.XmlId, err.Error())
			return
		}

		// Renew only certificates within configured limit
		if expiration.After(time.Now().Add(time.Hour * 24 * time.Duration(cfg.ConfigLetsEncrypt.RenewDaysBeforeExpiration))) {
			continue
		}

		log.Debugf("renewing certificate for xmlId = %s, version = %d, and auth type = %s ", ds.XmlId, ds.Version.Int64, keyObj.AuthType)

		newVersion := util.JSONIntStr(keyObj.Version.ToInt64() + 1)

		dsExpInfo.XmlId = keyObj.DeliveryService
		dsExpInfo.Version = keyObj.Version
		dsExpInfo.Expiration = expiration
		dsExpInfo.AuthType = keyObj.AuthType

		if keyObj.AuthType == tc.LetsEncryptAuthType || (keyObj.AuthType == tc.SelfSignedCertAuthType && cfg.ConfigLetsEncrypt.ConvertSelfSigned) {
			req := tc.DeliveryServiceLetsEncryptSSLKeysReq{
				DeliveryServiceSSLKeysReq: tc.DeliveryServiceSSLKeysReq{
					HostName:        &keyObj.Hostname,
					DeliveryService: &keyObj.DeliveryService,
					CDN:             &keyObj.CDN,
					Version:         &newVersion,
				},
			}

			if error := GetLetsEncryptCertificates(cfg, req, ctx, currentUser); error != nil {
				dsExpInfo.Error = error
			}
			keysFound.LetsEncryptExpirations = append(keysFound.LetsEncryptExpirations, dsExpInfo)

		} else if keyObj.AuthType == tc.SelfSignedCertAuthType {
			keysFound.SelfSignedExpirations = append(keysFound.SelfSignedExpirations, dsExpInfo)
		} else {
			keysFound.OtherExpirations = append(keysFound.OtherExpirations, dsExpInfo)
		}

	}

	if cfg.SMTP.Enabled && cfg.ConfigLetsEncrypt.SendExpEmail {
		errs := AlertExpiringCerts(keysFound, *cfg)
		if errs.Occurred() {
			log.Errorf("cert autorenewal: sending email: %s", errs)
			return
		}

	}
}

func AlertExpiringCerts(certsFound ExpirationSummary, config config.Config) apierrors.Errors {
	header := "From: " + config.ConfigTO.EmailFrom.String() + "\r\n" +
		"To: " + config.ConfigLetsEncrypt.Email + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
		"Subject: Certificate Expiration Summary\r\n\r\n"

	return api.SendEmailFromTemplate(config, header, certsFound, emailTemplateFile, config.ConfigLetsEncrypt.Email)
}

type ExistingCerts struct {
	Version sql.NullInt64
	XmlId   string
}
