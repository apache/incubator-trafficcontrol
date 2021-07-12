package orttest

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
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/cache-config/t3cutil"
)

func TestCheckReload(t *testing.T) {
	type argsResults struct {
		configs     []string
		packages    []string
		mode        string
		expected    string
		expectedErr bool
	}
	argsExpected := []argsResults{
		{
			configs:  []string{"/etc/trafficserver/remap.config", "/etc/trafficserver/parent.config"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"/etc/trafficserver/anything.foo"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"/opt/trafficserver/etc/trafficserver/anything.foo"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"/foo/bar/hdr_rw_foo.config"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"/foo/bar/uri_signing_dsname.config"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"/foo/bar/url_sig_dsname.config", "foo"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"plugin.config", "foo"},
			packages: nil,
			expected: "restart",
		},
		{
			configs:  []string{"/etc/trafficserver/anything.foo"},
			packages: []string{"anything"},
			expected: "restart",
		},
		{
			configs:  nil,
			packages: []string{"anything"},
			expected: "restart",
		},
		{
			configs:  nil,
			packages: []string{"anything", "anythingelse"},
			expected: "restart",
		},
		{
			configs:  []string{"/foo/bar/ssl_multicert.config"},
			packages: nil,
			expected: "reload",
		},
		{
			configs:  []string{"foo"},
			packages: nil,
			expected: "",
		},
		{
			configs:  []string{"/foo/bar/baz.config"},
			packages: nil,
			expected: "",
		},
	}

	for _, ae := range argsExpected {
		out, code := t3cCheckReload(ae.configs, ae.packages)
		out = strings.TrimSpace(out)
		if !ae.expectedErr && code != 0 {
			t.Errorf("expected configs %+v packages %+v would not error, actual: code %v output '%v'", ae.configs, ae.packages, code, out)
			continue
		} else if ae.expectedErr && code == 0 {
			t.Errorf("expected configs %+v packages %+v would error, actual: no error", ae.configs, ae.packages)
			continue
		}
		if out != ae.expected {
			t.Errorf("expected configs %+v packages %+v would need '%v', actual: '%v'", ae.configs, ae.packages, ae.expected, out)
		}
	}
}

func t3cCheckReload(changedConfigPaths []string, packagesInstalled []string) (string, int) {
	args := []string{
		"check", "reload",
		"--changed-config-paths=" + strings.Join(changedConfigPaths, ","),
		"--plugin-packages-installed=" + strings.Join(packagesInstalled, ","),
	}
	stdOut, _, exitCode := t3cutil.Do("t3c", args...)
	return string(stdOut), exitCode
}
