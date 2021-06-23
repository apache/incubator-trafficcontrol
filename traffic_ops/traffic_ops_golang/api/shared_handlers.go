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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

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

func checkIfOptionsDeleter(obj interface{}, params map[string]string) (bool, error, error, int) {
	optionsDeleter, ok := obj.(OptionsDeleter)
	if !ok {
		return false, nil, nil, http.StatusOK
	}
	options := optionsDeleter.DeleteKeyOptions()
	for key := range options {
		if params[key] != "" {
			return true, nil, nil, http.StatusOK
		}
	}
	name := reflect.TypeOf(obj).Elem().Name()[2:]
	return false, errors.New("Refusing to delete all resources of type " + name), nil, http.StatusBadRequest
}

// SetLastModifiedHeader sets the Last-Modified header in case the "useIMS" is set to true in the config,
// and if there is an "If-Modified-Since" header in the incoming request
func SetLastModifiedHeader(r *http.Request, useIMS bool) bool {
	if r == nil {
		return false
	}
	if r.Header.Get(rfc.IfModifiedSince) != "" && useIMS {
		return true
	}
	return false
}

// ReadHandler creates a handler function from the pointer to a struct implementing the Reader interface
//      this handler retrieves the user from the context
//      combines the path and query parameters
//      produces the proper status code based on the error code returned
//      marshals the structs returned into the proper response json
func ReadHandler(reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		useIMS := false
		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		interfacePtr := reflect.ValueOf(reader)
		if interfacePtr.Kind() != reflect.Ptr {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("reflect: can only indirect from a pointer"))
			return
		}
		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Reader)
		obj.SetInfo(inf)

		cfg, err := GetConfig(r.Context())
		if err != nil {
			log.Warnf("Couldnt get the config %v", err)
		}
		if cfg != nil {
			useIMS = cfg.UseIMS
		}
		results, userErr, sysErr, errCode, maxTime := obj.Read(r.Header, useIMS)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		if maxTime != nil && SetLastModifiedHeader(r, useIMS) {
			date := maxTime.Format(rfc.LastModifiedFormat)
			w.Header().Add(rfc.LastModified, date)
		}
		w.WriteHeader(errCode)
		WriteResp(w, r, results)
	}
}

// DeprecatedReadHandler creates a net/http.HandlerFunc for the passed Reader object, and adds a deprecation
// notice, optionally with a passed alternative route suggestion.
func DeprecatedReadHandler(reader Reader, alternative *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := CreateDeprecationAlerts(alternative)

		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			userErr = LogErr(r, http.StatusInternalServerError, userErr, sysErr)
			alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
			WriteAlerts(w, r, errCode, alerts)
			return
		}

		interfacePtr := reflect.ValueOf(reader)
		if interfacePtr.Kind() != reflect.Ptr {
			userErr = LogErr(r, http.StatusInternalServerError, nil, errors.New(" reflect: can only indirect from a pointer"))
			alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
			WriteAlerts(w, r, errCode, alerts)
			return
		}

		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Reader)
		obj.SetInfo(inf)

		results, userErr, sysErr, errCode, _ := obj.Read(r.Header, false)
		if userErr != nil || sysErr != nil {
			userErr = LogErr(r, http.StatusInternalServerError, userErr, sysErr)
			alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
			WriteAlerts(w, r, errCode, alerts)
			return
		}
		WriteAlertsObj(w, r, http.StatusOK, alerts, results)
	}
}

// UpdateHandler creates a handler function from the pointer to a struct implementing the Updater interface
//   this generic handler encapsulates the logic for handling:
//   *fetching the id from the path parameter
//   *current user
//   *decoding and validating the struct
//   *change log entry
//   *forming and writing the body over the wire
func UpdateHandler(updater Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		interfacePtr := reflect.ValueOf(updater)
		if interfacePtr.Kind() != reflect.Ptr {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("reflect: can only indirect from a pointer"))
			return
		}
		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Updater)
		obj.SetInfo(inf)

		if err := decodeAndValidateRequestBody(r, obj); err != nil {
			HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
			return
		}

		keyFields := obj.GetKeyFieldsInfo() //expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a {}interface of the right type. in most that will be [{Field:"id",Func: func(s string)({}interface,error){return strconv.Atoi(s)}}]
		// ignoring ok value -- will be checked after param processing

		keys := make(map[string]interface{}) // a map of keyField to keyValue where keyValue is an {}interface
		for _, kf := range keyFields {
			paramKey := inf.Params[kf.Field]
			if paramKey == "" {
				HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("missing key: "+kf.Field), nil)
				return
			}

			paramValue, err := kf.Func(paramKey)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+kf.Field), nil)
				return
			}

			if paramValue != "" {
				// if key's value provided in params,  overwrite it and ignore that provided in JSON
				keys[kf.Field] = paramValue
			}
		}

		// check that all keys were properly filled in
		obj.SetKeys(keys)
		_, ok := obj.GetKeys()
		if !ok {
			HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("unable to parse required keys from request body"), nil)
			return // TODO verify?
		}

		// if the object has tenancy enabled, check that user is able to access the tenant
		if t, ok := obj.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(inf.User)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant authorized: "+err.Error()))
				return
			}
			if !authorized {
				HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}

		userErr, sysErr, errCode = obj.Update(r.Header)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}

		if err := CreateChangeLog(ApiChange, Updated, obj, inf.User, inf.Tx.Tx); err != nil {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
			return
		}
		alerts := tc.CreateAlerts(tc.SuccessLevel, obj.GetType()+" was updated.")
		if alertsObj, hasAlerts := obj.(AlertsResponse); hasAlerts {
			alerts.AddAlerts(alertsObj.GetAlerts())
		}
		WriteAlertsObj(w, r, http.StatusOK, alerts, obj)
	}
}

// DeleteHandler creates a handler function from the pointer to a struct implementing the Deleter interface
//   this generic handler encapsulates the logic for handling:
//   *fetching the id from the path parameter
//   *current user
//   *change log entry
//   *forming and writing the body over the wire
func DeleteHandler(deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		interfacePtr := reflect.ValueOf(deleter)
		if interfacePtr.Kind() != reflect.Ptr {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("reflect: can only indirect from a pointer"))
			return
		}
		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Deleter)
		obj.SetInfo(inf)

		isOptionsDeleter, userErr, sysErr, errCode := checkIfOptionsDeleter(obj, inf.Params)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		var (
			keys = make(map[string]interface{})
			err  error
		)
		if isOptionsDeleter {
			for key, info := range obj.(OptionsDeleter).DeleteKeyOptions() {
				paramKey := inf.Params[key]
				if paramKey == "" {
					continue
				}
				switch reflect.ValueOf(info.Checker) {
				case reflect.ValueOf(IsInt):
					if keys[key], err = GetIntKey(paramKey); err != nil {
						HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+key), nil)
						return
					}
				case reflect.ValueOf(IsBool):
					if keys[key], err = strconv.ParseBool(paramKey); err != nil {
						HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+key), nil)
						return
					}
				default:
					keys[key] = paramKey
				}
			}
		} else {
			keyFields := obj.GetKeyFieldsInfo() // expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a interface{} of the right type. in most that will be [{Field:"id",Func: func(s string)(interface{},error){return strconv.Atoi(s)}}]
			for _, kf := range keyFields {
				paramKey := inf.Params[kf.Field]
				if paramKey == "" {
					HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("missing key: "+kf.Field), nil)
					return
				}

				paramValue, err := kf.Func(paramKey)
				if err != nil {
					HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+kf.Field), nil)
					return
				}
				keys[kf.Field] = paramValue
			}
		}
		obj.SetKeys(keys) // if the type assertion of a key fails it will be should be set to the zero value of the type and the delete should fail (this means the code is not written properly no changes of user input should cause this.)

		if t, ok := obj.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(inf.User)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant authorized: "+err.Error()))
				return
			}
			if !authorized {
				HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
				return
			}
		}

		if isOptionsDeleter {
			obj := reflect.New(objectType).Interface().(OptionsDeleter)
			obj.SetInfo(inf)
			userErr, sysErr, errCode = obj.OptionsDelete()
		} else {
			userErr, sysErr, errCode = obj.Delete()
		}
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}

		log.Debugf("changelog for delete on object")
		if err := CreateChangeLog(ApiChange, Deleted, obj, inf.User, inf.Tx.Tx); err != nil {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("inserting changelog: "+err.Error()))
			return
		}
		WriteRespAlert(w, r, tc.SuccessLevel, obj.GetType()+" was deleted.")
	}
}

// DeprecatedDeleteHandler creates a handler function from the pointer to a struct implementing the Deleter interface with a optional deprecation notice
//   this generic handler encapsulates the logic for handling:
//   *fetching the id from the path parameter
//   *current user
//   *change log entry
//   *forming and writing the body over the wire
func DeprecatedDeleteHandler(deleter Deleter, alternative *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, alternative)
			return
		}
		defer inf.Close()

		interfacePtr := reflect.ValueOf(deleter)
		if interfacePtr.Kind() != reflect.Ptr {
			HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("reflect: can only indirect from a pointer"), alternative)
			return
		}
		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Deleter)
		obj.SetInfo(inf)

		isOptionsDeleter, userErr, sysErr, errCode := checkIfOptionsDeleter(obj, inf.Params)
		if userErr != nil || sysErr != nil {
			HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, alternative)
			return
		}
		var (
			keys = make(map[string]interface{})
			err  error
		)
		if isOptionsDeleter {
			for key, info := range obj.(OptionsDeleter).DeleteKeyOptions() {
				paramKey := inf.Params[key]
				if paramKey == "" {
					continue
				}
				switch reflect.ValueOf(info.Checker) {
				case reflect.ValueOf(IsInt):
					if keys[key], err = GetIntKey(paramKey); err != nil {
						HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("missing key: "+key), nil, alternative)
						return
					}
				case reflect.ValueOf(IsBool):
					if keys[key], err = strconv.ParseBool(paramKey); err != nil {
						HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+key), nil, alternative)
						return
					}
				default:
					keys[key] = paramKey
				}
			}
		} else {
			keyFields := obj.GetKeyFieldsInfo() // expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a interface{} of the right type. in most that will be [{Field:"id",Func: func(s string)(interface{},error){return strconv.Atoi(s)}}]
			for _, kf := range keyFields {
				paramKey := inf.Params[kf.Field]
				if paramKey == "" {
					HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("missing key: "+kf.Field), nil, alternative)
					return
				}

				paramValue, err := kf.Func(paramKey)
				if err != nil {
					HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("failed to parse key: "+kf.Field), nil, alternative)
					return
				}
				keys[kf.Field] = paramValue
			}
		}
		obj.SetKeys(keys) // if the type assertion of a key fails it will be should be set to the zero value of the type and the delete should fail (this means the code is not written properly no changes of user input should cause this.)

		if t, ok := obj.(Tenantable); ok {
			authorized, err := t.IsTenantAuthorized(inf.User)
			if err != nil {
				HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant authorized: "+err.Error()), alternative)
				return
			}
			if !authorized {
				HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil, alternative)
				return
			}
		}

		if isOptionsDeleter {
			obj := reflect.New(objectType).Interface().(OptionsDeleter)
			obj.SetInfo(inf)
			userErr, sysErr, errCode = obj.OptionsDelete()
		} else {
			userErr, sysErr, errCode = obj.Delete()
		}

		if userErr != nil || sysErr != nil {
			HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, alternative)
			return
		}

		log.Debugf("changelog for delete on object")
		if err := CreateChangeLog(ApiChange, Deleted, obj, inf.User, inf.Tx.Tx); err != nil {
			HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("inserting changelog: "+err.Error()), alternative)
			return
		}
		alerts := CreateDeprecationAlerts(alternative)
		alerts.AddNewAlert(tc.SuccessLevel, obj.GetType()+" was deleted.")

		WriteAlerts(w, r, http.StatusOK, alerts)
	}
}

// CreateHandler creates a handler function from the pointer to a struct implementing the Creator interface
//   this generic handler encapsulates the logic for handling:
//   *current user
//   *decoding and validating the struct
//   *change log entry
//   *forming and writing the body over the wire
func CreateHandler(creator Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(w, r, nil, nil)
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		defer inf.Close()

		interfacePtr := reflect.ValueOf(creator)
		if interfacePtr.Kind() != reflect.Ptr {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("reflect: can only indirect from a pointer"))
			return
		}
		objectType := reflect.Indirect(interfacePtr).Type()
		obj := reflect.New(objectType).Interface().(Creator)
		obj.SetInfo(inf)

		if c, ok := obj.(MultipleCreator); ok && c.AllowMultipleCreates() {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
				return
			}

			objSlice, err := parseMultipleCreates(data, objectType, inf)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
				return
			}

			for _, objElemInt := range objSlice {
				objElem := reflect.ValueOf(objElemInt).Interface().(Creator)

				err = objElem.Validate()
				if err != nil {
					HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
					return
				}

				if t, ok := objElem.(Tenantable); ok {
					authorized, err := t.IsTenantAuthorized(inf.User)
					if err != nil {
						HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant authorized: "+err.Error()))
						return
					}
					if !authorized {
						HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
						return
					}
				}

				userErr, sysErr, errCode = objElem.Create()
				if userErr != nil || sysErr != nil {
					HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
					return
				}

				if err = CreateChangeLog(ApiChange, Created, objElem, inf.User, inf.Tx.Tx); err != nil {
					HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
					return
				}
			}
			if len(objSlice) == 0 {
				WriteRespAlert(w, r, tc.SuccessLevel, "No objects were provided in request.")
				return
			}
			var (
				responseObj interface{}
				message     string
			)
			if len(objSlice) == 1 {
				responseObj = objSlice[0]
				message = objSlice[0].GetType() + " was created."
			} else {
				message = objSlice[0].GetType() + "s were created."
			}
			alerts := tc.CreateAlerts(tc.SuccessLevel, message)
			if _, hasAlerts := objSlice[0].(AlertsResponse); hasAlerts {
				for _, objElem := range objSlice {
					alerts.AddAlerts(objElem.(AlertsResponse).GetAlerts())
				}
			}
			WriteAlertsObj(w, r, http.StatusOK, alerts, responseObj)

		} else {
			err := decodeAndValidateRequestBody(r, obj)
			if err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
				return
			}

			if t, ok := obj.(Tenantable); ok {
				authorized, err := t.IsTenantAuthorized(inf.User)
				if err != nil {
					HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenant authorized: "+err.Error()))
					return
				}
				if !authorized {
					HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
					return
				}
			}

			userErr, sysErr, errCode = obj.Create()
			if userErr != nil || sysErr != nil {
				HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
				return
			}

			if err = CreateChangeLog(ApiChange, Created, obj, inf.User, inf.Tx.Tx); err != nil {
				HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
				return
			}
			alerts := tc.CreateAlerts(tc.SuccessLevel, obj.GetType()+" was created.")
			if alertsObj, hasAlerts := obj.(AlertsResponse); hasAlerts {
				alerts.AddAlerts(alertsObj.GetAlerts())
			}
			WriteAlertsObj(w, r, http.StatusOK, alerts, obj)
		}
	}
}

func parseMultipleCreates(data []byte, desiredType reflect.Type, inf *Info) ([]Creator, error) {
	buf := ioutil.NopCloser(bytes.NewReader(data))

	var genericInt interface{}
	err := json.NewDecoder(buf).Decode(&genericInt)
	if err != nil {
		return nil, err
	}

	var creatorSlice []Creator

	_, ok := genericInt.([]interface{})
	var parseErr error = nil
	if !ok {
		singleCreator := reflect.New(desiredType).Interface().(Creator)
		singleCreator.SetInfo(inf)
		parseErr = json.Unmarshal(data, &singleCreator)
		creatorSlice = append(creatorSlice, singleCreator)
	} else {
		sliceOfT := reflect.SliceOf(desiredType)
		ptr := reflect.New(sliceOfT)
		parseErr = json.Unmarshal(data, ptr.Interface())

		for i := 0; i < reflect.Indirect(ptr).Len(); i++ {
			singleCreator := reflect.Indirect(ptr).Index(i).Addr().Interface().(Creator)
			singleCreator.SetInfo(inf)
			creatorSlice = append(creatorSlice, singleCreator)
		}
	}
	if parseErr != nil {
		return nil, parseErr
	}

	return creatorSlice, nil
}
