package client

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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// These are the API endpoints used by the various Delivery Service-related client methods.
const (
	// API_DELIVERY_SERVICES is the API path on which Traffic Ops serves Delivery Service
	// information. More specific information is typically found on sub-paths of this.
	apiDeliveryServices = "/deliveryservices"

	// APIDeliveryServiceId is the API path on which Traffic Ops serves information about
	// a specific Delivery Service identified by an integral, unique identifier. It is
	// intended to be used with fmt.Sprintf to insert its required path parameter (namely the ID
	// of the Delivery Service of interest).
	apiDeliveryServiceID = apiDeliveryServices + "/%d"

	// apiDeliveryServiceHealth is the API path on which Traffic Ops serves information about
	// the 'health' of a specific Delivery Service identified by an integral, unique identifier. It is
	// intended to be used with fmt.Sprintf to insert its required path parameter (namely the ID
	// of the Delivery Service of interest).
	apiDeliveryServiceHealth = apiDeliveryServiceID + "/health"

	// apiDeliveryServiceCapacity is the API path on which Traffic Ops serves information about
	// the 'capacity' of a specific Delivery Service identified by an integral, unique identifier. It is
	// intended to be used with fmt.Sprintf to insert its required path parameter (namely the ID
	// of the Delivery Service of interest).
	apiDeliveryServiceCapacity = apiDeliveryServiceID + "/capacity"

	// apiDeliveryServiceEligibleServers is the API path on which Traffic Ops serves information about
	// the servers which are eligible to be assigned to a specific Delivery Service identified by an integral,
	// unique identifier. It is intended to be used with fmt.Sprintf to insert its required path parameter
	// (namely the ID of the Delivery Service of interest).
	apiDeliveryServiceEligibleServers = apiDeliveryServiceID + "/servers/eligible"

	// apiDeliveryServicesSafeUpdate is the API path on which Traffic Ops provides the functionality to
	// update the "safe" subset of properties of a Delivery Service identified by an integral, unique
	// identifier. It is intended to be used with fmt.Sprintf to insert its required path parameter
	// (namely the ID of the Delivery Service of interest).
	apiDeliveryServicesSafeUpdate = apiDeliveryServiceID + "/safe"

	// apiAPIDeliveryServiceXMLIDSSLKeys is the API path on which Traffic Ops serves information about
	// and functionality relating to the SSL keys used by a Delivery Service identified by its XMLID. It is
	// intended to be used with fmt.Sprintf to insert its required path parameter (namely the XMLID
	// of the Delivery Service of interest).
	apiAPIDeliveryServiceXMLIDSSLKeys = apiDeliveryServices + "/xmlId/%s/sslkeys"

	// apiDeliveryServiceGenerateSSLKeys is the API path on which Traffic Ops will generate new SSL keys.
	apiDeliveryServiceGenerateSSLKeys = apiDeliveryServices + "/sslkeys/generate"

	// apiDeliveryServiceAddSSLKeys is the API path on which Traffic Ops will add SSL keys.
	apiDeliveryServiceAddSSLKeys = apiDeliveryServices + "/sslkeys/add"

	// apiDeliveryServiceURISigningKeys is the API path on which Traffic Ops serves information
	// about and functionality relating to the URI-signing keys used by a Delivery Service identified
	// by its XMLID. It is intended to be used with fmt.Sprintf to insert its required path parameter
	// (namely the XMLID of the Delivery Service of interest).
	apiDeliveryServicesURISigningKeys = apiDeliveryServices + "/%s/urisignkeys"

	// apiDeliveryServicesURLSignatureKeys is the API path on which Traffic Ops serves information
	// about and functionality relating to the URL-signing keys used by a Delivery Service identified
	// by its XMLID. It is intended to be used with fmt.Sprintf to insert its required path parameter
	// (namely the XMLID of the Delivery Service of interest).
	apiDeliveryServicesURLSignatureKeys = apiDeliveryServices + "/xmlId/%s/urlkeys"

	// apiDeliveryServicesURLSignatureKeysGenerate is the API path on which Traffic Ops provides
	// functionality to generate new URL-signing keys used by a Delivery Service identified
	// by its XMLID. It is intended to be used with fmt.Sprintf to insert its required path parameter
	// (namely the XMLID of the Delivery Service of interest).
	apiDeliveryServicesURLSignatureKeysGenerate = apiDeliveryServices + "/xmlId/%s/urlkeys/generate"

	// apiDeliveryServicesRegexes is the API path on which Traffic Ops serves Delivery Service
	// 'regex' (Regular Expression) information.
	apiDeliveryServicesRegexes = "/deliveryservices_regexes"

	// apiServerDeliveryServices is the API path on which Traffic Ops serves functionality
	// related to the associations a specific server and its assigned Delivery Services. It is
	// intended to be used with fmt.Sprintf to insert its required path parameter (namely the ID
	// of the server of interest).
	apiServerDeliveryServices = "/servers/%d/deliveryservices"

	// apiDeliveryServiceServer is the API path on which Traffic Ops serves functionality related
	// to the associations between Delivery Services and their assigned Server(s).
	apiDeliveryServiceServer = "/deliveryserviceserver"

	// apiDeliveryServicesServers is the API path on which Traffic Ops serves functionality related
	// to the associations between a Delivery Service and its assigned Server(s).
	apiDeliveryServicesServers = "/deliveryservices/%s/servers"
)

// GetDeliveryServicesByServer retrieves all Delivery Services assigned to the
// server with the given ID.
func (to *Session) GetDeliveryServicesByServer(id int, opts RequestOptions) (tc.DeliveryServicesResponseV4, toclientlib.ReqInf, error) {
	var data tc.DeliveryServicesResponseV4
	reqInf, err := to.get(fmt.Sprintf(apiServerDeliveryServices, id), opts, &data)
	return data, reqInf, err
}

// GetDeliveryServices returns (tenant-visible) Delivery Services.
func (to *Session) GetDeliveryServices(opts RequestOptions) (tc.DeliveryServicesResponseV4, toclientlib.ReqInf, error) {
	var data tc.DeliveryServicesResponseV4
	reqInf, err := to.get(apiDeliveryServices, opts, &data)
	return data, reqInf, err
}

// CreateDeliveryService creates the Delivery Service it's passed.
func (to *Session) CreateDeliveryService(ds tc.DeliveryServiceV4, opts RequestOptions) (tc.DeliveryServicesResponseV4, toclientlib.ReqInf, error) {
	var reqInf toclientlib.ReqInf
	var resp tc.DeliveryServicesResponseV4
	if ds.TypeID == 0 && ds.Type != nil {
		typeOpts := NewRequestOptions()
		typeOpts.QueryParameters.Set("name", ds.Type.String())
		ty, _, err := to.GetTypes(typeOpts)
		if err != nil {
			return resp, reqInf, err
		}
		if len(ty.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no Type named '%s'", ds.Type)
		}
		ds.TypeID = ty.Response[0].ID
	}

	if ds.CDNID == 0 && ds.CDNName != nil {
		cdnOpts := NewRequestOptions()
		cdnOpts.QueryParameters.Set("name", *ds.CDNName)
		cdns, _, err := to.GetCDNs(cdnOpts)
		if err != nil {
			err = fmt.Errorf("attempting to resolve CDN name '%s' to an ID: %w", *ds.CDNName, err)
			return resp, reqInf, err
		}
		if len(cdns.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no CDN named '%s'", *ds.CDNName)
		}
		ds.CDNID = cdns.Response[0].ID
	}

	if ds.ProfileID == nil && ds.ProfileName != nil {
		profileOpts := NewRequestOptions()
		profileOpts.QueryParameters.Set("name", *ds.ProfileName)
		profiles, _, err := to.GetProfiles(profileOpts)
		if err != nil {
			return resp, reqInf, fmt.Errorf("attempting to resolve Profile name '%s' to an ID: %w", *ds.ProfileName, err)
		}
		if len(profiles.Response) == 0 {
			return resp, reqInf, errors.New("no Profile named " + *ds.ProfileName)
		}
		ds.ProfileID = &profiles.Response[0].ID
	}

	if ds.TenantID == 0 && ds.Tenant != nil {
		tenantOpts := NewRequestOptions()
		tenantOpts.QueryParameters.Set("name", *ds.Tenant)
		ten, _, err := to.GetTenants(tenantOpts)
		if err != nil {
			return resp, reqInf, fmt.Errorf("attempting to resolve Tenant '%s' to an ID: %w", *ds.Tenant, err)
		}
		if len(ten.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no Tenant named '%s'", *ds.Tenant)
		}
		ds.TenantID = ten.Response[0].ID
	}

	reqInf, err := to.post(apiDeliveryServices, RequestOptions{Header: opts.Header}, ds, &resp)
	if err != nil {
		return resp, reqInf, err
	}

	return resp, reqInf, nil
}

// UpdateDeliveryService replaces the Delivery Service identified by the
// integral, unique identifier 'id' with the one it's passed.
func (to *Session) UpdateDeliveryService(id int, ds tc.DeliveryServiceV4, opts RequestOptions) (tc.DeliveryServicesResponseV4, toclientlib.ReqInf, error) {
	var data tc.DeliveryServicesResponseV4
	reqInf, err := to.put(fmt.Sprintf(apiDeliveryServiceID, id), opts, ds, &data)
	if err != nil {
		return data, reqInf, err
	}
	return data, reqInf, nil
}

// DeleteDeliveryService deletes the DeliveryService matching the ID it's passed.
func (to *Session) DeleteDeliveryService(id int, opts RequestOptions) (tc.DeleteDeliveryServiceResponse, toclientlib.ReqInf, error) {
	var data tc.DeleteDeliveryServiceResponse
	reqInf, err := to.del(fmt.Sprintf(apiDeliveryServiceID, id), opts, &data)
	return data, reqInf, err
}

// GetDeliveryServiceHealth gets the 'health' of the Delivery Service identified by the
// integral, unique identifier 'id'.
func (to *Session) GetDeliveryServiceHealth(id int, opts RequestOptions) (tc.DeliveryServiceHealthResponse, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceHealthResponse
	reqInf, err := to.get(fmt.Sprintf(apiDeliveryServiceHealth, id), opts, &data)
	return data, reqInf, err
}

// GetDeliveryServiceCapacity gets the 'capacity' of the Delivery Service identified by the
// integral, unique identifier 'id'.
func (to *Session) GetDeliveryServiceCapacity(id int, opts RequestOptions) (tc.DeliveryServiceCapacityResponse, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceCapacityResponse
	reqInf, err := to.get(fmt.Sprintf(apiDeliveryServiceCapacity, id), opts, &data)
	return data, reqInf, err
}

// GenerateSSLKeysForDS generates ssl keys for a given cdn.
func (to *Session) GenerateSSLKeysForDS(
	xmlid string,
	cdnName string,
	sslFields tc.SSLKeyRequestFields,
	opts RequestOptions,
) (tc.DeliveryServiceSSLKeysGenerationResponse, toclientlib.ReqInf, error) {
	version := util.JSONIntStr(1)
	request := tc.DeliveryServiceSSLKeysReq{
		BusinessUnit:    sslFields.BusinessUnit,
		CDN:             util.StrPtr(cdnName),
		City:            sslFields.City,
		Country:         sslFields.Country,
		DeliveryService: util.StrPtr(xmlid),
		HostName:        sslFields.HostName,
		Key:             util.StrPtr(xmlid),
		Organization:    sslFields.Organization,
		State:           sslFields.State,
		Version:         &version,
	}
	var response tc.DeliveryServiceSSLKeysGenerationResponse
	reqInf, err := to.post(apiDeliveryServiceGenerateSSLKeys, opts, request, &response)
	return response, reqInf, err
}

// AddSSLKeysForDS adds SSL Keys for the given Delivery Service.
func (to *Session) AddSSLKeysForDS(request tc.DeliveryServiceAddSSLKeysReq, opts RequestOptions) (tc.SSLKeysAddResponse, toclientlib.ReqInf, error) {
	var response tc.SSLKeysAddResponse
	reqInf, err := to.post(apiDeliveryServiceAddSSLKeys, opts, request, &response)
	return response, reqInf, err
}

// DeleteDeliveryServiceSSLKeys deletes the SSL Keys used by the Delivery
// Service identified by the passed XMLID.
func (to *Session) DeleteDeliveryServiceSSLKeys(xmlid string, opts RequestOptions) (tc.DeliveryServiceSSLKeysGenerationResponse, toclientlib.ReqInf, error) {
	var resp tc.DeliveryServiceSSLKeysGenerationResponse
	reqInf, err := to.del(fmt.Sprintf(apiAPIDeliveryServiceXMLIDSSLKeys, url.QueryEscape(xmlid)), opts, &resp)
	return resp, reqInf, err
}

// GetDeliveryServiceSSLKeys retrieves the SSL keys of the Delivery Service
// with the given XMLID.
func (to *Session) GetDeliveryServiceSSLKeys(xmlid string, opts RequestOptions) (tc.DeliveryServiceSSLKeysResponse, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceSSLKeysResponse
	reqInf, err := to.get(fmt.Sprintf(apiAPIDeliveryServiceXMLIDSSLKeys, url.QueryEscape(xmlid)), opts, &data)
	return data, reqInf, err
}

// GetDeliveryServicesEligible returns the servers eligible for assignment to the Delivery
// Service identified by the integral, unique identifier 'dsID'.
func (to *Session) GetDeliveryServicesEligible(dsID int, opts RequestOptions) (tc.DSServerResponseV4, toclientlib.ReqInf, error) {
	var resp tc.DSServerResponseV4
	reqInf, err := to.get(fmt.Sprintf(apiDeliveryServiceEligibleServers, dsID), opts, &resp)
	return resp, reqInf, err
}

// GetDeliveryServiceURLSignatureKeys returns the URL-signing keys used by the Delivery Service
// identified by the XMLID 'dsName'.
func (to *Session) GetDeliveryServiceURLSignatureKeys(dsName string, opts RequestOptions) (tc.URLSignatureKeysResponse, toclientlib.ReqInf, error) {
	var data tc.URLSignatureKeysResponse
	reqInf, err := to.get(fmt.Sprintf(apiDeliveryServicesURLSignatureKeys, dsName), opts, &data)
	return data, reqInf, err
}

// CreateDeliveryServiceURLSignatureKeys creates new URL-signing keys used by
// the Delivery Service identified by the XMLID 'dsName'.
func (to *Session) CreateDeliveryServiceURLSignatureKeys(dsName string, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqInf, err := to.post(fmt.Sprintf(apiDeliveryServicesURLSignatureKeysGenerate, url.PathEscape(dsName)), opts, nil, &alerts)
	return alerts, reqInf, err
}

// DeleteDeliveryServiceURLSignatureKeys deletes the URL-signing keys used by the Delivery Service
// identified by the XMLID 'dsName'.
func (to *Session) DeleteDeliveryServiceURLSignatureKeys(dsName string, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqInf, err := to.del(fmt.Sprintf(apiDeliveryServicesURLSignatureKeys, url.PathEscape(dsName)), opts, &alerts)
	return alerts, reqInf, err
}

// GetDeliveryServiceURISigningKeys returns the URI-signing keys used by the Delivery Service
// identified by the XMLID 'dsName'. The result is not parsed.
// Note that unlike most methods, this is incapable of returning alerts.
func (to *Session) GetDeliveryServiceURISigningKeys(dsName string, opts RequestOptions) ([]byte, toclientlib.ReqInf, error) {
	data := json.RawMessage{}
	reqInf, err := to.get(fmt.Sprintf(apiDeliveryServicesURISigningKeys, url.PathEscape(dsName)), opts, &data)
	return []byte(data), reqInf, err
}

// CreateDeliveryServiceURISigningKeys creates new URI-signing keys used by the Delivery Service
// identified by the XMLID 'dsXMLID'.
func (to *Session) CreateDeliveryServiceURISigningKeys(dsXMLID string, body map[string]tc.URISignerKeyset, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqInf, err := to.post(fmt.Sprintf(apiDeliveryServicesURISigningKeys, url.PathEscape(dsXMLID)), opts, body, &alerts)
	return alerts, reqInf, err
}

// DeleteDeliveryServiceURISigningKeys deletes the URI-signing keys used by the Delivery Service
// identified by the XMLID 'dsXMLID'.
func (to *Session) DeleteDeliveryServiceURISigningKeys(dsXMLID string, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqInf, err := to.del(fmt.Sprintf(apiDeliveryServicesURISigningKeys, url.PathEscape(dsXMLID)), opts, &alerts)
	return alerts, reqInf, err
}

// SafeDeliveryServiceUpdate updates the "safe" fields of the Delivery
// Service identified by the integral, unique identifier 'id'.
func (to *Session) SafeDeliveryServiceUpdate(
	id int,
	r tc.DeliveryServiceSafeUpdateRequest,
	opts RequestOptions,
) (tc.DeliveryServiceSafeUpdateResponseV4, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceSafeUpdateResponseV4
	reqInf, err := to.put(fmt.Sprintf(apiDeliveryServicesSafeUpdate, id), opts, r, &data)
	return data, reqInf, err
}
