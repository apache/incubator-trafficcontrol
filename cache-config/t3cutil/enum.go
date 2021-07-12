package t3cutil

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
	"strings"
)

// ServiceNeeds represents whether we need to reload or restart Traffic Server,
// as returned by t3c-check-reload.
//
// Note the default is success, not invalid.
// With enums, typically invalid should be the default, but in this case, this represents the return of an
// app which returns nothing on successes (which is typical of POSIX apps),
// so we want the default empty string to represent that.
// Hence, the default value of ServiceNeeds is success, not invalid
//
type ServiceNeeds string

const (
	ServiceNeedsNothing ServiceNeeds = "" // default is nothing, and print nothing if nothing needs done
	ServiceNeedsRestart ServiceNeeds = "restart"
	ServiceNeedsReload  ServiceNeeds = "reload"
	ServiceNeedsInvalid ServiceNeeds = "invalid"
)

func (sn ServiceNeeds) String() string { return string(sn) }

func StrToServiceNeeds(str string) ServiceNeeds {
	switch str {
	case string(ServiceNeedsNothing):
		return ServiceNeedsNothing
	case string(ServiceNeedsRestart):
		return ServiceNeedsRestart
	case string(ServiceNeedsReload):
		return ServiceNeedsReload
	default:
		return ServiceNeedsInvalid
	}
}

type ApplyServiceActionFlag string

const (
	ApplyServiceActionFlagNone    ApplyServiceActionFlag = "none"
	ApplyServiceActionFlagReload  ApplyServiceActionFlag = "reload"
	ApplyServiceActionFlagRestart ApplyServiceActionFlag = "restart"
	ApplyServiceActionFlagInvalid ApplyServiceActionFlag = ""
)

func (af ApplyServiceActionFlag) String() string { return string(af) }

func StrToApplyServiceActionFlag(str string) ApplyServiceActionFlag {
	switch str {
	case string(ApplyServiceActionFlagNone):
		return ApplyServiceActionFlagNone
	case string(ApplyServiceActionFlagReload):
		return ApplyServiceActionFlagReload
	case string(ApplyServiceActionFlagRestart):
		return ApplyServiceActionFlagRestart
	default:
		return ApplyServiceActionFlagInvalid
	}
}

type ApplyFilesFlag string

const (
	ApplyFilesFlagInvalid ApplyFilesFlag = ""
	ApplyFilesFlagAll     ApplyFilesFlag = "all"
	ApplyFilesFlagReval   ApplyFilesFlag = "reval"
)

func (ff ApplyFilesFlag) String() string {
	return string(ff)
}

func StrToApplyFilesFlag(str string) ApplyFilesFlag {
	switch ApplyFilesFlag(strings.TrimSpace(strings.ToLower(str))) {
	case ApplyFilesFlagAll:
		return ApplyFilesFlagAll
	case ApplyFilesFlagReval:
		return ApplyFilesFlagReval
	default:
		return ApplyFilesFlagInvalid
	}
}

// Mode is the t3c run mode - syncds, badass, etc.
type Mode string

const (
	ModeInvalid    Mode = ""
	ModeBadAss     Mode = "badass"
	ModeReport     Mode = "report"
	ModeRevalidate Mode = "revalidate"
	ModeSyncDS     Mode = "syncds"
)

func (md Mode) String() string {
	return string(md)
}

func StrToMode(str string) Mode {
	switch Mode(strings.ToLower(str)) {
	case ModeBadAss:
		return ModeBadAss
	case ModeReport:
		return ModeReport
	case ModeRevalidate:
		return ModeRevalidate
	case ModeSyncDS:
		return ModeSyncDS
	default:
		return ModeInvalid
	}
}
