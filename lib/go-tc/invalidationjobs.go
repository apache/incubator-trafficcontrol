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

import "errors"
import "fmt"
import "regexp"
import "database/sql"
import "math"
import "strconv"
import "strings"
import "time"

import "github.com/apache/trafficcontrol/lib/go-log"

import "github.com/go-ozzo/ozzo-validation"
import "github.com/go-ozzo/ozzo-validation/is"

// MaxTTL is the maximum value of TTL representable as a time.Duration object, which is used
// internally by InvalidationJobInput objects to store the TTL.
const MaxTTL = math.MaxInt64 / 3600000000000

var twoDays = time.Hour * 48

// ValidJobRegexPrefix matches the only valid prefixes for a relative-path Content Invalidation Job regex
var ValidJobRegexPrefix = regexp.MustCompile(`^\?/.*$`)

// InvalidationJob represents a content invalidation job as returned by the API.
type InvalidationJob struct {
	AssetURL        *string `json:"assetUrl"`
	CreatedBy       *string `json:"createdBy"`
	DeliveryService *string `json:"deliveryService"`
	ID              *uint64 `json:"id"`
	Keyword         *string `json:"keyword"`
	Parameters      *string `json:"parameters"`

	// StartTime is the time at which the job will come into effect. Must be in the future, but will
	// fail to Validate if it is further in the future than two days.
	StartTime *Time `json:"startTime"`
}

// InvalidationJobInput represents user input intending to create or modify a content invalidation job.
type InvalidationJobInput struct {

	// DeliveryService needs to be an identifier for a Delivery Service. It can be either a string - in which
	// case it is treated as an XML_ID - or a float64 (because that's the type used by encoding/json
	// to represent all JSON numbers) - in which case it's treated as an integral, unique identifier
	// (and any fractional part is discarded, i.e. 2.34 -> 2)
	DeliveryService *interface{} `json:"deliveryService"`

	// Regex is a regular expression which not only must be valid, but should also start with '/'
	// (or escaped: '\/')
	Regex *string `json:"regex"`

	// StartTime is the time at which the job will come into effect. Must be in the future.
	StartTime *Time `json:"startTime"`

	// TTL indicates the Time-to-Live of the job. This can be either a valid string for
	// time.ParseDuration, or a float64 indicating the number of hours. Note that regardless of the
	// actual value here, Traffic Ops will only consider it rounded down to the nearest natural
	// number
	TTL *interface{} `json:"ttl"`

	dsid *uint          `json:"-"`
	ttl  *time.Duration `json:"-"`
}

// UserInvalidationJobInput Represents legacy-style user input to the /user/current/jobs API endpoint.
// This is much less flexible than InvalidationJobInput, which should be used instead when possible.
type UserInvalidationJobInput struct {
	DSID  *uint   `json:"dsId"`
	Regex *string `json:"regex"`

	// StartTime is the time at which the job will come into effect. Must be in the future, but will
	// fail to Validate if it is further in the future than two days.
	StartTime *Time   `json:"startTime"`
	TTL       *uint64 `json:"ttl"`
	Urgent    *bool   `json:"urgent"`
}

// UserInvalidationJob is a full representation of content invalidation jobs as stored in the
// database, including several unused fields.
type UserInvalidationJob struct {

	// Agent is unused, and developers should never count on it containing or meaning anything.
	Agent    *uint   `json:"agent"`
	AssetURL *string `json:"assetUrl"`

	// AssetType is unused, and developers should never count on it containing or meaning anything.
	AssetType       *string `json:"assetType"`
	DeliveryService *string `json:"deliveryService"`
	EnteredTime     *Time   `json:"enteredTime"`
	ID              *uint   `json:"id"`
	Keyword         *string `json:"keyword"`

	// ObjectName is unused, and developers should never count on it containing or meaning anything.
	ObjectName *string `json:"objectName"`

	// ObjectType is unused, and developers should never count on it containing or meaning anything.
	ObjectType *string `json:"objectType"`
	Parameters *string `json:"parameters"`
	Username   *string `json:"username"`
}

// TTLHours gets the number of hours of the job's TTL - rounded down to the nearest natural number,
// or an error if it is an invalid value.
func (j *InvalidationJobInput) TTLHours() (uint, error) {
	if j.ttl != nil {
		return uint((*j.ttl).Hours()), nil
	}
	if j.TTL == nil {
		return 0, errors.New("Attempted to convert a nil TTL into hours")
	}

	var ret uint
	switch t := (*j.TTL).(type) {
	case float64:
		v := (*j.TTL).(float64)
		if v < 0 {
			return 0, errors.New("TTL cannot be negative!")
		}
		if v >= MaxTTL {
			return 0, fmt.Errorf("TTL cannot exceed %d hours!", MaxTTL)
		}
		ttl := time.Duration(int64(v * 3600000000000))
		j.ttl = &ttl
		ret = uint(ttl.Hours())

	case string:
		d, err := time.ParseDuration((*j.TTL).(string))
		if err != nil || d.Hours() < 1 {
			return 0, fmt.Errorf("Invalid duration entered for TTL! Must be at least one hour, but no more than %d hours!", MaxTTL)
		}
		j.ttl = &d
		ret = uint(d.Hours())

	default:
		log.Errorf("unsupported TTL key type: %T\n", t)
		return 0, errors.New("Unknown error occurred")
	}

	return ret, nil
}

// DSID gets the integral, unique identifier of the Delivery Service identified by
// InvalidationJobInput.DeliveryService
//
// This requires a transaction connected to a Traffic Ops database, because if DeliveryService is
// an xml_id, a database lookup will be necessary to get the unique, integral identifier. Thus,
// this method also checks for the existence of the identified Delivery Service, and will return
// an error if it does not exist.
func (j *InvalidationJobInput) DSID(tx *sql.Tx) (uint, error) {
	if j.dsid != nil {
		return *j.dsid, nil
	}

	if j.DeliveryService == nil {
		return 0, errors.New("Attempted to turn a nil DeliveryService into a DSID")
	}
	if tx == nil {
		return 0, errors.New("Attempted to turn a DeliveryService into a DSID with no DB connection")
	}

	var ret uint
	switch t := (*j.DeliveryService).(type) {
	case float64:
		v := (*j.DeliveryService).(float64)
		if v < 0 {
			return 0, errors.New("Delivery Service ID cannot be negative")
		}

		u := uint(v)
		var exists bool
		row := tx.QueryRow(`SELECT EXISTS(SELECT * FROM deliveryservice WHERE id=$1)`, u)
		if err := row.Scan(&exists); err != nil {
			log.Errorf("Error checking for deliveryservice existence in DSID: %v\n", err)
			return 0, errors.New("Unknown error occurred")
		} else if !exists {
			return 0, fmt.Errorf("No Delivery Service exists matching identifier: %v", *j.DeliveryService)
		}

		j.dsid = &u
		return u, nil

	case string:
		row := tx.QueryRow(`SELECT id FROM deliveryservice WHERE xml_id=$1`, *j.DeliveryService)
		if err := row.Scan(&ret); err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("No DeliveryService exists matching identifier: %v", *j.DeliveryService)
			}
			return 0, errors.New("Unknown error occurred")
		}
		j.dsid = &ret
		return ret, nil

	default:
		log.Errorf("unsupported DS key type: %T\n", t)
		return 0, errors.New("Unknown error occurred")

	}
}

// Validate, given a transaction connected to the Traffic Ops database, validates that the user input
// is correct. In particular, it enforces the constraints described on each field, as well as
// ensuring they actually exist. This method calls InvalidationJobInput.DSID to validate the
// DeliveryService field.
//
// This returns an error describing any and all problematic fields encountered during validation.
func (job *InvalidationJobInput) Validate(tx *sql.Tx) error {
	errs := []string{}
	err := validation.ValidateStruct(job,
		validation.Field(job.DeliveryService, validation.Required),
		validation.Field(job.Regex, validation.Required, validation.Match(ValidJobRegexPrefix)),
		validation.Field(job.StartTime, validation.Required),
		validation.Field(job.TTL, validation.Required),
	)

	if err != nil {
		errs = append(errs, err.Error())
	}

	if job.DeliveryService != nil {
		if _, err := job.DSID(tx); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if job.Regex != nil && *job.Regex != "" {
		if _, err := regexp.Compile(*job.Regex); err != nil {
			errs = append(errs, "'regex' is not a valid Regular Expression: "+err.Error())
		}
	}

	if job.StartTime != nil && job.StartTime.Time.Before(time.Now()) {
		errs = append(errs, "'startTime' must be in the future!")
	}

	if job.TTL != nil {
		if _, err := job.TTLHours(); err != nil {
			errs = append(errs, "'ttl' must be a number of hours, or a duration string e.g. '48h'!")
		}
	}

	return errors.New(strings.Join(errs, ", "))
}

// Validate checks that the InvalidationJob is valid, by ensuring all of its fields are well-defined.
//
// This returns an error describing any and all problematic fields encountered during validation.
func (job *InvalidationJob) Validate() error {
	err := validation.ValidateStruct(job,
		validation.Field(job.AssetURL, validation.Required, is.URL),
		validation.Field(job.CreatedBy, validation.Required),
		validation.Field(job.DeliveryService, validation.Required),
		validation.Field(job.ID, validation.Required),
		validation.Field(job.Keyword, validation.Required),
		validation.Field(job.Parameters, validation.Required),
		validation.Field(job.StartTime, validation.Required),
	)

	if job.StartTime == nil {
		return err
	}

	if job.StartTime.After(time.Now().Add(twoDays)) {
		e := errors.New("'startTime' must be within two days from now")
		if err == nil {
			return e
		}
		return fmt.Errorf("%v, %v", err, e)
	}

	if job.StartTime.Before(time.Now()) {
		e := errors.New("'startTime' cannot be in the past")
		if err == nil {
			return e
		}
		return fmt.Errorf("%v, %v", err, e)
	}

	return err
}

// Validate, given a transaction connected to the Traffic Ops database, validates that the user input
// is correct.
//
// This requires a database transaction to check that the DSID is a valid identifier for an existing
// Delivery Service.
//
// Returns an error describing any and all problematic fields encountered during validation.
func (job *UserInvalidationJobInput) Validate(tx *sql.Tx) error {
	errs := []string{}
	err := validation.ValidateStruct(job,
		validation.Field(job.StartTime, validation.Required),
		validation.Field(job.Regex, validation.Required, validation.Match(ValidJobRegexPrefix)),
		validation.Field(job.DSID, validation.Required),
		validation.Field(job.TTL, validation.Required),
	)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if job.StartTime != nil && job.StartTime.After(time.Now().Add(twoDays)) {
		errs = append(errs, "'startTime' must be within two days!")
	}

	if job.Regex != nil && *(job.Regex) != "" {
		if _, err := regexp.Compile(*(job.Regex)); err != nil {
			errs = append(errs, "'regex' is not a valid regular expression: "+err.Error())
		}
	}

	if job.DSID != nil {
		row := tx.QueryRow(`SELECT id FROM deliveryservice WHERE id = $1::bigint`, job.DSID)
		var id uint
		if err := row.Scan(&id); err != nil {
			log.Errorln(err.Error())
			errs = append(errs, "No Delivery Service corresponding to 'dsId'!")
		}
	}

	if job.TTL != nil {
		row := tx.QueryRow(`SELECT value FROM parameter WHERE name='maxRevalDurationDays' AND config_file='regex_revalidate.config'`)
		var max uint64
		err := row.Scan(&max)
		if err == sql.ErrNoRows && MaxTTL < *(job.TTL) {
			errs = append(errs, "'ttl' cannot exceed "+strconv.FormatUint(MaxTTL, 10)+"!")
		} else if err == nil && max < *(job.TTL) { //silently ignore other errors to
			errs = append(errs, "'ttl' cannot exceed "+strconv.FormatUint(max, 10)+"!")
		} else if *(job.TTL) < 1 {
			errs = append(errs, "'ttl' must be at least 1!")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}
	return nil
}
