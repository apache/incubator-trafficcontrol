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

package v3

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

const queryParamFormat = "?profileId=%d&parameterId=%d"

func TestProfileParameters(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Profiles, ProfileParameters}, func() {
		t.Run("Get /profileparameters with the If-Modified-Since HTTP header", GetTestProfileParametersIMS)
		t.Run("Basic GET request", GetTestProfileParameters)
		t.Run("Attempt to create an invalid Profile-Parameter relationship", InvalidCreateTestProfileParameters)
	})
}

func GetTestProfileParametersIMS(t *testing.T) {
	var header http.Header
	header = make(map[string][]string)
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)
	header.Set(rfc.IfModifiedSince, time)
	for _, pp := range testData.ProfileParameters {
		queryParams := fmt.Sprintf(queryParamFormat, pp.ProfileID, pp.ParameterID)
		_, reqInf, err := TOSession.GetProfileParameterByQueryParamsWithHdr(queryParams, header)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err.Error())
		}
		if reqInf.StatusCode != http.StatusNotModified {
			t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
		}
	}
}

func CreateTestProfileParameters(t *testing.T) {

	firstProfile := testData.Profiles[0]
	profileResp, _, err := TOSession.GetProfileByNameWithHdr(firstProfile.Name, nil)
	if err != nil {
		t.Errorf("cannot GET Profile by name: %v - %v", firstProfile.Name, err)
	}

	firstParameter := testData.Parameters[0]
	paramResp, _, err := TOSession.GetParameterByNameWithHdr(firstParameter.Name, nil)
	if err != nil {
		t.Errorf("cannot GET Parameter by name: %v - %v", firstParameter.Name, err)
	}

	profileID := profileResp[0].ID
	parameterID := paramResp[0].ID

	pp := tc.ProfileParameter{
		ProfileID:   profileID,
		ParameterID: parameterID,
	}
	resp, _, err := TOSession.CreateProfileParameter(pp)
	t.Log("Response: ", resp)
	if err != nil {
		t.Errorf("could not CREATE profile parameters: %v", err)
	}

}

func InvalidCreateTestProfileParameters(t *testing.T) {
	pp := tc.ProfileParameter{
		ProfileID:   0,
		ParameterID: 0,
	}
	_, _, err := TOSession.CreateProfileParameter(pp)
	if err == nil {
		t.Fatalf("creating invalid profile parameter - expected: error, actual: nil")
	}
	if !strings.Contains(err.Error(), "profileId") {
		t.Errorf("expected: error message to contain 'profileId', actual: %v", err)
	}
	if !strings.Contains(err.Error(), "parameterId") {
		t.Errorf("expected: error message to contain 'parameterId', actual: %v", err)
	}

}

func GetTestProfileParameters(t *testing.T) {

	for _, pp := range testData.ProfileParameters {
		queryParams := fmt.Sprintf(queryParamFormat, pp.ProfileID, pp.ParameterID)
		resp, _, err := TOSession.GetProfileParameterByQueryParamsWithHdr(queryParams, nil)
		if err != nil {
			t.Errorf("cannot GET Parameter by name: %v - %v", err, resp)
		}
	}
}

func DeleteTestProfileParameters(t *testing.T) {

	for _, pp := range testData.ProfileParameters {
		DeleteTestProfileParameter(t, pp)
	}
}

func DeleteTestProfileParameter(t *testing.T, pp tc.ProfileParameter) {

	queryParams := fmt.Sprintf(queryParamFormat, pp.ProfileID, pp.ParameterID)
	// Retrieve the PtofileParameter by profile so we can get the id for the Update
	resp, _, err := TOSession.GetProfileParameterByQueryParamsWithHdr(queryParams, nil)
	if err != nil {
		t.Errorf("cannot GET Parameter by profile: %v - %v", pp.Profile, err)
	}
	if len(resp) > 0 {
		respPP := resp[0]

		delResp, _, err := TOSession.DeleteParameterByProfileParameter(respPP.ProfileID, respPP.ParameterID)
		if err != nil {
			t.Errorf("cannot DELETE Parameter by profile: %v - %v", err, delResp)
		}

		// Retrieve the Parameter to see if it got deleted
		pps, _, err := TOSession.GetProfileParameterByQueryParamsWithHdr(queryParams, nil)
		if err != nil {
			t.Errorf("error deleting Parameter name: %s", err.Error())
		}
		if len(pps) > 0 {
			t.Errorf("expected Parameter Name: %s and ConfigFile: %s to be deleted", pp.Profile, pp.Parameter)
		}
	}
}
