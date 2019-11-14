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
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

type CRUDer interface {
	Create() (error, error, int)
	Read() ([]interface{}, error, error, int)
	Update() (error, error, int)
	Delete() (error, error, int)
	APIInfoer
	Identifier
	Validator
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
	APIInfoer
	Identifier
	Validator
}

type Reader interface {
	// Read returns the object to write to the user, any user error, any system error, and the HTTP error code to be returned if there was an error.
	Read() ([]interface{}, error, error, int)
	APIInfoer
	Identifier
}

type Updater interface {
	// Update returns any user error, any system error, and the HTTP error code to be returned if there was an error.
	Update() (error, error, int)
	APIInfoer
	Identifier
	Validator
}

type Deleter interface {
	// Delete returns any user error, any system error, and the HTTP error code to be returned if there was an error.
	Delete() (error, error, int)
	APIInfoer
	Identifier
}

type Validator interface {
	Validate() error
}

type Tenantable interface {
	IsTenantAuthorized(user *auth.CurrentUser) (bool, error)
}

// APIInfoer is an interface that guarantees the existance of a variable through its setters and getters.
// Every CRUD operation uses this login session context
type APIInfoer interface {
	SetInfo(*APIInfo)
	APIInfo() *APIInfo
}

// Modifieder allows Readers to set a LastModified time, to be returned to clients.
// If a Reader implements Modifieder, the "CRUDer" ReadHandler will use it to set an ETag and Last-Modified header.
type Modifieder interface {
	LastModified() time.Time
}
