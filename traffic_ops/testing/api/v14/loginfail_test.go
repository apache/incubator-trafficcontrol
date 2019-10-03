package v14

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
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"golang.org/x/net/publicsuffix"

	toclient "github.com/apache/trafficcontrol/traffic_ops/client"
)

func TestLoginFail(t *testing.T) {
	WithObjs(t, []TCObj{CDNs}, func() {
		PostTestLoginFail(t)
		LoginWithEmptyCredentialsTest(t)
	})
	WithObjs(t, []TCObj{Users, Roles})
}

func PostTestLoginFail(t *testing.T) {
	// This specifically tests a previous bug: auth failure returning a 200, causing the client to think the request succeeded, and deserialize no matching fields successfully, and return an empty object.

	userAgent := "to-api-v14-client-tests-loginfailtest"
	uninitializedTOClient, err := getUninitializedTOClient(Config.TrafficOps.Users.Admin, Config.TrafficOps.UserPassword, Config.TrafficOps.URL, userAgent, time.Second*time.Duration(Config.Default.Session.TimeoutInSecs))
	if err != nil {
		t.Fatalf("getting uninitialized client: %+v\n", err)
	}

	if len(testData.CDNs) < 1 {
		t.Fatalf("cannot test login: must have at least 1 test data cdn\n")
	}
	expectedCDN := testData.CDNs[0]
	actualCDNs, _, err := uninitializedTOClient.GetCDNByName(expectedCDN.Name)
	if err != nil {
		t.Fatalf("GetCDNByName err expected nil, actual '%+v'\n", err)
	}
	if len(actualCDNs) < 1 {
		t.Fatalf("uninitialized client should have retried login (possibly login failed with a 200, so it didn't try again, and the CDN request returned an auth failure with a 200, which the client reasonably thought was success, and deserialized with no matching keys, resulting in an empty object); len(actualCDNs) expected >1, actual 0")
	}
	actualCDN := actualCDNs[0]
	if expectedCDN.Name != actualCDN.Name {
		t.Fatalf("cdn.Name expected '%+v' actual '%+v'\n", expectedCDN.Name, actualCDN.Name)
	}
}

func LoginWithEmptyCredentialsTest(t *testing.T) {
	userAgent := "to-api-v14-client-tests-loginfailtest"
	_, _, err := toclient.LoginWithAgent(Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, "", true, userAgent, false, time.Second*time.Duration(Config.Default.Session.TimeoutInSecs))
	if err == nil {
		t.Fatalf("expected error when logging in with empty credentials, actual nil")
	}
}

func LoginWithTokenTest(t *testing.T) {
	userAgent := "to-api-v14-client-tests-loginfailtest"
	s, _, err := toclient.LoginWithToken(Config.TrafficOps.URL, "test", true, userAgent, false, time.Second*time.Duration(Config.Default.Session.TimeoutInSecs))
	if err != nil {
		t.Fatalf("unexpected error when logging in with a token: %v", err)
	}
	if s == nil {
		t.Fatalf("returned client was nil")
	}

	// disallowed token
	_, _, err := toclient.LoginWithToken(Config.TrafficOps.URL, "quest", true, userAgent, false, time.Second*time.Duration(Config.Default.Session.TimeoutInSecs))
	if err == nil {
		t.Fatalf("expected an error when logging in with a disallowed token, actual nil")
	}

	// nonexistent token
	_, _, err := toclient.LoginWithToken(Config.TrafficOps.URL, "notarealtoken", true, userAgent, false, time.Second*time.Duration(Config.Default.Session.TimeoutInSecs))
	if err == nil {
		t.Fatalf("expected an error when logging in with a nonexistent token, actual nil")
	}
}

func getUninitializedTOClient(user, pass, uri, agent string, reqTimeout time.Duration) (*toclient.Session, error) {
	insecure := true
	useCache := false
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}
	return toclient.NewSession(user, pass, uri, agent, &http.Client{
		Timeout: reqTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
		Jar: jar,
	}, useCache), nil
}
