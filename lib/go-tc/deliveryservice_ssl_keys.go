package tc

/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/lestrrat/go-jwx/jwk"
)

const (
	SelfSignedCertAuthType           = "Self Signed"
	CertificateAuthorityCertAuthType = "Certificate Authority"
	LetsEncryptAuthType              = "Lets Encrypt"
)

// SSLKeysAddResponse is a struct to store the response of addition of ssl keys for a DS,
// along with any alert messages
type SSLKeysAddResponse struct {
	Response string `json:"response"`
	Alerts
}

// DeliveryServiceSSLKeysResponse ...
type DeliveryServiceSSLKeysResponse struct {
	Response DeliveryServiceSSLKeys `json:"response"`
	Alerts
}

// DeliveryServiceSSLKeys ...
type DeliveryServiceSSLKeys struct {
	AuthType        string                            `json:"authType,omitempty"`
	CDN             string                            `json:"cdn,omitempty"`
	DeliveryService string                            `json:"deliveryservice,omitempty"`
	BusinessUnit    string                            `json:"businessUnit,omitempty"`
	City            string                            `json:"city,omitempty"`
	Organization    string                            `json:"organization,omitempty"`
	Hostname        string                            `json:"hostname,omitempty"`
	Country         string                            `json:"country,omitempty"`
	State           string                            `json:"state,omitempty"`
	Key             string                            `json:"key"`
	Version         util.JSONIntStr                   `json:"version"`
	Certificate     DeliveryServiceSSLKeysCertificate `json:"certificate,omitempty"`
}

type DeliveryServiceSSLKeysV15 struct {
	DeliveryServiceSSLKeys
	Expiration time.Time `json:"expiration,omitempty"`
}

type SSLKeyRequestFields struct {
	BusinessUnit *string `json:"businessUnit,omitempty"`
	City         *string `json:"city,omitempty"`
	Organization *string `json:"organization,omitempty"`
	HostName     *string `json:"hostname,omitempty"`
	Country      *string `json:"country,omitempty"`
	State        *string `json:"state,omitempty"`
	Version      *int    `json:"version,omitempty"`
}

type DeliveryServiceSSLKeysReq struct {
	AuthType        *string `json:"authType,omitempty"`
	CDN             *string `json:"cdn,omitempty"`
	DeliveryService *string `json:"deliveryservice,omitempty"`
	BusinessUnit    *string `json:"businessUnit,omitempty"`
	City            *string `json:"city,omitempty"`
	Organization    *string `json:"organization,omitempty"`
	HostName        *string `json:"hostname,omitempty"`
	Country         *string `json:"country,omitempty"`
	State           *string `json:"state,omitempty"`
	// Key is the XMLID of the delivery service
	Key         *string                            `json:"key"`
	Version     *util.JSONIntStr                   `json:"version"`
	Certificate *DeliveryServiceSSLKeysCertificate `json:"certificate,omitempty"`
}

// DeliveryServiceSSLKeysGenerationResponse is the type of a response from
// Traffic Ops to a request for generation of SSL Keys for a Delivery Service.
type DeliveryServiceSSLKeysGenerationResponse struct {
	Response string `json:"response"`
	Alerts
}

// DeliveryServiceSSLKeysCertificate ...
type DeliveryServiceSSLKeysCertificate struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
	CSR string `json:"csr"`
}

func (r *DeliveryServiceSSLKeysReq) Sanitize() {
	// DeliveryService and Key are the same value, so if the user sent one but not the other, set the missing one, in the principle of "be liberal in what you accept."
	if r.DeliveryService == nil && r.Key != nil {
		k := *r.Key // sqlx fails with aliased pointers, so make a new one
		r.DeliveryService = &k
	} else if r.Key == nil && r.DeliveryService != nil {
		k := *r.DeliveryService // sqlx fails with aliased pointers, so make a new one
		r.Key = &k
	}
}

// validateSharedRequiredRequestFields validates the request fields that are shared and required by both 'add' and 'generate' requests
func (r *DeliveryServiceSSLKeysReq) validateSharedRequiredRequestFields() []string {
	errs := []string{}
	if checkNilOrEmpty(r.CDN) {
		errs = append(errs, "cdn required")
	}
	if r.Version == nil {
		errs = append(errs, "version required")
	}
	if checkNilOrEmpty(r.Key) {
		errs = append(errs, "key required")
	}
	if checkNilOrEmpty(r.DeliveryService) {
		errs = append(errs, "deliveryservice required")
	}
	if r.Key != nil && r.DeliveryService != nil && *r.Key != *r.DeliveryService {
		errs = append(errs, "deliveryservice and key must match")
	}
	if checkNilOrEmpty(r.HostName) {
		errs = append(errs, "hostname required")
	}
	return errs
}

type DeliveryServiceAddSSLKeysReq struct {
	DeliveryServiceSSLKeysReq
}

func (r *DeliveryServiceAddSSLKeysReq) Validate(tx *sql.Tx) error {
	r.Sanitize()
	errs := r.validateSharedRequiredRequestFields()
	if r.Certificate == nil {
		errs = append(errs, "certificate required")
	} else {
		if r.Certificate.Key == "" {
			errs = append(errs, "certificate.key required")
		}
		if r.Certificate.Crt == "" {
			errs = append(errs, "certificate.crt required")
		}
		if r.Certificate.CSR == "" && (r.AuthType == nil || *r.AuthType != LetsEncryptAuthType) {
			errs = append(errs, "certificate.csr required")
		}
	}
	if len(errs) > 0 {
		return errors.New("missing fields: " + strings.Join(errs, "; "))
	}
	return nil
}

type DeliveryServiceGenSSLKeysReq struct {
	DeliveryServiceSSLKeysReq
}

func (r *DeliveryServiceGenSSLKeysReq) Validate(tx *sql.Tx) error {
	r.Sanitize()
	errs := r.validateSharedRequiredRequestFields()
	if checkNilOrEmpty(r.BusinessUnit) {
		errs = append(errs, "businessUnit required")
	}
	if checkNilOrEmpty(r.City) {
		errs = append(errs, "city required")
	}
	if checkNilOrEmpty(r.Organization) {
		errs = append(errs, "organization required")
	}
	if checkNilOrEmpty(r.Country) {
		errs = append(errs, "country required")
	}
	if checkNilOrEmpty(r.State) {
		errs = append(errs, "state required")
	}
	if len(errs) > 0 {
		return errors.New("missing fields: " + strings.Join(errs, "; "))
	}
	return nil
}

type DeliveryServiceAcmeSSLKeysReq struct {
	DeliveryServiceSSLKeysReq
}

func (r *DeliveryServiceAcmeSSLKeysReq) Validate(tx *sql.Tx) error {
	r.Sanitize()
	errs := r.validateSharedRequiredRequestFields()
	if len(errs) > 0 {
		return errors.New("missing fields: " + strings.Join(errs, "; "))
	}
	errs = r.validateAcmeSpecificFields()
	if len(errs) > 0 {
		return errors.New("missing fields: " + strings.Join(errs, "; "))
	}
	return nil
}

func (r *DeliveryServiceAcmeSSLKeysReq) validateAcmeSpecificFields() []string {
	errs := []string{}
	if checkNilOrEmpty(r.AuthType) {
		errs = append(errs, "authType required")
	}
	return errs
}

func checkNilOrEmpty(s *string) bool {
	return s == nil || *s == ""
}

// URISignerKeyset is the container for the CDN URI signing keys.
type URISignerKeyset struct {
	RenewalKid *string               `json:"renewal_kid"`
	Keys       []jwk.EssentialHeader `json:"keys"`
}

// TrafficVaultPing represents the status of a given Traffic Vault server.
type TrafficVaultPing struct {
	Status string `json:"status"`
	Server string `json:"server"`
}

// TrafficVaultPingResponse represents the JSON HTTP response returned by the /vault/ping route.
type TrafficVaultPingResponse struct {
	Response TrafficVaultPing `json:"response"`
	Alerts
}

type URLSigKeys map[string]string

// URLSignatureKeysResponse is the type of a response from Traffic Ops to a request
// for the URL Signing keys of a Delivery Service - in API version 4.0.
type URLSignatureKeysResponse struct {
	Response URLSigKeys `json:"response"`
	Alerts
}

type CDNSSLKeysResp []CDNSSLKey

type CDNSSLKey struct {
	DeliveryService string        `json:"deliveryservice"`
	HostName        string        `json:"hostname"`
	Certificate     CDNSSLKeyCert `json:"certificate"`
}

type CDNSSLKeyCert struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

type CDNGenerateKSKReq struct {
	ExpirationDays *uint64    `json:"expirationDays"`
	EffectiveDate  *time.Time `json:"effectiveDate"`
}

func (r *CDNGenerateKSKReq) Validate(tx *sql.Tx) error {
	r.Sanitize()
	errs := []string{}
	if r.ExpirationDays == nil || *r.ExpirationDays == 0 {
		errs = append(errs, "expiration missing")
	}
	// effective date is optional
	if len(errs) > 0 {
		return errors.New("missing fields: " + strings.Join(errs, "; "))
	}
	return nil
}

func (r *CDNGenerateKSKReq) Sanitize() {
	if r.EffectiveDate == nil {
		now := time.Now()
		r.EffectiveDate = &now
	}
}
