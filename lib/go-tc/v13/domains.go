package v13

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

type DomainsResponse struct {
	Response []Domain `json:"response"`
}

type Domain struct {

	ProfileID int `json:"profileId" db:"profile_id"`

	ParameterID int `json:"parameterId" db:"parameter_id"`

	ProfileName string `json:"profileName" db:"profile_name"`

	ProfileDescription string `json:"profileDescription" db:"profile_description"`

	// DomainName of the CDN
	DomainName string `json:"domainName" db:"domain_name"`
}

// DomainNullable - a struct version that allows for all fields to be null, mostly used by the API side
type DomainNullable struct {
	ProfileID          *int    `json:"profileId" db:"profile_id"`
	ParameterID        *int    `json:"parameterId" db:"parameter_id"`
	ProfileName        *string `json:"profileName" db:"profile_name"`
	ProfileDescription *string `json:"profileDescription" db:"profile_description"`
	DomainName         *string `json:"domainName" db:"domain_name"`
}
