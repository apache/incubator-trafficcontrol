// Package api provides general purpose tools for implementing the Traffic Ops
// API.
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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tocookie"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault/backends/disabled"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type errorConstant string

// Error implements the error interface for errorConstants.
func (e errorConstant) Error() string {
	return string(e)
}

// NilRequestError is returned by Info methods when the request internally
// referred to by the Info cannot be found.
const NilRequestError = errorConstant("method called on Info with nil request")

// NilTransactionError is returned by Info methods when the transaction
// internally referred to by the Info cannot be found.
const NilTransactionError = errorConstant("method called on Info with nil transaction")

// ResourceModifiedError is a user-safe error that indicates a precondition
// failure.
const ResourceModifiedError = errorConstant("resource was modified since the time specified by the request headers")

// GoneErr is the error returned in Alerts when an endpoint has been removed,
// but the API version to which it belonged is still (partially) supported.
const GoneErr = errorConstant("this endpoint is no longer available; please consult documentation")

// Common context.Context value keys.
const (
	DBContextKey           = "db"
	ConfigContextKey       = "context"
	ReqIDContextKey        = "reqid"
	APIRespWrittenKey      = "respwritten"
	PathParamsKey          = "pathParams"
	TrafficVaultContextKey = "tv"
)

const influxServersQuery = `
SELECT (host_name||'.'||domain_name) as fqdn,
       tcp_port,
       https_port
FROM server
WHERE type in ( SELECT id
                FROM type
                WHERE name='INFLUXDB'
              )
AND status=(SELECT id FROM status WHERE name='ONLINE')
`

// Response is the basic structure of any given response from the Traffic Ops
// API.
type Response struct {
	tc.Alerts
	// Response is the actual object, the desired output of the client's
	// request.
	Response interface{} `json:"response"`
}

// ResponseWithSummary is an API response that includes the special "Summary"
// section as laid out in the API guidelines.
type ResponseWithSummary struct {
	Response
	Summary struct {
		Count uint64 `json:"count"`
	} `json:"summary"`
}

// GoneHandler is an http.Handler function that just writes a 410 Gone response
// back to the client, along with an error-level alert stating that the endpoint
// is no longer available.
func GoneHandler(w http.ResponseWriter, r *http.Request) {
	HandleErr(w, r, nil, http.StatusGone, GoneErr, nil)
}

// WriteResp takes any object, serializes it as JSON, and writes that to w.
// This is a helper for the common case; not using this in unusual cases is
// perfectly acceptable.
func WriteResp(w http.ResponseWriter, r *http.Request, v interface{}) {
	resp := Response{Response: v}
	WriteRespRaw(w, r, resp)
}

// WriteRespRaw acts like WriteResp, but doesn't wrap the object in a `{"response":` object. This should be used to respond with endpoints which don't wrap their response in a "response" object.
func WriteRespRaw(w http.ResponseWriter, r *http.Request, v interface{}) {
	if respWritten(r) {
		log.Errorf("WriteRespRaw called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	bts, err := json.Marshal(v)
	if err != nil {
		err = fmt.Errorf("marshalling JSON (raw) for %T: %w", v, err)
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, err)
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.Write(append(bts, '\n'))
}

// WriteRespWithSummary writes a JSON-encoded representation of an arbitrary
// object to the provided writer, and cleans up the corresponding request
// object. It also provides a "summary" section to the response object that
// contains the given "count".
func WriteRespWithSummary(w http.ResponseWriter, r *http.Request, v interface{}, count uint64) {
	var resp ResponseWithSummary
	resp.Response.Response = v
	resp.Summary.Count = count

	WriteRespRaw(w, r, resp)
}

// WriteRespVals is like WriteResp, but also takes a map of root-level values to write. The API most commonly needs these for meta-parameters, like size, limit, and orderby.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
// TODO: get rid of/deprecate this; it violates API guidelines.
func WriteRespVals(w http.ResponseWriter, r *http.Request, v interface{}, vals map[string]interface{}) {
	if respWritten(r) {
		log.Errorf("WriteRespVals called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	vals["response"] = v
	respBts, err := json.Marshal(vals)
	if err != nil {
		err = fmt.Errorf("marshalling JSON for %T: %w", v, err)
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(append(respBts, '\n'))
}

// WriteIMSHitResp writes a response to 'w' for an IMS request "hit", using the
// passed time as the Last-Modified date.
func WriteIMSHitResp(w http.ResponseWriter, r *http.Request, t time.Time) {
	if respWritten(r) {
		log.Errorf("WriteIMSHitResp called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	w.Header().Add(rfc.LastModified, t.Format(rfc.LastModifiedFormat))
	w.WriteHeader(http.StatusNotModified)
}

// HandleErr handles an API error, rolling back the transaction, writing the given statusCode and userErr to the user, and logging the sysErr. If userErr is nil, the text of the HTTP statusCode is written.
//
// The tx may be nil, if there is no transaction. Passing a nil tx is strongly discouraged if a transaction exists, because it will result in copy-paste errors for the common Info use case.
//
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func HandleErr(w http.ResponseWriter, r *http.Request, tx *sql.Tx, statusCode int, userErr error, sysErr error) {
	if respWritten(r) {
		log.Errorf("HandleErr called after a write already occurred! Attempting to write the error anyway! Path %s", r.URL.Path)
		// Don't return, attempt to rollback and write the error anyway
	}
	setRespWritten(r)

	if tx != nil {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Errorln("rolling back transaction: " + err.Error())
		}
	}
	handleSimpleErr(w, r, statusCode, userErr, sysErr)
}

func HandleErrOptionalDeprecation(w http.ResponseWriter, r *http.Request, tx *sql.Tx, statusCode int, userErr error, sysErr error, deprecated bool, alternative *string) {
	if deprecated {
		HandleDeprecatedErr(w, r, tx, statusCode, userErr, sysErr, alternative)
	} else {
		HandleErr(w, r, tx, statusCode, userErr, sysErr)
	}
}

// HandleDeprecatedErr handles an API error, adding a deprecation alert, rolling back the transaction, writing the given statusCode and userErr to the user, and logging the sysErr. If userErr is nil, the text of the HTTP statusCode is written.
//
// The alternative may be nil if there is no alternative and the deprecation message will be selected appropriately.
//
// The tx may be nil, if there is no transaction. Passing a nil tx is strongly discouraged if a transaction exists, because it will result in copy-paste errors for the common Info use case.
//
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func HandleDeprecatedErr(w http.ResponseWriter, r *http.Request, tx *sql.Tx, statusCode int, userErr error, sysErr error, alternative *string) {
	if respWritten(r) {
		log.Errorf("HandleDeprecatedErr called after a write already occurred! Attempting to write the error anyway! Path %s", r.URL.Path)
		// Don't return, attempt to rollback and write the error anyway
	}

	if tx != nil {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Errorln("rolling back transaction: " + err.Error())
		}
	}

	alerts := CreateDeprecationAlerts(alternative)

	userErr = LogErr(r, statusCode, userErr, sysErr)
	alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
	WriteAlerts(w, r, statusCode, alerts)
}

// LogErr handles the logging of errors and setting up possibly nil errors without actually writing anything to a
// http.ResponseWriter, unlike handleSimpleErr. It returns the userErr which will be initialized to the
// http.StatusText of errCode if it was passed as nil - otherwise left alone.
func LogErr(r *http.Request, errCode int, userErr error, sysErr error) error {
	if sysErr != nil {
		log.Errorf(r.RemoteAddr + " " + sysErr.Error())
	}
	if userErr == nil {
		userErr = errors.New(http.StatusText(errCode))
	}
	log.Debugln(userErr.Error())
	*r = *r.WithContext(context.WithValue(r.Context(), tc.StatusKey, errCode))
	return userErr
}

// handleSimpleErr is a helper for HandleErr.
// This exists to prevent exposing HandleErr calls in this file with nil transactions, which might be copy-pasted creating bugs.
func handleSimpleErr(w http.ResponseWriter, r *http.Request, statusCode int, userErr error, sysErr error) {
	userErr = LogErr(r, statusCode, userErr, sysErr)

	respBts, err := json.Marshal(tc.CreateErrorAlerts(userErr))
	if err != nil {
		log.Errorln("marshalling error: " + err.Error())
		w.Write(append([]byte(http.StatusText(http.StatusInternalServerError)), '\n'))
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.Write(append(respBts, '\n'))
}

// RespWriter is a helper to allow a one-line response, for endpoints with a function that returns the object that needs to be written and an error.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func RespWriter(w http.ResponseWriter, r *http.Request, tx *sql.Tx) func(v interface{}, err error) {
	return func(v interface{}, err error) {
		if err != nil {
			HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
			return
		}
		WriteResp(w, r, v)
	}
}

// RespWriterVals is like RespWriter, but also takes a map of root-level values to write. The API most commonly needs these for meta-parameters, like size, limit, and orderby.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func RespWriterVals(w http.ResponseWriter, r *http.Request, tx *sql.Tx, vals map[string]interface{}) func(v interface{}, err error) {
	return func(v interface{}, err error) {
		if err != nil {
			HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
			return
		}
		WriteRespVals(w, r, v, vals)
	}
}

// WriteRespAlert creates an alert, serializes it as JSON, and writes that to w.
// This is a helper for the common case; not using this in unusual cases is
// perfectly acceptable.
func WriteRespAlert(w http.ResponseWriter, r *http.Request, level tc.AlertLevel, msg string) {
	if respWritten(r) {
		log.Errorf("WriteRespAlert called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	resp := struct{ tc.Alerts }{tc.CreateAlerts(level, msg)}
	respBts, err := json.Marshal(resp)
	if err != nil {
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, errors.New("marshalling JSON: "+err.Error()))
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.Write(append(respBts, '\n'))
}

// WriteRespAlertNotFound creates an alert indicating that the resource was not found and writes that to w.
func WriteRespAlertNotFound(w http.ResponseWriter, r *http.Request) {
	if respWritten(r) {
		log.Errorf("WriteRespAlert called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	resp := struct{ tc.Alerts }{tc.CreateAlerts(tc.ErrorLevel, "Resource not found.")}
	respBts, err := json.Marshal(resp)
	if err != nil {
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, errors.New("marshalling JSON: "+err.Error()))
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.WriteHeader(http.StatusNotFound)
	w.Write(append(respBts, '\n'))
}

// WriteRespAlertObj Writes the given alert, and the given response object.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func WriteRespAlertObj(w http.ResponseWriter, r *http.Request, level tc.AlertLevel, msg string, obj interface{}) {
	if respWritten(r) {
		log.Errorf("WriteRespAlertObj called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	resp := struct {
		tc.Alerts
		Response interface{} `json:"response"`
	}{
		Alerts:   tc.CreateAlerts(level, msg),
		Response: obj,
	}
	respBts, err := json.Marshal(resp)
	if err != nil {
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, errors.New("marshalling JSON: "+err.Error()))
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	_, _ = w.Write(append(respBts, '\n'))
}

func WriteAlerts(w http.ResponseWriter, r *http.Request, code int, alerts tc.Alerts) {
	if respWritten(r) {
		log.Errorf("WriteAlerts called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.WriteHeader(code)
	if alerts.HasAlerts() {
		resp, err := json.Marshal(alerts)
		if err != nil {
			handleSimpleErr(w, r, http.StatusInternalServerError, nil, fmt.Errorf("marshalling JSON: %v", err))
			return
		}
		_, _ = w.Write(append(resp, '\n'))
	}
}

func WriteAlertsObj(w http.ResponseWriter, r *http.Request, code int, alerts tc.Alerts, obj interface{}) {
	if !alerts.HasAlerts() {
		WriteResp(w, r, obj)
		w.WriteHeader(code)
		return
	}
	if respWritten(r) {
		log.Errorf("WriteAlertsObj called after a write already occurred! Not double-writing! Path %s", r.URL.Path)
		return
	}
	setRespWritten(r)

	resp := struct {
		tc.Alerts
		Response interface{} `json:"response"`
	}{
		Alerts:   alerts,
		Response: obj,
	}
	respBts, err := json.Marshal(resp)
	if err != nil {
		handleSimpleErr(w, r, http.StatusInternalServerError, nil, fmt.Errorf("marshalling JSON: %v", err))
		return
	}
	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.WriteHeader(code)
	w.Write(append(respBts, '\n'))
}

// IntParams parses integer parameters, and returns map of the given params, or an error if any integer param is not an integer. The intParams may be nil if no integer parameters are required. Note this does not check existence; if an integer paramter is required, it should be included in the requiredParams given to NewInfo.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func IntParams(params map[string]string, intParamNames []string) (map[string]int, error) {
	intParams := map[string]int{}
	for _, intParam := range intParamNames {
		valStr, ok := params[intParam]
		if !ok {
			continue
		}
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			return nil, errors.New("parameter '" + intParam + "'" + " not an integer")
		}
		intParams[intParam] = valInt
	}
	return intParams, nil
}

// ParamsHaveRequired checks that params have all the required parameters, and returns nil on success, or an error providing information on which params are missing.
func ParamsHaveRequired(params map[string]string, required []string) error {
	missing := []string{}
	for _, requiredParam := range required {
		if _, ok := params[requiredParam]; !ok {
			missing = append(missing, requiredParam)
		}
	}
	if len(missing) > 0 {
		return errors.New("missing required parameters: " + strings.Join(missing, ", "))
	}
	return nil
}

// StripParamJSON removes ".json" trailing any parameter value, and returns the modified params.
// This allows the API handlers to transparently accept /id.json routes, as allowed by the 1.x API.
func StripParamJSON(params map[string]string) map[string]string {
	for name, val := range params {
		if strings.HasSuffix(val, ".json") {
			params[name] = val[:len(val)-len(".json")]
		}
	}
	return params
}

// AllParams takes the request (in which the router has inserted context for path parameters), and an array of parameters required to be integers, and returns the map of combined parameters, and the map of int parameters; or a user or system error and the HTTP error code. The intParams may be nil if no integer parameters are required.
// This is a helper for the common case; not using this in unusual cases is perfectly acceptable.
func AllParams(req *http.Request, required []string, ints []string) (map[string]string, map[string]int, error, error, int) {
	params, err := GetCombinedParams(req)
	if err != nil {
		return nil, nil, nil, errors.New("getting combined URI parameters: " + err.Error()), http.StatusInternalServerError
	}
	params = StripParamJSON(params)
	if err := ParamsHaveRequired(params, required); err != nil {
		return nil, nil, errors.New("required parameters missing: " + err.Error()), nil, http.StatusBadRequest
	}
	intParams, err := IntParams(params, ints)
	if err != nil {
		return nil, nil, errors.New("getting integer parameters: " + err.Error()), nil, http.StatusBadRequest
	}
	return params, intParams, nil, nil, 0
}

// ParseValidator objects can make use of api.Parse to handle parsing and
// validating at the same time.
//
// TODO: Rework validation to be able to return system-level errors
type ParseValidator interface {
	Validate(tx *sql.Tx) error
}

// Parse decodes a JSON object from r into v, validating and sanitizing the
// input. Use this function instead of the json package when writing API
// endpoints to safely decode and validate PUT and POST requests.
//
// TODO: change to take data loaded from db, to remove sql from tc package.
func Parse(r io.Reader, tx *sql.Tx, v ParseValidator) error {
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return fmt.Errorf("decoding: %v", err)
	}
	if err := v.Validate(tx); err != nil {
		return fmt.Errorf("validating: %v", err)
	}
	return nil
}

// SendMail sends an email msg to the address identified by to. The msg parameter should be an
// RFC822-style email with headers first, a blank line, and then the message body. The lines of msg
// should be CRLF terminated. The msg headers should usually include fields such as "From", "To",
// "Subject", and "Cc". Sending "Bcc" messages is accomplished by including an email address in the
// to parameter but not including it in the msg headers.
// The cfg parameter is used to set things like the "From" field, as well as for connection
// and authentication with an external SMTP server.
// SendMail returns (in order) an HTTP status code, a user-friendly error, and an error fit for
// logging to system error logs. If either the user or system error is non-nil, the operation failed,
// and the HTTP status code indicates the type of failure.
func SendMail(to rfc.EmailAddress, msg []byte, cfg *config.Config) (int, error, error) {
	if !cfg.SMTP.Enabled {
		return http.StatusInternalServerError, nil, errors.New("SMTP is not enabled; mail cannot be sent")
	}
	var auth smtp.Auth
	if cfg.SMTP.User != "" {
		auth = LoginAuth("", cfg.SMTP.User, cfg.SMTP.Password, strings.Split(cfg.SMTP.Address, ":")[0])
	}
	err := smtp.SendMail(cfg.SMTP.Address, auth, cfg.ConfigTO.EmailFrom.Address.Address, []string{to.Address.Address}, msg)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("Failed to send email: %v", err)
	}
	return http.StatusOK, nil, nil
}

// getRequestedAPIVersion returns the requested API Version from the request if it exists, or returns "0.0" otherwise.
func getRequestedAPIVersion(path string) Version {
	v := Version{
		Major: 0,
		Minor: 0,
	}
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 || strings.ToLower(pathParts[1]) != "api" {
		return v // path doesn't start with `/api`, so it's not an api request.
	}
	if len(pathParts) < 3 {
		return v // path starts with `/api` but not `/api/{version}`, so it's an api request, and an unknown/nonexistent version.
	}
	version := pathParts[2]

	versionParts := strings.Split(version, ".")
	if len(versionParts) != 2 {
		return v // version part of the path isn't in 'major.minor' format.
	}
	majorVersion, err := strconv.ParseUint(versionParts[0], 10, 64)
	if err != nil {
		return v // version major part is not a number.
	}
	minorVersion, err := strconv.ParseUint(versionParts[1], 10, 64)
	if err != nil {
		return v // version minor part is not a number.
	}
	return Version{Major: majorVersion, Minor: minorVersion}
}

// GetDB returns the database from the context. This should very rarely be needed, rather `NewInfo` should always be used to get a transaction, except in extenuating circumstances.
func GetDB(ctx context.Context) (*sqlx.DB, error) {
	val := ctx.Value(DBContextKey)
	if val != nil {
		switch v := val.(type) {
		case *sqlx.DB:
			return v, nil
		default:
			return nil, fmt.Errorf("Tx found with bad type: %T", v)
		}
	}
	return nil, errors.New("No db found in Context")
}

func GetConfig(ctx context.Context) (*config.Config, error) {
	val := ctx.Value(ConfigContextKey)
	if val != nil {
		switch v := val.(type) {
		case *config.Config:
			return v, nil
		default:
			return nil, fmt.Errorf("Config found with bad type: %T", v)
		}
	}
	return nil, errors.New("No config found in Context")
}

func GetTrafficVault(ctx context.Context) (trafficvault.TrafficVault, error) {
	val := ctx.Value(TrafficVaultContextKey)
	if val != nil {
		switch v := val.(type) {
		case trafficvault.TrafficVault:
			return v, nil
		default:
			return nil, fmt.Errorf("TrafficVault found with bad type: %T", v)
		}
	}
	// this return should never be reached because a non-nil TrafficVault should always be included in the request context
	return &disabled.Disabled{}, errors.New("no Traffic Vault found in Context")
}

func getReqID(ctx context.Context) (uint64, error) {
	val := ctx.Value(ReqIDContextKey)
	if val != nil {
		switch v := val.(type) {
		case uint64:
			return v, nil
		default:
			return 0, fmt.Errorf("ReqID found with bad type: %T", v)
		}
	}
	return 0, errors.New("No ReqID found in Context")
}

// setRespWritten sets the APIRespWrittenKey key in the Context of the given Request.
// This is used to indicate that a response has been written with an API helper, and to prevent double-write errors.
// If an API helper which responds is called after another response helper was already called, all API helpers will log an error, and not write the second response, except HandleErr, which will write its error anyway, along with its status code.
func setRespWritten(r *http.Request) {
	*r = *r.WithContext(context.WithValue(r.Context(), APIRespWrittenKey, struct{}{}))
}

// respWritten gets the APIRespWrittenKey key, which indicates whether an API response helper was previously called.
// This is used to prevent double-write errors. See setRespWritten.
func respWritten(r *http.Request) bool {
	return r.Context().Value(APIRespWrittenKey) != nil
}

// small helper function to help with parsing below
func toCamelCase(str string) string {
	mutable := []byte(str)
	for i := 0; i < len(str); i++ {
		if mutable[i] == '_' && i+1 < len(str) {
			mutable[i+1] = strings.ToUpper(string(str[i+1]))[0]
		}
	}
	return strings.Replace(string(mutable[:]), "_", "", -1)
}

// parses pq errors for any trigger based conflicts
func parseTriggerConflicts(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`^(.*?)conflicts`)
	match := pattern.FindStringSubmatch(err.Message)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("%s", toCamelCase(match[0])), nil, http.StatusBadRequest
}

// parses pq errors for not null constraint
func parseNotNullConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`null value in column "(.+)" violates not-null constraint`)
	match := pattern.FindStringSubmatch(err.Message)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("%s is a required field", toCamelCase(match[1])), nil, http.StatusBadRequest
}

// parses pq errors for empty string check constraint
func parseEmptyConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`new row for relation "[^"]*" violates check constraint "(.*)_empty"`)
	match := pattern.FindStringSubmatch(err.Message)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("%s cannot be ", match[1]), nil, http.StatusBadRequest
}

// parses pq errors for violated foreign key constraints
func parseNotPresentFKConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`Key \(.+\)=\(.+\) is not present in table "(.+)"`)
	match := pattern.FindStringSubmatch(err.Detail)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("%s not found", match[1]), nil, http.StatusNotFound
}

// parses pq errors for uniqueness constraint violations
func parseUniqueConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`Key \((.+)\)=\((.+)\) already exists`)
	match := pattern.FindStringSubmatch(err.Detail)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("%v %s '%s' already exists.", err.Table, match[1], match[2]), nil, http.StatusBadRequest
}

// parses pq errors for database enum constraint violations
func parseEnumConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`invalid input value for enum (.+): \"(.+)\"`)
	match := pattern.FindStringSubmatch(err.Message)
	if match == nil {
		return nil, nil, http.StatusOK
	}
	return fmt.Errorf("invalid enum value %s for field %s.", match[2], match[1]), nil, http.StatusBadRequest
}

// parses pq errors for ON DELETE RESTRICT fk constraint violations
//
// Note: This method would also catch an ON UPDATE RESTRICT fk constraint,
// but only an error message appropriate for delete is returned. Currently,
// no API endpoint can trigger an ON UPDATE RESTRICT fk constraint since
// no API endpoint updates the primary key of any table.
//
// ATM I'm not sure if there is significance in restricting either of the table
// names that are captured in the regex to not contain any underscores.
// This function fixes issues like #3410. If an error message needs to be made
// for tables with underscores in particular, it should be made into an issue
// and this function should be updated then. At the moment, there are no documented
// issues for this case, so I won't include it.
//
// It may be helpful to look at constraints for api_capability, role_capability,
// and user_role for examples.
//
func parseRestrictFKConstraint(err *pq.Error) (error, error, int) {
	pattern := regexp.MustCompile(`update or delete on table "([a-z_]+)" violates foreign key constraint ".+" on table "([a-z_]+)"`)
	match := pattern.FindStringSubmatch(err.Message)
	if match == nil {
		return nil, nil, http.StatusOK
	}

	// small heuristic for grammar
	article := "a"
	switch match[2][0] {
	case 'a', 'e', 'i', 'o':
		article = "an"
	}
	return fmt.Errorf("cannot delete %s because it is being used by %s %s", match[1], article, match[2]), nil, http.StatusBadRequest
}

// ParseDBError parses pq errors for database constraint violations, and returns the (userErr, sysErr, httpCode) format expected by the API helpers.
func ParseDBError(ierr error) (error, error, int) {

	err, ok := ierr.(*pq.Error)
	if !ok {
		log.Errorf("a non-pq error was given")
		return nil, ierr, http.StatusInternalServerError
	}

	if usrErr, sysErr, errCode := parseNotPresentFKConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseUniqueConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseRestrictFKConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseNotNullConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseEmptyConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseEnumConstraint(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	if usrErr, sysErr, errCode := parseTriggerConflicts(err); errCode != http.StatusOK {
		return usrErr, sysErr, errCode
	}

	return nil, err, http.StatusInternalServerError
}

// GetUserFromReq returns the current user, any user error, any system error, and an error code to be returned if either error was not nil.
// This also uses the given ResponseWriter to refresh the cookie, if it was valid.
func GetUserFromReq(w http.ResponseWriter, r *http.Request, secret string) (auth.CurrentUser, error, error, int) {
	cookie, err := r.Cookie(tocookie.Name)
	if err != nil {
		return auth.CurrentUser{}, errors.New("Unauthorized, please log in."), errors.New("error getting cookie: " + err.Error()), http.StatusUnauthorized
	}

	if cookie == nil {
		return auth.CurrentUser{}, errors.New("Unauthorized, please log in."), nil, http.StatusUnauthorized
	}

	oldCookie, err := tocookie.Parse(secret, cookie.Value)
	if err != nil {
		return auth.CurrentUser{}, errors.New("Unauthorized, please log in."), errors.New("error parsing cookie: " + err.Error()), http.StatusUnauthorized
	}

	username := oldCookie.AuthData
	if username == "" {
		return auth.CurrentUser{}, errors.New("Unauthorized, please log in."), nil, http.StatusUnauthorized
	}
	db := (*sqlx.DB)(nil)
	val := r.Context().Value(DBContextKey)
	if val == nil {
		return auth.CurrentUser{}, nil, errors.New("request context db missing"), http.StatusInternalServerError
	}
	switch v := val.(type) {
	case *sqlx.DB:
		db = v
	default:
		return auth.CurrentUser{}, nil, fmt.Errorf("request context db unknown type %T", val), http.StatusInternalServerError
	}

	cfg, err := GetConfig(r.Context())
	if err != nil {
		return auth.CurrentUser{}, nil, errors.New("request context config missing"), http.StatusInternalServerError
	}

	user, userErr, sysErr, code := auth.GetCurrentUserFromDB(db, username, time.Duration(cfg.DBQueryTimeoutSeconds)*time.Second)
	if userErr != nil || sysErr != nil {
		return auth.CurrentUser{}, userErr, sysErr, code
	}

	duration := tocookie.DefaultDuration
	newCookie := tocookie.GetCookie(oldCookie.AuthData, duration, secret)
	http.SetCookie(w, newCookie)
	return user, nil, nil, http.StatusOK
}

func AddUserToReq(r *http.Request, u auth.CurrentUser) {
	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey, u)
	*r = *r.WithContext(ctx)
}

// SendEmailFromTemplate allows a user to input an html template to format an email.  It parses the template and creates a message before calling the SendMail method.
// SendEmailFromTemplate returns (in order) an HTTP status code, a user-friendly error, and an error fit for
// logging to system error logs. If either the user or system error is non-nil, the operation failed,
// and the HTTP status code indicates the type of failure.
func SendEmailFromTemplate(config config.Config, header string, data interface{}, templateFile string, toEmail string) (int, error, error) {
	email := rfc.EmailAddress{
		Address: mail.Address{Name: "", Address: toEmail},
	}

	msgBodyBuffer, err := parseTemplate(templateFile, data)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	msg := append([]byte(header), msgBodyBuffer.Bytes()...)

	return SendMail(email, msg, &config)

}

func parseTemplate(templateFileName string, data interface{}) (*bytes.Buffer, error) {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, data)

	return buf, err
}

type loginAuth struct {
	identity, username, password, host string
}

func LoginAuth(identity, username, password, host string) smtp.Auth {
	return &loginAuth{identity, username, password, host}
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "LOGIN", resp, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	command := string(fromServer)
	command = strings.TrimSpace(command)
	command = strings.TrimSuffix(command, ":")
	command = strings.ToLower(command)

	if more {
		if command == "username" {
			return []byte(a.username), nil
		} else if command == "password" {
			return []byte(a.password), nil
		} else {
			return nil, fmt.Errorf("unexpected server challenge: %s", command)
		}
	}
	return nil, nil
}

// CreateDeprecationAlerts creates a deprecation notice with an optional alternative route suggestion.
func CreateDeprecationAlerts(alternative *string) tc.Alerts {
	if alternative != nil {
		return tc.CreateAlerts(tc.WarnLevel, fmt.Sprintf("This endpoint is deprecated, please use %s instead", *alternative))
	} else {
		return tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, and will be removed in the future")
	}
}

// CheckIfUnModified checks to see if the resource was modified since the "If-Unmodified-Since" header value in the request.
// In case it was, the 412 error code is returned. If some other error was encountered while checking, the appropriate error code along with
// error details is returned. If the resource was not modified since the specified time, the UPDATE proceeds in the normal fashion.
func CheckIfUnModified(h http.Header, tx *sqlx.Tx, ID int, tableName string) (error, error, int) {
	_, okIUS := h[rfc.IfUnmodifiedSince]
	_, okIM := h[rfc.IfMatch]
	if !okIUS && !okIM {
		return nil, nil, http.StatusOK
	}
	existingLastUpdated, found, err := GetLastUpdated(tx, ID, tableName)
	if err == nil && found == false {
		return errors.New("no " + tableName + " found with this id"), nil, http.StatusNotFound
	}
	if err != nil {
		return nil, errors.New("error getting last updated: " + err.Error()), http.StatusInternalServerError
	}
	if !IsUnmodified(h, *existingLastUpdated) {
		return ResourceModifiedError, nil, http.StatusPreconditionFailed
	}
	return nil, nil, http.StatusOK
}

// CheckIfUnModifiedByName checks to see if the resource was modified since the "If-Unmodified-Since" header value in the request.
// In case it was, the 412 error code is returned. If some other error was encountered while checking, the appropriate error code along with
// error details is returned. If the resource was not modified since the specified time, the UPDATE proceeds in the normal fashion.
func CheckIfUnModifiedByName(h http.Header, tx *sqlx.Tx, name string, tableName string) (error, error, int) {
	_, okIUS := h[rfc.IfUnmodifiedSince]
	_, okIM := h[rfc.IfMatch]
	if !okIUS && !okIM {
		return nil, nil, http.StatusOK
	}
	existingLastUpdated, found, err := GetLastUpdatedByName(tx, name, tableName)
	if err == nil && found == false {
		return errors.New("no " + tableName + " found with this name"), nil, http.StatusNotFound
	}
	if err != nil {
		return nil, errors.New(tableName + "update: querying: " + err.Error()), http.StatusInternalServerError
	}
	if !IsUnmodified(h, *existingLastUpdated) {
		return ResourceModifiedError, nil, http.StatusPreconditionFailed
	}
	return nil, nil, http.StatusOK
}

// GetLastUpdated checks for the resource by ID in the database, and returns its last_updated timestamp, if available.
func GetLastUpdated(tx *sqlx.Tx, ID int, tableName string) (*time.Time, bool, error) {
	return getLastUpdatedByIdentifier(tx, "id", ID, tableName)
}

// GetLastUpdatedByName checks for the resource by name in the database, and returns its last_updated timestamp, if available.
func GetLastUpdatedByName(tx *sqlx.Tx, name string, tableName string) (*time.Time, bool, error) {
	return getLastUpdatedByIdentifier(tx, "name", name, tableName)
}

func getLastUpdatedByIdentifier(tx *sqlx.Tx, IDColumn string, IDValue interface{}, tableName string) (*time.Time, bool, error) {
	lastUpdated := time.Time{}
	found := false
	rows, err := tx.Query(fmt.Sprintf(`select last_updated from %s where %s = $1`, pq.QuoteIdentifier(tableName), pq.QuoteIdentifier(IDColumn)), IDValue)
	if err != nil {
		return nil, found, errors.New("querying last_updated: " + err.Error())
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, found, nil
	}
	found = true
	if err := rows.Scan(&lastUpdated); err != nil {
		return nil, found, errors.New("scanning last_updated: " + err.Error())
	}
	return &lastUpdated, found, nil
}

// IsUnmodified returns a boolean, saying whether or not the resource in question was modified since the time specified in the headers.
func IsUnmodified(h http.Header, lastUpdated time.Time) bool {
	unmodifiedTime, ok := rfc.GetUnmodifiedTime(h)
	if !ok {
		return true // no IUS/IM header: unmodified, proceed with normal update
	}
	return !lastUpdated.After(unmodifiedTime)
}

// FormatLastModified trims the time string and formats it according to RFC1123.
func FormatLastModified(t time.Time) string {
	return rfc.FormatHTTPDate(t.Truncate(time.Second).Add(time.Second))
}

// AddLastModifiedHdr adds the "last modified" header to the response.
func AddLastModifiedHdr(w http.ResponseWriter, t time.Time) {
	w.Header().Add(rfc.LastModified, FormatLastModified(t))
}

// DefaultSort sorts alphabetically for a given readerType (eg: TOCDN, TODeliveryService, TOOrigin etc).
func DefaultSort(readerType *Info, param string) {
	if readerType == nil {
		return
	}
	if _, ok := readerType.Params["orderby"]; !ok {
		readerType.Params["orderby"] = param
	}
	return
}
