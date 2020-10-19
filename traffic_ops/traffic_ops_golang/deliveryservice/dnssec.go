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
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"

	"github.com/miekg/dns"
)

func PutDNSSecKeys(tx *sql.Tx, cfg *config.Config, xmlID string, cdnName string, exampleURLs []string) apierrors.Errors {
	errs := apierrors.New()
	keys, ok, err := riaksvc.GetDNSSECKeys(cdnName, tx, cfg.RiakAuthOptions, cfg.RiakPort)
	if err != nil {
		errs.SetSystemError("getting DNSSec keys from Riak: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	} else if !ok {
		errs.UserError = fmt.Errorf("there are no DNSSec keys for the CDN %s which is required to create keys for the deliveryservice", cdnName)
		errs.Code = http.StatusBadRequest
		return errs
	}
	cdnKeys, ok := keys[cdnName]
	// TODO warn and continue?
	if !ok {
		errs.UserError = fmt.Errorf("there are no DNSSec keys for the CDN %s which is required to create keys for the deliveryservice", cdnName)
		errs.Code = http.StatusBadRequest
		return errs
	}
	kExp := getKeyExpiration(cdnKeys.KSK, dnssecDefaultKSKExpiration)
	zExp := getKeyExpiration(cdnKeys.ZSK, dnssecDefaultZSKExpiration)
	overrideTTL := false
	dsKeys, err := CreateDNSSECKeys(tx, cfg, xmlID, exampleURLs, cdnKeys, kExp, zExp, dnssecDefaultTTL, overrideTTL)
	if err != nil {
		errs.SetSystemError("creating DNSSEC keys for delivery service '" + xmlID + "': " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}
	keys[xmlID] = dsKeys
	if err := riaksvc.PutDNSSECKeys(keys, cdnName, tx, cfg.RiakAuthOptions, cfg.RiakPort); err != nil {
		errs.SetSystemError("putting Riak DNSSEC keys: " + err.Error())
		errs.Code = http.StatusInternalServerError
	}
	return errs
}

// CreateDNSSECKeys creates DNSSEC keys for the given delivery service, updating existing keys if they exist. The overrideTTL parameter determines whether to reuse existing key TTLs if they exist, or to override existing TTLs with the ttl parameter's value.
func CreateDNSSECKeys(tx *sql.Tx, cfg *config.Config, xmlID string, exampleURLs []string, cdnKeys tc.DNSSECKeySetV11, kskExpiration time.Duration, zskExpiration time.Duration, ttl time.Duration, overrideTTL bool) (tc.DNSSECKeySetV11, error) {
	if len(cdnKeys.ZSK) == 0 {
		return tc.DNSSECKeySetV11{}, errors.New("getting DNSSec keys from Riak: no DNSSec ZSK keys for CDN")
	}
	if len(cdnKeys.KSK) == 0 {
		return tc.DNSSECKeySetV11{}, errors.New("getting DNSSec keys from Riak: no DNSSec ZSK keys for CDN")
	}
	if !overrideTTL {
		ttl = getKeyTTL(cdnKeys.KSK, ttl)
	}
	dsName, err := GetDSDomainName(exampleURLs)
	if err != nil {
		return tc.DNSSECKeySetV11{}, errors.New("creating DS domain name: " + err.Error())
	}
	inception := time.Now()
	zExpiration := inception.Add(zskExpiration)
	kExpiration := inception.Add(kskExpiration)

	tld := false
	effectiveDate := inception
	zsk, err := GetDNSSECKeysV11(tc.DNSSECZSKType, dsName, ttl, inception, zExpiration, tc.DNSSECKeyStatusNew, effectiveDate, tld)
	if err != nil {
		return tc.DNSSECKeySetV11{}, errors.New("getting DNSSEC keys for ZSK: " + err.Error())
	}
	ksk, err := GetDNSSECKeysV11(tc.DNSSECKSKType, dsName, ttl, inception, kExpiration, tc.DNSSECKeyStatusNew, effectiveDate, tld)
	if err != nil {
		return tc.DNSSECKeySetV11{}, errors.New("getting DNSSEC keys for KSK: " + err.Error())
	}
	return tc.DNSSECKeySetV11{ZSK: []tc.DNSSECKeyV11{zsk}, KSK: []tc.DNSSECKeyV11{ksk}}, nil
}

func GetDNSSECKeysV11(keyType string, dsName string, ttl time.Duration, inception time.Time, expiration time.Time, status string, effectiveDate time.Time, tld bool) (tc.DNSSECKeyV11, error) {
	key := tc.DNSSECKeyV11{
		InceptionDateUnix:  inception.Unix(),
		ExpirationDateUnix: expiration.Unix(),
		Name:               dsName,
		TTLSeconds:         uint64(ttl / time.Second),
		Status:             status,
		EffectiveDateUnix:  effectiveDate.Unix(),
	}
	isKSK := keyType != tc.DNSSECZSKType
	err := error(nil)
	key.Public, key.Private, key.DSRecord, err = genKeys(dsName, isKSK, ttl, tld)
	return key, err
}

// genKeys generates keys for DNSSEC for a delivery service. Returns the public key, private key, and DS record (which will be nil if ksk or tld is false).
// This emulates the old Perl Traffic Ops behavior: the public key is of the RFC1035 single-line zone file format, base64 encoded; the private key is of the BIND private-key-file format, base64 encoded; the DSRecord contains the algorithm, digest type, and digest.
func genKeys(dsName string, ksk bool, ttl time.Duration, tld bool) (string, string, *tc.DNSSECKeyDSRecordV11, error) {
	bits := 1024
	flags := 256
	algorithm := dns.RSASHA1 // 5 - http://www.iana.org/assignments/dns-sec-alg-numbers/dns-sec-alg-numbers.xhtml
	protocol := 3

	if ksk {
		flags |= 1
		bits *= 2
	}

	// Note: currently, the Router appears to hard-code this in what it generates for the DS record (or at least the "Publish this" log message).
	// DO NOT change this, without verifying the Router works correctly with this digest/type, and specifically with the text generated by MakeDSRecordText inserted in the parent resolver.
	digestType := dns.SHA256

	dnskey := dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   dsName,
			Rrtype: dns.TypeDNSKEY,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl / time.Second),
		},
		Flags:     uint16(flags),
		Protocol:  uint8(protocol),
		Algorithm: algorithm,
	}

	priKey, err := dnskey.Generate(bits)
	if err != nil {
		return "", "", nil, errors.New("error generating DNS key: " + err.Error())
	}

	priKeyStr := dnskey.PrivateKeyString(priKey) // BIND9 private-key-file format; cooresponds to Perl Net::DNS::SEC::Private->generate_rsa.dump_rsa_priv
	priKeyStrBase64 := base64.StdEncoding.EncodeToString([]byte(priKeyStr))

	pubKeyStr := dnskey.String() // RFC1035 single-line zone file format; cooresponds to Perl Net::DNS::RR.plain
	pubKeyStrBase64 := base64.StdEncoding.EncodeToString([]byte(pubKeyStr))

	keyDS := (*tc.DNSSECKeyDSRecordV11)(nil)
	if ksk && tld {
		dsRecord := dnskey.ToDS(digestType)
		if dsRecord == nil {
			return "", "", nil, fmt.Errorf("creating DS record from DNSKEY record: converting dnskey %++v to DS failed", dnskey)
		}
		keyDS = &tc.DNSSECKeyDSRecordV11{Algorithm: int64(dsRecord.Algorithm), DigestType: int64(dsRecord.DigestType), Digest: dsRecord.Digest}
	}

	return pubKeyStrBase64, priKeyStrBase64, keyDS, nil
}

// TODO change ttl to time.Duration

func GetDSDomainName(dsExampleURLs []string) (string, error) {
	// TODO move somewhere generic
	if len(dsExampleURLs) == 0 {
		return "", errors.New("no example URLs")
	}

	dsName := dsExampleURLs[0] + "."
	firstDot := strings.Index(dsName, ".")
	if firstDot == -1 {
		return "", errors.New("malformed example URL, no dots")
	}
	if len(dsName) < firstDot+2 {
		return "", errors.New("malformed example URL, nothing after first dot")
	}
	dsName = dsName[firstDot+1:]
	dsName = strings.ToLower(dsName)
	return dsName, nil
}

const dnssecDefaultKSKExpiration = time.Duration(365) * time.Hour * 24
const dnssecDefaultZSKExpiration = time.Duration(30) * time.Hour * 24
const dnssecDefaultTTL = 30

func getKeyExpiration(keys []tc.DNSSECKeyV11, defaultExpiration time.Duration) time.Duration {
	for _, key := range keys {
		if key.Status != tc.DNSSECKeyStatusNew {
			continue
		}
		return time.Duration(key.ExpirationDateUnix-key.InceptionDateUnix) * time.Second
	}
	return defaultExpiration
}

func getKeyTTL(keys []tc.DNSSECKeyV11, defaultTTL time.Duration) time.Duration {
	for _, key := range keys {
		if key.Status != tc.DNSSECKeyStatusNew {
			continue
		}
		return time.Duration(key.TTLSeconds) * time.Second
	}
	return defaultTTL
}

// MakeDNSSECKeySetFromRiakKeySet creates a DNSSECKeySet (as served by Traffic Ops) from a DNSSECKeysRiak (as stored in Riak), adding any computed data.
// Notably, this adds the full DS Record text to CDN KSKs
func MakeDNSSECKeysFromRiakKeys(riakKeys tc.DNSSECKeysRiak, dsTTL time.Duration) (tc.DNSSECKeys, error) {
	keys := map[string]tc.DNSSECKeySet{}
	for name, riakKeySet := range riakKeys {
		newKeySet := tc.DNSSECKeySet{}
		for _, zsk := range riakKeySet.ZSK {
			newZSK := tc.DNSSECKey{DNSSECKeyV11: zsk}
			// ZSKs don't have DSRecords, so we don't need to check here
			newKeySet.ZSK = append(newKeySet.ZSK, newZSK)
		}
		for _, ksk := range riakKeySet.KSK {
			newKSK := tc.DNSSECKey{DNSSECKeyV11: ksk}
			if ksk.DSRecord != nil {
				newKSK.DSRecord = &tc.DNSSECKeyDSRecord{DNSSECKeyDSRecordV11: *ksk.DSRecord}
				err := error(nil)
				newKSK.DSRecord.Text, err = MakeDSRecordText(ksk, dsTTL)
				if err != nil {
					return tc.DNSSECKeys{}, errors.New("making DS record text: " + err.Error())
				}
			}
			newKeySet.KSK = append(newKeySet.KSK, newKSK)
		}
		keys[name] = newKeySet
	}
	return tc.DNSSECKeys(keys), nil
}

func MakeDSRecordText(ksk tc.DNSSECKeyV11, ttl time.Duration) (string, error) {
	kskPublic := strings.Replace(ksk.Public, `\n`, "", -1) // note this is replacing the actual string slash-n not a newline. Because Perl.
	kskPublic = strings.Replace(kskPublic, "\n", "", -1)
	kskPublicBts := []byte(kskPublic)
	publicKeyBtsLen := base64.StdEncoding.DecodedLen(len(kskPublicBts))
	publicKeyBts := make([]byte, publicKeyBtsLen)
	publicKeyBtsLen, err := base64.StdEncoding.Decode(publicKeyBts, kskPublicBts)
	if err != nil {
		return "", errors.New("decoding ksk public key base64: " + err.Error())
	}
	publicKeyBts = publicKeyBts[:publicKeyBtsLen]

	// ksk.Public isn't just the public key, it's the RFC 1035 single-line zone file format: "name ttl IN DNSKEY flags protocol algorithm keyBytes".
	fields := strings.Fields(string(publicKeyBts))
	if len(fields) < 8 {
		return "", errors.New("malformed ksk public key: not enough fields")
	}
	flagsStr := fields[4]
	protocolStr := fields[5]
	flags, err := strconv.Atoi(flagsStr)
	if err != nil {
		return "", errors.New("malformed ksk public key: flags '" + flagsStr + "' not a number")
	}
	protocol, err := strconv.Atoi(protocolStr)
	if err != nil {
		return "", errors.New("malformed ksk public key: protocol '" + protocolStr + "' not a number")
	}

	realPublicKey := fields[7] // the Riak ksk.Public key is actually the RFC1035 single-line zone file format. For which the 7th field from 0 is the actual public key.

	dnsKey := dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   ksk.Name,
			Rrtype: dns.TypeDNSKEY,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl / time.Second), // NOT ksk.TTLSeconds, which is the DNSKEY TTL. The DS has its own TTL, which in Traffic Ops (as of this writing) comes from the Parameter name 'tld.ttls.DS' config 'CRConfig.json' assigned to the profile of a Router on this CDN.
		},
		Flags:     uint16(flags),
		Protocol:  uint8(protocol),
		Algorithm: uint8(ksk.DSRecord.Algorithm),
		PublicKey: realPublicKey,
	}

	ds := dnsKey.ToDS(uint8(ksk.DSRecord.DigestType))
	if ds == nil {
		return "", errors.New("failed to convert DNSKEY to DS record (is some field in the KSK invalid?)")
	}
	return ds.String(), nil
}
