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
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/lib/pq"
)

func TestCamelCase(t *testing.T) {
	testStrings := []string{"hello_world", "trailing_underscore_", "w_h_a_t____"}
	expected := []string{"helloWorld", "trailingUnderscore", "wHAT"}
	for i, str := range testStrings {
		if toCamelCase(str) != expected[i] {
			t.Errorf("expected: %v error, actual: %v", expected[i], toCamelCase(str))
		}
	}
}

// TestRespWrittenAfterErrFails tests that a WriteResp called after HandleErr will not be written.
func TestRespWrittenAfterErrFails(t *testing.T) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}
	tx := (*sql.Tx)(nil)

	expectedCode := http.StatusUnauthorized
	expectedUserErr := errors.New("user unauthorized")

	HandleErr(w, r, tx, expectedCode, expectedUserErr, nil)
	WriteResp(w, r, "should not be written")

	actualCode := w.Code
	statusVal := r.Context().Value(tc.StatusKey)
	statusInt, ok := statusVal.(int)
	if ok {
		actualCode = statusInt
	}

	if actualCode != expectedCode {
		t.Errorf("code expected: %+v, actual %+v", expectedCode, actualCode)
	}

	alerts := tc.Alerts{}
	if err := json.Unmarshal(w.Body, &alerts); err != nil {
		t.Fatalf("unmarshalling actual body: %v", err)
	}
	for _, alert := range alerts.Alerts {
		if string(alert.Level) != tc.ErrorLevel.String() {
			t.Errorf("alert level expected: '%s', actual: '%s'", tc.ErrorLevel.String(), alert.Level)
		}
	}
}

func TestWriteAlertsObjEmpty(t *testing.T) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}
	a := tc.Alerts{}
	code := http.StatusAlreadyReported

	WriteAlertsObj(w, r, code, a, code)

	resp := struct {
		Response interface{} `json:"response"`
	}{code}
	serialized, _ := json.Marshal(resp)
	if !bytes.Equal(append(serialized[:], '\n'), w.Body[:]) {
		t.Error("expected response to only include object")
	}
	writeAlertsCodeTest(t, *w, code)
}

func TestWriteAlertsObj(t *testing.T) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}
	a := tc.CreateAlerts(tc.WarnLevel, "test")
	code := http.StatusAlreadyReported

	WriteAlertsObj(w, r, code, a, code)

	resp := struct {
		tc.Alerts
		Response interface{} `json:"response"`
	}{a, code}
	serialized, _ := json.Marshal(resp)
	if !bytes.Equal(append(serialized[:], '\n'), w.Body[:]) {
		t.Error("expected response to include alert")
	}
	writeAlertsCodeTest(t, *w, code)
}

func writeAlertsCodeTest(t *testing.T, w MockHTTPResponseWriter, code int) {
	if w.Body == nil || len(w.Body) == 0 {
		t.Error("expected response body to be written to")
	}
	if w.Code != code {
		t.Errorf("expected response code %v, got %v", code, w.Code)
	}
}

func TestWriteResp(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteResp(w, r, "foo")
	})
}

func TestWriteRespRaw(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespRaw(w, r, "foo")
	})
}

func TestWriteRespVals(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespVals(w, r, "foo", map[string]interface{}{"a": "b"})
	})
}

func TestRespWriter(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		RespWriter(w, r, nil)("foo", nil)
	})
}

func TestRespWriterVals(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		RespWriterVals(w, r, nil, map[string]interface{}{"a": "b"})("foo", nil)
	})
}

func TestWriteRespAlert(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespAlert(w, r, tc.ErrorLevel, "foo error")
	})
}

func TestWriteRespAlertObj(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespAlertObj(w, r, tc.ErrorLevel, "foo error", "bar")
	})
}

// apiWriteTest tests that an API write func succeeds and writes a body and a 200.
func apiWriteTest(t *testing.T, write func(w http.ResponseWriter, r *http.Request)) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}

	write(w, r)

	if w.Code == 0 {
		w.Code = http.StatusOK // emulate behavior of w.Write
	}

	actualCode := w.Code
	statusVal := r.Context().Value(tc.StatusKey)
	statusInt, ok := statusVal.(int)
	if ok {
		actualCode = statusInt
	}

	expectedCode := http.StatusOK

	if actualCode != expectedCode {
		t.Errorf("code expected: %+v, actual %+v", expectedCode, actualCode)
	}

	if len(w.Body) == 0 {
		t.Errorf("body len expected: >0, actual 0")
	}
}

type MockHTTPResponseWriter struct {
	Code int
	Body []byte
}

func (i *MockHTTPResponseWriter) WriteHeader(rc int) {
	i.Code = rc
}

func (i *MockHTTPResponseWriter) Write(b []byte) (int, error) {
	i.Body = append(i.Body, b...)
	return len(b), nil
}

func (i *MockHTTPResponseWriter) Header() http.Header {
	return http.Header{}
}

func TestParseRestrictFKConstraint(t *testing.T) {
	var testCases = []struct {
		description        string
		storageError       pq.Error
		expectedReturnCode int
	}{
		{
			description: "FK Constraint Error",
			storageError: pq.Error{
				Message: "update or delete on table \"foo\" violates foreign key constraint \"fk_foo_bar\" on table \"bar\"",
			},
			expectedReturnCode: http.StatusBadRequest,
		},
		{
			description: "FK Constraint Error with underscores in table name",
			storageError: pq.Error{
				Message: "update or delete on table \"foo_ser\" violates foreign key constraint \"fk_foo_bar\" on table \"bar_cap\"",
			},
			expectedReturnCode: http.StatusBadRequest,
		},
		{
			description: "Non FK Constraint Error",
			storageError: pq.Error{
				Message: "connection error",
			},
			expectedReturnCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Log("Starting test scenario: ", tc.description)
			errs := parseRestrictFKConstraint(&tc.storageError)
			if errs.Code != tc.expectedReturnCode {
				t.Errorf("code expected: %v, actual %v", tc.expectedReturnCode, errs.Code)
			}
		})
	}
}
