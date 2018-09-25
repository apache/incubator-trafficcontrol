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

// A List of Roles Response
// swagger:response RolesResponse
// in: body
type RolesResponse struct {
	// in: body
	Response []Role `json:"response"`
}

// A Single Role Response for Update and Create to depict what changed
// swagger:response RoleResponse
// in: body
type RoleResponse struct {
	// in: body
	Response Role `json:"response"`
}

// Role ...
type Role struct {
	// ID of the Role
	//
	// required: true
	ID *int `json:"id" db:"id"`

	// Name of the Role
	//
	// required: true
	Name *string `json:"name" db:"name"`

	// Description of the Role
	//
	// required: true
	Description *string `json:"description" db:"description"`

	// Priv Level of the Role. This is no longer used, and new versions of Traffic Ops always return 0.
	//
	// required: false
	PrivLevel *int `json:"privLevel"`

	// Capabilities associated with the Role
	//
	// required: true
	Capabilities *[]string `json:"capabilities" db:"-"`
}
