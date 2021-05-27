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

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	type tester struct {
		st   RequestStatus
		name string
	}
	tests := []tester{
		tester{RequestStatus("foo"), "invalid"},
		tester{RequestStatusDraft, "draft"},
		tester{RequestStatusSubmitted, "submitted"},
		tester{RequestStatusRejected, "rejected"},
		tester{RequestStatusPending, "pending"},
		tester{RequestStatusComplete, "complete"},
	}

	for _, tst := range tests {
		v, _ := RequestStatusFromString(tst.name)
		if tst.name != string(v) {
			t.Errorf("%v: expected %s, got %s", tst, tst.name, string(v))
		}
	}
}

func TestStatusTransition(t *testing.T) {
	bad := errors.New("bad error")
	var validTests = [][]error{
		// To:  Dra  Sub  Rej  Pen  Com   // From:
		[]error{nil, nil, bad, bad, bad}, // Draft
		[]error{nil, nil, nil, nil, nil}, // Submitted
		[]error{bad, bad, bad, bad, bad}, // Rejected
		[]error{bad, bad, bad, nil, nil}, // Pending
		[]error{bad, bad, bad, bad, bad}, // Complete
	}

	// test all transitions
	for i := range validTests {
		from := RequestStatuses[i]
		for j, exp := range validTests[i] {
			to := RequestStatuses[j]
			if exp != nil {
				if from == RequestStatusRejected || from == RequestStatusComplete {
					exp = errors.New(string(from) + " request cannot be changed")
				} else {
					exp = errors.New("invalid transition from " + string(from) + " to " + string(to))
				}
			}
			got := from.ValidTransition(to)
			if got == exp {
				continue
			}

			if got != nil && exp != nil && got.Error() == exp.Error() {
				continue
			}

			t.Errorf("%s -> %s : expected %++v, got %++v", string(from), string(to), exp, got)
		}
	}
}

func TestRequestStatusJSON(t *testing.T) {
	b, err := json.Marshal(RequestStatusDraft)
	if err != nil {
		t.Errorf("Error marshalling %v: %s", RequestStatusDraft, err.Error())
	}

	exp := []byte(`"draft"`)
	if !bytes.Equal(exp, b) {
		t.Errorf("expected %s, got %s", exp, string(b))
	}

	var r RequestStatus
	err = json.Unmarshal([]byte(b), &r)
	if err != nil {
		t.Errorf("Error unmarshalling %s: %v", b, err)
	}
	if r != RequestStatusDraft {
		t.Errorf("expected %v, got %v", RequestStatusDraft, r)
	}
}

func ExampleDSRChangeType_UnmarshalJSON() {
	var dsrct DSRChangeType
	raw := `"CREATE"`
	if err := json.Unmarshal([]byte(raw), &dsrct); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Parsed DSRCT: '%s'\n", dsrct.String())

	raw = `"something invalid"`
	if err := json.Unmarshal([]byte(raw), &dsrct); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Parsed DSRCT: '%s'\n", dsrct.String())

	// Output: Parsed DSRCT: 'create'
	// Error: invalid Delivery Service Request changeType: 'something invalid'
}

func ExampleDeliveryServiceRequestV40_String() {
	var dsr DeliveryServiceRequestV40
	fmt.Println(dsr.String())

	// Output: DeliveryServiceRequestV40(Assignee=<nil>, AssigneeID=<nil>, Author="", AuthorID=<nil>, ChangeType="", CreatedAt=0001-01-01T00:00:00Z, ID=<nil>, LastEditedBy="", LastEditedByID=<nil>, LastUpdated=0001-01-01T00:00:00Z, Status="")
}

func TestDeliveryServiceRequestV40_Downgrade(t *testing.T) {
	xmlid := "xmlid"
	dsr := DeliveryServiceRequestV40{
		Assignee:       nil,
		AssigneeID:     nil,
		Author:         "author",
		AuthorID:       nil,
		ChangeType:     DSRChangeTypeCreate,
		CreatedAt:      time.Time{},
		ID:             nil,
		LastEditedBy:   "last edited by",
		LastEditedByID: nil,
		LastUpdated:    time.Now(),
		Requested:      &DeliveryServiceV4{},
		Status:         RequestStatusComplete,
	}
	dsr.Requested.XMLID = xmlid

	downgraded := dsr.Downgrade()
	if downgraded.Assignee != nil {
		t.Errorf("Incorrect Assignee; want: <nil>, got: %s", *downgraded.Assignee)
	}
	if downgraded.AssigneeID != nil {
		t.Errorf("Incorrect Assignee ID; want: <nil>, got: %d", *downgraded.AssigneeID)
	}
	if downgraded.Author == nil {
		t.Errorf("Incorrect Author; want: '%s', got: <nil>", dsr.Author)
	} else if *downgraded.Author != dsr.Author {
		t.Errorf("Incorrect Author; want: '%s', got: '%s'", dsr.Author, *downgraded.Author)
	}
	if downgraded.AuthorID != nil {
		t.Errorf("Incorrect AuthorID; want: <nil>, got: %v", *downgraded.AuthorID)
	}
	if downgraded.ChangeType == nil {
		t.Errorf("Incorrect ChangeType; want: '%s', got: <nil>", dsr.ChangeType)
	} else if *downgraded.ChangeType != dsr.ChangeType.String() {
		t.Errorf("Incorrect ChangeType; want: '%s', got: '%s'", dsr.ChangeType, *downgraded.ChangeType)
	}
	if downgraded.CreatedAt == nil {
		t.Errorf("Incorrect CreatedAt; want: %v, got: <nil>", dsr.CreatedAt)
	} else if !dsr.CreatedAt.Equal(downgraded.CreatedAt.Time) {
		t.Errorf("Incorrect CreatedAt; want: %v, got: %v", dsr.CreatedAt, *downgraded.CreatedAt)
	}
	if downgraded.DeliveryService == nil {
		t.Errorf("DeliveryService was unexpectedly nil")
	}
	if downgraded.ID != nil {
		t.Errorf("Incorrect ID; want: <nil>, got: %d", *downgraded.ID)
	}
	if downgraded.LastEditedBy == nil {
		t.Errorf("Incorrect LastEditedBy; want: '%s', got: <nil>", dsr.LastEditedBy)
	} else if *downgraded.LastEditedBy != dsr.LastEditedBy {
		t.Errorf("Incorrect LastEditedBy; want: '%s', got: '%s'", dsr.LastEditedBy, *downgraded.LastEditedBy)
	}
	if downgraded.LastEditedByID != nil {
		t.Errorf("Incorrect LastEditedByID; want: <nil>, got: %d", *downgraded.LastEditedByID)
	}
	if downgraded.LastUpdated == nil {
		t.Errorf("Incorrect LastUpdated; want: %v, got: <nil>", dsr.LastUpdated)
	} else if !dsr.LastUpdated.Equal(downgraded.LastUpdated.Time) {
		t.Errorf("Incorrect LastUpdated; want: %v, got: %v", dsr.LastUpdated, *downgraded.LastUpdated)
	}
	if downgraded.Status == nil {
		t.Errorf("Incorrect Status; want: '%s', got: <nil>", dsr.Status)
	} else if *downgraded.Status != dsr.Status {
		t.Errorf("Incorrect Status; want: '%s', got: '%s'", dsr.Status, *downgraded.Status)
	}
	if downgraded.XMLID == nil {
		t.Errorf("Incorrect XMLID; want: '%s', got: <nil>", xmlid)
	} else if *downgraded.XMLID != xmlid {
		t.Errorf("Incorrect XMLID; want: '%s', got: '%s'", xmlid, *downgraded.XMLID)
	}
}

func ExampleDeliveryServiceRequestV40_SetXMLID() {
	var dsr DeliveryServiceRequestV40
	fmt.Println(dsr.XMLID == "")

	dsr.Requested = new(DeliveryServiceV4)
	dsr.Requested.XMLID = "test"
	dsr.SetXMLID()

	fmt.Println(dsr.XMLID)

	// Output: true
	// test
}
