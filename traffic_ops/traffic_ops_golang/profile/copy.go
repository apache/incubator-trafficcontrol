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
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profileparameter"
)

// CopyProfileHandler creates a new profile and parameters from an existing profile.
func CopyProfileHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	p := tc.ProfileCopyResponse{
		Response: tc.ProfileCopy{
			ExistingName: inf.Params["existing_profile"],
			Name:         inf.Params["new_profile"],
		},
	}

	errs = copyProfile(inf, &p.Response)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	errs = copyParameters(inf, &p.Response)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	successMsg := fmt.Sprintf("created new profile [%s] from existing profile [%s]", p.Response.Name, p.Response.ExistingName)
	api.CreateChangeLogRawTx(api.ApiChange, successMsg, inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, successMsg, p.Response)
}

func copyProfile(inf *api.APIInfo, p *tc.ProfileCopy) apierrors.Errors {
	// check if the newProfile already exists
	ok, err := tc.ProfileExistsByName(p.Name, inf.Tx.Tx)
	if ok {
		return apierrors.Errors{
			UserError: fmt.Errorf("profile with name %s already exists", p.Name),
			Code:      http.StatusBadRequest,
		}
	}
	if err != nil {
		return apierrors.Errors{
			SystemError: err,
			Code:        http.StatusInternalServerError,
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

	profiles, errs, _ := toProfile.Read(nil, false)
	if errs.Occurred() {
		return errs
	}

	if len(profiles) == 0 {
		return apierrors.Errors{
			UserError: fmt.Errorf("profile with name %s does not exist", p.ExistingName),
			Code:      http.StatusNotFound,
		}
	} else if len(profiles) > 1 {
		return apierrors.Errors{
			SystemError: fmt.Errorf("multiple profiles with name %s returned", p.ExistingName),
			Code:        http.StatusInternalServerError,
		}
	}

	// use existing CRUD helpers to create the new profile
	toProfile.ProfileNullable = profiles[0].(tc.ProfileNullable)
	toProfile.ProfileNullable.Name = &p.Name
	errs = api.GenericCreate(toProfile)
	if errs.Occurred() {
		return errs
	}

	p.ExistingID = *profiles[0].(tc.ProfileNullable).ID
	p.ID = *toProfile.ProfileNullable.ID
	p.Description = *toProfile.ProfileNullable.Description
	log.Infof("created new profile [%s] from existing profile [%s]", p.Name, p.ExistingName)
	return apierrors.New()
}

func copyParameters(inf *api.APIInfo, p *tc.ProfileCopy) apierrors.Errors {
	// use existing ProfileParameter CRUD helpers to find parameters for the existing profile
	inf.Params = map[string]string{
		"profileId": fmt.Sprintf("%d", p.ExistingID),
	}

	toParam := &profileparameter.TOProfileParameter{
		api.APIInfoImpl{
			ReqInfo: inf,
		},
		tc.ProfileParameterNullable{},
	}

	parameters, errs, _ := toParam.Read(nil, false)
	if errs.Occurred() {
		return errs
	}

	var newParams int
	for _, parameter := range parameters {
		param := parameter.(*tc.ProfileParametersNullable)

		// Use existing ProfileParameter CRUD helpers to associate
		// parameters to new profile.
		toParam.ProfileParameterNullable.ProfileID = &p.ID
		toParam.ProfileParameterNullable.ParameterID = param.Parameter
		errs := toParam.Create()
		if errs.Occurred() {
			return errs
		}
		newParams++
	}

	log.Infof("profile [%s] was assigned to %d parameters", p.Name, newParams)
	return apierrors.New()
}
