package api

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

const PathParamsKey = "pathParams"
const DBKey = "db"
const ConfigKey = "cfg"

type KeyFieldInfo struct {
	Field string
	Func  func(string) (interface{}, error)
}

func GetIntKey(s string) (interface{}, error) {
	if strings.HasSuffix(s, ".json") {
		s = s[:len(s)-len(".json")]
	}
	return strconv.Atoi(s)
}

func GetStringKey(s string) (interface{}, error) {
	return s, nil
}

func GetPathParams(ctx context.Context) (map[string]string, error) {
	val := ctx.Value(PathParamsKey)
	if val != nil {
		switch v := val.(type) {
		case map[string]string:
			return v, nil
		default:
			return nil, fmt.Errorf("path parameters found with bad type: %T", v)
		}
	}
	return nil, errors.New("no PathParams found in Context")
}

func IsInt(s string) error {
	_, err := strconv.Atoi(s)
	if err != nil {
		err = errors.New("cannot parse to integer")
	}
	return err
}

func IsBool(s string) error {
	_, err := strconv.ParseBool(s)
	if err != nil {
		err = errors.New("cannot parse to boolean")
	}
	return err
}

func GetCombinedParams(r *http.Request) (map[string]string, error) {
	combinedParams := make(map[string]string)
	q := r.URL.Query()
	for k, v := range q {
		combinedParams[k] = v[0] //we take the first value and do not support multiple keys in query parameters
	}

	ctx := r.Context()
	pathParams, err := GetPathParams(ctx)
	if err != nil {
		return combinedParams, fmt.Errorf("no path parameters: %s", err)
	}
	//path parameters will overwrite query parameters
	for k, v := range pathParams {
		combinedParams[k] = v
	}

	return combinedParams, nil
}

// decodeAndValidateRequestBody decodes and validates a pointer to a struct implementing the Validator interface
func decodeAndValidateRequestBody(r *http.Request, v Validator) error {
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return v.Validate()
}

// ReadHandler creates a handler function from the pointer to a struct implementing the Reader interface
//      this handler retrieves the user from the context
//      combines the path and query parameters
//      produces the proper status code based on the error code returned
//      marshals the structs returned into the proper response json
func ReadHandler(typeFactory CRUDFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//create error function with ResponseWriter and Request
		handleErrs := tc.GetHandleErrorsFunc(w, r)

		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		// Load the PathParams into the query parameters for pass through
		params, err := GetCombinedParams(r)
		if err != nil {
			log.Errorf("unable to get parameters from request: %s", err)
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		reader := typeFactory(inf)

		results, errs, errType := reader.Read(params)
		if len(errs) > 0 {
			tc.HandleErrorsWithType(errs, errType, handleErrs)
			return
		}
		resp := struct {
			Response []interface{} `json:"response"`
		}{results}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", respBts)
	}
}

// ReadOnlyHandler creates a handler function from the pointer to a struct implementing the Reader interface
//      this handler retrieves the user from the context
//      combines the path and query parameters
//      produces the proper status code based on the error code returned
//      marshals the structs returned into the proper response json
func ReadOnlyHandler(typeFactory func(reqInfo *APIInfo) Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//create error function with ResponseWriter and Request
		handleErrs := tc.GetHandleErrorsFunc(w, r)

		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		// Load the PathParams into the query parameters for pass through
		params, err := GetCombinedParams(r)
		if err != nil {
			log.Errorf("unable to get parameters from request: %s", err)
			handleErrs(http.StatusInternalServerError, err)
		}

		reader := typeFactory(inf)

		results, errs, errType := reader.Read(params)
		if len(errs) > 0 {
			tc.HandleErrorsWithType(errs, errType, handleErrs)
			return
		}
		resp := struct {
			Response []interface{} `json:"response"`
		}{results}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", respBts)
	}
}

// UpdateHandler creates a handler function from the pointer to a struct implementing the Updater interface
//   this generic handler encapsulates the logic for handling:
//   *fetching the id from the path parameter
//   *current user
//   *decoding and validating the struct
//   *change log entry
//   *forming and writing the body over the wire
func UpdateHandler(typeFactory CRUDFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//create error function with ResponseWriter and Request
		handleErrs := tc.GetHandleErrorsFunc(w, r)

		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		//decode the body and validate the request struct
		u := typeFactory(inf)
		//collect path parameters and user from context
		ctx := r.Context()
		params, err := GetCombinedParams(r)
		if err != nil {
			log.Errorf("received error trying to get path parameters: %s", err)
			handleErrs(http.StatusInternalServerError, err)
			return
		}
		user, err := auth.GetCurrentUser(ctx)
		if err != nil {
			log.Errorf("unable to retrieve current user from context: %s", err)
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		//decode the body and validate the request struct
		err = decodeAndValidateRequestBody(r, u)
		if err != nil {
			handleErrs(http.StatusBadRequest, err)
			return
		}

		keyFields := u.GetKeyFieldsInfo() //expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a {}interface of the right type. in most that will be [{Field:"id",Func: func(s string)({}interface,error){return strconv.Atoi(s)}}]
		keys, ok := u.GetKeys()           // a map of keyField to keyValue where keyValue is an {}interface
		if !ok {
			log.Errorf("unable to parse keys from request: %++v", u)
			handleErrs(http.StatusBadRequest, errors.New("unable to parse required keys from request body"))
			return // TODO verify?
		}
		for _, kf := range keyFields {
			paramKey := params[kf.Field]
			if paramKey == "" {
				log.Errorf("missing key: %s", kf.Field)
				handleErrs(http.StatusBadRequest, errors.New("missing key: "+kf.Field))
				return
			}

			paramValue, err := kf.Func(paramKey)
			if err != nil {
				log.Errorf("failed to parse key %s: %s", kf.Field, err)
				handleErrs(http.StatusBadRequest, errors.New("failed to parse key: "+kf.Field))
				return
			}

			if paramValue != "" {
				// if key's value provided in params,  overwrite it and ignore that provided in JSON
				keys[kf.Field] = paramValue
				u.SetKeys(keys)
				continue
			}

			if paramValue != keys[kf.Field] {
				handleErrs(http.StatusBadRequest, errors.New("key in body does not match key in params"))
				return
			}
		}

		// if the object has tenancy enabled, check that user is able to access the tenant
		if t, ok := u.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(user)
			if err != nil {
				handleErrs(http.StatusBadRequest, err)
				return
			}
			if !authorized {
				handleErrs(http.StatusForbidden, errors.New("not authorized on this tenant"))
				return
			}
		}

		//run the update and handle any error
		err, errType := u.Update()
		if err != nil {
			tc.HandleErrorsWithType([]error{err}, errType, handleErrs)
			return
		}
		//auditing here
		if err := CreateChangeLog(ApiChange, Updated, u, inf.User, inf.Tx); err != nil {
			HandleErr(w, r, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
			return
		}
		*inf.CommitTx = true
		//form response to send across the wire
		resp := struct {
			Response interface{} `json:"response"`
			tc.Alerts
		}{u, tc.CreateAlerts(tc.SuccessLevel, u.GetType()+" was updated.")}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(tc.ContentType, tc.ApplicationJson)
		fmt.Fprintf(w, "%s", respBts)
	}
}

// DeleteHandler creates a handler function from the pointer to a struct implementing the Deleter interface
//   this generic handler encapsulates the logic for handling:
//   *fetching the id from the path parameter
//   *current user
//   *change log entry
//   *forming and writing the body over the wire
func DeleteHandler(typeFactory CRUDFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)

		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		d := typeFactory(inf)

		params, err := GetCombinedParams(r)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		keyFields := d.GetKeyFieldsInfo() // expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a interface{} of the right type. in most that will be [{Field:"id",Func: func(s string)(interface{},error){return strconv.Atoi(s)}}]
		keys := make(map[string]interface{})
		for _, kf := range keyFields {
			paramKey := params[kf.Field]
			if paramKey == "" {
				log.Errorf("missing key: %s", kf.Field)
				handleErrs(http.StatusBadRequest, errors.New("missing key: "+kf.Field))
				return
			}

			paramValue, err := kf.Func(paramKey)
			if err != nil {
				log.Errorf("failed to parse key %s: %s", kf.Field, err)
				handleErrs(http.StatusBadRequest, errors.New("failed to parse key: "+kf.Field))
			}
			keys[kf.Field] = paramValue
		}
		d.SetKeys(keys) // if the type assertion of a key fails it will be should be set to the zero value of the type and the delete should fail (this means the code is not written properly no changes of user input should cause this.)

		// if the object has tenancy enabled, check that user is able to access the tenant
		if t, ok := d.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(inf.User)
			if err != nil {
				handleErrs(http.StatusBadRequest, err)
				return
			}
			if !authorized {
				handleErrs(http.StatusForbidden, errors.New("not authorized on this tenant"))
				return
			}
		}

		log.Debugf("calling delete on object: %++v", d) //should have id set now
		err, errType := d.Delete()
		if err != nil {
			log.Errorf("error deleting: %++v", err)
			tc.HandleErrorsWithType([]error{err}, errType, handleErrs)
			return
		}
		//audit here
		log.Debugf("changelog for delete on object")
		if err = CreateChangeLog(ApiChange, Deleted, d, inf.User, inf.Tx); err != nil {
			HandleErr(w, r, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
			return
		}
		*inf.CommitTx = true
		//
		resp := struct {
			tc.Alerts
		}{tc.CreateAlerts(tc.SuccessLevel, d.GetType()+" was deleted.")}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(tc.ContentType, tc.ApplicationJson)
		fmt.Fprintf(w, "%s", respBts)
	}
}

// CreateHandler creates a handler function from the pointer to a struct implementing the Creator interface
//   this generic handler encapsulates the logic for handling:
//   *current user
//   *decoding and validating the struct
//   *change log entry
//   *forming and writing the body over the wire
func CreateHandler(typeConstructor CRUDFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)

		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		i := typeConstructor(inf)
		//decode the body and validate the request struct
		err := decodeAndValidateRequestBody(r, i)

		if err != nil {
			handleErrs(http.StatusBadRequest, err)
			return
		}

		log.Debugf("%++v", i)
		//now we have a validated local object to insert

		// if the object has tenancy enabled, check that user is able to access the tenant
		if t, ok := i.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(inf.User)
			if err != nil {
				handleErrs(http.StatusBadRequest, err)
				return
			}
			if !authorized {
				handleErrs(http.StatusForbidden, errors.New("not authorized on this tenant"))
				return
			}
		}

		err, errType := i.Create()
		if err != nil {
			tc.HandleErrorsWithType([]error{err}, errType, handleErrs)
			return
		}

		if err = CreateChangeLog(ApiChange, Created, i, inf.User, inf.Tx); err != nil {
			HandleErr(w, r, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
			return
		}
		*inf.CommitTx = true

		resp := struct {
			Response interface{} `json:"response"`
			tc.Alerts
		}{i, tc.CreateAlerts(tc.SuccessLevel, i.GetType()+" was created.")}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(tc.ContentType, tc.ApplicationJson)
		fmt.Fprintf(w, "%s", respBts)
	}
}
