package v13

import tc "github.com/apache/trafficcontrol/lib/go-tc"

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

// A List of Statuses Response that depict the state of a server
// swagger:response StatusesResponse
type StatusesResponse struct {
	// in: body
	Response []Status `json:"response"`
}

// A Single Status Response for Update and Create to depict what changed
// swagger:response StatusResponse
// in: body
type StatusResponse struct {
	// in: body
	Response Status `json:"response"`
}

// A Single Statuses Response for Update and Create to depict what changed
// swagger:model Statuses
type Status struct {

	// The Statuses to retrieve
	//
	// description of the status type
	//
	Description string `json:"description" db:"description"`

	// ID of the Status
	//
	// required: true
	ID int `json:"id" db:"id"`

	// The Time / Date this server entry was last updated
	//
	LastUpdated tc.TimeNoMod `json:"lastUpdated" db:"last_updated"`

	// enum: ["OFFLINE", "ONLINE", "ADMIN_DOWN", "REPORTED", "CCR_IGNORE", "PRE_PROD"]
	Name string `json:"name" db:"name"`
}
