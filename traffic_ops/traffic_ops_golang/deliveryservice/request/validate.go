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
	"database/sql"
	"errors"
	"fmt"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"

	validation "github.com/go-ozzo/ozzo-validation"
)

// validateLegacy ensures all required fields are present and in correct form.
// Also checks request JSON is complete and valid.
func validateLegacy(dsr tc.DeliveryServiceRequestNullable, tx *sql.Tx) error {
	if tx == nil {
		log.Errorln("validating a legacy Delivery Service Request: nil transaction was passed")
	}

	fromStatus := tc.RequestStatusDraft
	if dsr.ID != nil && *dsr.ID > 0 {
		err := tx.QueryRow(`SELECT status FROM deliveryservice_request WHERE id=$1`, *dsr.ID).Scan(&fromStatus)

		if err != nil {
			log.Errorf("querying for dsr by ID %d: %v", *dsr.ID, err)
			return errors.New("unknown error")
		}
	}

	validTransition := func(s interface{}) error {
		if s == nil {
			return errors.New("cannot transition to nil status")
		}
		toStatus, ok := s.(*tc.RequestStatus)
		if !ok {
			return fmt.Errorf("Expected *tc.RequestStatus type,  got %T", s)
		}
		return fromStatus.ValidTransition(*toStatus)
	}

	errMap := validation.Errors{
		"changeType":      validation.Validate(dsr.ChangeType, validation.Required),
		"deliveryservice": validation.Validate(dsr.DeliveryService, validation.Required),
		"status":          validation.Validate(dsr.Status, validation.Required, validation.By(validTransition)),
	}
	errs := tovalidate.ToErrors(errMap)
	// ensure the deliveryservice requested is valid
	upgraded := dsr.DeliveryService.UpgradeToV4()
	e := deliveryservice.Validate(tx, &upgraded)

	errs = append(errs, e)

	return util.JoinErrs(errs)
}

// validateV4 validates a DSR, returning - in order - a user-facing error that
// should be shown to the client, and a system error.
func validateV4(dsr tc.DeliveryServiceRequestV40, tx *sql.Tx) (error, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	fromStatus := tc.RequestStatusDraft
	if dsr.ID != nil && *dsr.ID > 0 {
		if err := tx.QueryRow(`SELECT status FROM deliveryservice_request WHERE id=$1`, *dsr.ID).Scan(&fromStatus); err != nil {
			return nil, err
		}
	}

	err := validation.ValidateStruct(&dsr,
		validation.Field(&dsr.ChangeType, validation.Required),
		validation.Field(&dsr.Status, validation.By(
			func(s interface{}) error {
				if s == nil {
					return errors.New("required")
				}
				toStatus, ok := s.(tc.RequestStatus)
				if !ok {
					return fmt.Errorf("expected RequestStatus type, got %T", s)
				}
				return fromStatus.ValidTransition(toStatus)
			},
		)),
		validation.Field(&dsr.Requested, validation.By(
			func(r interface{}) error {
				if dsr.ChangeType != tc.DSRChangeTypeUpdate && dsr.ChangeType != tc.DSRChangeTypeCreate {
					return nil
				}
				if r == nil {
					return fmt.Errorf("required for changeType='%s'", dsr.ChangeType)
				}
				ds, ok := r.(*tc.DeliveryServiceV4)
				if !ok {
					return fmt.Errorf("expected a Delivery Service, got %T", r)
				}
				if ds == nil {
					return fmt.Errorf("required for changeType='%s'", dsr.ChangeType)
				}
				err := deliveryservice.Validate(tx, ds)
				if err == nil {
					dsr.XMLID = ds.XMLID
				}
				return err
			},
		)),
		validation.Field(&dsr.Original, validation.By(
			func(o interface{}) error {
				if dsr.ChangeType != tc.DSRChangeTypeDelete {
					return nil
				}
				if o == nil {
					return fmt.Errorf("required for changeType='%s'", dsr.ChangeType)
				}
				ds, ok := o.(*tc.DeliveryServiceV4)
				if !ok {
					return fmt.Errorf("expected a Delivery Service, got %T", o)
				}
				if ds == nil {
					return fmt.Errorf("required for changeType='%s'", dsr.ChangeType)
				}
				if ds.ID == nil {
					return errors.New("must be identified (specify ID)")
				}
				return nil
				// I don't really think we need to validate this, since it's
				// being deleted. ID is sufficient to fully identify it.
				// return ds.Validate(tx)
			},
		)),
		validation.Field(&dsr.Assignee, validation.By(
			func(a interface{}) error {
				if a == nil {
					return nil
				}
				assignee, ok := a.(*string)
				if !ok {
					return fmt.Errorf("expected string, got %T", a)
				}
				if assignee == nil {
					return nil
				}
				var id int
				if err := tx.QueryRow(`SELECT id FROM tm_user WHERE username=$1`, *assignee).Scan(&id); err != nil {
					if err == sql.ErrNoRows {
						return fmt.Errorf("no such user '%s'", *assignee)
					}
					// TODO: allow ParseValidators to return system errors?
					return errors.New("unknown error")
				}
				dsr.AssigneeID = new(int)
				*dsr.AssigneeID = id
				return nil
			},
		)),
	)
	if err != nil {
		return err, nil
	}

	return err, nil
}
