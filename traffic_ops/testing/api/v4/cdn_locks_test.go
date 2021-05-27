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
	"net/url"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestCDNLocks(t *testing.T) {
	WithObjs(t, []TCObj{Tenants, Roles, Users, CDNs}, func() {
		CRDCdnLocks(t)
		AdminCdnLocks(t)
	})
}

func getCDNName(t *testing.T) string {
	cdnResp, _, err := TOSession.GetCDNs(client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't get CDNs: %v", err)
	}
	if len(cdnResp.Response) < 1 {
		t.Fatalf("no valid CDNs in response")
	}
	return cdnResp.Response[0].Name
}

func CRDCdnLocks(t *testing.T) {
	cdn := getCDNName(t)
	// CREATE
	var cdnLock tc.CDNLock
	cdnLock.CDN = cdn
	cdnLock.UserName = TOSession.UserName
	cdnLock.Message = util.StrPtr("snapping cdn")
	cdnLock.Soft = util.BoolPtr(true)
	cdnLockResp, _, err := TOSession.CreateCdnLock(cdnLock, client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't create cdn lock: %v", err)
	}
	if cdnLockResp.Response.UserName != cdnLock.UserName {
		t.Errorf("expected username %v, got %v", cdnLock.UserName, cdnLockResp.Response.UserName)
	}
	if cdnLockResp.Response.CDN != cdnLock.CDN {
		t.Errorf("expected cdn %v, got %v", cdnLock.CDN, cdnLockResp.Response.CDN)
	}
	if cdnLockResp.Response.Message == nil {
		t.Errorf("expected a valid message, but got nothing")
	}
	if cdnLockResp.Response.Message != nil && *cdnLockResp.Response.Message != *cdnLock.Message {
		t.Errorf("expected Message %v, got %v", *cdnLock.Message, *cdnLockResp.Response.Message)
	}
	if cdnLockResp.Response.Soft == nil {
		t.Errorf("expected a valid soft/hard setting, but got nothing")
	}
	if cdnLockResp.Response.Soft != nil && *cdnLockResp.Response.Soft != *cdnLock.Soft {
		t.Errorf("expected 'Soft' to be %v, got %v", *cdnLock.Soft, *cdnLockResp.Response.Soft)
	}

	// READ
	cdnLocksReadResp, _, err := TOSession.GetCdnLocks(client.RequestOptions{})
	if err != nil {
		t.Fatalf("could not get CDN Locks: %v", err)
	}
	if len(cdnLocksReadResp.Response) != 1 {
		t.Fatalf("expected to get back one CDN lock, but got %d instead", len(cdnLocksReadResp.Response))
	}
	if cdnLocksReadResp.Response[0].UserName != cdnLock.UserName {
		t.Errorf("expected username %v, got %v", cdnLock.UserName, cdnLocksReadResp.Response[0].UserName)
	}
	if cdnLocksReadResp.Response[0].CDN != cdnLock.CDN {
		t.Errorf("expected cdn %v, got %v", cdnLock.CDN, cdnLocksReadResp.Response[0].CDN)
	}
	if cdnLocksReadResp.Response[0].Message == nil {
		t.Errorf("expected a valid message, but got nothing")
	}
	if cdnLocksReadResp.Response[0].Message != nil && *cdnLocksReadResp.Response[0].Message != *cdnLock.Message {
		t.Errorf("expected Message %v, got %v", *cdnLock.Message, *cdnLocksReadResp.Response[0].Message)
	}
	if cdnLocksReadResp.Response[0].Soft == nil {
		t.Errorf("expected a valid soft/hard setting, but got nothing")
	}
	if cdnLocksReadResp.Response[0].Soft != nil && *cdnLocksReadResp.Response[0].Soft != *cdnLock.Soft {
		t.Errorf("expected 'Soft' to be %v, got %v", *cdnLock.Soft, *cdnLocksReadResp.Response[0].Soft)
	}

	// DELETE
	_, reqInf, err := TOSession.DeleteCdnLocks(client.RequestOptions{QueryParameters: url.Values{"cdn": []string{cdnLock.CDN}}})
	if err != nil {
		t.Fatalf("couldn't delete cdn lock, err: %v", err)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Errorf("expected status code of 200, but got %d instead", reqInf.StatusCode)
	}

}

func AdminCdnLocks(t *testing.T) {
	resp, _, err := TOSession.GetTenants(client.RequestOptions{})
	if err != nil {
		t.Fatalf("could not GET tenants: %v", err)
	}
	if len(resp.Response) == 0 {
		t.Fatalf("didn't get any tenant in response")
	}

	// Create a new user with operations level privileges
	user1 := tc.User{
		Username:             util.StrPtr("lock_user1"),
		RegistrationSent:     tc.TimeNoModFromTime(time.Now()),
		LocalPassword:        util.StrPtr("test_pa$$word"),
		ConfirmLocalPassword: util.StrPtr("test_pa$$word"),
		RoleName:             util.StrPtr("operations"),
	}
	user1.Email = util.StrPtr("email@domain.com")
	user1.TenantID = util.IntPtr(resp.Response[0].ID)
	user1.FullName = util.StrPtr("firstName LastName")
	_, _, err = TOSession.CreateUser(user1, client.RequestOptions{})
	if err != nil {
		t.Fatalf("could not create test user with username: %s", *user1.Username)
	}
	defer ForceDeleteTestUsersByUsernames(t, []string{"lock_user1"})

	// Establish a session with the newly created non admin level user
	userSession, _, err := client.LoginWithAgent(Config.TrafficOps.URL, *user1.Username, *user1.LocalPassword, true, "to-api-v4-client-tests", false, toReqTimeout)
	if err != nil {
		t.Fatalf("could not login with user lock_user1: %v", err)
	}
	cdn := getCDNName(t)
	// Create a lock for this user
	_, _, err = userSession.CreateCdnLock(tc.CDNLock{
		CDN:     cdn,
		Message: util.StrPtr("test lock"),
		Soft:    util.BoolPtr(true),
	}, client.RequestOptions{})
	if err != nil {
		t.Fatalf("couldn't create cdn lock: %v", err)
	}

	// Non admin user trying to hit an admin privileged endpoint -> this should fail
	_, reqInf, err := userSession.AdminDeleteCdnLocks(client.RequestOptions{QueryParameters: url.Values{"cdn": []string{cdn}}})
	if err == nil {
		t.Fatalf("expected error when a non admin user tries to hit an admin privileged endpoint, but got nothing")
	}
	if reqInf.StatusCode != http.StatusForbidden {
		t.Fatalf("expected a 403 status code, but got %d instead", reqInf.StatusCode)
	}

	// Try to delete another user's lock by hitting the normal DELETE endpoint for cdn_locks -> this should fail
	_, reqInf, err = TOSession.DeleteCdnLocks(client.RequestOptions{QueryParameters: url.Values{"cdn": []string{cdn}}})
	if err == nil {
		t.Fatalf("expected an error while deleting other user's lock, but got nothing")
	}
	if reqInf.StatusCode != http.StatusNotFound {
		t.Fatalf("expected a 404 status code, but got %d instead", reqInf.StatusCode)
	}

	// Now try to delete another user's lock by hitting the admin DELETE endpoint for cdn_locks -> this should pass
	_, reqInf, err = TOSession.AdminDeleteCdnLocks(client.RequestOptions{QueryParameters: url.Values{"cdn": []string{cdn}}})
	if err != nil {
		t.Fatalf("expected no error while deleting other user's lock using admin endpoint, but got %v", err)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status code, but got %d instead", reqInf.StatusCode)
	}
}
