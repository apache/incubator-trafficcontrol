package main

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
	"compress/gzip"
	"encoding/json"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/tocookie"
	"github.com/jmoiron/sqlx"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// TestWrapHeaders checks that appropriate default headers are added to a request
func TestWrapHeaders(t *testing.T) {
	body := "We are here!!"
	f := wrapHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", ".", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	// Call to add the headers
	f(w, r)
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}

	expected := map[string]http.Header{
		"Access-Control-Allow-Credentials": nil,
		"Access-Control-Allow-Headers":     nil,
		"Access-Control-Allow-Methods":     nil,
		"Access-Control-Allow-Origin":      nil,
		"Content-Type":                     nil,
		"Whole-Content-Sha512":             nil,
		"X-Server-Name":                    nil,
	}

	if len(expected) != len(w.HeaderMap) {
		t.Error("Expected", len(expected), "header, got", len(w.HeaderMap))
	}
	m := w.Header()
	for k := range expected {
		if _, ok := m[k]; !ok {
			t.Error("Expected header", k, "not found")
		}
	}
}

// TestGzip checks that if Accept-Encoding contains "gzip" that the body is indeed gzip'd
func TestGzip(t *testing.T) {
	body := "am I gzip'd?"
	gz := bytes.Buffer{}
	zw := gzip.NewWriter(&gz)

	if _, err := zw.Write([]byte(body)); err != nil {
		t.Error("Error gzipping", err)
	}

	if err := zw.Close(); err != nil {
		t.Error("Error closing gzipper", err)
	}

	f := wrapHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "/", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	f(w, r)

	// body should not be gzip'd
	if bytes.Compare(w.Body.Bytes(), []byte(body)) != 0 {
		t.Error("Expected body to be NOT gzip'd!")
	}

	// Call with gzip
	w = httptest.NewRecorder()
	r.Header.Add("Accept-Encoding", "gzip")
	f(w, r)
	if bytes.Compare(w.Body.Bytes(), gz.Bytes()) != 0 {
		t.Error("Expected body to be gzip'd!")
	}
}

func TestWrapAuth(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	defer mockDB.Close()
	db := sqlx.NewDb(mockDB, "sqlmock")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	userName := "user1"
	secret := "secret"

	rows := sqlmock.NewRows([]string{"priv_level"})
	rows.AddRow(30)
	mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(userName).WillReturnRows(rows)

	sqlStatement, err := preparePrivLevelStmt(db)
	if err != nil {
		t.Fatalf("could not create priv statement: %v\n", err)
	}

	authBase := AuthBase{false, secret, sqlStatement, nil}

	cookie := tocookie.New(userName, time.Now().Add(time.Minute), secret)

	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		privLevel, err := getPrivLevel(ctx)
		if err != nil {
			t.Fatalf("unable to get privLevel: %v", err)
			return
		}
		userName, err := getUserName(ctx)
		if err != nil {
			t.Fatalf("unable to get userName: %v", err)
			return
		}

		response := struct {
			PrivLevel int
			UserName  string
		}{privLevel, userName}

		respBts, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("unable to marshal: %v", err)
			return
		}

		w.Header().Set(tc.ContentType, tc.ApplicationJson)
		fmt.Fprintf(w, "%s", respBts)
	}

	authWrapper := authBase.GetWrapper(15)

	f := authWrapper(handler)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "/", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	r.Header.Add("Cookie", tocookie.Name+"="+cookie)

	expected := struct {
		PrivLevel int
		UserName  string
	}{30, userName}

	expectedBody, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("unable to marshal: %v", err)
	}

	f(w, r)

	if bytes.Compare(w.Body.Bytes(), expectedBody) != 0 {
		t.Errorf("received: %s\n expected: %s\n", w.Body.Bytes(), expectedBody)
	}

	w = httptest.NewRecorder()
	r, err = http.NewRequest("", "/", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	f(w, r)

	expectedError := "Unauthorized"

	if *debugLogging {
		fmt.Printf("received: %s\n expected: %s\n", w.Body.Bytes(), expectedError)
	}

	if bytes.Compare(w.Body.Bytes(), []byte(expectedError)) != 0 {
		t.Errorf("received: %s\n expected: %s\n", w.Body.Bytes(), expectedError)
	}
}

// TODO: TestWrapAccessLog
