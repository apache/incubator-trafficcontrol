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

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

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

// commonUserFields is unexported, but its contents are still visible when it is embedded
// LastUpdated is a new field for some structs
type commonUserFields struct {
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
	Username             *string    `json:"username" db:"username"`
	RegistrationSent     *TimeNoMod `json:"registrationSent" db:"registration_sent"`
	LocalPassword        *string    `json:"localPasswd,omitempty" db:"local_passwd"`
	ConfirmLocalPassword *string    `json:"confirmLocalPasswd,omitempty" db:"confirm_local_passwd"`
	// NOTE: RoleName db:"-" tag is required due to clashing with the DB query here:
	// https://github.com/apache/trafficcontrol/blob/3b5dd406bf1a0bb456c062b0f6a465ec0617d8ef/traffic_ops/traffic_ops_golang/user/user.go#L197
	// It's done that way in order to maintain "rolename" vs "roleName" JSON field capitalization for the different users APIs.
	// TODO: make the breaking API change to make all user APIs use "roleName" consistently.
	RoleName *string `json:"roleName,omitempty" db:"-"`
	commonUserFields
}

// UserV40 contains ChangeLogCount field.
type UserV40 struct {
	User
	ChangeLogCount    *int       `json:"changeLogCount" db:"change_log_count"`
	LastAuthenticated *time.Time `json:"lastAuthenticated" db:"last_authenticated"`
}

// UserCurrent represents the profile for the authenticated user.
type UserCurrent struct {
	UserName  *string `json:"username"`
	LocalUser *bool   `json:"localUser"`
	RoleName  *string `json:"roleName"`
	commonUserFields
}

// UserCurrentV40 contains LastAuthenticated field.
type UserCurrentV40 struct {
	UserCurrent
	LastAuthenticated *time.Time `json:"lastAuthenticated" db:"last_authenticated"`
}

// CurrentUserUpdateRequest differs from a regular User/UserCurrent in that many of its fields are
// *parsed* but not *unmarshaled*. This allows a handler to distinguish between "null" and
// "undefined" values.
type CurrentUserUpdateRequest struct {
	// User, for whatever reason, contains all of the actual data.
	User CurrentUserUpdateRequestUser `json:"user"`
}

// CurrentUserUpdateRequestUser holds all of the actual data in a request to update the current user.
type CurrentUserUpdateRequestUser struct {
	AddressLine1       json.RawMessage `json:"addressLine1"`
	AddressLine2       json.RawMessage `json:"addressLine2"`
	City               json.RawMessage `json:"city"`
	Company            json.RawMessage `json:"company"`
	ConfirmLocalPasswd *string         `json:"confirmLocalPasswd"`
	Country            json.RawMessage `json:"country"`
	Email              json.RawMessage `json:"email"`
	FullName           json.RawMessage `json:"fullName"`
	GID                json.RawMessage `json:"gid"`
	ID                 json.RawMessage `json:"id"`
	LocalPasswd        *string         `json:"localPasswd"`
	PhoneNumber        json.RawMessage `json:"phoneNumber"`
	PostalCode         json.RawMessage `json:"postalCode"`
	PublicSSHKey       json.RawMessage `json:"publicSshKey"`
	Role               json.RawMessage `json:"role"`
	StateOrProvince    json.RawMessage `json:"stateOrProvince"`
	TenantID           json.RawMessage `json:"tenantId"`
	UID                json.RawMessage `json:"uid"`
	Username           json.RawMessage `json:"username"`
}

// UnmarshalAndValidate validates the request and returns a User into which the request's information
// has been unmarshalled.
func (u *CurrentUserUpdateRequestUser) UnmarshalAndValidate(user *User) error {
	errs := []error{}
	if u.AddressLine1 != nil {
		if err := json.Unmarshal(u.AddressLine1, &user.AddressLine1); err != nil {
			errs = append(errs, fmt.Errorf("addressLine1: %v", err))
		}
	}

	if u.AddressLine2 != nil {
		if err := json.Unmarshal(u.AddressLine2, &user.AddressLine2); err != nil {
			errs = append(errs, fmt.Errorf("addressLine2: %v", err))
		}
	}

	if u.City != nil {
		if err := json.Unmarshal(u.City, &user.City); err != nil {
			errs = append(errs, fmt.Errorf("city: %v", err))
		}
	}

	if u.Company != nil {
		if err := json.Unmarshal(u.Company, &user.Company); err != nil {
			errs = append(errs, fmt.Errorf("company: %v", err))
		}
	}

	user.ConfirmLocalPassword = u.ConfirmLocalPasswd
	user.LocalPassword = u.LocalPasswd

	if u.Country != nil {
		if err := json.Unmarshal(u.Country, &user.Country); err != nil {
			errs = append(errs, fmt.Errorf("country: %v", err))
		}
	}

	if u.Email != nil {
		if err := json.Unmarshal(u.Email, &user.Email); err != nil {
			errs = append(errs, fmt.Errorf("email: %v", err))
		} else if user.Email == nil || *user.Email == "" {
			errs = append(errs, errors.New("email: cannot be null or an empty string"))
		} else if err = validation.Validate(*user.Email, is.Email); err != nil {
			errs = append(errs, err)
		}
	}

	if u.FullName != nil {
		if err := json.Unmarshal(u.FullName, &user.FullName); err != nil {
			errs = append(errs, fmt.Errorf("fullName: %v", err))
		} else if user.FullName == nil || *user.FullName == "" {
			// Perl enforced this
			errs = append(errs, errors.New("fullName: cannot be set to 'null' or empty string"))
		}
	}

	if u.GID != nil {
		if err := json.Unmarshal(u.GID, &user.GID); err != nil {
			errs = append(errs, fmt.Errorf("gid: %v", err))
		}
	}

	if u.ID != nil {
		var uid int
		if err := json.Unmarshal(u.ID, &uid); err != nil {
			errs = append(errs, fmt.Errorf("id: %v", err))
		} else if user.ID != nil && *user.ID != uid {
			errs = append(errs, errors.New("id: cannot change user id"))
		} else {
			user.ID = &uid
		}
	}

	if u.PhoneNumber != nil {
		if err := json.Unmarshal(u.PhoneNumber, &user.PhoneNumber); err != nil {
			errs = append(errs, fmt.Errorf("phoneNumber: %v", err))
		}
	}

	if u.PostalCode != nil {
		if err := json.Unmarshal(u.PostalCode, &user.PostalCode); err != nil {
			errs = append(errs, fmt.Errorf("postalCode: %v", err))
		}
	}

	if u.PublicSSHKey != nil {
		if err := json.Unmarshal(u.PublicSSHKey, &user.PublicSSHKey); err != nil {
			errs = append(errs, fmt.Errorf("publicSshKey: %v", err))
		}
	}

	if u.Role != nil {
		if err := json.Unmarshal(u.Role, &user.Role); err != nil {
			errs = append(errs, fmt.Errorf("role: %v", err))
		} else if user.Role == nil {
			errs = append(errs, errors.New("role: cannot be null"))
		}
	}

	if u.StateOrProvince != nil {
		if err := json.Unmarshal(u.StateOrProvince, &user.StateOrProvince); err != nil {
			errs = append(errs, fmt.Errorf("stateOrProvince: %v", err))
		}
	}

	if u.TenantID != nil {
		if err := json.Unmarshal(u.TenantID, &user.TenantID); err != nil {
			errs = append(errs, fmt.Errorf("tenantID: %v", err))
		} else if user.TenantID == nil {
			errs = append(errs, errors.New("tenantID: cannot be null"))
		}
	}

	if u.UID != nil {
		if err := json.Unmarshal(u.UID, &user.UID); err != nil {
			errs = append(errs, fmt.Errorf("uid: %v", err))
		}
	}

	if u.Username != nil {
		if err := json.Unmarshal(u.Username, &user.Username); err != nil {
			errs = append(errs, fmt.Errorf("username: %v", err))
		} else if user.Username == nil || *user.Username == "" {
			errs = append(errs, errors.New("username: cannot be null or empty string"))
		}
	}

	return util.JoinErrs(errs)
}

// ------------------- Response structs -------------------- //
//  Response structs should only be used in the client       //
//  The client's use of these will eventually be deprecated  //
// --------------------------------------------------------- //

// UsersResponseV13 is the Traffic Ops API version 1.3 variant of UserResponse.
// It is unused.
type UsersResponseV13 struct {
	Response []UserV13 `json:"response"`
	Alerts
}

// UsersResponse can hold a Traffic Ops API response to a request to get a list of users.
type UsersResponse struct {
	Response []User `json:"response"`
	Alerts
}

// UsersResponseV40 is the Traffic Ops API version 4.0 variant of UserResponse.
type UsersResponseV40 struct {
	Response []UserV40 `json:"response"`
	Alerts
}

// CreateUserResponse can hold a Traffic Ops API response to a POST request to create a user.
type CreateUserResponse struct {
	Response User `json:"response"`
	Alerts
}

// UpdateUserResponse can hold a Traffic Ops API response to a PUT request to update a user.
type UpdateUserResponse struct {
	Response User `json:"response"`
	Alerts
}

// DeleteUserResponse can theoretically hold a Traffic Ops API response to a
// DELETE request to update a user. It is unused.
type DeleteUserResponse struct {
	Alerts
}

// UserCurrentResponse can hold a Traffic Ops API response to a request to get
// or update the current user.
type UserCurrentResponse struct {
	Response UserCurrent `json:"response"`
	Alerts
}

// UserCurrentResponseV40 is the Traffic Ops API version 4.0 variant of UserResponse.
type UserCurrentResponseV40 struct {
	Response UserCurrentV40 `json:"response"`
	Alerts
}

// UserDeliveryServiceDeleteResponse can hold a Traffic Ops API response to
// a request to remove a delivery service from a user.
type UserDeliveryServiceDeleteResponse struct {
	Alerts
}

// UserPasswordResetRequest can hold  Traffic Ops API request to reset a user's password.
type UserPasswordResetRequest struct {
	Email rfc.EmailAddress `json:"email"`
}

// UserRegistrationRequest is the request submitted by operators when they want to register a new
// user.
type UserRegistrationRequest struct {
	Email rfc.EmailAddress `json:"email"`
	// Role - despite being named "Role" - is actually merely the *ID* of a Role to give the new user.
	Role     uint `json:"role"`
	TenantID uint `json:"tenantId"`
}

// Validate implements the
// github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api.ParseValidator interface.
func (urr *UserRegistrationRequest) Validate(tx *sql.Tx) error {
	var errs = []error{}
	if urr.Role == 0 {
		errs = append(errs, errors.New("role: required and cannot be zero."))
	}

	if urr.TenantID == 0 {
		errs = append(errs, errors.New("tenantId: required and cannot be zero."))
	}

	// This can only happen if an email isn't present in the request; the JSON parse handles actually
	// invalid email addresses.
	if urr.Email.Address.Address == "" {
		errs = append(errs, errors.New("email: required"))
	}

	return util.JoinErrs(errs)
}
