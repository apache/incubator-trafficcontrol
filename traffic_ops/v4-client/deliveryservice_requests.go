package client

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
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// apiDSRequests is the API version-relative path to the
// /deliveryservice_requests API endpoint.
const apiDSRequests = "/deliveryservice_requests"

// CreateDeliveryServiceRequest creates the given Delivery Service Request.
func (to *Session) CreateDeliveryServiceRequest(dsr tc.DeliveryServiceRequestV4, opts RequestOptions) (tc.DeliveryServiceRequestResponseV4, toclientlib.ReqInf, error) {
	var resp tc.DeliveryServiceRequestResponseV4
	if dsr.AssigneeID == nil && dsr.Assignee != nil {
		assigneeOpts := NewRequestOptions()
		assigneeOpts.QueryParameters.Set("username", *dsr.Assignee)
		res, reqInf, err := to.GetUsers(assigneeOpts)
		if err != nil {
			return resp, reqInf, err
		}
		if len(res.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no user with username '%s'", *dsr.Assignee)
		}
		dsr.AssigneeID = res.Response[0].ID
	}

	if dsr.AuthorID == nil && dsr.Author != "" {
		authorOpts := NewRequestOptions()
		authorOpts.QueryParameters.Set("username", dsr.Author)
		res, reqInf, err := to.GetUsers(authorOpts)
		if err != nil {
			return resp, reqInf, err
		}
		if len(res.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no user with name '%s'", dsr.Author)
		}
		dsr.AuthorID = res.Response[0].ID
	}

	var ds *tc.DeliveryServiceV4
	if dsr.ChangeType == tc.DSRChangeTypeDelete {
		ds = dsr.Original
	} else {
		ds = dsr.Requested
	}

	if ds.TypeID == 0 && ds.Type.String() != "" {
		typeOpts := NewRequestOptions()
		typeOpts.QueryParameters.Set("name", ds.Type.String())
		ty, reqInf, err := to.GetTypes(typeOpts)
		if err != nil || len(ty.Response) == 0 {
			return resp, reqInf, errors.New("no type named " + ds.Type.String())
		}
		ds.TypeID = ty.Response[0].ID
	}

	if ds.CDNID == 0 && ds.CDNName != nil {
		cdnOpts := NewRequestOptions()
		cdnOpts.QueryParameters.Set("name", *ds.CDNName)
		cdns, reqInf, err := to.GetCDNs(cdnOpts)
		if err != nil || len(cdns.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no CDN named '%s'", *ds.CDNName)
		}
		ds.CDNID = cdns.Response[0].ID
	}

	if ds.ProfileID == nil && ds.ProfileName != nil {
		profileOpts := NewRequestOptions()
		profileOpts.QueryParameters.Set("name", *ds.ProfileName)
		profiles, reqInf, err := to.GetProfiles(profileOpts)
		if err != nil || len(profiles.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no Profile named '%s'", *ds.ProfileName)
		}
		ds.ProfileID = &profiles.Response[0].ID
	}

	if ds.TenantID == 0 && ds.Tenant != nil {
		tenantOpts := NewRequestOptions()
		tenantOpts.QueryParameters.Set("name", *ds.Tenant)
		ten, reqInf, err := to.GetTenants(tenantOpts)
		if err != nil || len(ten.Response) == 0 {
			return resp, reqInf, fmt.Errorf("no Tenant named '%s'", *ds.Tenant)
		}
		ds.TenantID = ten.Response[0].ID
	}

	reqInf, err := to.post(apiDSRequests, opts, dsr, &resp)
	return resp, reqInf, err
}

// GetDeliveryServiceRequests retrieves Delivery Service Requests available to session user.
func (to *Session) GetDeliveryServiceRequests(opts RequestOptions) (tc.DeliveryServiceRequestsResponseV4, toclientlib.ReqInf, error) {
	var data tc.DeliveryServiceRequestsResponseV4
	reqInf, err := to.get(apiDSRequests, opts, &data)
	return data, reqInf, err
}

// DeleteDeliveryServiceRequest deletes the Delivery Service Request with the given ID.
func (to *Session) DeleteDeliveryServiceRequest(id int, opts RequestOptions) (tc.DeliveryServiceRequestResponseV4, toclientlib.ReqInf, error) {
	if opts.QueryParameters == nil {
		opts.QueryParameters = url.Values{}
	}
	opts.QueryParameters.Set("id", strconv.Itoa(id))
	var resp tc.DeliveryServiceRequestResponseV4
	reqInf, err := to.del(apiDSRequests, opts, &resp)
	return resp, reqInf, err
}

// UpdateDeliveryServiceRequest replaces the existing DSR that has the given
// ID with the DSR passed.
func (to *Session) UpdateDeliveryServiceRequest(id int, dsr tc.DeliveryServiceRequestV4, opts RequestOptions) (tc.DeliveryServiceRequestResponseV4, toclientlib.ReqInf, error) {
	if opts.QueryParameters == nil {
		opts.QueryParameters = url.Values{}
	}
	opts.QueryParameters.Set("id", strconv.Itoa(id))

	var payload tc.DeliveryServiceRequestResponseV4
	reqInf, err := to.put(apiDSRequests, opts, dsr, &payload)

	return payload, reqInf, err
}
