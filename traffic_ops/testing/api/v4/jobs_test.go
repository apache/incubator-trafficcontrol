package v4

/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestJobs(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, DeliveryServices}, func() {
		CreateTestJobs(t)
		CreateTestInvalidationJobs(t)
		CreateTestInvalidJob(t)
		GetTestJobsQueryParams(t)
		GetTestJobs(t)
		GetTestInvalidationJobs(t)
		JobCollisionWarningTest(t)
	})
}

func CreateTestJobs(t *testing.T) {
	toDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, toDSes.Alerts)
	}

	for i, job := range testData.InvalidationJobs {
		job.StartTime = &tc.Time{
			Time:  time.Now().Add(time.Minute).UTC(),
			Valid: true,
		}
		testData.InvalidationJobs[i] = job
	}

	for _, job := range testData.InvalidationJobs {
		request := tc.InvalidationJobInput{
			DeliveryService: job.DeliveryService,
			Regex:           job.Regex,
			StartTime:       job.StartTime,
			TTL:             job.TTL,
		}
		resp, _, err := TOSession.CreateInvalidationJob(request, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create job: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func JobCollisionWarningTest(t *testing.T) {
	if len(testData.DeliveryServices) < 1 {
		t.Fatal("Need at least one Delivery Service to test Invalidation Job collisions")
	}
	xmlID := testData.DeliveryServices[0].XMLID

	startTime := tc.Time{
		Time:  time.Now().Add(time.Hour),
		Valid: true,
	}
	firstJob := tc.InvalidationJobInput{
		DeliveryService: util.InterfacePtr(&xmlID),
		Regex:           util.StrPtr(`/\.*([A-Z]0?)`),
		TTL:             util.InterfacePtr(16),
		StartTime:       &startTime,
	}

	resp, _, err := TOSession.CreateInvalidationJob(firstJob, client.RequestOptions{})
	if err != nil {
		t.Fatalf("Unexpected error creating a content invalidation Job: %v - alerts: %+v", err, resp.Alerts)
	}

	newTime := tc.Time{
		Time:  startTime.Time.Add(time.Hour),
		Valid: true,
	}
	newJob := tc.InvalidationJobInput{
		DeliveryService: firstJob.DeliveryService,
		Regex:           firstJob.Regex,
		TTL:             firstJob.TTL,
		StartTime:       &newTime,
	}

	alerts, _, err := TOSession.CreateInvalidationJob(newJob, client.RequestOptions{})
	if err != nil {
		t.Fatalf("expected invalidation job create to succeed: %v - %+v", err, alerts.Alerts)
	}

	if len(alerts.Alerts) < 2 {
		t.Fatalf("expected at least 2 alerts on creation, got %v", len(alerts.Alerts))
	}

	found := false
	for _, alert := range alerts.Alerts {
		if alert.Level == tc.WarnLevel.String() && strings.Contains(alert.Text, *firstJob.Regex) {
			found = true
		}
	}
	if !found {
		t.Error("Expected a warning-level error about the regular expression, but couldn't find one")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("deliveryService", xmlID)
	jobs, _, err := TOSession.GetInvalidationJobs(opts)
	if err != nil {
		t.Fatalf("unable to get invalidation jobs: %v - alerts: %+v", err, jobs.Alerts)
	}

	var realJob *tc.InvalidationJob
	for i, job := range jobs.Response {
		if job.StartTime == nil || job.DeliveryService == nil || job.CreatedBy == nil {
			t.Error("Traffic Ops returned a representation of a content invalidation Job that had null or undefined Start Time and/or Delivery Service and/or Created By")
			continue
		}
		diff := newJob.StartTime.Time.Sub(job.StartTime.Time)
		if *job.DeliveryService == xmlID && *job.CreatedBy == "admin" && diff < time.Second {
			realJob = &jobs.Response[i]
			break
		}
	}

	if realJob == nil || realJob.ID == nil || *realJob.ID == 0 {
		t.Fatal("could not find new job")
	}

	newTime.Time = startTime.Time.Add(time.Hour * 2)
	realJob.StartTime = &newTime
	alerts, _, err = TOSession.UpdateInvalidationJob(*realJob, client.RequestOptions{})
	if err != nil {
		t.Fatalf("expected invalidation job update to succeed: %v - alerts: %+v", err, alerts.Alerts)
	}

	if len(alerts.Alerts) < 2 {
		t.Fatalf("expected at least 2 alerts on update, got %v", len(alerts.Alerts))
	}

	found = false
	for _, alert := range alerts.Alerts {
		if alert.Level == tc.WarnLevel.String() && strings.Contains(alert.Text, *firstJob.Regex) {
			found = true
		}
	}
	if !found {
		t.Error("Expected a warning-level error about the regular expression, but couldn't find one")
	}
}

func CreateTestInvalidationJobs(t *testing.T) {
	toDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, toDSes.Alerts)
	}
	dsNameIDs := make(map[string]int64, len(toDSes.Response))
	for _, ds := range toDSes.Response {
		if ds.ID == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service that had null or undefined ID")
			continue
		}
		dsNameIDs[ds.XMLID] = int64(*ds.ID)
	}

	for _, job := range testData.InvalidationJobs {
		if job.DeliveryService == nil {
			t.Error("Found a Job in the test data that has null or undefined Delivery Service")
		}
		_, ok := dsNameIDs[(*job.DeliveryService).(string)]
		if !ok {
			t.Fatalf("can't create test data job: delivery service '%v' not found in Traffic Ops", job.DeliveryService)
		}
		if alerts, _, err := TOSession.CreateInvalidationJob(job, client.RequestOptions{}); err != nil {
			t.Errorf("could not create job: %v - alerts: %+v", err, alerts)
		}
	}
}

func CreateTestInvalidJob(t *testing.T) {
	toDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, toDSes.Alerts)
	}
	dsNameIDs := make(map[string]int64, len(toDSes.Response))
	for _, ds := range toDSes.Response {
		if ds.ID == nil {
			t.Error("Traffic Ops returned a representation of a Delivery Service that had null or undefined ID")
			continue
		}
		dsNameIDs[ds.XMLID] = int64(*ds.ID)
	}

	if len(testData.InvalidationJobs) < 1 {
		t.Fatal("Need at least one Invalidation Job to test creating an invalid Job")
	}
	job := testData.InvalidationJobs[0]
	if job.DeliveryService == nil {
		t.Fatal("Found a Job in the testing data that has null or undefined Delivery Service")
	}
	_, ok := dsNameIDs[(*job.DeliveryService).(string)]
	if !ok {
		t.Fatalf("can't create test data job: delivery service '%v' not found in Traffic Ops", job.DeliveryService)
	}
	maxRevalDays := 0
	foundMaxRevalDays := false
	for _, p := range testData.Parameters {
		if p.Name != "maxRevalDurationDays" {
			continue
		}
		maxRevalDays, err = strconv.Atoi(p.Value)
		if err != nil {
			t.Fatalf("unable to parse maxRevalDurationDays value '%s' to int", p.Value)
		}
		foundMaxRevalDays = true
	}
	if !foundMaxRevalDays {
		t.Fatalf("expected: parameter named maxRevalDurationDays, actual: not found")
	}
	tooHigh := interface{}((maxRevalDays * 24) + 1)
	job.TTL = &tooHigh
	_, reqInf, err := TOSession.CreateInvalidationJob(job, client.RequestOptions{})
	if err == nil {
		t.Error("creating invalid job (TTL higher than maxRevalDurationDays) - expected: error, actual: nil error")
	}
	if reqInf.StatusCode < http.StatusBadRequest || reqInf.StatusCode >= http.StatusInternalServerError {
		t.Errorf("creating invalid job (TTL higher than maxRevalDurationDays) - expected: 400-level status code, actual: %d", reqInf.StatusCode)
	}
}

func GetTestJobsQueryParams(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("deliveryService", "ds2")
	toJobs, _, err := TOSession.GetInvalidationJobs(opts)
	if err != nil {
		t.Fatalf("error getting jobs for Delivery Service 'ds2': %v - alerts: %+v", err, toJobs.Alerts)
	}
	foundOne := false
	for _, j := range toJobs.Response {
		if j.DeliveryService == nil {
			t.Error("expected: non-nil DeliveryService pointer, actual: nil")
		} else if *j.DeliveryService != "ds2" {
			t.Errorf("expected: DeliveryService == ds2, actual: DeliveryService == %s", *j.DeliveryService)
		} else {
			foundOne = true
		}
	}
	if !foundOne {
		t.Error("expected: to find at least one job with deliveryService == ds2, actual: found none")
	}
}

func GetTestJobs(t *testing.T) {
	toJobs, _, err := TOSession.GetInvalidationJobs(client.RequestOptions{})
	if err != nil {
		t.Fatalf("error getting jobs: %v - alerts: %+v", err, toJobs.Alerts)
	}

	toDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, toDSes.Alerts)
	}

	for i, testJob := range testData.InvalidationJobs {
		found := false
		if testJob.DeliveryService == nil {
			t.Errorf("test job (index %v) has nil delivery service", i)
			continue
		} else if testJob.Regex == nil {
			t.Errorf("test job (index %v) has nil regex", i)
			continue
		}
		for j, toJob := range toJobs.Response {
			if toJob.DeliveryService == nil {
				t.Errorf("to job (index %v) has nil delivery service", j)
				continue
			}
			if toJob.AssetURL == nil {
				t.Errorf("to job (index %v) has nil asset url", j)
				continue
			}
			if *toJob.DeliveryService != *testJob.DeliveryService {
				continue
			}
			if !strings.HasSuffix(*toJob.AssetURL, *testJob.Regex) {
				continue
			}
			toJobTime := toJob.StartTime.Round(time.Minute)
			testJobTime := testJob.StartTime.Round(time.Minute)
			if !toJobTime.Equal(testJobTime) {
				t.Errorf("test job ds %v regex %s start time expected '%+v' actual '%+v'", *testJob.DeliveryService, *testJob.Regex, testJobTime, toJobTime)
				continue
			}
			found = true
			break
		}
		if !found {
			t.Errorf("test job ds %v regex %s expected: exists, actual: not found", *testJob.DeliveryService, *testJob.Regex)
		}
	}
}

func GetTestInvalidationJobs(t *testing.T) {
	jobs, _, err := TOSession.GetInvalidationJobs(client.RequestOptions{})
	if err != nil {
		t.Fatalf("error getting invalidation jobs: %v - alerts: %+v", err, jobs.Alerts)
	}

	toDSes, _, err := TOSession.GetDeliveryServices(client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot get Delivery Services: %v - alerts: %+v", err, toDSes.Alerts)
	}

	for _, ds := range toDSes.Response {
		if ds.ID == nil {
			t.Fatal("Erroneous Delivery Service - has invalid ID: <nil>")
		}
		if *ds.ID <= 0 {
			t.Fatalf("Erroneous Delivery Service - has invalid ID: %d", *ds.ID)
		}
	}

	for _, testJob := range testData.InvalidationJobs {
		found := false
		for _, toJob := range jobs.Response {
			if *toJob.DeliveryService != (*testJob.DeliveryService).(string) {
				continue
			}
			if !strings.HasSuffix(*toJob.AssetURL, *testJob.Regex) {
				continue
			}
			if !toJob.StartTime.Round(time.Minute).Equal(testJob.StartTime.Round(time.Minute)) {
				t.Errorf("test invalidation job start time expected '%+v' actual '%+v'", testJob.StartTime, toJob.StartTime)
				continue
			}
			found = true
			break
		}
		if !found {
			t.Errorf("expected a test job %+v to exist, but it didn't", testJob)
		}
	}
}
