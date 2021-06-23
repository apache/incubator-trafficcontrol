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
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/certificate"
	"github.com/go-acme/lego/challenge"
	"github.com/go-acme/lego/challenge/dns01"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"github.com/jmoiron/sqlx"
)

const validAccountStatus = "valid"
const AcmeTimeout = time.Minute * 20
const API_ACME_GENERATE_LE = "/deliveryservices/sslkeys/generate/acme"

// MyUser stores the user's information for use in ACME protocol.
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

// GetEmail returns a user's email for use in ACME protocol.
func (u *MyUser) GetEmail() string {
	return u.Email
}

// GetRegistration returns a user's registration for use in ACME protocol.
func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}

// GetPrivateKey returns a user's private key for use in ACME protocol.
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// DNSProviderTrafficRouter is used in the lego library and contains a database in order to store the DNS challenges for ACME protocol.
type DNSProviderTrafficRouter struct {
	db *sqlx.DB
}

// NewDNSProviderTrafficRouter returns a new DNSProviderTrafficRouter object.
func NewDNSProviderTrafficRouter() *DNSProviderTrafficRouter {
	return &DNSProviderTrafficRouter{}
}

// Timeout returns timeout information for the lego library including the timeout duration and the interval between checks.
func (d *DNSProviderTrafficRouter) Timeout() (timeout, interval time.Duration) {
	return AcmeTimeout, time.Second * 30
}

// Present inserts the DNS challenge record into the database to be used by Traffic Router. This is used in the lego library.
func (d *DNSProviderTrafficRouter) Present(domain, token, keyAuth string) error {
	tx, err := d.db.Begin()
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	q := `INSERT INTO dnschallenges (fqdn, record) VALUES ($1, $2)`
	response, err := tx.Exec(q, fqdn, value)
	tx.Commit()
	if err != nil {
		log.Errorf("Inserting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		return fmt.Errorf("Inserting dns txt record for fqdn '"+fqdn+"' record '"+value+"': %v", err)
	} else {
		rows, err := response.RowsAffected()
		if err != nil {
			log.Errorf("Determining rows affected dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return fmt.Errorf("Determining rows affected dns txt record for fqdn '"+fqdn+"' record '"+value+"': %v", err)
		}
		if rows == 0 {
			log.Errorf("Zero rows affected when inserting dns txt record for fqdn '" + fqdn + "' record '" + value)
			return errors.New("Zero rows affected when inserting dns txt record for fqdn '" + fqdn + "' record '" + value)
		}
	}

	return nil
}

// CleanUp removes the DNS challenge record from the database after the challenge has completed. This is used in the lego library.
func (d *DNSProviderTrafficRouter) CleanUp(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	tx, err := d.db.Begin()

	q := `DELETE FROM dnschallenges WHERE fqdn = $1 and record = $2`
	response, err := tx.Exec(q, fqdn, value)
	tx.Commit()
	if err != nil {
		log.Errorf("Deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		return fmt.Errorf("Deleting dns txt record for fqdn '"+fqdn+"' record '"+value+"': %v", err)
	} else {
		rows, err := response.RowsAffected()
		if err != nil {
			log.Errorf("Determining rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return fmt.Errorf("Determining rows affected when deleting dns txt record for fqdn '"+fqdn+"' record '"+value+"': %v", err)
		}
		if rows == 0 {
			log.Errorf("Zero rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value)
			return errors.New("Zero rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value)
		}
	}

	return nil
}

// GenerateAcmeCertificates gets and saves certificates using ACME protocol from a give ACME provider.
func GenerateAcmeCertificates(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateAcmeCertificates: Traffic Vault is not configured"))
		return
	}
	ctx, cancelTx := context.WithTimeout(r.Context(), AcmeTimeout)

	req := tc.DeliveryServiceAcmeSSLKeysReq{}
	if err := api.Parse(r.Body, nil, &req); err != nil {
		defer cancelTx()
		api.HandleErr(w, r, nil, http.StatusBadRequest, fmt.Errorf("parsing request: %v", err), nil)
		return
	}
	if *req.DeliveryService == "" {
		req.DeliveryService = req.Key
	}

	dsID, cdnName, ok, err := dbhelpers.GetDSIDAndCDNFromName(inf.Tx.Tx, *req.DeliveryService)
	if err != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("deliveryservice.GenerateLetsEncryptCertificates: getting DS ID from name: %v", err))
		return
	} else if !ok {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+*req.DeliveryService), nil)
		return
	}

	userErr, sysErr, errCode = tenant.CheckID(inf.Tx.Tx, inf.User, dsID)
	if userErr != nil || sysErr != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	_, ok, err = dbhelpers.GetCDNIDFromName(inf.Tx.Tx, tc.CDNName(*req.CDN))
	if err != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("checking CDN existence: %v", err))
		return
	} else if !ok {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("cdn not found with name "+*req.CDN), nil)
		return
	}

	if cdnName != tc.CDNName(*req.CDN) {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("delivery service not in cdn"), nil)
		return
	}

	asyncStatusId, errCode, userErr, sysErr := api.InsertAsyncStatus(inf.Tx.Tx, "ACME async job has started.")
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
	}

	go GetAcmeCertificates(inf.Config, req, ctx, cancelTx, true, inf.User, asyncStatusId, inf.Vault)

	var alerts tc.Alerts
	alerts.AddAlert(tc.Alert{
		Text:  "Beginning async ACME call for " + *req.DeliveryService + " using " + *req.AuthType + ". This may take a few minutes. Status updates can be found here: " + api.CurrentAsyncEndpoint + strconv.Itoa(asyncStatusId),
		Level: tc.SuccessLevel.String(),
	})

	w.Header().Add("Location", api.CurrentAsyncEndpoint+strconv.Itoa(asyncStatusId))
	api.WriteAlerts(w, r, http.StatusAccepted, alerts)
}

// GenerateLetsEncryptCertificates gets and saves new certificates from Let's Encrypt.
func GenerateLetsEncryptCertificates(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateLetsEncryptCertificates: Traffic Vault is not configured"))
		return
	}

	ctx, cancelTx := context.WithTimeout(r.Context(), AcmeTimeout)

	req := tc.DeliveryServiceAcmeSSLKeysReq{}
	if req.AuthType == nil {
		req.AuthType = new(string)
		*req.AuthType = tc.LetsEncryptAuthType
	}

	if err := api.Parse(r.Body, nil, &req); err != nil {
		defer cancelTx()
		api.HandleErr(w, r, nil, http.StatusBadRequest, fmt.Errorf("parsing request: %v", err), nil)
		return
	}
	if *req.DeliveryService == "" {
		req.DeliveryService = req.Key
	}

	dsID, cdnName, ok, err := dbhelpers.GetDSIDAndCDNFromName(inf.Tx.Tx, *req.DeliveryService)
	if err != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("deliveryservice.GenerateLetsEncryptCertificates: getting DS ID from name: %v", err))
		return
	} else if !ok {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+*req.DeliveryService), nil)
		return
	}

	userErr, sysErr, errCode = tenant.CheckID(inf.Tx.Tx, inf.User, dsID)
	if userErr != nil || sysErr != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	_, ok, err = dbhelpers.GetCDNIDFromName(inf.Tx.Tx, tc.CDNName(*req.CDN))
	if err != nil {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("checking CDN existence: %v", err))
		return
	} else if !ok {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("cdn not found with name "+*req.CDN), nil)
		return
	}

	if cdnName != tc.CDNName(*req.CDN) {
		defer cancelTx()
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("delivery service not in cdn"), nil)
		return
	}

	asyncStatusId, errCode, userErr, sysErr := api.InsertAsyncStatus(inf.Tx.Tx, "ACME async job has started.")
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
	}

	go GetAcmeCertificates(inf.Config, req, ctx, cancelTx, true, inf.User, asyncStatusId, inf.Vault)

	var alerts tc.Alerts
	alerts.AddAlerts(api.CreateDeprecationAlerts(util.StrPtr(API_ACME_GENERATE_LE)))
	alerts.AddAlert(tc.Alert{
		Text:  "Beginning async call to Let's Encrypt for " + *req.DeliveryService + ". This may take a few minutes. Status updates can be found here: " + api.CurrentAsyncEndpoint + strconv.Itoa(asyncStatusId),
		Level: tc.SuccessLevel.String(),
	})

	w.Header().Add("Location", api.CurrentAsyncEndpoint+strconv.Itoa(asyncStatusId))
	api.WriteAlerts(w, r, http.StatusAccepted, alerts)
}

// GetAcmeCertificates gets or creates an ACME account based on the provider, then gets new certificates for the delivery service requested and saves them to Vault.
func GetAcmeCertificates(cfg *config.Config, req tc.DeliveryServiceAcmeSSLKeysReq, ctx context.Context, cancelTx context.CancelFunc, shouldCancelTx bool, currentUser *auth.CurrentUser, asyncStatusId int, tv trafficvault.TrafficVault) error {
	defer func() {
		if shouldCancelTx {
			defer cancelTx()
		}
		if err := recover(); err != nil {
			db, dbErr := api.GetDB(ctx)
			if dbErr != nil {
				log.Errorf(*req.DeliveryService+": Error getting db for recover async update: %s", dbErr.Error())
				log.Errorf("panic: (err: %v) stacktrace:\n%s\n", err, util.Stacktrace())
				return
			}

			if asyncErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asyncErr != nil {
				log.Errorf("updating async status for id %v: %v", asyncStatusId, asyncErr)
			}
			log.Errorf("panic: (err: %v) stacktrace:\n%s\n", err, util.Stacktrace())
			return
		}
	}()

	db, err := api.GetDB(ctx)
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting db: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting tx: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}
	userTx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting userTx: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}
	defer userTx.Commit()

	logTx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting logTx: %s", err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}
	defer logTx.Commit()

	domainName := *req.HostName
	deliveryService := *req.DeliveryService
	provider := *req.AuthType

	dsID, ok, err := getDSIDFromName(tx, *req.DeliveryService)
	if err != nil {
		log.Errorf("deliveryservice.GenerateSSLKeys: getting DS ID from name " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return fmt.Errorf("deliveryservice.GenerateSSLKeys: getting DS ID from name: %v", err)
	} else if !ok {
		log.Errorf("no DS with name " + *req.DeliveryService)
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return errors.New("no DS with name " + *req.DeliveryService)
	}
	tx.Commit()

	if cfg == nil {
		log.Errorf("acme: config was nil for provider %s", provider)
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return errors.New("acme: config was nil")
	}

	var account *config.ConfigAcmeAccount
	if provider == tc.LetsEncryptAuthType {
		letsEncryptAccount := config.ConfigAcmeAccount{
			UserEmail:    cfg.ConfigLetsEncrypt.Email,
			AcmeProvider: tc.LetsEncryptAuthType,
		}

		if strings.EqualFold(cfg.ConfigLetsEncrypt.Environment, "staging") {
			letsEncryptAccount.AcmeUrl = lego.LEDirectoryStaging // provides certificate signed by invalid authority for testing purposes
		} else {
			letsEncryptAccount.AcmeUrl = lego.LEDirectoryProduction // provides certificate signed by valid LE authority
		}
		account = &letsEncryptAccount
	} else {
		acmeAccount := GetAcmeAccountConfig(cfg, provider)
		if acmeAccount == nil {
			log.Errorf("acme: no account information found for %s", provider)
			api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
			if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
				log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
			}
			return errors.New("No acme account information in cdn.conf for " + provider)
		}
		account = acmeAccount
	}

	client, err := GetAcmeClient(account, userTx, db)
	if err != nil {
		log.Errorf("acme: getting acme client for provider %s: %v", provider, err)
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return fmt.Errorf("getting acme client: %v", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf(deliveryService + ": Error generating private key: " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}
	request := certificate.ObtainRequest{
		Domains:    []string{domainName},
		Bundle:     true,
		PrivateKey: priv,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Errorf(deliveryService+": Error obtaining acme certificate from %s: %s", provider, err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}

	// Save certs into Traffic Vault
	dsSSLKeys := tc.DeliveryServiceSSLKeys{
		AuthType:        provider,
		CDN:             *req.CDN,
		DeliveryService: *req.DeliveryService,
		Key:             *req.DeliveryService,
		Hostname:        *req.HostName,
		Version:         *req.Version,
	}

	keyPem, err := ConvertPrivateKeyToKeyPem(priv)
	if err != nil {
		log.Errorf(deliveryService + ": Error converting private key to PEM: " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return err
	}

	// remove extra line if LE returns it
	trimmedCert := bytes.ReplaceAll(certificates.Certificate, []byte("\n\n"), []byte("\n"))

	dsSSLKeys.Certificate = tc.DeliveryServiceSSLKeysCertificate{
		Crt: string(EncodePEMToLegacyPerlRiakFormat(trimmedCert)),
		Key: string(EncodePEMToLegacyPerlRiakFormat(keyPem)),
		CSR: string(EncodePEMToLegacyPerlRiakFormat([]byte("ACME Generated"))),
	}

	if err := tv.PutDeliveryServiceSSLKeys(dsSSLKeys, tx, context.Background()); err != nil {
		log.Errorf("Error putting ACME certificate in Traffic Vault: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with "+provider, currentUser, logTx)
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return fmt.Errorf(deliveryService+": putting keys in Traffic Vault: %v", err)
	}

	tx2, err := db.Begin()
	if err != nil {
		log.Errorf("starting sql transaction for delivery service " + *req.DeliveryService + ": " + err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return fmt.Errorf("starting sql transaction for delivery service "+*req.DeliveryService+": %v", err)
	}

	if err := updateSSLKeyVersion(*req.DeliveryService, req.Version.ToInt64(), tx2); err != nil {
		log.Errorf("updating SSL key version for delivery service '" + *req.DeliveryService + "': " + err.Error())
		if asycErr := api.UpdateAsyncStatus(db, api.AsyncFailed, "ACME renewal failed.", asyncStatusId, true); asycErr != nil {
			log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
		}
		return fmt.Errorf("updating SSL key version for delivery service '"+*req.DeliveryService+"': %v", err)
	}
	tx2.Commit()

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: Added SSL keys with "+provider, currentUser, logTx)
	if asycErr := api.UpdateAsyncStatus(db, api.AsyncSucceeded, "ACME renewal complete.", asyncStatusId, true); asycErr != nil {
		log.Errorf("updating async status for id %v: %v", asyncStatusId, asycErr)
	}
	return nil
}

// GetAcmeAccountConfig returns the ACME account information from cdn.conf for a given provider.
func GetAcmeAccountConfig(cfg *config.Config, acmeProvider string) *config.ConfigAcmeAccount {
	if acmeProvider == tc.LetsEncryptAuthType {
		letsEncryptAccount := config.ConfigAcmeAccount{
			UserEmail:    cfg.ConfigLetsEncrypt.Email,
			AcmeProvider: tc.LetsEncryptAuthType,
		}
		if strings.EqualFold(cfg.ConfigLetsEncrypt.Environment, "staging") {
			letsEncryptAccount.AcmeUrl = lego.LEDirectoryStaging // provides certificate signed by invalid authority for testing purposes
		} else {
			letsEncryptAccount.AcmeUrl = lego.LEDirectoryProduction // provides certificate signed by valid LE authority
		}
		return &letsEncryptAccount
	}
	for _, acmeCfg := range cfg.AcmeAccounts {
		if acmeCfg.AcmeProvider == acmeProvider {
			return &acmeCfg
		}
	}
	return nil
}

// GetAcmeClient uses the ACME account information in either cdn.conf or the database to create and register an ACME client.
func GetAcmeClient(acmeAccount *config.ConfigAcmeAccount, userTx *sql.Tx, db *sqlx.DB) (*lego.Client, error) {
	if acmeAccount.UserEmail == "" {
		log.Errorf("An email address must be provided to use ACME with %v", acmeAccount.AcmeProvider)
		return nil, errors.New("An email address must be provided to use ACME with " + acmeAccount.AcmeProvider)
	}
	storedAcmeInfo, err := getStoredAcmeAccountInfo(userTx, acmeAccount.UserEmail, acmeAccount.AcmeProvider)
	if err != nil {
		log.Errorf("Error finding stored ACME information: %s", err.Error())
		return nil, err
	}

	myUser := MyUser{}
	foundPreviousAccount := false
	userPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("Error generating private key: %s", err.Error())
		return nil, err
	}

	if storedAcmeInfo == nil || acmeAccount.UserEmail == "" {
		myUser = MyUser{
			key:   userPrivateKey,
			Email: acmeAccount.UserEmail,
		}
	} else {
		foundPreviousAccount = true
		myUser = MyUser{
			key:   &storedAcmeInfo.PrivateKey,
			Email: storedAcmeInfo.Email,
			Registration: &registration.Resource{
				URI: storedAcmeInfo.URI,
			},
		}
	}

	config := lego.NewConfig(&myUser)
	config.CADirURL = acmeAccount.AcmeUrl
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		log.Errorf("Error creating acme client: %s", err.Error())
		return nil, err
	}

	if acmeAccount.AcmeProvider == tc.LetsEncryptAuthType {
		client.Challenge.Remove(challenge.HTTP01)
		client.Challenge.Remove(challenge.TLSALPN01)
		trafficRouterDns := NewDNSProviderTrafficRouter()
		trafficRouterDns.db = db
		if err != nil {
			log.Errorf("Error creating Traffic Router DNS provider: %s", err.Error())
			return nil, err
		}
		client.Challenge.SetDNS01Provider(trafficRouterDns)
	}

	if foundPreviousAccount {
		log.Debugf("Found existing account with %s", acmeAccount.AcmeProvider)
		reg, err := client.Registration.QueryRegistration()
		if err != nil {
			log.Errorf("Error querying %s for existing account: %s", acmeAccount.AcmeProvider, err.Error())
			return nil, err
		}
		myUser.Registration = reg
		if reg.Body.Status != validAccountStatus {
			log.Debugf("Account found with %s is not valid.", acmeAccount.AcmeProvider)
			foundPreviousAccount = false
		}
	}
	if !foundPreviousAccount {
		if acmeAccount.Kid != "" && acmeAccount.HmacEncoded != "" {
			reg, err := client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
				TermsOfServiceAgreed: true,
				Kid:                  acmeAccount.Kid,
				HmacEncoded:          acmeAccount.HmacEncoded,
			})
			if err != nil {
				log.Errorf("Error registering acme client with external account binding: %s", err.Error())
				return nil, err
			}
			myUser.Registration = reg
			log.Debugf("Creating a new account with %s", acmeAccount.AcmeProvider)
		} else {
			reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err != nil {
				log.Errorf("Error registering acme client: %s", err.Error())
				return nil, err
			}
			myUser.Registration = reg
			log.Debugf("Creating a new account with %s", acmeAccount.AcmeProvider)
		}

		// save account info
		userKeyPem, err := ConvertPrivateKeyToKeyPem(userPrivateKey)
		if err != nil {
			return nil, err
		}
		err = storeAcmeAccountInfo(userTx, myUser.Email, string(userKeyPem), myUser.Registration.URI, acmeAccount.AcmeProvider)
		if err != nil {
			log.Errorf("storing user account info: " + err.Error())
			return nil, fmt.Errorf("storing user account info: %v", err)
		}
	}

	return client, nil
}

// ConvertPrivateKeyToKeyPem converts an rsa.PrivateKey to be PEM encoded.
func ConvertPrivateKeyToKeyPem(userPrivateKey *rsa.PrivateKey) ([]byte, error) {
	userKeyDer := x509.MarshalPKCS1PrivateKey(userPrivateKey)
	if userKeyDer == nil {
		log.Errorf("marshalling private key: nil der")
		return nil, errors.New("marshalling private key: nil der")
	}
	userKeyBuf := bytes.Buffer{}
	if err := pem.Encode(&userKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: userKeyDer}); err != nil {
		log.Errorf("pem-encoding private key: " + err.Error())
		return nil, fmt.Errorf("pem-encoding private key: %v", err)
	}
	return userKeyBuf.Bytes(), nil
}

// AcmeInfo contains the information that will be stored for an ACME account.
type AcmeInfo struct {
	Email      string `db:"email"`
	Key        string `db:"private_key"`
	URI        string `db:"uri"`
	PrivateKey rsa.PrivateKey
}

func getStoredAcmeAccountInfo(tx *sql.Tx, email string, provider string) (*AcmeInfo, error) {
	acmeInfo := AcmeInfo{}
	selectQuery := `SELECT email, private_key, uri FROM acme_account WHERE email = $1 AND provider = $2 LIMIT 1`
	if err := tx.QueryRow(selectQuery, email, provider).Scan(&acmeInfo.Email, &acmeInfo.Key, &acmeInfo.URI); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting ACME account record: %v", err)
	}

	decodedKeyBlock, _ := pem.Decode([]byte(acmeInfo.Key))
	decodedKey, err := x509.ParsePKCS1PrivateKey(decodedKeyBlock.Bytes)
	if err != nil {
		return nil, errors.New("decoding private key for user account")
	}
	acmeInfo.PrivateKey = *decodedKey

	return &acmeInfo, nil
}

func storeAcmeAccountInfo(tx *sql.Tx, email string, privateKey string, uri string, provider string) error {
	q := `INSERT INTO acme_account (email, private_key, uri, provider) VALUES ($1, $2, $3, $4)`
	response, err := tx.Exec(q, email, privateKey, uri, provider)
	if err != nil {
		return err
	}

	rows, err := response.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("zero rows affected when inserting Let's Encrypt account information")
	}

	return nil
}
