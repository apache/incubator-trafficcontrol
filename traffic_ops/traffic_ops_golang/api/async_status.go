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
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	AsyncSucceeded = "SUCCEEDED"
	AsyncFailed    = "FAILED"
	AsyncPending   = "PENDING"
)

const CurrentAsyncEndpoint = "/api/4.0/async_status/"

type AsyncStatus struct {
	Id        int        `json:"id,omitempty" db:"id"`
	Status    string     `json:"status,omitempty" db:"status"`
	StartTime time.Time  `json:"start_time,omitempty" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`
	Message   *string    `json:"message,omitempty" db:"message"`
}

const selectAsyncStatusQuery = `SELECT id, status, message, start_time, end_time from async_status WHERE id = $1`
const insertAsyncStatusQuery = `INSERT INTO async_status (status, message) VALUES ($1, $2) RETURNING id`
const updateAsyncStatusEndTimeQuery = `UPDATE async_status SET status = $1, message = $2, end_time = now() WHERE id = $3`
const updateAsyncStatusQuery = `UPDATE async_status SET status = $1, message = $2 WHERE id = $3`

// GetAsyncStatus returns the status of an asynchronous job.
func GetAsyncStatus(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	asyncStatusId := inf.Params["id"]

	rows, err := inf.Tx.Tx.Query(selectAsyncStatusQuery, asyncStatusId)
	if err != nil {
		HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	defer rows.Close()

	asyncStatus := AsyncStatus{}
	rowCount := 0
	for rows.Next() {
		rowCount++
		err := rows.Scan(&asyncStatus.Id, &asyncStatus.Status, &asyncStatus.Message, &asyncStatus.StartTime, &asyncStatus.EndTime)
		if err != nil {
			HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
			return
		}
	}

	if rowCount == 0 {
		HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, errors.New("async status not found"))
		return
	}

	WriteResp(w, r, asyncStatus)
}

// InsertAsyncStatus inserts a new status for an asynchronous job.
func InsertAsyncStatus(tx *sql.Tx, message string) (int, int, error, error) {
	defer tx.Commit()

	resultRows, err := tx.Query(insertAsyncStatusQuery, AsyncPending, message)
	if err != nil {
		userErr, sysErr, errCode := ParseDBError(err)
		return 0, errCode, userErr, sysErr
	}
	defer resultRows.Close()

	var asyncStatusId int

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&asyncStatusId); err != nil {
			return 0, http.StatusInternalServerError, nil, err
		}
	}
	if rowsAffected == 0 {
		return 0, http.StatusInternalServerError, nil, errors.New("async status create: no status was inserted, no id was returned")
	} else if rowsAffected > 1 {
		return 0, http.StatusInternalServerError, nil, errors.New("too many ids returned from async status insert")
	}

	return asyncStatusId, http.StatusOK, nil, nil
}

// UpdateAsyncStatus updates the status table for an asynchronous job.
func UpdateAsyncStatus(db *sqlx.DB, newStatus string, newMessage string, asyncStatusId int, finished bool) error {
	if asyncStatusId == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	q := updateAsyncStatusQuery
	if finished {
		q = updateAsyncStatusEndTimeQuery
	}
	_, err = tx.Exec(q, newStatus, newMessage, asyncStatusId)
	if err != nil {
		return err
	}

	return nil
}
