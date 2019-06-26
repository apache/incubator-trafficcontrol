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
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
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

// copy interface{} from src to dst by marshalling and unmarshalling into json
// This should allow us to do the following:
//
// func (obj GenericCreator) GenericCreate() {
//
//		...
//
//		oddInterfaces, usrErr, sysErr, errCode := Read()
//		ConvertFrom(obj, oddInterfaces[0])
// }
//
// Basically, we are given an interface we know nothing about and we want it to represent
// our object fully. We must use our `obj` handle, since it gets used further up the
// function call stack. I guess we technically know what the interface is, but we can't
// match it generically to the type we want easily.
//
func ConvertFrom(dst interface{}, src interface{}) {
	blob, err := json.Marshal(src)
	if err != nil {
		log.Errorf("converting interface: %v", err)
	}
	err = json.Unmarshal(blob, dst)
	if err != nil {
		log.Errorf("converting interface: %v", err)
	}
}

// ReadHandler creates a handler function from the pointer to a struct implementing the Reader interface
//      this handler retrieves the user from the context
//      combines the path and query parameters
//      produces the proper status code based on the error code returned
//      marshals the structs returned into the proper response json
func ReadHandler(reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
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

		results, userErr, sysErr, errCode := obj.Read()
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		WriteResp(w, r, results)
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
		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
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

		userErr, sysErr, errCode = obj.Update()
		if userErr != nil || sysErr != nil {
			HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}

		if err := CreateChangeLog(ApiChange, Updated, obj, inf.User, inf.Tx.Tx); err != nil {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, tc.DBError, errors.New("inserting changelog: "+err.Error()))
			return
		}
		WriteRespAlertObj(w, r, tc.SuccessLevel, obj.GetType()+" was updated.", obj)
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
		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
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

		keyFields := obj.GetKeyFieldsInfo() // expecting a slice of the key fields info which is a struct with the field name and a function to convert a string into a interface{} of the right type. in most that will be [{Field:"id",Func: func(s string)(interface{},error){return strconv.Atoi(s)}}]
		keys := make(map[string]interface{})
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

		userErr, sysErr, errCode = obj.Delete()
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

// CreateHandler creates a handler function from the pointer to a struct implementing the Creator interface
//   this generic handler encapsulates the logic for handling:
//   *current user
//   *decoding and validating the struct
//   *change log entry
//   *forming and writing the body over the wire
func CreateHandler(creator Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, userErr, sysErr, errCode := NewInfo(r, nil, nil)
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
		WriteRespAlertObj(w, r, tc.SuccessLevel, obj.GetType()+" was created.", obj)
	}
}
