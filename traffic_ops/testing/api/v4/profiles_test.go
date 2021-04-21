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

package v4

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	tc "github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

func TestProfiles(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Profiles, Parameters}, func() {
		CreateBadProfiles(t)
		UpdateTestProfiles(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfUnmodifiedSince, time)
		UpdateTestProfilesWithHeaders(t, header)
		GetTestProfilesIMS(t)
		GetTestProfiles(t)
		GetTestProfilesWithParameters(t)
		ImportProfile(t)
		CopyProfile(t)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestProfilesWithHeaders(t, header)
		GetTestPaginationSupportProfiles(t)
	})
}

func UpdateTestProfilesWithHeaders(t *testing.T, header http.Header) {
	if len(testData.Profiles) > 0 {
		firstProfile := testData.Profiles[0]
		// Retrieve the Profile by name so we can get the id for the Update
		resp, _, err := TOSession.GetProfileByName(firstProfile.Name, header)
		if err != nil {
			t.Errorf("cannot GET Profile by name: %v - %v", firstProfile.Name, err)
		}
		if len(resp) > 0 {
			remoteProfile := resp[0]
			_, reqInf, err := TOSession.UpdateProfile(remoteProfile.ID, remoteProfile, header)
			if err == nil {
				t.Errorf("Expected error about precondition failed, but got none")
			}
			if reqInf.StatusCode != http.StatusPreconditionFailed {
				t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
			}
		}
	} else {
		t.Errorf("No data available to update")
	}
}

func GetTestProfilesIMS(t *testing.T) {
	var header http.Header
	header = make(map[string][]string)
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)
	header.Set(rfc.IfModifiedSince, time)
	for _, pr := range testData.Profiles {
		_, reqInf, err := TOSession.GetProfileByName(pr.Name, header)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if reqInf.StatusCode != http.StatusNotModified {
			t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
		}
	}
}

// CreateBadProfiles ensures that profiles can't be created with bad values
func CreateBadProfiles(t *testing.T) {

	// blank profile
	prs := []tc.Profile{
		{Type: "", Name: "", Description: "", CDNID: 0},
		{Type: "ATS_PROFILE", Name: "badprofile", Description: "description", CDNID: 0},
		{Type: "ATS_PROFILE", Name: "badprofile", Description: "", CDNID: 1},
		{Type: "ATS_PROFILE", Name: "", Description: "description", CDNID: 1},
		{Type: "", Name: "badprofile", Description: "description", CDNID: 1},
	}

	for _, pr := range prs {
		resp, _, err := TOSession.CreateProfile(pr)

		if err == nil {
			t.Errorf("Creating bad profile succeeded: %+v\nResponse is %+v", pr, resp)
		}
	}
}

func CopyProfile(t *testing.T) {
	testCases := []struct {
		description  string
		profile      tc.ProfileCopy
		expectedResp string
		err          string
	}{
		{
			description: "copy profile",
			profile: tc.ProfileCopy{
				Name:         "profile-2",
				ExistingName: "EDGE1",
			},
			expectedResp: "created new profile [profile-2] from existing profile [EDGE1]",
		},
		{
			description: "existing profile does not exist",
			profile: tc.ProfileCopy{
				Name:         "profile-3",
				ExistingName: "bogus",
			},
			err: "profile with name bogus does not exist",
		},
		{
			description: "new profile already exists",
			profile: tc.ProfileCopy{
				Name:         "EDGE2",
				ExistingName: "EDGE1",
			},
			err: "profile with name EDGE2 already exists",
		},
	}

	var newProfileNames []string
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			resp, _, err := TOSession.CopyProfile(c.profile)
			if c.err != "" {
				if err != nil && !strings.Contains(err.Error(), c.err) {
					t.Fatalf("got err= %s; expected err= %s", err, c.err)
				}
			} else if err != nil {
				t.Fatalf("got err= %s; expected err= nil", err)
			}

			if err == nil {
				if got, want := resp.Alerts.ToStrings()[0], c.expectedResp; got != want {
					t.Fatalf("got= %s; expected= %s", got, want)
				}

				newProfileNames = append(newProfileNames, c.profile.Name)
			}
		})
	}

	// Cleanup profiles
	for _, name := range newProfileNames {
		profiles, _, err := TOSession.GetProfileByName(name, nil)
		if err != nil {
			t.Fatalf("got err= %s; expected err= nil", err)
		}
		if len(profiles) == 0 {
			t.Errorf("could not GET profile %+v: not found", name)
		}
		_, _, err = TOSession.DeleteProfile(profiles[0].ID)
		if err != nil {
			t.Fatalf("got err= %s; expected err= nil", err)
		}
	}
}

func CreateTestProfiles(t *testing.T) {

	for _, pr := range testData.Profiles {
		_, _, err := TOSession.CreateProfile(pr)

		if err != nil {
			t.Errorf("could not CREATE profiles with name: %s %v", pr.Name, err)
		}
		profiles, _, err := TOSession.GetProfileByName(pr.Name, nil)
		if err != nil {
			t.Errorf("could not GET profile with name: %s %v", pr.Name, err)
		}
		if len(profiles) == 0 {
			t.Errorf("could not GET profile %+v: not found", pr)
		}
		profileID := profiles[0].ID

		opts := client.NewRequestOptions()
		for _, param := range pr.Parameters {
			if param.Name == nil || param.Value == nil || param.ConfigFile == nil {
				t.Errorf("invalid parameter specification: %+v", param)
				continue
			}
			alerts, _, err := TOSession.CreateParameter(tc.Parameter{Name: *param.Name, Value: *param.Value, ConfigFile: *param.ConfigFile}, client.RequestOptions{})
			if err != nil {
				found := false
				for _, alert := range alerts.Alerts {
					if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
						found = true
						break
					}
				}
				// ok if already exists
				if !found {
					t.Errorf("Unexpected error creating Parameter %+v: %v - alerts: %+v", param, err, alerts.Alerts)
					continue
				}
			}
			opts.QueryParameters.Set("name", *param.Name)
			opts.QueryParameters.Set("configFile", *param.ConfigFile)
			opts.QueryParameters.Set("value", *param.Value)
			p, _, err := TOSession.GetParameters(opts)
			if err != nil {
				t.Errorf("could not get Parameter %+v: %v - alerts: %+v", param, err, p.Alerts)
			}
			if len(p.Response) == 0 {
				t.Fatalf("could not get parameter %+v: not found", param)
			}
			req := tc.ProfileParameterCreationRequestV4{ProfileID: profileID, ParameterID: p.Response[0].ID}
			alerts, _, err = TOSession.CreateProfileParameter(req, client.RequestOptions{})
			if err != nil {
				t.Errorf("could not associate Parameter %+v with Profile #%d: %v - alerts: %+v", param, profileID, err, alerts.Alerts)
			}
		}

	}
}

func UpdateTestProfiles(t *testing.T) {

	firstProfile := testData.Profiles[0]
	// Retrieve the Profile by name so we can get the id for the Update
	resp, _, err := TOSession.GetProfileByName(firstProfile.Name, nil)
	if err != nil {
		t.Errorf("cannot GET Profile by name: %v - %v", firstProfile.Name, err)
	}
	if len(resp) > 0 {
		cdns, _, err := TOSession.GetCDNByName("cdn2", nil)
		remoteProfile := resp[0]
		oldName := remoteProfile.Name

		if len(cdns) > 0 {
			expectedProfileDesc := "UPDATED"
			expectedCDNId := cdns[0].ID
			expectedName := "testing"
			expectedRoutingDisabled := true
			expectedType := "TR_PROFILE"

			remoteProfile.Description = expectedProfileDesc
			remoteProfile.Type = expectedType
			remoteProfile.CDNID = expectedCDNId
			remoteProfile.Name = expectedName
			remoteProfile.RoutingDisabled = expectedRoutingDisabled

			var alert tc.Alerts
			alert, _, err = TOSession.UpdateProfile(remoteProfile.ID, remoteProfile, nil)
			if err != nil {
				t.Errorf("cannot UPDATE Profile by id: %v - %v", err, alert)
			}

			// Retrieve the Profile to check Profile name got updated
			resp, _, err = TOSession.GetProfileByID(remoteProfile.ID, nil)
			if err != nil {
				t.Errorf("cannot GET Profile by name: %v - %v", firstProfile.Name, err)
			}

			if len(resp) > 0 {
				respProfile := resp[0]
				if respProfile.Description != expectedProfileDesc {
					t.Errorf("results do not match actual: %s, expected: %s", respProfile.Description, expectedProfileDesc)
				}
				if respProfile.Type != expectedType {
					t.Errorf("results do not match actual: %s, expected: %s", respProfile.Type, expectedType)
				}
				if respProfile.CDNID != expectedCDNId {
					t.Errorf("results do not match actual: %d, expected: %d", respProfile.CDNID, expectedCDNId)
				}
				if respProfile.RoutingDisabled != expectedRoutingDisabled {
					t.Errorf("results do not match actual: %t, expected: %t", respProfile.RoutingDisabled, expectedRoutingDisabled)
				}
				if respProfile.Name != expectedName {
					t.Errorf("results do not match actual: %v, expected: %v", respProfile.Name, expectedName)
				}
				respProfile.Name = oldName
				alert, _, err = TOSession.UpdateProfile(respProfile.ID, respProfile, nil)
				if err != nil {
					t.Errorf("cannot UPDATE Profile by id: %v - %v", err, alert)
				}
			}
		}
	}
}

func GetTestProfiles(t *testing.T) {

	for _, pr := range testData.Profiles {
		resp, _, err := TOSession.GetProfileByName(pr.Name, nil)
		if err != nil {
			t.Errorf("cannot GET Profile by name: %v - %v", err, resp)
		}
		profileID := resp[0].ID

		// TODO: figure out what the 'Parameter' field of a Profile is and how
		// passing it to this is supposed to work.
		// resp, _, err = TOSession.GetProfileByParameter(pr.Parameter)
		// if err != nil {
		// 	t.Errorf("cannot GET Profile by param: %v - %v", err, resp)
		// }

		resp, _, err = TOSession.GetProfilesByCDNID(pr.CDNID, nil)
		if err != nil {
			t.Errorf("cannot GET Profile by cdn: %v - %v", err, resp)
		}

		// Export Profile
		exportResp, _, err := TOSession.ExportProfile(profileID)
		if err != nil {
			t.Errorf("error exporting Profile: %v - %v", profileID, err)
		}
		if exportResp == nil {
			t.Error("error exporting Profile: response nil")
		}
	}
}

func ImportProfile(t *testing.T) {
	// Get ID of Profile to export
	resp, _, err := TOSession.GetProfileByName(testData.Profiles[0].Name, nil)
	if err != nil {
		t.Fatalf("cannot GET Profile by name: %v - %v", err, resp)
	}
	if resp == nil {
		t.Fatal("error getting Profile: response nil")
	}
	if len(resp) != 1 {
		t.Fatalf("Profiles expected 1, actual %v", len(resp))
	}
	profileID := resp[0].ID

	// Export Profile to import
	exportResp, _, err := TOSession.ExportProfile(profileID)
	if err != nil {
		t.Fatalf("error exporting Profile: %v - %v", profileID, err)
	}
	if exportResp == nil {
		t.Fatal("error exporting Profile: response nil")
	}

	// Modify Profile and import

	// Add parameter and change name
	profile := exportResp.Profile
	profile.Name = util.StrPtr("TestProfileImport")

	newParam := tc.ProfileExportImportParameterNullable{
		ConfigFile: util.StrPtr("config_file_import_test"),
		Name:       util.StrPtr("param_import_test"),
		Value:      util.StrPtr("import_test"),
	}
	parameters := append(exportResp.Parameters, newParam)
	// Import Profile
	importReq := tc.ProfileImportRequest{
		Profile:    profile,
		Parameters: parameters,
	}
	importResp, _, err := TOSession.ImportProfile(&importReq)
	if err != nil {
		t.Fatalf("error importing Profile: %v - %v", profileID, err)
	}
	if importResp == nil {
		t.Error("error importing Profile: response nil")
	}

	// Add newly create profile and parameter to testData so it gets deleted
	testData.Profiles = append(testData.Profiles, tc.Profile{
		Name:        *profile.Name,
		CDNName:     *profile.CDNName,
		Description: *profile.Description,
		Type:        *profile.Type,
	})

	testData.Parameters = append(testData.Parameters, tc.Parameter{
		ConfigFile: *newParam.ConfigFile,
		Name:       *newParam.Name,
		Value:      *newParam.Value,
	})
}

func GetTestProfilesWithParameters(t *testing.T) {
	firstProfile := testData.Profiles[0]
	resp, _, err := TOSession.GetProfileByName(firstProfile.Name, nil)
	if err != nil {
		t.Errorf("cannot GET Profile by name: %v - %v", err, resp)
		return
	}
	if len(resp) == 0 {
		t.Errorf("cannot GET Profile by name: not found - %v", resp)
		return
	}
	respProfile := resp[0]
	// query by name does not retrieve associated parameters.  But query by id does.
	resp, _, err = TOSession.GetProfileByID(respProfile.ID, nil)
	if err != nil {
		t.Errorf("cannot GET Profile by name: %v - %v", err, resp)
	}
	if len(resp) > 0 {
		respProfile = resp[0]
		respParameters := respProfile.Parameters
		if len(respParameters) == 0 {
			t.Errorf("expected a profile with parameters to be retrieved: %v - %v", err, respParameters)
		}
	}
}

func DeleteTestProfiles(t *testing.T) {

	for _, pr := range testData.Profiles {
		// Retrieve the Profile by name so we can get the id for the Update
		resp, _, err := TOSession.GetProfileByName(pr.Name, nil)
		if err != nil {
			t.Errorf("cannot GET Profile by name: %s - %v", pr.Name, err)
			continue
		}
		if len(resp) == 0 {
			t.Errorf("cannot GET Profile by name: not found - %s", pr.Name)
			continue
		}

		profileID := resp[0].ID
		// query by name does not retrieve associated parameters.  But query by id does.
		resp, _, err = TOSession.GetProfileByID(profileID, nil)
		if err != nil {
			t.Errorf("cannot GET Profile by id: %v - %v", err, resp)
			continue
		}
		if len(resp) < 1 {
			t.Errorf("Traffic Ops returned no Profiles with ID %d", profileID)
			continue
		}
		// delete any profile_parameter associations first
		// the parameter is what's being deleted, but the delete is cascaded to profile_parameter
		for _, param := range resp[0].Parameters {
			if param.ID == nil {
				t.Error("Traffic Ops responded with a representation of a Parameter with null or undefined ID")
				continue
			}
			alerts, _, err := TOSession.DeleteParameter(*param.ID, client.RequestOptions{})
			if err != nil {
				t.Errorf("cannot delete Parameter #%d: %v - alerts: %+v", *param.ID, err, alerts.Alerts)
			}
		}
		delResp, _, err := TOSession.DeleteProfile(profileID)
		if err != nil {
			t.Errorf("cannot DELETE Profile by name: %v - %v", err, delResp)
		}
		//time.Sleep(1 * time.Second)

		// Retrieve the Profile to see if it got deleted
		prs, _, err := TOSession.GetProfileByName(pr.Name, nil)
		if err != nil {
			t.Errorf("error deleting Profile name: %s", err)
		}
		if len(prs) > 0 {
			t.Errorf("expected Profile Name: %s to be deleted", pr.Name)
		}

		// Attempt to export Profile
		_, _, err = TOSession.ExportProfile(profileID)
		if err == nil {
			t.Errorf("export deleted profile %s - expected: error, actual: nil", pr.Name)
		}
	}
}

func GetTestPaginationSupportProfiles(t *testing.T) {
	qparams := url.Values{}
	qparams.Set("orderby", "id")
	profiles, _, err := TOSession.GetProfiles(qparams, nil)
	if err != nil {
		t.Errorf("cannot GET Profiles: %v", err)
	}

	if len(profiles) > 0 {
		qparams = url.Values{}
		qparams.Set("orderby", "id")
		qparams.Set("limit", "1")
		profilesWithLimit, _, err := TOSession.GetProfiles(qparams, nil)
		if err == nil {
			if !reflect.DeepEqual(profiles[:1], profilesWithLimit) {
				t.Error("expected GET Profiles with limit = 1 to return first result")
			}
		} else {
			t.Error("Error in getting Profiles by limit")
		}
		if len(profiles) > 1 {
			qparams = url.Values{}
			qparams.Set("orderby", "id")
			qparams.Set("limit", "1")
			qparams.Set("offset", "1")
			profilesWithOffset, _, err := TOSession.GetProfiles(qparams, nil)
			if err == nil {
				if !reflect.DeepEqual(profiles[1:2], profilesWithOffset) {
					t.Error("expected GET Profiles with limit = 1, offset = 1 to return second result")
				}
			} else {
				t.Error("Error in getting Profiles by limit and offset")
			}

			qparams = url.Values{}
			qparams.Set("orderby", "id")
			qparams.Set("limit", "1")
			qparams.Set("page", "2")
			profilesWithPage, _, err := TOSession.GetProfiles(qparams, nil)
			if err == nil {
				if !reflect.DeepEqual(profiles[1:2], profilesWithPage) {
					t.Error("expected GET Profiles with limit = 1, page = 2 to return second result")
				}
			} else {
				t.Error("Error in getting Profiles by limit and page")
			}
		} else {
			t.Errorf("only one Profiles found, so offset functionality can't test")
		}
	} else {
		t.Errorf("No Profiles found to check pagination")
	}

	qparams = url.Values{}
	qparams.Set("limit", "-2")
	_, _, err = TOSession.GetProfiles(qparams, nil)
	if err == nil {
		t.Error("expected GET Profiles to return an error when limit is not bigger than -1")
	} else if !strings.Contains(err.Error(), "must be bigger than -1") {
		t.Errorf("expected GET Profiles to return an error for limit is not bigger than -1, actual error: " + err.Error())
	}

	qparams = url.Values{}
	qparams.Set("limit", "1")
	qparams.Set("offset", "0")
	_, _, err = TOSession.GetProfiles(qparams, nil)
	if err == nil {
		t.Error("expected GET Profiles to return an error when offset is not a positive integer")
	} else if !strings.Contains(err.Error(), "must be a positive integer") {
		t.Errorf("expected GET Profiles to return an error for offset is not a positive integer, actual error: " + err.Error())
	}

	qparams = url.Values{}
	qparams.Set("limit", "1")
	qparams.Set("page", "0")
	_, _, err = TOSession.GetProfiles(qparams, nil)
	if err == nil {
		t.Error("expected GET Profiles to return an error when page is not a positive integer")
	} else if !strings.Contains(err.Error(), "must be a positive integer") {
		t.Errorf("expected GET Profiles to return an error for page is not a positive integer, actual error: " + err.Error())
	}
}
