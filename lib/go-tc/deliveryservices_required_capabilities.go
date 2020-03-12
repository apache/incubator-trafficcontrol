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

//  DeliveryServiceRequiredCapabilitiesResponse contains information about a given DeliveryServiceRequiredCapability in Traffic Ops.
type DeliveryServiceRequiredCapabilitiesResponse struct {
	Alerts
	Response DeliveryServicesRequiredCapability `json:"response,omitempty"`
}

// DeliveryServiceRequiredCapabilitiesDetailResponse contains the result data from a GET request.
type DeliveryServiceRequiredCapabilitiesDetailResponse struct {
	Alerts
	Response []DeliveryServicesRequiredCapability `json:"response,omitempty"`
}

// DeliveryServicesRequiredCapability represents an association between a required capability and a delivery service.
type DeliveryServicesRequiredCapability struct {
	LastUpdated        *TimeNoMod `json:"lastUpdated" db:"last_updated"`
	DeliveryServiceID  *int       `json:"deliveryServiceID" db:"deliveryservice_id"`
	RequiredCapability *string    `json:"requiredCapability" db:"required_capability"`
	XMLID              *string    `json:"xmlID,omitempty" db:"xml_id"`
}
