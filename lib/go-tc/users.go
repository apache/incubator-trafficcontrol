package tc

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

import "github.com/apache/trafficcontrol/lib/go-rfc"

// UserCredentials contains Traffic Ops login credentials
type UserCredentials struct {
	Username string `json:"u"`
	Password string `json:"p"`
}

// UserToken represents a request payload containing a UUID token for authentication
type UserToken struct {
	Token string `json:"t"`
}

// UserV13 contains non-nullable TO user information
type UserV13 struct {
	Username         string    `json:"username"`
	PublicSSHKey     string    `json:"publicSshKey"`
	Role             int       `json:"role"`
	RoleName         string    `json:"rolename"`
	ID               int       `json:"id"`
	UID              int       `json:"uid"`
	GID              int       `json:"gid"`
	Company          string    `json:"company"`
	Email            string    `json:"email"`
	FullName         string    `json:"fullName"`
	NewUser          bool      `json:"newUser"`
	LastUpdated      string    `json:"lastUpdated"`
	AddressLine1     string    `json:"addressLine1"`
	AddressLine2     string    `json:"addressLine2"`
	City             string    `json:"city"`
	Country          string    `json:"country"`
	PhoneNumber      string    `json:"phoneNumber"`
	PostalCode       string    `json:"postalCode"`
	RegistrationSent TimeNoMod `json:"registrationSent"`
	StateOrProvince  string    `json:"stateOrProvince"`
	Tenant           string    `json:"tenant"`
	TenantID         int       `json:"tenantId"`
}

// LastUpdated is a new field for some structs
type CommonUserFields struct {
	AddressLine1    *string `json:"addressLine1" db:"address_line1"`
	AddressLine2    *string `json:"addressLine2" db:"address_line2"`
	City            *string `json:"city" db:"city"`
	Company         *string `json:"company" db:"company"`
	Country         *string `json:"country" db:"country"`
	Email           *string `json:"email" db:"email"`
	FullName        *string `json:"fullName" db:"full_name"`
	GID             *int    `json:"gid"`
	ID              *int    `json:"id" db:"id"`
	NewUser         *bool   `json:"newUser" db:"new_user"`
	PhoneNumber     *string `json:"phoneNumber" db:"phone_number"`
	PostalCode      *string `json:"postalCode" db:"postal_code"`
	PublicSSHKey    *string `json:"publicSshKey" db:"public_ssh_key"`
	Role            *int    `json:"role" db:"role"`
	StateOrProvince *string `json:"stateOrProvince" db:"state_or_province"`
	Tenant          *string `json:"tenant"`
	TenantID        *int    `json:"tenantId" db:"tenant_id"`
	Token           *string `json:"-" db:"token"`
	UID             *int    `json:"uid"`
	//Username        *string    `json:"username" db:"username"`  //not including major change due to naming incompatibility
	LastUpdated *TimeNoMod `json:"lastUpdated" db:"last_updated"`
}

// User fields in v14 have been updated to be nullable
type User struct {
	Username         *string    `json:"username" db:"username"`
	RegistrationSent *TimeNoMod `json:"registrationSent" db:"registration_sent"`
	LocalPassword    *string    `json:"localPasswd,omitempty" db:"local_passwd"`
	RoleName         *string    `json:"roleName,omitempty" db:"-"`
	CommonUserFields
}

// UserCurrent represents the profile for the authenticated user
type UserCurrent struct {
	UserName  *string `json:"username"`
	LocalUser *bool   `json:"localUser"`
	RoleName  *string `json:"roleName"`
	CommonUserFields
}

// ------------------- Response structs -------------------- //
//  Response structs should only be used in the client       //
//  The client's use of these will eventually be deprecated  //
// --------------------------------------------------------- //

type UsersResponseV13 struct {
	Response []UserV13 `json:"response"`
}

type UsersResponse struct {
	Response []User `json:"response"`
}

type CreateUserResponse struct {
	Response User `json:"response"`
	Alerts
}

type UpdateUserResponse struct {
	Response User `json:"response"`
	Alerts
}

type DeleteUserResponse struct {
	Alerts
}

type UserCurrentResponse struct {
	Response UserCurrent `json:"response"`
}

type UserDeliveryServiceDeleteResponse struct {
	Alerts
}

type UserPasswordResetRequest struct {
	Email rfc.EmailAddress `json:"email"`
}
