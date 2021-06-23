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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
)

const (
	PemCertEndMarker  = "-----END CERTIFICATE-----"
	hostnameKeyDepMsg = "This endpoint is deprecated, please use '/deliveryservices/xmlId/{{XMLID}}/sslkeys' instead"
)

// AddSSLKeys adds the given ssl keys to the given delivery service.
func AddSSLKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("adding SSL keys to Traffic Vault for delivery service: Traffic Vault is not configured"))
		return
	}
	req := tc.DeliveryServiceAddSSLKeysReq{}
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
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.AddSSLKeys: getting DS ID from name "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+*req.DeliveryService), nil)
		return
	}

	// ECDSA keys support is only permitted for DNS delivery services
	// Traffic Router (HTTP* delivery service types) do not support ECDSA keys
	dsType, dsFound, err := getDSType(inf.Tx.Tx, *req.Key)
	allowEC := false
	if err == nil && dsFound && dsType.IsDNS() {
		allowEC = true
	}

	certChain, certPrivateKey, isUnknownAuth, isVerifiedChainNotEqual, err := verifyCertKeyPair(req.Certificate.Crt, req.Certificate.Key, "", allowEC)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("verifying certificate: "+err.Error()), nil)
		return
	}
	req.Certificate.Crt = certChain
	req.Certificate.Key = certPrivateKey

	base64EncodeCertificate(req.Certificate)

	authType := ""
	if req.AuthType != nil {
		authType = *req.AuthType
	}
	dsSSLKeys := tc.DeliveryServiceSSLKeys{
		CDN:             *req.CDN,
		DeliveryService: *req.DeliveryService,
		Hostname:        *req.HostName,
		Key:             *req.Key,
		Version:         *req.Version,
		Certificate:     *req.Certificate,
		AuthType:        authType,
	}

	if err := inf.Vault.PutDeliveryServiceSSLKeys(dsSSLKeys, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("putting SSL keys in Traffic Vault for delivery service '"+*req.DeliveryService+"': "+err.Error()))
		return
	}
	if err := updateSSLKeyVersion(*req.DeliveryService, req.Version.ToInt64(), inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("adding SSL keys to delivery service '"+*req.DeliveryService+"': "+err.Error()))
		return
	}

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+*req.DeliveryService+", ID: "+strconv.Itoa(dsID)+", ACTION: Added/Updated SSL keys", inf.User, inf.Tx.Tx)

	if isUnknownAuth {
		api.WriteRespAlert(w, r, tc.WarnLevel, "WARNING: SSL keys were successfully added for '"+*req.DeliveryService+"', but the input certificate may be invalid (certificate is signed by an unknown authority)")
		return
	}
	if isVerifiedChainNotEqual {
		api.WriteRespAlert(w, r, tc.WarnLevel, "WARNING: SSL keys were successfully added for '"+*req.DeliveryService+"', but the input certificate may be invalid (certificate verification produced a different chain)")
		return
	}

	api.WriteResp(w, r, "Successfully added ssl keys for "+*req.DeliveryService)
}

// GetSSLKeysByHostName fetches the ssl keys for a deliveryservice specified by the fully qualified hostname
func GetSSLKeysByHostName(w http.ResponseWriter, r *http.Request) {
	inf, xmlID, err := getXmlIDFromRequest(w, r)
	if inf != nil {
		defer inf.Close()
	}
	if err != nil {
		return
	}
	getSSLKeysByXMLIDHelper(xmlID, inf.Vault, tc.CreateAlerts(tc.WarnLevel, hostnameKeyDepMsg), inf, w, r)
}

func getXmlIDFromRequest(w http.ResponseWriter, r *http.Request) (*api.Info, string, error) {
	alerts := tc.CreateAlerts(tc.WarnLevel, hostnameKeyDepMsg)
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"hostname"}, nil)
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return inf, "", errors.New("getting XML ID from request")
	}

	if !inf.Config.TrafficVaultEnabled {
		userErr = api.LogErr(r, http.StatusInternalServerError, nil, errors.New("getting SSL keys from Traffic Vault by host name: Traffic Vault is not configured"))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return inf, "", errors.New("getting XML ID from request")
	}

	hostName := inf.Params["hostname"]
	xmlID, userErr, sysErr, errCode := getXmlIdFromHostname(inf, hostName)
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return inf, "", errors.New("getting XML ID from request")
	}
	return inf, xmlID, nil
}

// GetSSLKeysByHostNameV15 fetches the ssl keys for a deliveryservice specified by the fully qualified hostname. V15 includes expiration date.
func GetSSLKeysByHostNameV15(w http.ResponseWriter, r *http.Request) {
	inf, xmlID, err := getXmlIDFromRequest(w, r)
	if inf != nil {
		defer inf.Close()
	}
	if err != nil {
		return
	}
	getSSLKeysByXMLIDHelperV15(xmlID, tc.CreateAlerts(tc.WarnLevel, hostnameKeyDepMsg), inf, w, r)
}

func getXmlIdFromHostname(inf *api.Info, hostName string) (string, error, error, int) {
	domainName := ""
	hostRegex := ""
	strArr := strings.Split(hostName, ".")
	ln := len(strArr)
	if ln > 1 {
		for i := 2; i < ln-1; i++ {
			domainName += strArr[i] + "."
		}
		domainName += strArr[ln-1]
		hostRegex = `.*\.` + strArr[1] + `\..*`
	}

	// lookup the cdnID
	cdnID, ok, err := getCDNIDByDomainname(domainName, inf.Tx.Tx)
	if err != nil {
		return "", nil, errors.New("getting cdn id by domain name: " + err.Error()), http.StatusInternalServerError
	}
	if !ok {
		return "", errors.New("a CDN does not exist for the domain: " + domainName + " parsed from hostname: " + hostName), nil, http.StatusNotFound
	}
	// now lookup the deliveryservice xmlID
	xmlID, ok, err := getXMLID(cdnID, hostRegex, inf.Tx.Tx)
	if err != nil {
		return "", nil, errors.New("getting xml id: " + err.Error()), http.StatusInternalServerError
	}
	if !ok {
		return "", errors.New("a delivery service does not exist for a host with hostname of " + hostName), nil, http.StatusNotFound
	}

	return xmlID, nil, nil, http.StatusOK
}

// GetSSLKeysByXMLID fetches the deliveryservice ssl keys by the specified xmlID.
func GetSSLKeysByXMLID(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"xmlid"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting SSL keys from Traffic Vault by xml id: Traffic Vault is not configured"))
		return
	}
	xmlID := inf.Params["xmlid"]
	getSSLKeysByXMLIDHelper(xmlID, inf.Vault, tc.Alerts{}, inf, w, r)
}

func getSSLKeysByXMLIDHelper(xmlID string, tv trafficvault.TrafficVault, alerts tc.Alerts, inf *api.Info, w http.ResponseWriter, r *http.Request) {
	version := inf.Params["version"]
	decode := inf.Params["decode"]
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	keyObjV15, ok, err := tv.GetDeliveryServiceSSLKeys(xmlID, version, inf.Tx.Tx, r.Context())
	if err != nil {
		userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New("getting ssl keys: "+err.Error()))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	keyObj := keyObjV15.DeliveryServiceSSLKeys
	if !ok {
		keyObj = tc.DeliveryServiceSSLKeys{}
	}
	if decode != "" && decode != "0" { // the Perl version checked the decode string as: if ( $decode )
		err = Base64DecodeCertificate(&keyObj.Certificate)
		if err != nil {
			userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New("getting SSL keys for XMLID '"+xmlID+"': "+err.Error()))
			alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
			api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
			return
		}
	}

	if len(alerts.Alerts) == 0 {
		api.WriteResp(w, r, keyObj)
	} else {
		api.WriteAlertsObj(w, r, http.StatusOK, alerts, keyObj)
	}
}

// GetSSLKeysByXMLIDV15 fetches the deliveryservice ssl keys by the specified xmlID. V15 includes expiration date.
func GetSSLKeysByXMLIDV15(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"xmlid"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting SSL keys from Traffic Vault by xml id: Traffic Vault is not configured"))
		return
	}
	xmlID := inf.Params["xmlid"]
	getSSLKeysByXMLIDHelperV15(xmlID, tc.Alerts{}, inf, w, r)
}

func getSSLKeysByXMLIDHelperV15(xmlID string, alerts tc.Alerts, inf *api.Info, w http.ResponseWriter, r *http.Request) {
	version := inf.Params["version"]
	decode := inf.Params["decode"]
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	keyObj, ok, err := inf.Vault.GetDeliveryServiceSSLKeys(xmlID, version, inf.Tx.Tx, r.Context())
	if err != nil {
		userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New("getting ssl keys: "+err.Error()))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	if !ok {
		keyObj = tc.DeliveryServiceSSLKeysV15{}
	} else {
		parsedCert := keyObj.Certificate
		err = Base64DecodeCertificate(&parsedCert)
		if err != nil {
			userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New("getting SSL keys for XMLID '"+xmlID+"': "+err.Error()))
			alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
			api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
			return
		}
		if decode != "" && decode != "0" { // the Perl version checked the decode string as: if ( $decode )
			keyObj.Certificate = parsedCert
		}

		if keyObj.Certificate.Crt != "" && keyObj.Expiration.IsZero() {
			exp, err := parseExpirationFromCert([]byte(parsedCert.Crt))
			if err != nil {
				userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New(xmlID+": "+err.Error()))
				alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
				api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
				return
			}
			keyObj.Expiration = exp
		}
	}

	if len(alerts.Alerts) == 0 {
		api.WriteResp(w, r, keyObj)
	} else {
		api.WriteAlertsObj(w, r, http.StatusOK, alerts, keyObj)
	}
}

func parseExpirationFromCert(cert []byte) (time.Time, error) {
	block, _ := pem.Decode(cert)
	if block == nil {
		return time.Time{}, errors.New("Error decoding cert to parse expiration")
	}

	x509cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, errors.New("Error parsing cert to get expiration - " + err.Error())
	}

	return x509cert.NotAfter, nil
}

func Base64DecodeCertificate(cert *tc.DeliveryServiceSSLKeysCertificate) error {
	csrDec, err := base64.StdEncoding.DecodeString(cert.CSR)
	if err != nil {
		return errors.New("base64 decoding csr: " + err.Error())
	}
	cert.CSR = string(csrDec)
	crtDec, err := base64.StdEncoding.DecodeString(cert.Crt)
	if err != nil {
		return errors.New("base64 decoding crt: " + err.Error())
	}
	cert.Crt = string(crtDec)
	keyDec, err := base64.StdEncoding.DecodeString(cert.Key)
	if err != nil {
		return errors.New("base64 decoding key: " + err.Error())
	}
	cert.Key = string(keyDec)
	return nil
}

func base64EncodeCertificate(cert *tc.DeliveryServiceSSLKeysCertificate) {
	cert.CSR = base64.StdEncoding.EncodeToString([]byte(cert.CSR))
	cert.Crt = base64.StdEncoding.EncodeToString([]byte(cert.Crt))
	cert.Key = base64.StdEncoding.EncodeToString([]byte(cert.Key))
}

// DeleteSSLKeys deletes a Delivery Service's sslkeys via a DELETE method
func DeleteSSLKeys(w http.ResponseWriter, r *http.Request) {
	deleteSSLKeys(w, r, false)
}

// DeleteSSLKeysDeprecated deletes a Delivery Service's sslkeys via a deprecated GET method
func DeleteSSLKeysDeprecated(w http.ResponseWriter, r *http.Request) {
	deleteSSLKeys(w, r, true)
}

func deleteSSLKeys(w http.ResponseWriter, r *http.Request, deprecated bool) {
	alt := "DELETE /deliveryservices/xmlId/:xmlid/sslkeys"
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"xmlid"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, errCode, userErr, sysErr, deprecated, &alt)
		return
	}
	defer inf.Close()
	if !inf.Config.TrafficVaultEnabled {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: Traffic Vault is not configured"), deprecated, &alt)
		return
	}
	xmlID := inf.Params["xmlid"]
	dsID, ok, err := getDSIDFromName(inf.Tx.Tx, xmlID)
	if err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.DeleteSSLKeys: getting DS ID from name "+err.Error()), deprecated, &alt)
		return
	} else if !ok {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no DS with name "+xmlID), nil, deprecated, &alt)
		return
	}
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, errCode, userErr, sysErr, deprecated, &alt)
		return
	}
	if err := inf.Vault.DeleteDeliveryServiceSSLKeys(xmlID, inf.Params["version"], inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("deliveryservice.DeleteSSLKeys: deleting SSL keys: "+err.Error()), deprecated, &alt)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+xmlID+", ID: "+strconv.Itoa(dsID)+", ACTION: Deleted SSL keys", inf.User, inf.Tx.Tx)
	if deprecated {
		api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(&alt), "Successfully deleted ssl keys for "+xmlID)
		return
	}
	api.WriteResp(w, r, "Successfully deleted ssl keys for "+xmlID)
}

func updateSSLKeyVersion(xmlID string, version int64, tx *sql.Tx) error {
	q := `UPDATE deliveryservice SET ssl_key_version = $1 WHERE xml_id = $2`
	if _, err := tx.Exec(q, version, xmlID); err != nil {
		return errors.New("updating delivery service ssl_key_version: " + err.Error())
	}
	return nil
}

// returns the cdn_id found by domainname.
func getCDNIDByDomainname(domainName string, tx *sql.Tx) (int64, bool, error) {
	cdnID := int64(0)
	if err := tx.QueryRow(`SELECT id from cdn WHERE domain_name = $1`, domainName).Scan(&cdnID); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return cdnID, true, nil
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

// returns a delivery service xmlId for a cdn by host regex.
func getXMLID(cdnID int64, hostRegex string, tx *sql.Tx) (string, bool, error) {
	q := `
SELECT ds.xml_id from deliveryservice ds
JOIN deliveryservice_regex dr on ds.id = dr.deliveryservice AND ds.cdn_id = $1
JOIN regex r on r.id = dr.regex
WHERE r.pattern = $2
`
	xmlID := ""
	if err := tx.QueryRow(q, cdnID, hostRegex).Scan(&xmlID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying xml id: " + err.Error())
	}
	return xmlID, true, nil
}

// verify the server certificate chain and return the
// certificate and its chain in the proper order. Returns a verified
// and ordered certificate and CA chain.
// If the cert verification returns UnknownAuthorityError, return true to
// indicate that the certs are signed by an unknown authority (e.g. self-signed). Otherwise, return false.
// If the chain returned from Certificate.Verify() does not match the input chain,
// return true. Otherwise, return false.
func verifyCertKeyPair(pemCertificate string, pemPrivateKey string, rootCA string, allowEC bool) (string, string, bool, bool, error) {
	// decode, verify, and order certs for storage
	cleanPemPrivateKey := ""
	certs := strings.SplitAfter(pemCertificate, PemCertEndMarker)
	if len(certs) <= 1 {
		return "", "", false, false, errors.New("no certificate chain to verify")
	}

	// decode and verify the server certificate
	block, _ := pem.Decode([]byte(certs[0]))
	if block == nil {
		return "", "", false, false, errors.New("could not decode pem-encoded server certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", false, false, errors.New("could not parse the server certificate: " + err.Error())
	}

	// Common x509 certificate validation
	err = commonX509CertificateValidation(cert)
	if err != nil {
		return "", "", false, false, err
	}

	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		var rsaPrivateKey *rsa.PrivateKey

		// RSA is both a digital signature and encryption algorithm, hence the key encipherment
		// usage must be indicated in the certificate.
		// The keyUsage and extended Key Usage does not exist in version 1 of the x509 specificication.
		if cert.Version > 1 && !(cert.KeyUsage&x509.KeyUsageKeyEncipherment > 0) {
			return "", "", false, false, errors.New("cert/key (rsa) validation: no keyEncipherment keyUsage extension present in x509v3 server certificate")
		}

		// Extract the RSA public key from the x509 certificate
		certPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok || certPublicKey == nil {
			return "", "", false, false, errors.New("cert/key (rsa) validation error: could not extract public RSA key from certificate")
		}

		// Attempt to decode the RSA private key
		rsaPrivateKey, cleanPemPrivateKey, err = decodeRSAPrivateKey(pemPrivateKey)
		if err != nil {
			return "", "", false, false, err
		}

		// Check RSA private key modulus against the x509 RSA public key modulus
		if rsaPrivateKey != nil && certPublicKey != nil && !bytes.Equal(rsaPrivateKey.N.Bytes(), certPublicKey.N.Bytes()) {
			return "", "", false, false, errors.New("cert/key (rsa) mismatch error: RSA public N modulus value mismatch")
		}

	case x509.ECDSA:
		var ecdsaPrivateKey *ecdsa.PrivateKey

		// Only permit ECDSA support for DNS* DSTypes until the Traffic Router can support it
		if !allowEC {
			return "", "", false, false, errors.New("cert/key validation error: ECDSA public key algorithm unsupported for non-DNS delivery service type")
		}

		// DSA and ECDSA is not an encryption algorithm and only a signing algorithm, hence the
		// certificate only needs to have the DigitalSignature KeyUsage indicated.
		if cert.Version > 1 && !(cert.KeyUsage&x509.KeyUsageDigitalSignature > 0) {
			return "", "", false, false, errors.New("cert/key (ecdsa) validation error: no digitalSignature keyUsage extension present in x509v3 server certificate")
		}

		// Attempt to decode the ECDSA private key
		ecdsaPrivateKey, cleanPemPrivateKey, err = decodeECDSAPrivateKey(pemPrivateKey)
		if err != nil {
			return "", "", false, false, err
		}

		// Extract the ECDSA public key from the x509 certificate
		certPublicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok || certPublicKey == nil {
			return "", "", false, false, errors.New("cert/key (ecdsa) validation error: could not get extract public ECDSA key from certificate")
		}

		// Compare the ECDSA curve name contained within the x509.PublicKey against the curve name indicated in the private key
		if certPublicKey.Params().Name != ecdsaPrivateKey.Params().Name {
			return "", "", false, false, errors.New("cert/key (ecdsa) mismatch error: ECDSA curve name in cert does not match curve name in private key")
		}

		// Verify that ECDSA public value X matches in both the cert.PublicKey and the private key.
		if !bytes.Equal(certPublicKey.X.Bytes(), ecdsaPrivateKey.X.Bytes()) {
			return "", "", false, false, errors.New("cert/key (ecdsa) mismatch error: ECDSA public X value mismatch")
		}

		// Verify that ECDSA public value Y matches in both the cert.PublicKey and the private key.
		if !bytes.Equal(certPublicKey.Y.Bytes(), ecdsaPrivateKey.Y.Bytes()) {
			return "", "", false, false, errors.New("cert/key (ecdsa) mismatch error: ECDSA public Y value mismatch")
		}

	case x509.DSA:
		return "", "", false, false, errors.New("cert/key validation error: DSA public key algorithm unsupported")

	case x509.UnknownPublicKeyAlgorithm:
		fallthrough
	default:
		return "", "", false, false, errors.New("cert/key validation error: Unknown public key algorithm")
	}

	bundle := ""
	for i := 0; i < len(certs)-1; i++ {
		bundle += certs[i]
	}

	intermediatePool := x509.NewCertPool()
	if !intermediatePool.AppendCertsFromPEM([]byte(bundle)) {
		return "", "", false, false, errors.New("certificate CA bundle is empty")
	}

	opts := x509.VerifyOptions{
		Intermediates: intermediatePool,
	}

	if rootCA != "" {
		// verify the certificate chain.
		rootPool := x509.NewCertPool()
		if !rootPool.AppendCertsFromPEM([]byte(rootCA)) {
			return "", "", false, false, errors.New("unable to parse root CA certificate")
		}
		opts.Roots = rootPool
	}

	chain, err := cert.Verify(opts)
	if err != nil {
		if _, ok := err.(x509.UnknownAuthorityError); ok {
			return pemCertificate, cleanPemPrivateKey, true, false, nil
		}
		return "", "", false, false, errors.New("could not verify the certificate chain: " + err.Error())
	}
	if len(chain) < 1 {
		return "", "", false, false, errors.New("can't find valid chain for cert in file in request")
	}
	pemEncodedChain := ""
	for _, link := range chain[0] {
		// Include all certificates in the chain, since verification was successful.
		block := &pem.Block{Type: "CERTIFICATE", Bytes: link.Raw}
		pemEncodedChain += string(pem.EncodeToMemory(block))
	}

	if len(pemEncodedChain) < 1 {
		return "", "", false, false, errors.New("invalid empty certificate chain in request")
	}

	if pemEncodedChain != pemCertificate {
		return pemCertificate, cleanPemPrivateKey, false, true, nil
	}

	return pemCertificate, cleanPemPrivateKey, false, false, nil
}

func commonX509CertificateValidation(cert *x509.Certificate) error {

	// validate certificate is a server auth certificate if the extension is present
	if cert.Version > 1 {
		serverAuthExtKeyUsageFound := false
		for _, certExtKeyUsage := range cert.ExtKeyUsage {
			if certExtKeyUsage == x509.ExtKeyUsageServerAuth {
				serverAuthExtKeyUsageFound = true
				break
			}
		}

		if !serverAuthExtKeyUsageFound {
			return errors.New("certificate (x509v3) validation error: server certificate missing 'serverAuth' extended key usage")
		}
	}

	// ensure that the certificate uses a supported PKI algorithm and a public key is present.
	if cert.PublicKey == nil {
		return errors.New("certificate validation error: no PKI public key found")
	}
	if cert.PublicKeyAlgorithm == x509.UnknownPublicKeyAlgorithm {
		return errors.New("certificate validation error: unknown PKI algorithm")
	}

	// ensure that the certificate is signed with supported algorithm
	if len(cert.Signature) == 0 {
		return errors.New("certificate validation error: no signature found")
	}
	if cert.SignatureAlgorithm == x509.UnknownSignatureAlgorithm {
		return errors.New("certificate validation error: unknown signature algorithm")
	}

	return nil
}

// Common privateKey validation logic.
// Reject unsupported encrypted private keys
func commonPrivateKeyValidation(block *pem.Block) error {

	if block == nil {
		return errors.New("private key validation error: could not decode pem-encoded private key")
	}

	// Check for encrypted keys or other unsupported key types
	if strings.Contains(block.Type, "ENCRYPTED") {
		return errors.New("private key validation error: encrypted private key not supported - block type: " + block.Type)
	}

	// Check block headers for encryption.
	for _, value := range block.Headers {
		if strings.Contains(value, "ENCRYPTED") {
			return errors.New("private key validation error: encrypted private key not supported - header: " + value)
		}
	}

	return nil
}

// decode the private key
// check for proper algorithm.
// check for correct number of keys
// return private key object, cleaned private key PEM, or any errors.
func decodeRSAPrivateKey(pemPrivateKey string) (*rsa.PrivateKey, string, error) {

	// Remove any white space before decoding
	var trimmedPrivateKey = strings.TrimSpace(pemPrivateKey)

	// Capture all key decode errors and collapse them at the end
	var decodeErrors = make([]error, 0)

	// RSA Private Key
	var rsaPrivateKey *rsa.PrivateKey = nil

	// Check for proper key count before attempting to decode.
	blockCount := strings.Count(trimmedPrivateKey, "\n-----END")
	if blockCount < 1 {
		return nil, "", errors.New("private key validation error: no RSA private key PEM blocks found")
	}
	if blockCount > 1 {
		return nil, "", errors.New("private key validation error: multiple private key PEM blocks found")
	}

	// Attempt to decode pem encoded text into PEM block.
	block, _ := pem.Decode([]byte(trimmedPrivateKey))

	// Check that the key was decoded and validate key isn't encrypted and
	// other common validation shared between PKI algorithms
	err := commonPrivateKeyValidation(block)
	if err != nil {
		return nil, "", err
	}

	// Decode PKCS#8 - RSA Private Key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		decodeErrors = append(decodeErrors, errors.New("private key validation error: parse pkcs#8 error: "+err.Error()))
	}

	// Determine if the privateKey is of the correct type
	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok || rsaPrivateKey == nil {
		decodeErrors = append(decodeErrors, fmt.Errorf("private key validation error: incorrect private key type: %T", privateKey))
	} else {
		return rsaPrivateKey, trimmedPrivateKey, nil
	}

	// Decode PKCS#1 - RSA Private Key
	rsaPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil || rsaPrivateKey == nil {
		decodeErrors = append(decodeErrors, errors.New("private key validation error: parse pkcs#1 error: "+err.Error()))
		return nil, "", util.JoinErrsSep(decodeErrors, ", ")
	}

	return rsaPrivateKey, trimmedPrivateKey, nil
}

// decode the private key
// check for proper algorithm.
// check for correct number of keys
// return private key object, cleaned private key PEM, or any errors.
func decodeECDSAPrivateKey(pemPrivateKey string) (*ecdsa.PrivateKey, string, error) {

	var ecdsaPrivateKey *ecdsa.PrivateKey = nil

	// Remove any white space before decoding
	var trimmedPrivateKey = strings.TrimSpace(pemPrivateKey)

	// Capture all key decode errors and collapse them at the end
	var decodeErrors = make([]error, 0)

	// Check for proper key count before attempting to decode.
	// ECDSA keys can have 1 or 2 PEM blocks if the 'EC PARAM' block is included.
	var blockCount = strings.Count(trimmedPrivateKey, "\n-----END")

	if blockCount < 1 {
		return nil, "", errors.New("private key validation error: no EC private key PEM blocks found")
	}

	if blockCount > 2 {
		return nil, "", errors.New("private key validation error: too many EC related PEM blocks found")
	}

	// Attempt to decode pem encoded text into PEM block.
	var pemData = []byte(trimmedPrivateKey)
	for len(pemData) > 0 {
		var block *pem.Block = nil

		// Check for at least one END marker
		if strings.Count(string(pemData), "\n-----END") == 0 {
			break
		}

		// Attempt to decode the first PEM Block
		block, pemData = pem.Decode(pemData)
		if block == nil {
			return nil, "", errors.New("private key validation error: could not decode pem-encoded block")
		}

		// Check that the key was decoded and validate key isn't encrypted and
		// other common validation shared between PKI algorithms
		err := commonPrivateKeyValidation(block)
		if err != nil {
			return nil, "", err
		}

		// Check if this pem block has 'KEY' contained in the type and try to decode it.
		if !strings.Contains(block.Type, "KEY") {
			continue
		}

		// First try to parse an EC key the normal way, before attempting PKCS8
		ecdsaPrivateKey, err = x509.ParseECPrivateKey(block.Bytes)
		if ecdsaPrivateKey == nil || err != nil {
			decodeErrors = append(decodeErrors, errors.New("private key validation error: failed to parse EC ANSI X9.62: "+err.Error()))
		} else {
			return ecdsaPrivateKey, trimmedPrivateKey, nil
		}

		// Second, try to parse PEM block as a PKCS#8 formatted RSA Private Key.
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			decodeErrors = append(decodeErrors, errors.New("private key validation error: parse pkcs#8 error: %s"+err.Error()))
			return nil, "", util.JoinErrsSep(decodeErrors, ", ")
		}

		// Make sure the privateKey is of the correct type (ecdsa.PrivateKey)
		ecdsaPrivateKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok || ecdsaPrivateKey == nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("private key validation error: incorrect private key type: %T", privateKey))
			return nil, "", util.JoinErrsSep(decodeErrors, ", ")
		}

		return ecdsaPrivateKey, trimmedPrivateKey, nil
	}

	return nil, "", errors.New("private key validation error: no ECDSA private keys found")
}
