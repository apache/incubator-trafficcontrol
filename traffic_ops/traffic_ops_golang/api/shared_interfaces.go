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
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

type CRUDer interface {
	Create() (error, error, int)
	Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time)
	Update(http.Header) (error, error, int)
	Delete() (error, error, int)
	Infoer
	Identifier
	Validator
}

type AlertsResponse interface {
	// GetAlerts retrieves an array of alerts that were generated over the course of handling an endpoint.
	GetAlerts() tc.Alerts
}

type Identifier interface {

	// Getters and Setters for key data
	// The current common case is a single numerical id
	SetKeys(map[string]interface{})
	GetKeys() (map[string]interface{}, bool)

	// GetType gives the name of the implementing struct
	GetType() string

	// GetAuditName returns the name of an object instance. If no name is availible, the id should be returned. "unknown" is the final case
	GetAuditName() string

	// This should define the key getters and setters
	GetKeyFieldsInfo() []KeyFieldInfo
}

type Creator interface {
	// Create returns any user error, any system error, and the HTTP error code to be returned if there was an error.
	Create() (error, error, int)
	Infoer
	Identifier
	Validator
}

// MultipleCreator indicates whether an object using the shared handlers allows an array of objects in the POST
type MultipleCreator interface {
	AllowMultipleCreates() bool
}

type Reader interface {
	// Read returns the object to write to the user, any user error, any system error, and the HTTP error code to be returned if there was an error.
	Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time)
	Infoer
}

type Updater interface {
	// Update returns any user error, any system error, and the HTTP error code to be returned if there was an error.
	Update(h http.Header) (error, error, int)
	Infoer
	Identifier
	Validator
}

type Deleter interface {
	// Delete returns any user error, any system error, and the HTTP error code to be returned if there was an error.
	Delete() (error, error, int)
	Infoer
	Identifier
}

// OptionsDeleter calls the OptionsDelete() generic CRUD function, unlike Deleter, which calls Delete().
type OptionsDeleter interface {
	// OptionsDelete returns any user error, any system error, and the HTTP error code to be returned if there was an
	// error.
	OptionsDelete() (error, error, int)
	Infoer
	Identifier
	DeleteKeyOptions() map[string]dbhelpers.WhereColumnInfo
}

type Validator interface {
	Validate() error
}

type Tenantable interface {
	IsTenantAuthorized(user *auth.CurrentUser) (bool, error)
}

// Infoer is an interface that guarantees the existance of a variable through its setters and getters.
// Every CRUD operation uses this login session context
type Infoer interface {
	SetInfo(*Info)
	Info() *Info
}

// InfoerImpl is an implementation of the Infoer interface.
type InfoerImpl struct {
	// ReqInfo is the stored API Info.
	ReqInfo *Info `json:"-"`
}

// Info returns the InfoerImpl's stored API Info.
func (i InfoerImpl) Info() *Info {
	return i.ReqInfo
}

// SetInfo sets the InfoerImpl's stored API Info.
func (i *InfoerImpl) SetInfo(inf *Info) {
	i.ReqInfo = inf
}
