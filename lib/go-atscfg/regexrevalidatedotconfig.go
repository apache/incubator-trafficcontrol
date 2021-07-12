package atscfg

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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const RegexRemapPrefix = "regex_remap_"
const CacheUrlPrefix = "cacheurl_"
const RefetchSuffix = "##REFETCH##"
const RefreshSuffix = "##REFRESH##"

const RemapFile = "remap.config"

const RegexRevalidateFileName = "regex_revalidate.config"
const RegexRevalidateMaxRevalDurationDaysParamName = "maxRevalDurationDays"
const DefaultMaxRevalDurationDays = 90
const JobKeywordPurge = "PURGE"
const RegexRevalidateMinTTL = time.Hour

const ContentTypeRegexRevalidateDotConfig = ContentTypeTextASCII
const LineCommentRegexRevalidateDotConfig = LineCommentHash

// RegexRevalidateDotConfigOpts contains settings to configure generation options.
type RegexRevalidateDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeRegexRevalidateDotConfig(
	server *Server,
	deliveryServices []DeliveryService,
	globalParams []tc.Parameter,
	jobs []tc.Job,
	opt *RegexRevalidateDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &RegexRevalidateDotConfigOpts{}
	}
	warnings := []string{}

	if server.CDNName == nil {
		return Cfg{}, makeErr(warnings, "server CDNName missing")
	}

	params := paramsToMultiMap(filterParams(globalParams, RegexRevalidateFileName, "", "", ""))

	dsNames := map[string]struct{}{}
	for _, ds := range deliveryServices {
		if ds.XMLID == nil {
			warnings = append(warnings, "got Delivery Service from Traffic Ops with a nil xmlId! Skipping!")
			continue
		}
		dsNames[*ds.XMLID] = struct{}{}
	}

	dsJobs := []tc.Job{}
	for _, job := range jobs {
		if _, ok := dsNames[job.DeliveryService]; !ok {
			continue
		}
		dsJobs = append(dsJobs, job)
	}

	// TODO: add cdn, startTime query params to /jobs endpoint

	maxDays := DefaultMaxRevalDurationDays
	if maxDaysStrs := params[RegexRevalidateMaxRevalDurationDaysParamName]; len(maxDaysStrs) > 0 {
		sort.Strings(maxDaysStrs)
		err := error(nil)
		if maxDays, err = strconv.Atoi(maxDaysStrs[0]); err != nil { // just use the first, if there were multiple params
			warnings = append(warnings, "max days param '"+maxDaysStrs[0]+"' is not an integer, using default value!")
			maxDays = DefaultMaxRevalDurationDays
		}
	}

	maxReval := time.Duration(maxDays) * time.Hour * 24

	cfgJobs, jobWarns := filterJobs(dsJobs, maxReval, RegexRevalidateMinTTL)
	warnings = append(warnings, jobWarns...)

	txt := makeHdrComment(opt.HdrComment)
	for _, job := range cfgJobs {
		txt += job.AssetURL + " " + strconv.FormatInt(job.PurgeEnd.Unix(), 10)
		if job.Type != "" {
			txt += " " + job.Type
		}
		txt += "\n"
	}

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeRegexRevalidateDotConfig,
		LineComment: LineCommentRegexRevalidateDotConfig,
		Warnings:    warnings,
	}, nil
}

type revalJob struct {
	AssetURL string
	PurgeEnd time.Time
	Type     string // MISS or STALE (default)
}

type jobsSort []revalJob

func (jb jobsSort) Len() int      { return len(jb) }
func (jb jobsSort) Swap(i, j int) { jb[i], jb[j] = jb[j], jb[i] }
func (jb jobsSort) Less(i, j int) bool {
	if jb[i].AssetURL == jb[j].AssetURL {
		return jb[i].PurgeEnd.Before(jb[j].PurgeEnd)
	}
	return strings.Compare(jb[i].AssetURL, jb[j].AssetURL) < 0
}

// filterJobs returns only jobs which:
//   - have a non-null deliveryservice
//   - have parameters of the form TTL:%dh
//   - have a start time later than (now + maxReval days). That is, we don't query jobs older than maxReval in the past.
//   - are "purge" jobs
//   - have a start_time+ttl > now. That is, jobs that haven't expired yet.
// Returns the filtered jobs, and any warnings.
func filterJobs(tc_jobs []tc.Job, maxReval time.Duration, minTTL time.Duration) ([]revalJob, []string) {
	warnings := []string{}

	jobMap := map[string]revalJob{}

	for _, tc_job := range tc_jobs {
		if tc_job.DeliveryService == "" {
			continue
		}
		if !strings.HasPrefix(tc_job.Parameters, `TTL:`) {
			continue
		}
		if !strings.HasSuffix(tc_job.Parameters, `h`) {
			continue
		}

		ttlHoursStr := tc_job.Parameters
		ttlHoursStr = strings.TrimPrefix(ttlHoursStr, `TTL:`)
		ttlHoursStr = strings.TrimSuffix(ttlHoursStr, `h`)
		ttlHours, err := strconv.Atoi(ttlHoursStr)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("job %+v has unexpected parameters ttl format, config generation skipping!\n", tc_job))
			continue
		}

		ttl := time.Duration(ttlHours) * time.Hour
		if ttl > maxReval {
			ttl = maxReval
		} else if ttl < minTTL {
			ttl = minTTL
		}

		jobStartTime, err := time.Parse(tc.JobTimeFormat, tc_job.StartTime)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("job %+v has unexpected time format, config generation skipping!\n", tc_job))
			continue
		}

		if jobStartTime.Add(maxReval).Before(time.Now()) {
			continue
		}

		if jobStartTime.Add(ttl).Before(time.Now()) {
			continue
		}
		if tc_job.Keyword != JobKeywordPurge {
			continue
		}

		// process the __REFETCH__ keyword
		assetURL := tc_job.AssetURL
		var jobType string

		if strings.HasSuffix(assetURL, RefetchSuffix) {
			assetURL = strings.TrimSuffix(assetURL, RefetchSuffix)
			jobType = "MISS"
		} else if strings.HasSuffix(assetURL, RefreshSuffix) { // also default
			assetURL = strings.TrimSuffix(assetURL, RefreshSuffix)
			jobType = "STALE"
		}

		purgeEnd := jobStartTime.Add(ttl)

		if rjob, ok := jobMap[assetURL]; !ok || purgeEnd.After(rjob.PurgeEnd) {
			jobMap[assetURL] = revalJob{AssetURL: assetURL, PurgeEnd: purgeEnd, Type: jobType}
		}
	}

	newJobs := []revalJob{}
	for _, rjob := range jobMap {
		newJobs = append(newJobs, rjob)
	}
	sort.Sort(jobsSort(newJobs))

	return newJobs, warnings
}
