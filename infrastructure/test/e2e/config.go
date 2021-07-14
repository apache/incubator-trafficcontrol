package e2e

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
	"encoding/json"
	"errors"
	"os"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

type Config struct {
	TOURI      string `json:"traffic_ops_uri"`
	TOUser     string `json:"traffic_ops_user"`
	TOPass     string `json:"traffic_ops_pass"`
	TOInsecure bool   `json:"traffic_ops_insecure"`
	// IPV4Only is whether to use IPv6 in tests, or to intentionally exclude IPv6 addresses. This is likely required for Docker testing.
	IPV4Only bool `json:"ipv4_only"`
	// DSAssets maps delivery services to a path on that DS which serves a valid file, which may be requested for testing.
	DSAssets map[tc.DeliveryServiceName]string `json:"ds_assets"`
	ConfigLog
}

type ConfigLog struct {
	LogLocationError   string `json:"log_location_error"`
	LogLocationWarning string `json:"log_location_warning"`
	LogLocationInfo    string `json:"log_location_info"`
	LogLocationDebug   string `json:"log_location_debug"`
	LogLocationEvent   string `json:"log_location_event"`
}

func (c *ConfigLog) ErrorLog() log.LogLocation   { return log.LogLocation(c.LogLocationError) }
func (c *ConfigLog) WarningLog() log.LogLocation { return log.LogLocation(c.LogLocationWarning) }
func (c *ConfigLog) InfoLog() log.LogLocation    { return log.LogLocation(c.LogLocationInfo) }
func (c *ConfigLog) DebugLog() log.LogLocation   { return log.LogLocation(c.LogLocationDebug) }
func (c *ConfigLog) EventLog() log.LogLocation   { return log.LogLocation(c.LogLocationEvent) }

func LoadConfig(fileName string) (*Config, error) {
	c := &Config{}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.New("opening file: " + err.Error())
	}
	if err := json.NewDecoder(f).Decode(c); err != nil {
		return nil, errors.New("decoding file: " + err.Error())
	}
	return c, nil
}
