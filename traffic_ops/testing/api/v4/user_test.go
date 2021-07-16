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
	"bytes"
	"fmt"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestUsers(t *testing.T) {
	WithObjs(t, []TCObj{Tenants, Parameters, Users}, func() {
		GetTestUsersIMS(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, time)
		SortTestUsers(t)
		UpdateTestUsers(t)
		GetTestUsersIMSAfterChange(t, header)
		RolenameCapitalizationTest(t)
		OpsUpdateAdminTest(t)
		UserSelfUpdateTest(t)
		UserUpdateOwnRoleTest(t)
		GetTestUsers(t)
		GetTestUserCurrent(t)
		UserTenancyTest(t)
		if includeSystemTests {
			// UserRegistrationTest deletes test users before registering new users, so it must come after the other user tests.
			UserRegistrationTest(t)
		}
	})
}

func GetTestUsersIMSAfterChange(t *testing.T, header http.Header) {
	opts := client.NewRequestOptions()
	opts.Header = header
	resp, reqInf, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}

	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)
	opts.Header.Set(rfc.IfModifiedSince, timeStr)

	resp, reqInf, err = TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

const SessionUserName = "admin" // TODO make dynamic?

func GetTestUsersIMS(t *testing.T) {
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)

	resp, reqInf, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func CreateTestUsers(t *testing.T) {
	for _, user := range testData.Users {
		resp, _, err := TOSession.CreateUser(user, client.RequestOptions{})
		if err != nil {
			t.Errorf("could not create user: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func RolenameCapitalizationTest(t *testing.T) {

	roles, _, err := TOSession.GetRoles(client.RequestOptions{})
	if err != nil {
		t.Errorf("could not get roles: %v - alerts: %+v", err, roles.Alerts)
	}
	if len(roles.Response) == 0 {
		t.Fatal("there should be at least one role to test the user")
	}
	if roles.Response[0].ID == nil {
		t.Fatal("Traffic Ops returned a representation of a Role that had null or undefined ID")
	}

	tenants, _, err := TOSession.GetTenants(client.RequestOptions{})
	if err != nil {
		t.Errorf("could not get tenants: %v", err)
	}
	if len(tenants.Response) == 0 {
		t.Fatal("there should be at least one tenant to test the user")
	}

	// this user never does anything, so the role and tenant aren't important
	blob := fmt.Sprintf(`
	{
		"username": "test_user",
		"email": "cooldude6@example.com",
		"fullName": "full name is required",
		"localPasswd": "better_twelve",
		"confirmLocalPasswd": "better_twelve",
		"role": %d,
		"tenantId": %d
	}`, *roles.Response[0].ID, tenants.Response[0].ID)

	reader := strings.NewReader(blob)
	request, err := http.NewRequest("POST", fmt.Sprintf("%v%s/users", TOSession.URL, TestAPIBase), reader)
	if err != nil {
		t.Errorf("could not make new request: %v", err)
	}
	resp, err := TOSession.Client.Do(request)
	if err != nil {
		t.Errorf("could not do request: %v", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	strResp := buf.String()
	if !strings.Contains(strResp, "roleName") {
		t.Error("incorrect json was returned for POST")
	}

	request, err = http.NewRequest("GET", fmt.Sprintf("%v%s/users?username=test_user", TOSession.URL, TestAPIBase), nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	resp, _ = TOSession.Client.Do(request)

	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	strResp = buf.String()
	if !strings.Contains(strResp, "rolename") {
		t.Error("incorrect json was returned for GET")
	}

}

func OpsUpdateAdminTest(t *testing.T) {
	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	opsTOClient, _, err := client.LoginWithAgent(TOSession.URL, "opsuser", "pa$$word", true, "to-api-v3-client-tests/opsuser", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to get log in with opsuser: %v", err.Error())
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("username", "admin")
	resp, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Errorf("cannot get users filtered by username 'admin': %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one user to exist with username 'admin', found: %d", len(resp.Response))
	}
	user := resp.Response[0]
	if user.ID == nil {
		t.Fatal("Traffic Ops returned a representation for the 'admin' user with null or undefined ID")
	}

	fullName := "oops"
	email := "oops@ops.net"
	user.FullName = &fullName
	user.Email = &email

	_, _, err = opsTOClient.UpdateUser(*user.ID, user, client.RequestOptions{})
	if err == nil {
		t.Error("ops user incorrectly updated an admin")
	}
}

func SortTestUsers(t *testing.T) {
	resp, _, err := TOSession.GetUsers(client.RequestOptions{})
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, resp.Alerts)
	}
	sortedList := make([]string, 0, len(resp.Response))
	for _, user := range resp.Response {
		if user.Username == nil {
			t.Errorf("Traffic Ops returned a representation for a user with null or undefined username")
			continue
		}
		sortedList = append(sortedList, *user.Username)
	}

	if !sort.StringsAreSorted(sortedList) {
		t.Errorf("list is not sorted by their names: %v", sortedList)
	}
}

func UserRegistrationTest(t *testing.T) {
	ForceDeleteTestUsers(t)
	var emails []string
	opts := client.NewRequestOptions()
	for _, user := range testData.Users {
		if user.Tenant == nil || user.Role == nil || user.Email == nil {
			t.Error("Found User in the testing data with null or undefined Tenant and/or Role ID and/or Email address")
			continue
		}
		opts.QueryParameters.Set("name", *user.Tenant)
		resp, _, err := TOSession.GetTenants(opts)
		if err != nil {
			t.Fatalf("could not get Tenants filtered by name '%s': %v - alerts: %+v", *user.Tenant, err, resp.Alerts)
		}
		if len(resp.Response) != 1 {
			t.Fatalf("Expected exactly one Tenant to exist with the name '%s', found: %d", *user.Tenant, len(resp.Response))
		}
		tenant := resp.Response[0]

		regResp, _, err := TOSession.RegisterNewUser(uint(tenant.ID), uint(*user.Role), rfc.EmailAddress{Address: mail.Address{Address: *user.Email}}, client.RequestOptions{})
		if err != nil {
			t.Fatalf("could not register user: %v - alerts: %+v", err, regResp.Alerts)
		}
		emails = append(emails, fmt.Sprintf(`'%v'`, *user.Email))
	}

	db, err := OpenConnection()
	if err != nil {
		t.Error("cannot open db")
	}
	defer db.Close()
	q := `DELETE FROM tm_user WHERE email IN (` + strings.Join(emails, ",") + `)`
	if err := execSQL(db, q); err != nil {
		t.Errorf("cannot execute SQL to delete registered users: %s; SQL is %s", err.Error(), q)
	}
}

func UserSelfUpdateTest(t *testing.T) {
	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	opsTOClient, _, err := client.LoginWithAgent(TOSession.URL, "opsuser", "pa$$word", true, "to-api-v3-client-tests/opsuser", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to get log in with opsuser: %v", err.Error())
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("username", "opsuser")
	resp, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("cannot get users filtered by username 'opsuser': %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) < 1 {
		t.Fatalf("no users returned when requesting user 'opsuser'")
	}
	user := resp.Response[0]

	if user.ID == nil {
		t.Fatalf("user 'opsuser' has a null or missing ID - cannot proceed")
	}

	user.FullName = util.StrPtr("Oops-man")
	user.Email = util.StrPtr("operator@not.example.com")

	updateResp, _, err := opsTOClient.UpdateUser(*user.ID, user, client.RequestOptions{})
	if err != nil {
		t.Fatalf("cannot update user: %v - alerts: %+v", err, updateResp.Alerts)
	}

	// Make sure it got updated
	opts.QueryParameters.Del("username")
	opts.QueryParameters.Set("id", strconv.Itoa(*user.ID))
	resp2, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("cannot get users filtered by ID %d: %v - alerts: %+v", *user.ID, err, resp2.Alerts)
	}
	if len(resp2.Response) < 1 {
		t.Fatalf("no results returned when requesting user #%d", *user.ID)
	}
	updatedUser := resp2.Response[0]

	if updatedUser.FullName == nil {
		t.Errorf("user was not correctly updated, FullName is null or missing")
	} else if *updatedUser.FullName != "Oops-man" {
		t.Errorf("results do not match actual: '%s', expected: 'Oops-man'\n", *updatedUser.FullName)
	}

	if updatedUser.Email == nil {
		t.Errorf("user was not correctly updated, Email is null or missing")
	} else if *updatedUser.Email != "operator@not.example.com" {
		t.Errorf("results do not match actual: '%s', expected: 'operator@not.example.com'\n", *updatedUser.Email)
	}

	// Same thing using /user/current
	user.FullName = util.StrPtr("ops-man")
	user.Email = util.StrPtr("operator@example.com")
	updateResp, _, err = opsTOClient.UpdateCurrentUser(user, client.RequestOptions{})
	if err != nil {
		t.Fatalf("error updating current user: %v - alerts: %+v", err, updateResp.Alerts)
	}

	// Make sure it got updated
	resp2, _, err = TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("error getting user #%d: %v - alerts: %+v", *user.ID, err, resp2.Alerts)
	}

	if len(resp2.Response) < 1 {
		t.Fatalf("no user returned when requesting user #%d", *user.ID)
	}

	if resp2.Response[0].FullName == nil {
		t.Errorf("FullName missing or null after update")
	} else if *resp2.Response[0].FullName != "ops-man" {
		t.Errorf("Expected FullName to be 'ops-man', but it was '%s'", *resp2.Response[0].FullName)
	}

	if resp2.Response[0].Email == nil {
		t.Errorf("Email missing or null after update")
	} else if *resp2.Response[0].Email != "operator@example.com" {
		t.Errorf("Expected Email to be restored to 'operator@example.com', but it was '%s'", *resp2.Response[0].Email)
	}

	// now test using an invalid email address
	currentEmail := *user.Email
	user.Email = new(string)
	updateResp, _, err = TOSession.UpdateCurrentUser(user, client.RequestOptions{})
	if err == nil {
		t.Fatal("error was expected updating user with email: '' - got none")
	}

	// Ensure it wasn't actually updated
	resp2, _, err = TOSession.GetUsers(opts)
	if err != nil {
		t.Fatalf("error getting user #%d: %v - alerts: %+v", *user.ID, err, resp2.Alerts)
	}

	if len(resp2.Response) < 1 {
		t.Fatalf("no user returned when requesting user #%d", *user.ID)
	}

	if resp2.Response[0].Email == nil {
		t.Errorf("Email missing or null after update")
	} else if *resp2.Response[0].Email != currentEmail {
		t.Errorf("Expected Email to still be '%s', but it was '%s'", currentEmail, *resp2.Response[0].Email)
	}
}

func UserUpdateOwnRoleTest(t *testing.T) {
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("username", SessionUserName)
	resp, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Errorf("cannot get users filtered by username '%s': %v - alerts: %+v", SessionUserName, err, resp.Alerts)
	}
	user := resp.Response[0]
	if user.Role == nil || user.ID == nil {
		t.Fatalf("Traffic Ops returned a representation for user '%s' with null or undefined Role ID and/or ID", SessionUserName)
	}

	*user.Role = *user.Role + 1
	_, _, err = TOSession.UpdateUser(*user.ID, user, client.RequestOptions{})
	if err == nil {
		t.Error("user incorrectly updated their role")
	}
}

func UpdateTestUsers(t *testing.T) {
	if len(testData.Users) < 1 {
		t.Fatal("Need at least one User to test updating users")
	}
	if testData.Users[0].Username == nil {
		t.Fatal("Found a user in the test data with null or undefined username")
	}
	firstUsername := *testData.Users[0].Username

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("username", firstUsername)
	resp, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Errorf("cannot get users filtered by username '%s': %v - alerts: %+v", firstUsername, err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one user to exist with username '%s', found: %d", firstUsername, len(resp.Response))
	}
	user := resp.Response[0]
	if user.City == nil || user.ID == nil {
		t.Fatal("Traffic Ops returned a representation for a user with null or undefined ID and/or City")
	}
	newCity := "kidz kable kown"
	*user.City = newCity

	var updateResp tc.UpdateUserResponse
	updateResp, _, err = TOSession.UpdateUser(*user.ID, user, client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot update user: %v - alerts: %+v", err, updateResp.Alerts)
	}

	// Make sure it got updated
	opts.QueryParameters.Del("username")
	opts.QueryParameters.Set("id", strconv.Itoa(*user.ID))
	resp2, _, err := TOSession.GetUsers(opts)
	if err != nil {
		t.Errorf("cannot get users filtered by id %d: %v - alerts: %+v", *user.ID, err, resp2.Alerts)
	}
	if len(resp2.Response) != 1 {
		t.Fatalf("Expected exactly one user to exist with ID %d, found: %d", *user.ID, len(resp2.Response))
	}
	updatedUser := resp2.Response[0]
	if updatedUser.City == nil {
		t.Error("Traffic Ops returned a representation of a user with null or undefined City")
	} else if *updatedUser.City != newCity {
		t.Errorf("results do not match actual: %s, expected: %s", *updatedUser.City, newCity)
	}

	if user.RegistrationSent == nil {
		if updatedUser.RegistrationSent != nil {
			t.Errorf("Updated user has registration sent time when original did not (and no registration was sent): %s", *updatedUser.RegistrationSent)
		}
	} else if updatedUser.RegistrationSent == nil {
		t.Errorf("Updated user was supposed to have registration sent time '%s', but it had null or undefined", *user.RegistrationSent)
	} else if *resp.Response[0].RegistrationSent != *resp2.Response[0].RegistrationSent {
		t.Errorf("registration_sent value shouldn't have been updated, expectd: %s, got: %s", *resp.Response[0].RegistrationSent, *resp2.Response[0].RegistrationSent)
	}

}

func GetTestUsers(t *testing.T) {
	resp, _, err := TOSession.GetUsers(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get users: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) < 1 {
		t.Fatalf("expected a users list, got nothing")
	}
	if resp.Response[0].LastAuthenticated == nil {
		t.Errorf("current user's authenticated time, expected: '%s' actual: %v", resp.Response[0].LastAuthenticated, nil)
	}
}

func GetTestUserCurrent(t *testing.T) {
	user, _, err := TOSession.GetUserCurrent(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get current user: %v - alerts: %+v", err, user.Alerts)
	}
	if user.Response.UserName == nil {
		t.Errorf("current user expected: '%s' actual: %v", SessionUserName, nil)
	} else if *user.Response.UserName != SessionUserName {
		t.Errorf("current user expected: '%s' actual: '%s'", SessionUserName, *user.Response.UserName)
	}
	if user.Response.LastAuthenticated == nil {
		t.Errorf("current user's authenticated time, expected: '%s' actual: %v", user.Response.LastAuthenticated, nil)
	}
}

func UserTenancyTest(t *testing.T) {
	users, _, err := TOSession.GetUsers(client.RequestOptions{})
	if err != nil {
		t.Errorf("cannot get users: %v - alerts: %+v", err, users.Alerts)
	}
	tenant3Found := false
	tenant4Found := false
	tenant3Username := "tenant3user"
	tenant4Username := "tenant4user"
	tenant3User := tc.UserV40{}

	// assert admin user can view tenant3user and tenant4user
	for _, user := range users.Response {
		if user.Username == nil || user.ID == nil {
			t.Error("Traffic Ops returned a representation for a user with null or undefined username and/or ID")
			continue
		}
		if *user.Username == tenant3Username {
			tenant3Found = true
			tenant3User = user
		} else if *user.Username == tenant4Username {
			tenant4Found = true
		}
		if tenant3Found && tenant4Found {
			break
		}
	}
	if !tenant3Found || !tenant4Found {
		t.Error("expected admin to be able to view tenants: tenant3 and tenant4")
	}

	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	tenant4TOClient, _, err := client.LoginWithAgent(TOSession.URL, "tenant4user", "pa$$word", true, "to-api-v3-client-tests/tenant4user", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to log in with tenant4user: %v", err.Error())
	}

	usersReadableByTenant4, _, err := tenant4TOClient.GetUsers(client.RequestOptions{})
	if err != nil {
		t.Errorf("tenant4user cannot get users: %v - alerts: %+v", err, usersReadableByTenant4.Alerts)
	}

	tenant4canReadItself := false
	for _, user := range usersReadableByTenant4.Response {
		if user.Username == nil {
			t.Error("Traffic Ops returned a representation of a user with null or undefined username")
			continue
		}
		// assert that tenant4user cannot read tenant3user
		if *user.Username == tenant3Username {
			t.Error("expected tenant4user to be unable to read tenant3user")
		}
		// assert that tenant4user can read itself
		if *user.Username == tenant4Username {
			tenant4canReadItself = true
		}
	}
	if !tenant4canReadItself {
		t.Error("expected tenant4user to be able to read itself")
	}

	// assert that tenant4user cannot update tenant3user
	if _, _, err = tenant4TOClient.UpdateUser(*tenant3User.ID, tenant3User, client.RequestOptions{}); err == nil {
		t.Error("expected tenant4user to be unable to update tenant4user")
	}

	// assert that tenant4user cannot create a user outside of its tenant
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", "root")
	resp, _, err := TOSession.GetTenants(opts)
	if err != nil {
		t.Errorf("Unexpected error getting the root Tenant: %v - alerts: %+v", err, resp.Alerts)
	}
	if len(resp.Response) != 1 {
		t.Fatalf("Expected exactly one Tenant to exist with the name 'root', found: %d", len(resp.Response))
	}
	rootTenant := resp.Response[0]

	if len(testData.Users) < 1 {
		t.Fatal("Need at least one User to continue testing User Tenancy")
	}
	newUser := testData.Users[0]
	newUser.Email = util.StrPtr("testusertenancy@example.com")
	newUser.Username = util.StrPtr("testusertenancy")
	newUser.TenantID = &rootTenant.ID
	if _, _, err = tenant4TOClient.CreateUser(newUser, client.RequestOptions{}); err == nil {
		t.Error("expected tenant4user to be unable to create a new user in the root tenant")
	}
}

// ForceDeleteTestUsers forcibly deletes the users from the db.
func ForceDeleteTestUsers(t *testing.T) {

	// NOTE: Special circumstances!  This should *NOT* be done without a really good reason!
	//  Connects directly to the DB to remove users rather than going thru the client.
	//  This is required here because the DeleteUser action does not really delete users,  but disables them.
	db, err := OpenConnection()
	if err != nil {
		t.Error("cannot open db")
	}
	defer db.Close()

	var usernames []string
	for _, user := range testData.Users {
		usernames = append(usernames, `'`+*user.Username+`'`)
	}

	// there is a constraint that prevents users from being deleted when they have a log
	q := `DELETE FROM log WHERE NOT tm_user = (SELECT id FROM tm_user WHERE username = 'admin')`
	err = execSQL(db, q)
	if err != nil {
		t.Errorf("cannot execute SQL: %s; SQL is %s", err.Error(), q)
	}

	q = `DELETE FROM tm_user WHERE username IN (` + strings.Join(usernames, ",") + `)`
	err = execSQL(db, q)
	if err != nil {
		t.Errorf("cannot execute SQL: %s; SQL is %s", err.Error(), q)
	}
}

func ForceDeleteTestUsersByUsernames(t *testing.T, usernames []string) {

	// NOTE: Special circumstances!  This should *NOT* be done without a really good reason!
	//  Connects directly to the DB to remove users rather than going thru the client.
	//  This is required here because the DeleteUser action does not really delete users,  but disables them.
	db, err := OpenConnection()
	if err != nil {
		t.Error("cannot open db")
	}
	defer db.Close()

	for i, u := range usernames {
		usernames[i] = `'` + u + `'`
	}
	// there is a constraint that prevents users from being deleted when they have a log
	q := `DELETE FROM log WHERE NOT tm_user = (SELECT id FROM tm_user WHERE username = 'admin')`
	err = execSQL(db, q)
	if err != nil {
		t.Errorf("cannot execute SQL: %s; SQL is %s", err.Error(), q)
	}

	q = `DELETE FROM tm_user WHERE username IN (` + strings.Join(usernames, ",") + `)`
	err = execSQL(db, q)
	if err != nil {
		t.Errorf("cannot execute SQL: %s; SQL is %s", err.Error(), q)
	}
}

func DeleteTestUsers(t *testing.T) {
	opts := client.NewRequestOptions()
	for _, user := range testData.Users {
		if user.Username == nil {
			t.Error("Found a user in the testing data with null or undefined username")
			continue
		}
		opts.QueryParameters.Set("username", *user.Username)
		resp, _, err := TOSession.GetUsers(opts)
		if err != nil {
			t.Errorf("cannot get users filtered by username '%s': %v - alerts: %+v", *user.Username, err, resp.Alerts)
		}
		if len(resp.Response) > 0 {
			respUser := resp.Response[0]
			if respUser.ID == nil {
				t.Error("Traffic Ops returned a representation for a user with null or undefined ID")
				continue
			}

			delResp, _, err := TOSession.DeleteUser(*respUser.ID, client.RequestOptions{})
			if err != nil {
				t.Errorf("cannot delete user '%s': %v - alerts: %+v", *user.Username, err, delResp.Alerts)
			}

			// Make sure it got deleted
			resp, _, err := TOSession.GetUsers(opts)
			if err != nil {
				t.Errorf("error getting users filtered by username after supposed deletion: %v - alerts: %+v", err, resp.Alerts)
			}
			if len(resp.Response) > 0 {
				t.Errorf("expected user: %s to be deleted", *user.Username)
			}
		}
	}
}
