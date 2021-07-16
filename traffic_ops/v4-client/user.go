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
	"fmt"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/toclientlib"
)

// UserCurrentResponseV4 is an alias to avoid client breaking changes. In-case of a minor or major version change, we replace the below alias with a new structure.
type UserCurrentResponseV4 = tc.UserCurrentResponseV40

// GetUsers retrieves all (Tenant-accessible) Users stored in Traffic Ops.
func (to *Session) GetUsers(opts RequestOptions) (tc.UsersResponseV40, toclientlib.ReqInf, error) {
	data := tc.UsersResponseV40{}
	route := "/users"
	inf, err := to.get(route, opts, &data)
	return data, inf, err
}

// GetUserCurrent retrieves the currently authenticated User.
func (to *Session) GetUserCurrent(opts RequestOptions) (UserCurrentResponseV4, toclientlib.ReqInf, error) {
	route := `/user/current`
	resp := UserCurrentResponseV4{}
	reqInf, err := to.get(route, opts, &resp)
	return resp, reqInf, err
}

// UpdateCurrentUser replaces the current user data with the provided tc.UserV40 structure.
func (to *Session) UpdateCurrentUser(u tc.UserV40, opts RequestOptions) (tc.UpdateUserResponse, toclientlib.ReqInf, error) {
	user := struct {
		User tc.UserV40 `json:"user"`
	}{u}
	var clientResp tc.UpdateUserResponse
	reqInf, err := to.put("/user/current", opts, user, &clientResp)
	return clientResp, reqInf, err
}

// CreateUser creates the given user.
func (to *Session) CreateUser(user tc.UserV40, opts RequestOptions) (tc.CreateUserResponse, toclientlib.ReqInf, error) {
	if user.TenantID == nil && user.Tenant != nil {
		innerOpts := NewRequestOptions()
		innerOpts.QueryParameters.Set("name", *user.Tenant)
		tenant, _, err := to.GetTenants(innerOpts)
		if err != nil {
			return tc.CreateUserResponse{Alerts: tenant.Alerts}, toclientlib.ReqInf{}, fmt.Errorf("resolving Tenant name '%s' to an ID: %w", *user.Tenant, err)
		}
		if len(tenant.Response) < 1 {
			return tc.CreateUserResponse{Alerts: tenant.Alerts}, toclientlib.ReqInf{}, fmt.Errorf("no such Tenant: '%s'", *user.Tenant)
		}
		user.TenantID = &tenant.Response[0].ID
	}

	if user.RoleName != nil && *user.RoleName != "" {
		innerOpts := NewRequestOptions()
		innerOpts.QueryParameters.Set("name", *user.RoleName)
		roles, _, err := to.GetRoles(innerOpts)
		if err != nil {
			return tc.CreateUserResponse{Alerts: roles.Alerts}, toclientlib.ReqInf{}, fmt.Errorf("resolving Role name '%s' to an ID: %w", *user.RoleName, err)
		}
		if len(roles.Response) == 0 || roles.Response[0].ID == nil {
			return tc.CreateUserResponse{Alerts: roles.Alerts}, toclientlib.ReqInf{}, fmt.Errorf("no such Role: '%s'", *user.RoleName)
		}
		user.Role = roles.Response[0].ID
	}

	route := "/users"
	var clientResp tc.CreateUserResponse
	reqInf, err := to.post(route, opts, user, &clientResp)
	return clientResp, reqInf, err
}

// UpdateUser replaces the User identified by 'id' with the one provided.
func (to *Session) UpdateUser(id int, u tc.UserV40, opts RequestOptions) (tc.UpdateUserResponse, toclientlib.ReqInf, error) {
	route := "/users/" + strconv.Itoa(id)
	var clientResp tc.UpdateUserResponse
	reqInf, err := to.put(route, opts, u, &clientResp)
	return clientResp, reqInf, err
}

// DeleteUser deletes the User with the given ID.
func (to *Session) DeleteUser(id int, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	route := "/users/" + strconv.Itoa(id)
	var alerts tc.Alerts
	reqInf, err := to.del(route, opts, &alerts)
	return alerts, reqInf, err
}

// RegisterNewUser requests the registration of a new user with the given tenant ID and role ID,
// through their email.
func (to *Session) RegisterNewUser(tenantID uint, roleID uint, email rfc.EmailAddress, opts RequestOptions) (tc.Alerts, toclientlib.ReqInf, error) {
	var alerts tc.Alerts
	reqBody := tc.UserRegistrationRequest{
		Email:    email,
		TenantID: tenantID,
		Role:     roleID,
	}
	reqInf, err := to.post("/users/register", opts, reqBody, &alerts)
	return alerts, reqInf, err
}
