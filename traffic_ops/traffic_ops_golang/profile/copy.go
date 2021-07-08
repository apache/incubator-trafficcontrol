package profile

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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profileparameter"
)

type errorDetails struct {
	userErr error
	sysErr  error
	errCode int
}

// CopyProfileHandler creates a new profile and parameters from an existing profile.
func CopyProfileHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	p := tc.ProfileCopyResponse{
		Response: tc.ProfileCopy{
			ExistingName: inf.Params["existing_profile"],
			Name:         inf.Params["new_profile"],
		},
	}
	errs := copyProfile(inf, &p.Response)
	if errs.userErr != nil || errs.sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errs.errCode, errs.userErr, errs.sysErr)
		return
	}

	errs = copyParameters(inf, &p.Response)
	if errs.userErr != nil || errs.sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errs.errCode, errs.userErr, errs.sysErr)
		return
	}

	successMsg := fmt.Sprintf("created new profile [%s] from existing profile [%s]", p.Response.Name, p.Response.ExistingName)
	api.CreateChangeLogRawTx(api.ApiChange, successMsg, inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, successMsg, p.Response)
}

func copyProfile(inf *api.APIInfo, p *tc.ProfileCopy) errorDetails {
	if inf == nil || p == nil {
		return errorDetails{
			sysErr:  errors.New("copyProfile received nil APIInfo or ProfileCopy reference"),
			errCode: http.StatusInternalServerError,
		}
	}
	if strings.Contains(p.Name, " ") {
		return errorDetails{
			userErr: errors.New("new Profile name cannot contain spaces"),
			errCode: http.StatusBadRequest,
		}
	}
	// check if the newProfile already exists
	ok, err := tc.ProfileExistsByName(p.Name, inf.Tx.Tx)
	if ok {
		return errorDetails{
			userErr: fmt.Errorf("profile with name %s already exists", p.Name),
			errCode: http.StatusBadRequest,
		}
	}
	if err != nil {
		return errorDetails{
			sysErr:  err,
			errCode: http.StatusInternalServerError,
		}
	}

	// use existing CRUD helpers to get the existing profile
	inf.Params = map[string]string{
		"name": p.ExistingName,
	}
	toProfile := &TOProfile{
		api.APIInfoImpl{
			ReqInfo: inf,
		},
		tc.ProfileNullable{},
	}

	profiles, userErr, sysErr, errCode, _ := toProfile.Read(nil, false)
	if userErr != nil || sysErr != nil {
		return errorDetails{
			userErr: userErr,
			sysErr:  sysErr,
			errCode: errCode,
		}
	}

	if len(profiles) == 0 {
		return errorDetails{
			userErr: fmt.Errorf("profile with name %s does not exist", p.ExistingName),
			errCode: http.StatusNotFound,
		}
	} else if len(profiles) > 1 {
		return errorDetails{
			sysErr:  fmt.Errorf("multiple profiles with name %s returned", p.ExistingName),
			errCode: http.StatusInternalServerError,
		}
	}

	cdnName, err := dbhelpers.GetCDNNameFromProfileName(inf.Tx.Tx, p.ExistingName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errorDetails{
				userErr: errors.New("no cdn for the given profile"),
				sysErr:  nil,
				errCode: http.StatusBadRequest,
			}
		}
		return errorDetails{
			userErr: nil,
			sysErr:  err,
			errCode: http.StatusInternalServerError,
		}
	}
	userErr, sysErr, errCode = dbhelpers.CheckIfCurrentUserCanModifyCDN(inf.Tx.Tx, string(cdnName), inf.User.UserName)
	if userErr != nil || sysErr != nil {
		return errorDetails{
			userErr: userErr,
			sysErr:  sysErr,
			errCode: errCode,
		}
	}
	// use existing CRUD helpers to create the new profile
	toProfile.ProfileNullable = profiles[0].(tc.ProfileNullable)
	toProfile.ProfileNullable.Name = &p.Name
	userErr, sysErr, errCode = api.GenericCreate(toProfile)
	if userErr != nil || sysErr != nil {
		return errorDetails{
			userErr: userErr,
			sysErr:  sysErr,
			errCode: errCode,
		}
	}

	p.ExistingID = *profiles[0].(tc.ProfileNullable).ID
	p.ID = *toProfile.ProfileNullable.ID
	p.Description = *toProfile.ProfileNullable.Description
	log.Infof("created new profile [%s] from existing profile [%s]", p.Name, p.ExistingName)
	return errorDetails{}
}

func copyParameters(inf *api.APIInfo, p *tc.ProfileCopy) errorDetails {
	// use existing ProfileParameter CRUD helpers to find parameters for the existing profile
	inf.Params = map[string]string{
		"profileId": fmt.Sprintf("%d", p.ExistingID),
	}

	toParam := &profileparameter.TOProfileParameter{
		APIInfoImpl: api.APIInfoImpl{
			ReqInfo: inf,
		},
	}

	parameters, userErr, sysErr, errCode, _ := toParam.Read(nil, false)
	if userErr != nil || sysErr != nil {
		return errorDetails{
			userErr: userErr,
			sysErr:  sysErr,
			errCode: errCode,
		}
	}

	var newParams int
	for _, parameter := range parameters {
		param := parameter.(*tc.ProfileParametersNullable)

		// Use existing ProfileParameter CRUD helpers to associate
		// parameters to new profile.
		toParam.ProfileParameterNullable.ProfileID = &p.ID
		toParam.ProfileParameterNullable.ParameterID = param.Parameter
		userErr, sysErr, errCode := toParam.Create()
		if userErr != nil || sysErr != nil {
			return errorDetails{
				userErr: userErr,
				sysErr:  sysErr,
				errCode: errCode,
			}
		}
		newParams++
	}

	log.Infof("profile [%s] was assigned to %d parameters", p.Name, newParams)
	return errorDetails{}
}
