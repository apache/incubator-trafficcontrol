package apierrors

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
	"errors"
	"fmt"
	"net/http"
)

// Errors represents a set of errors to be handled by the API.
type Errors struct {
	// Code is the HTTP response code. If no error has occurred, this *should*
	// be http.StatusOK.
	Code int
	// SystemError is an error that occurred internally; not safe for exposure
	// to the user.
	SystemError error
	// UserError is an error that should be shown to the user to explain why
	// their request failed.
	UserError error
}

// New constructs a new Errors where no error has actually occurred.
func New() Errors {
	return Errors{
		Code:        http.StatusOK,
		SystemError: nil,
		UserError:   nil,
	}
}

// Occurred returns whether at least one error has occurred (is non-nil).
func (e Errors) Occurred() bool {
	return e.SystemError != nil || e.UserError != nil
}

// SetSystemError sets the Errors' system-level error to a new error containing
// the passed message.
func (e *Errors) SetSystemError(err string) {
	e.SystemError = errors.New(err)
}

// SetUserError sets the Errors' user-level error to a new error containing the
// passed message.
func (e *Errors) SetUserError(err string) {
	e.UserError = errors.New(err)
}

func (e Errors) String() string {
	return fmt.Sprintf("Errors(Code=%d, SystemError='%v', UserError='%v')", e.Code, e.SystemError, e.UserError)
}
