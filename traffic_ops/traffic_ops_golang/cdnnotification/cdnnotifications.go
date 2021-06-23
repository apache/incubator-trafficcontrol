package cdnnotification

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
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

const readQuery = `
SELECT cn.id,
	cn.cdn,
	cn.last_updated,
	cn.user,
	cn.notification
FROM cdn_notification as cn
INNER JOIN cdn ON cdn.name = cn.cdn
INNER JOIN tm_user ON tm_user.username = cn.user
`

const insertQuery = `
INSERT INTO cdn_notification (cdn, "user", notification)
VALUES ($1, $2, $3)
RETURNING cdn_notification.id,
cdn_notification.cdn,
cdn_notification.last_updated,
cdn_notification.user,
cdn_notification.notification
`

const deleteQuery = `
DELETE FROM cdn_notification
WHERE cdn_notification.id = $1
RETURNING cdn_notification.id,
cdn_notification.cdn,
cdn_notification.last_updated,
cdn_notification.user,
cdn_notification.notification
`

// Read is the handler for GET requests to /cdn_notifications.
func Read(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cdnNotifications := []tc.CDNNotification{}

	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"id":   dbhelpers.WhereColumnInfo{Column: "cn.id", Checker: api.IsInt},
		"cdn":  dbhelpers.WhereColumnInfo{Column: "cdn.name"},
		"user": dbhelpers.WhereColumnInfo{Column: "tm_user.username"},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToQueryCols)
	if len(errs) > 0 {
		sysErr = util.JoinErrs(errs)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	query := readQuery + where + orderBy + pagination
	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		if sysErr != nil {
			sysErr = fmt.Errorf("notification read query: %v", sysErr)
		}

		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var n tc.CDNNotification
		if err = rows.Scan(&n.ID, &n.CDN, &n.LastUpdated, &n.User, &n.Notification); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("scanning cdn notifications: "+err.Error()))
			return
		}
		cdnNotifications = append(cdnNotifications, n)
	}

	api.WriteResp(w, r, cdnNotifications)
}

// Create is the handler for POST requests to /cdn_notifications.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var req tc.CDNNotificationRequest
	if userErr = api.Parse(r.Body, tx, &req); userErr != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, nil)
		return
	}

	var resp tc.CDNNotification
	err := tx.QueryRow(insertQuery, req.CDN, inf.User.UserName, req.Notification).Scan(&resp.ID, &resp.CDN, &resp.LastUpdated, &resp.User, &resp.Notification)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	changeLogMsg := fmt.Sprintf("CDN_NOTIFICATION: %s, CDN: %s, ACTION: Created", resp.Notification, resp.CDN)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)

	alertMsg := fmt.Sprintf("CDN notification created [ User = %s ] for CDN: %s", resp.User, resp.CDN)
	alerts := tc.CreateAlerts(tc.SuccessLevel, alertMsg)
	api.WriteAlertsObj(w, r, http.StatusCreated, alerts, resp)
}

// Delete is the handler for DELETE requests to /cdn_notifications.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	alert, respObj, userErr, sysErr, statusCode := deleteCDNNotification(inf)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
		return
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, alert.Text, respObj)
}

func deleteCDNNotification(inf *api.Info) (tc.Alert, tc.CDNNotification, error, error, int) {
	var userErr error
	var sysErr error
	var statusCode = http.StatusOK
	var alert tc.Alert
	var result tc.CDNNotification

	err := inf.Tx.Tx.QueryRow(deleteQuery, inf.Params["id"]).Scan(&result.ID, &result.CDN, &result.LastUpdated, &result.User, &result.Notification)
	if err != nil {
		if err == sql.ErrNoRows {
			userErr = fmt.Errorf("No CDN Notification for %s", inf.Params["id"])
			statusCode = http.StatusNotFound
		} else {
			userErr, sysErr, statusCode = api.ParseDBError(err)
		}

		return alert, result, userErr, sysErr, statusCode
	}

	changeLogMsg := fmt.Sprintf("CDN_NOTIFICATION: %s, CDN: %s, ACTION: Deleted", result.Notification, result.CDN)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, inf.Tx.Tx)

	alertMsg := fmt.Sprintf("CDN notification deleted [ User = %s ] for CDN: %s", result.User, result.CDN)
	alert = tc.Alert{
		Level: tc.SuccessLevel.String(),
		Text:  alertMsg,
	}

	return alert, result, userErr, sysErr, statusCode
}
