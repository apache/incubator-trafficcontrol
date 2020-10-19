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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/certificate"
	"github.com/go-acme/lego/challenge"
	"github.com/go-acme/lego/challenge/dns01"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"github.com/jmoiron/sqlx"
)

type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

const LetsEncryptTimeout = time.Minute * 20

func (u *MyUser) GetEmail() string {
	return u.Email
}

func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type DNSProviderTrafficRouter struct {
	db *sqlx.DB
}

func NewDNSProviderTrafficRouter() *DNSProviderTrafficRouter {
	return &DNSProviderTrafficRouter{}
}

func (d *DNSProviderTrafficRouter) Timeout() (timeout, interval time.Duration) {
	return LetsEncryptTimeout, time.Second * 30
}

func (d *DNSProviderTrafficRouter) Present(domain, token, keyAuth string) error {
	tx, err := d.db.Begin()
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	q := `INSERT INTO dnschallenges (fqdn, record) VALUES ($1, $2)`
	response, err := tx.Exec(q, fqdn, value)
	tx.Commit()
	if err != nil {
		log.Errorf("Inserting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		return errors.New("Inserting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
	} else {
		rows, err := response.RowsAffected()
		if err != nil {
			log.Errorf("Determining rows affected dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return errors.New("Determining rows affected dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		}
		if rows == 0 {
			log.Errorf("Zero rows affected when inserting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return errors.New("Zero rows affected when inserting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		}
	}

	return nil
}

func (d *DNSProviderTrafficRouter) CleanUp(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	tx, err := d.db.Begin()

	q := `DELETE FROM dnschallenges WHERE fqdn = $1 and record = $2`
	response, err := tx.Exec(q, fqdn, value)
	tx.Commit()
	if err != nil {
		log.Errorf("Deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		return errors.New("Deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
	} else {
		rows, err := response.RowsAffected()
		if err != nil {
			log.Errorf("Determining rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return errors.New("Determining rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		}
		if rows == 0 {
			log.Errorf("Zero rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
			return errors.New("Zero rows affected when deleting dns txt record for fqdn '" + fqdn + "' record '" + value + "': " + err.Error())
		}
	}

	return nil
}

func GenerateLetsEncryptCertificates(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	ctx, _ := context.WithTimeout(r.Context(), LetsEncryptTimeout)

	req := tc.DeliveryServiceLetsEncryptSSLKeysReq{}
	if err := api.Parse(r.Body, nil, &req); err != nil {
		api.HandleErr(w, r, nil, http.StatusBadRequest, errors.New("parsing request: "+err.Error()), nil)
		return
	}
	if *req.DeliveryService == "" {
		req.DeliveryService = req.Key
	}

	dsID, cdnName, ok, err := dbhelpers.GetDSIDAndCDNFromName(inf.Tx.Tx, *req.DeliveryService)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.GenerateLetsEncryptCertificates: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+*req.DeliveryService), nil)
		return
	}

	errs = tenant.CheckID(inf.Tx.Tx, inf.User, dsID)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	_, ok, err = dbhelpers.GetCDNIDFromName(inf.Tx.Tx, tc.CDNName(*req.CDN))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking CDN existence: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("cdn not found with name "+*req.CDN), nil)
		return
	}

	if cdnName != tc.CDNName(*req.CDN) {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("delivery service not in cdn"), nil)
		return
	}

	go GetLetsEncryptCertificates(inf.Config, req, ctx, inf.User)

	api.WriteRespAlert(w, r, tc.InfoLevel, "Beginning async call to Let's Encrypt for "+*req.DeliveryService+".  This may take a few minutes.")

}

func GetLetsEncryptCertificates(cfg *config.Config, req tc.DeliveryServiceLetsEncryptSSLKeysReq, ctx context.Context, currentUser *auth.CurrentUser) error {

	db, err := api.GetDB(ctx)
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting db: %s", err.Error())
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting tx: %s", err.Error())
		return err
	}
	userTx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting userTx: %s", err.Error())
		return err
	}
	defer userTx.Commit()

	logTx, err := db.Begin()
	if err != nil {
		log.Errorf(*req.DeliveryService+": Error getting logTx: %s", err.Error())
		return err
	}
	defer logTx.Commit()

	domainName := *req.HostName
	deliveryService := *req.DeliveryService

	dsID, ok, err := getDSIDFromName(tx, *req.DeliveryService)
	if err != nil {
		log.Errorf("deliveryservice.GenerateSSLKeys: getting DS ID from name " + err.Error() + " " + ctx.Err().Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("deliveryservice.GenerateSSLKeys: getting DS ID from name " + err.Error())
	} else if !ok {
		log.Errorf("no DS with name " + *req.DeliveryService)
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("no DS with name " + *req.DeliveryService)
	}
	tx.Commit()

	storedLEInfo, err := getStoredLetsEncryptInfo(userTx, cfg.ConfigLetsEncrypt.Email)
	if err != nil {
		log.Errorf(deliveryService+": Error finding stored LE information: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return err
	}

	myUser := MyUser{}
	foundPreviousAccount := false
	userPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf(deliveryService+": Error generating private key: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return err
	}
	if storedLEInfo == nil || cfg.ConfigLetsEncrypt.Email == "" {

		myUser = MyUser{
			key:   userPrivateKey,
			Email: cfg.ConfigLetsEncrypt.Email,
		}
	} else {
		foundPreviousAccount = true
		myUser = MyUser{
			key:   &storedLEInfo.PrivateKey,
			Email: cfg.ConfigLetsEncrypt.Email,
			Registration: &registration.Resource{
				URI: storedLEInfo.URI,
			},
		}
	}

	config := lego.NewConfig(&myUser)
	if strings.EqualFold(cfg.ConfigLetsEncrypt.Environment, "staging") {
		config.CADirURL = lego.LEDirectoryStaging // provides certificate signed by invalid authority for testing purposes
	} else {
		config.CADirURL = lego.LEDirectoryProduction // provides certificate signed by valid LE authority
	}

	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		log.Errorf(deliveryService+": Error creating lets encrypt client: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return err
	}

	client.Challenge.Remove(challenge.HTTP01)
	client.Challenge.Remove(challenge.TLSALPN01)
	trafficRouterDns := NewDNSProviderTrafficRouter()
	trafficRouterDns.db = db
	if err != nil {
		log.Errorf(deliveryService+": Error creating Traffic Router DNS provider: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return err
	}
	client.Challenge.SetDNS01Provider(trafficRouterDns)

	if foundPreviousAccount {
		log.Debugf("Found existing account with Let's Encrypt")
		reg, err := client.Registration.QueryRegistration()
		if err != nil {
			log.Errorf(deliveryService+": Error querying Lets Encrypt for existing account: %s", err.Error())
			api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
			return err
		}
		myUser.Registration = reg
		if reg.Body.Status != "valid" {
			log.Debugf("Account found with Let's Encrypt is not valid.")
			foundPreviousAccount = false
		}
	}
	if !foundPreviousAccount {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			log.Errorf(deliveryService+": Error registering lets encrypt client: %s", err.Error())
			api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
			return err
		}
		myUser.Registration = reg
		log.Debugf("Creating a new account with Let's Encrypt")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf(deliveryService + ": Error generating private key")
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return err
	}
	request := certificate.ObtainRequest{
		Domains:    []string{domainName},
		Bundle:     true,
		PrivateKey: priv,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Errorf(deliveryService+": Error obtaining lets encrypt certificate: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt "+err.Error(), currentUser, logTx)
		return err
	}

	// Save certs into Riak
	dsSSLKeys := tc.DeliveryServiceSSLKeys{
		AuthType:        tc.LetsEncryptAuthType,
		CDN:             *req.CDN,
		DeliveryService: *req.DeliveryService,
		Key:             *req.DeliveryService,
		Hostname:        *req.HostName,
		Version:         *req.Version,
	}

	keyDer := x509.MarshalPKCS1PrivateKey(priv)
	if keyDer == nil {
		log.Errorf("marshalling private key: nil der")
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("marshalling private key: nil der")
	}
	keyBuf := bytes.Buffer{}
	if err := pem.Encode(&keyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDer}); err != nil {
		log.Errorf("pem-encoding private key: " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("pem-encoding private key: " + err.Error())
	}
	keyPem := keyBuf.Bytes()

	dsSSLKeys.Certificate = tc.DeliveryServiceSSLKeysCertificate{Crt: string(EncodePEMToLegacyPerlRiakFormat(certificates.Certificate)), Key: string(EncodePEMToLegacyPerlRiakFormat(keyPem)), CSR: ""}
	if err := riaksvc.PutDeliveryServiceSSLKeysObj(dsSSLKeys, tx, cfg.RiakAuthOptions, cfg.RiakPort); err != nil {
		log.Errorf("Error posting lets encrypt certificate to riak: %s", err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New(deliveryService + ": putting riak keys: " + err.Error())
	}

	tx2, err := db.Begin()
	if err != nil {
		log.Errorf("starting sql transaction for delivery service " + *req.DeliveryService + ": " + err.Error())
		return errors.New("starting sql transaction for delivery service " + *req.DeliveryService + ": " + err.Error())
	}

	if err := updateSSLKeyVersion(*req.DeliveryService, req.Version.ToInt64(), tx2); err != nil {
		log.Errorf("updating SSL key version for delivery service '" + *req.DeliveryService + "': " + err.Error())
		return errors.New("updating SSL key version for delivery service '" + *req.DeliveryService + "': " + err.Error())
	}
	tx2.Commit()

	if foundPreviousAccount {
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: Added SSL keys with Lets Encrypt", currentUser, logTx)
		return nil
	}

	userKeyDer := x509.MarshalPKCS1PrivateKey(userPrivateKey)
	if userKeyDer == nil {
		log.Errorf("marshalling private key: nil der")
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("marshalling private key: nil der")
	}
	userKeyBuf := bytes.Buffer{}
	if err := pem.Encode(&userKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: userKeyDer}); err != nil {
		log.Errorf("pem-encoding private key: " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("pem-encoding private key: " + err.Error())
	}
	userKeyPem := userKeyBuf.Bytes()
	err = storeLEAccountInfo(userTx, myUser.Email, string(userKeyPem), myUser.Registration.URI)
	if err != nil {
		log.Errorf("storing user account info: " + err.Error())
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: FAILED to add SSL keys with Lets Encrypt", currentUser, logTx)
		return errors.New("storing user account info: " + err.Error())
	}

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: Added SSL keys with Lets Encrypt", currentUser, logTx)

	return nil
}

func getStoredLetsEncryptInfo(tx *sql.Tx, email string) (*LEInfo, error) {
	leInfo := LEInfo{}
	selectQuery := `SELECT email, private_key, uri FROM lets_encrypt_account WHERE email = $1 LIMIT 1`
	if err := tx.QueryRow(selectQuery, email).Scan(&leInfo.Email, &leInfo.Key, &leInfo.URI); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.New("getting lets encrypt account record: " + err.Error())
	}

	decodedKeyBlock, _ := pem.Decode([]byte(leInfo.Key))
	decodedKey, err := x509.ParsePKCS1PrivateKey(decodedKeyBlock.Bytes)
	if err != nil {
		return nil, errors.New("decoding private key for user account")
	}
	leInfo.PrivateKey = *decodedKey

	return &leInfo, nil
}

func storeLEAccountInfo(tx *sql.Tx, email string, privateKey string, uri string) error {
	q := `INSERT INTO lets_encrypt_account (email, private_key, uri) VALUES ($1, $2, $3)`
	response, err := tx.Exec(q, email, privateKey, uri)
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

type LEInfo struct {
	Email      string `db:"email"`
	Key        string `db:"private_key"`
	URI        string `db:"uri"`
	PrivateKey rsa.PrivateKey
}
