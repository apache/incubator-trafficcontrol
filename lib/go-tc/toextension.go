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

import "encoding/json"

// TOExtensionNullable represents a servercheck extension used by Traffic Ops
type TOExtensionNullable struct {
	ID                   *int            `json:"id"`
	Name                 *string         `json:"name"`
	Version              *string         `json:"version"`
	InfoURL              *string         `json:"info_url"`
	ScriptFile           *string         `json:"script_file"`
	IsActive             *bool           `json:"isactive"`
	AdditionConfigJSON   json.RawMessage `json:"additional_config_json"`
	Description          *string         `json:"description"`
	ServercheckShortName *string         `json:"servercheck_short_name"`
	Type                 *string         `json:"type"`
}

// TOExtensionResponse represents the response from Traffic Ops when creating a TOExtension
type TOExtensionResponse struct {
	Response []TOExtensionNullable `json:"response"`
}
