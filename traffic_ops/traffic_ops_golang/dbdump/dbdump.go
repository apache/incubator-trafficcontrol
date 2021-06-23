package dbdump

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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

func filename() string {
	host, err := os.Hostname()
	if err != nil {
		host = "UNKNOWN"
		log.Warnf("Unable to determine hostname: %v", err)
	}

	return fmt.Sprintf("to-backup-%s-%s.pg_dump", host, time.Now().Format(time.RFC3339))
}

func DBDump(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	conf := inf.Config.DB

	pgdump, err := exec.LookPath("pg_dump")
	if err != nil {
		sysErr = fmt.Errorf("Looking up 'pg_dump' executable: %v", err)
		userErr = errors.New("'pg_dump' not available")
		errCode = http.StatusServiceUnavailable
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	cmd := exec.Cmd{
		Path: pgdump,
		Args: []string{
			"--blobs",
			"--no-owner",
			"--format=c",
			fmt.Sprintf("--host=%s", conf.Hostname),
			fmt.Sprintf("--port=%s", conf.Port),
			fmt.Sprintf("--username=%s", conf.User),
			conf.DBName,
		},
		Env: []string{
			fmt.Sprintf("PGPASSWORD=%s", conf.Password),
		},
	}

	out, err := cmd.Output()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			sysErr = fmt.Errorf("subprocess encountered an error, stderr: %s", err.(*exec.ExitError).Stderr)
		default:
			sysErr = fmt.Errorf("subprocess encountered an error: %v", err)
		}
		userErr = errors.New("Subprocess encountered an error")
		errCode = http.StatusBadGateway
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	w.Header().Set(rfc.ContentType, "application/octet-stream;type=pg_dump-data")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename()))
	if out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	w.Write(out)
}
