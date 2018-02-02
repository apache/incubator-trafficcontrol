package request

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
	"encoding/json"

	tc "github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/tovalidate"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
)

// Validate ensures all required fields are present and in correct form.  Also checks request JSON is complete and valid
func (req *TODeliveryServiceRequest) Validate(db *sqlx.DB) []error {
	validation.NewStringRule(tovalidate.NoPeriods, "cannot contain periods")
	fromStatus := req.Status

	validTransition := func(s interface{}) error {
		toStatus, err := tc.RequestStatusFromString(s.(string))
		if err != nil {
			return err
		}
		return fromStatus.ValidTransition(toStatus)
	}
	errMap := validation.Errors{
		"changeType":      validation.Validate(req.ChangeType, validation.Required),
		"deliveryservice": validation.Validate(req.DeliveryService, validation.Required),
		"status": validation.Validate(req.Status, validation.Required,
			validation.By(validTransition)),
	}

	errs := tovalidate.ToErrors(errMap)

	var ds deliveryservice.TODeliveryService
	err := json.Unmarshal([]byte(req.DeliveryService), &ds)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	// ensure the deliveryservice requested is valid
	e := ds.Validate(db)
	errs = append(errs, e...)

	return errs
}
