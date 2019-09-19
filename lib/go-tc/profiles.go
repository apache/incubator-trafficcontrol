package tc

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/lib/pq"
)

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

// ProfilesResponse ...
type ProfilesResponse struct {
	Response []Profile `json:"response"`
}

// A Single Profile Response for Update and Create to depict what changed
// swagger:response ProfileResponse
// in: body
type ProfileResponse struct {
	// in: body
	Response Profile `json:"response"`
}

// Profile ...
type Profile struct {
	ID              int                 `json:"id" db:"id"`
	LastUpdated     TimeNoMod           `json:"lastUpdated"`
	Name            string              `json:"name"`
	Parameter       string              `json:"param"`
	Description     string              `json:"description"`
	CDNName         string              `json:"cdnName"`
	CDNID           int                 `json:"cdn"`
	RoutingDisabled bool                `json:"routingDisabled"`
	Type            string              `json:"type"`
	Parameters      []ParameterNullable `json:"params,omitempty"`
}

type ProfileNullable struct {

	// Unique identifier for the Profile
	//
	ID *int `json:"id" db:"id"`

	// LastUpdated
	//
	LastUpdated *TimeNoMod `json:"lastUpdated" db:"last_updated"`

	// The Profile name
	//
	Name *string `json:"name" db:"name"`

	// The Profile Description
	//
	Description *string `json:"description" db:"description"`

	// The CDN name associated with the Profile
	//
	CDNName *string `json:"cdnName" db:"cdn_name"`

	// The CDN id associated with the Profile
	//
	CDNID *int `json:"cdn" db:"cdn"`

	// Enables
	//
	RoutingDisabled *bool `json:"routingDisabled" db:"routing_disabled"`

	// The Type name associated with the Profile
	//
	Type *string `json:"type" db:"type"`

	// Parameters associated to the profile
	//
	Parameters []ParameterNullable `json:"params,omitempty"`
}

type ProfileTrimmed struct {
	Name string `json:"name"`
}

// ProfileExportImportNullable is an object of the form used by Traffic Ops
// to represent exported and imported profiles.
type ProfileExportImportNullable struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CDNName     *string `json:"cdn"`
	Type        *string `json:"type"`
}

// ProfileExportResponse is an object of the form used by Traffic Ops
// to represent exported profile response
type ProfileExportResponse struct {
	// Parameters associated to the profile
	//
	Profile ProfileExportImportNullable `json:"profile"`

	// Parameters associated to the profile
	//
	Parameters []ProfileExportImportParameterNullable `json:"parameters"`
}

// ProfileImportRequest is an object of the form used by Traffic Ops
// to represent a request to import a profile
type ProfileImportRequest struct {
	// Parameters associated to the profile
	//
	Profile ProfileExportImportNullable `json:"profile"`

	// Parameters associated to the profile
	//
	Parameters []ProfileExportImportParameterNullable `json:"parameters"`
}

// ProfileImportResponse is an object of the form used by Traffic Ops
// to represent a response from importing a profile
type ProfileImportResponse struct {
	Response ProfileImportResponseObj `json:"response"`
	Alerts
}

type ProfileImportResponseObj struct {
	ProfileExportImportNullable
	ID *int64 `json:"id"`
}

// Validate validates an profile import request
func (profileImport *ProfileImportRequest) Validate(tx *sql.Tx) error {

	profile := profileImport.Profile

	// Profile fields are valid
	errs := tovalidate.ToErrors(validation.Errors{
		"name":        validation.Validate(profile.Name, validation.Required),
		"description": validation.Validate(profile.Description, validation.Required),
		"cdnName":     validation.Validate(profile.CDNName, validation.Required),
		"type":        validation.Validate(profile.Type, validation.Required),
	})

	// Validate CDN exist
	if profile.CDNName != nil {
		if ok, err := CDNExistsByName(*profile.CDNName, tx); err != nil {
			errs = append(errs, fmt.Errorf("checking cdn name %v existence: %v", *profile.CDNName, err.Error()))
		} else if !ok {
			errs = append(errs, fmt.Errorf("%v CDN does not exist", *profile.CDNName))
		}
	}

	// Validate profile does not already exist
	if profile.Name != nil {
		if ok, err := ProfileExistsByName(*profile.Name, tx); err != nil {
			errs = append(errs, fmt.Errorf("checking profile name %v existence: %v", *profile.Name, err.Error()))
		} else if ok {
			errs = append(errs, fmt.Errorf("A profile with the name \"%v\" already exists", *profile.Name))
		}
	}

	// Validate all parameters
	// export/import does not include secure flag
	// default value to not flag on validation
	secure := 1
	for i, pp := range profileImport.Parameters {
		if ppErrs := validateProfileParamPostFields(pp.ConfigFile, pp.Name, pp.Value, &secure); len(ppErrs) > 0 {
			for _, err := range ppErrs {
				errs = append(errs, errors.New("parameter "+strconv.Itoa(i)+": "+err.Error()))
			}
		}
	}

	if len(errs) > 0 {
		return util.JoinErrs(errs)
	}

	return nil
}

// ProfilesExistByIDs returns whether profiles exist for all the given ids, and any error.
// TODO move to helper package.
func ProfilesExistByIDs(ids []int64, tx *sql.Tx) (bool, error) {
	count := 0
	if err := tx.QueryRow(`SELECT count(*) from profile where id = ANY($1)`, pq.Array(ids)).Scan(&count); err != nil {
		return false, errors.New("querying profiles existence from id: " + err.Error())
	}
	return count == len(ids), nil
}

// ProfileExistsByID returns whether a profile with the given id exists, and any error.
// TODO move to helper package.
func ProfileExistsByID(id int64, tx *sql.Tx) (bool, error) {
	count := 0
	if err := tx.QueryRow(`SELECT count(*) from profile where id = $1`, id).Scan(&count); err != nil {
		return false, errors.New("querying profile existence from id: " + err.Error())
	}
	return count > 0, nil
}

// ProfileExistsByName returns whether a profile with the given name exists, and any error.
// TODO move to helper package.
func ProfileExistsByName(name string, tx *sql.Tx) (bool, error) {
	count := 0
	if err := tx.QueryRow(`SELECT count(*) from profile where name = $1`, name).Scan(&count); err != nil {
		return false, errors.New("querying profile existence from name: " + err.Error())
	}
	return count > 0, nil
}
