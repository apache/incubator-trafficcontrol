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

package v13

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

var (
	db *sql.DB
)

// OpenConnection ...
func OpenConnection() (*sql.DB, error) {
	var err error
	sslStr := "require"
	if !Config.TrafficOpsDB.SSL {
		sslStr = "disable"
	}

	db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", Config.TrafficOpsDB.User, Config.TrafficOpsDB.Password, Config.TrafficOpsDB.Hostname, Config.TrafficOpsDB.Name, sslStr))

	if err != nil {
		log.Errorf("opening database: %v\n", err)
		return nil, fmt.Errorf("transaction failed: %s", err)
	}
	return db, err
}

// SetupTestData ...
func SetupTestData(*sql.DB) error {
	var err error

	err = SetupRoles(db)
	if err != nil {
		fmt.Printf("\nError setting up roles %s - %s, %v\n", Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, err)
		os.Exit(1)
	}

	err = SetupCapabilities(db)
	if err != nil {
		fmt.Printf("\nError setting up capabilities %s - %s, %v\n", Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, err)
		os.Exit(1)
	}

	err = SetupRoleCapabilities(db)
	if err != nil {
		fmt.Printf("\nError setting up roleCapabilities %s - %s, %v\n", Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, err)
		os.Exit(1)
	}

	err = SetupTenants(db)
	if err != nil {
		fmt.Printf("\nError setting up tenant %s - %s, %v\n", Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, err)
		os.Exit(1)
	}

	err = SetupTmusers(db)
	if err != nil {
		fmt.Printf("\nError setting up tm_user %s - %s, %v\n", Config.TrafficOps.URL, Config.TrafficOps.Users.Admin, err)
		os.Exit(1)
	}

	return err
}

// SetupRoles ...
func SetupRoles(db *sql.DB) error {

	sqlStmt := `
INSERT INTO role (name, description, priv_level) VALUES ('disallowed','Block all access',0) ON CONFLICT DO NOTHING;
INSERT INTO role (name, description, priv_level) VALUES ('read-only user','Block all access', 10) ON CONFLICT DO NOTHING;
INSERT INTO role (name, description, priv_level) VALUES ('operations','Block all access', 20) ON CONFLICT DO NOTHING;
INSERT INTO role (name, description, priv_level) VALUES ('admin','super-user', 30) ON CONFLICT DO NOTHING;
INSERT INTO role (name, description, priv_level) VALUES ('portal','Portal User', 2) ON CONFLICT DO NOTHING;
INSERT INTO role (name, description, priv_level) VALUES ('federation','Role for Secondary CZF', 15) ON CONFLICT DO NOTHING;
`
	err := execSQL(db, sqlStmt, "role")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

func SetupCapabilities(db *sql.DB) error {
	sqlStmt := `
insert into capability (name, description) values ('auth', 'Ability to authenticate') ON CONFLICT (name) DO NOTHING;
-- api endpoints
insert into capability (name, description) values ('api-endpoints-read', 'Ability to view api endpoints') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('api-endpoints-write', 'Ability to edit api endpoints') ON CONFLICT (name) DO NOTHING;
-- asns
insert into capability (name, description) values ('asns-read', 'Ability to view asns') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('asns-write', 'Ability to edit asns') ON CONFLICT (name) DO NOTHING;
-- cache config files
insert into capability (name, description) values ('cache-config-files-read', 'Ability to view cache config files') ON CONFLICT (name) DO NOTHING;
-- cache groups
insert into capability (name, description) values ('cache-groups-read', 'Ability to view cache groups') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('cache-groups-write', 'Ability to edit cache groups') ON CONFLICT (name) DO NOTHING;
-- capabilities
insert into capability (name, description) values ('capabilities-read', 'Ability to view capabilities') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('capabilities-write', 'Ability to edit capabilities') ON CONFLICT (name) DO NOTHING;
-- cdns
insert into capability (name, description) values ('cdns-read', 'Ability to view cdns') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('cdns-write', 'Ability to edit cdns') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('cdns-snapshot', 'Ability to snapshot a cdn') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('cdn-security-keys-read', 'Ability to view cdn security keys') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('cdn-security-keys-write', 'Ability to edit cdn security keys') ON CONFLICT (name) DO NOTHING;
-- change logs
insert into capability (name, description) values ('change-logs-read', 'Ability to view change logs') ON CONFLICT (name) DO NOTHING;
-- coordinates
insert into capability (name, description) values ('coordinates-read', 'Ability to view coordinates') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('coordinates-write', 'Ability to edit coordinates') ON CONFLICT (name) DO NOTHING;
-- delivery services
insert into capability (name, description) values ('delivery-services-read', 'Ability to view delivery services') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('delivery-services-write', 'Ability to view delivery services') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('delivery-service-security-keys-read', 'Ability to view delivery service security keys') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('delivery-service-security-keys-write', 'Ability to edit delivery service security keys') ON CONFLICT (name) DO NOTHING;
-- delivery service requests
insert into capability (name, description) values ('delivery-service-requests-read', 'Ability to view delivery service requests') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('delivery-service-requests-write', 'Ability to edit delivery service requests') ON CONFLICT (name) DO NOTHING;
-- delivery service servers
insert into capability (name, description) values ('delivery-service-servers-read', 'Ability to view delivery service / server assignments') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('delivery-service-servers-write', 'Ability to edit delivery service / server assignments') ON CONFLICT (name) DO NOTHING;
-- divisions
insert into capability (name, description) values ('divisions-read', 'Ability to view divisions') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('divisions-write', 'Ability to edit divisions') ON CONFLICT (name) DO NOTHING;
-- extensions
insert into capability (name, description) values ('to-extensions-read', 'Ability to view extensions') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('to-extensions-write', 'Ability to edit extensions') ON CONFLICT (name) DO NOTHING;
-- federations
insert into capability (name, description) values ('federations-read', 'Ability to view federations') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('federations-write', 'Ability to edit federations') ON CONFLICT (name) DO NOTHING;
-- hardware info
insert into capability (name, description) values ('hwinfo-read', 'Ability to view hardware info') ON CONFLICT (name) DO NOTHING;
-- iso
insert into capability (name, description) values ('iso-generate', 'Ability to generate isos') ON CONFLICT (name) DO NOTHING;
-- jobs
insert into capability (name, description) values ('jobs-read', 'Ability to view jobs (invalidation requests)') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('jobs-write', 'Ability to edit jobs (invalidation requests)') ON CONFLICT (name) DO NOTHING;
-- misc
insert into capability (name, description) values ('db-dump', 'Ability to get a copy of the database') ON CONFLICT (name) DO NOTHING;
-- origins
insert into capability (name, description) values ('origins-read', 'Ability to view origins') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('origins-write', 'Ability to edit origins') ON CONFLICT (name) DO NOTHING;
-- parameters
insert into capability (name, description) values ('parameters-read', 'Ability to view parameters') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('parameters-write', 'Ability to edit parameters') ON CONFLICT (name) DO NOTHING;
-- phys locations
insert into capability (name, description) values ('phys-locations-read', 'Ability to view phys locations') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('phys-locations-write', 'Ability to edit phys locations') ON CONFLICT (name) DO NOTHING;
-- profiles
insert into capability (name, description) values ('profiles-read', 'Ability to view profiles') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('profiles-write', 'Ability to edit profiles') ON CONFLICT (name) DO NOTHING;
-- regions
insert into capability (name, description) values ('regions-read', 'Ability to view regions') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('regions-write', 'Ability to edit regions') ON CONFLICT (name) DO NOTHING;
-- riak
insert into capability (name, description) values ('riak', 'Riak') ON CONFLICT (name) DO NOTHING;
-- roles
insert into capability (name, description) values ('roles-read', 'Ability to view roles') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('roles-write', 'ABILITY TO EDIT ROLES.') ON CONFLICT (name) DO NOTHING;
-- servers
insert into capability (name, description) values ('servers-read', 'Ability to view servers') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('servers-write', 'Ability to edit servers') ON CONFLICT (name) DO NOTHING;
-- stats
insert into capability (name, description) values ('stats-read', 'Ability to view cache stats') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('stats-write', 'Ability to edit cache stats') ON CONFLICT (name) DO NOTHING;
-- statuses
insert into capability (name, description) values ('statuses-read', 'Ability to view statuses') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('statuses-write', 'Ability to edit statuses') ON CONFLICT (name) DO NOTHING;
-- static dns entries
insert into capability (name, description) values ('static-dns-entries-read', 'Ability to view static dns entries') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('static-dns-entries-write', 'Ability to edit static dns entries') ON CONFLICT (name) DO NOTHING;
-- steering
insert into capability (name, description) values ('steering-read', 'Ability to view steering') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('steering-write', 'Ability to edit steering') ON CONFLICT (name) DO NOTHING;
-- steering targets
insert into capability (name, description) values ('steering-targets-read', 'Ability to view steering targets') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('steering-targets-write', 'Ability to edit steering targets') ON CONFLICT (name) DO NOTHING;
-- system info
insert into capability (name, description) values ('system-info-read', 'Ability to view system info') ON CONFLICT (name) DO NOTHING;
-- tenants
insert into capability (name, description) values ('tenants-read', 'Ability to view tenants') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('tenants-write', 'Ability to edit tenants') ON CONFLICT (name) DO NOTHING;
-- types
insert into capability (name, description) values ('types-read', 'Ability to view types') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('types-write', 'Ability to edit types') ON CONFLICT (name) DO NOTHING;
-- users
insert into capability (name, description) values ('users-register', 'Ability to register new users') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('users-read', 'Ability to view users') ON CONFLICT (name) DO NOTHING;
insert into capability (name, description) values ('users-write', 'Ability to edit users') ON CONFLICT (name) DO NOTHING;

INSERT INTO capability (name, description) values ('about-read', 'Ability to read the server about information') ON CONFLICT (name) DO NOTHING;

INSERT INTO capability (name, description) values ('parameters-read-secure', 'Ability to view secure parameter values') ON CONFLICT (name) DO NOTHING;
INSERT INTO capability (name, description) values ('servers-read-secure', 'Ability to view secure server values') ON CONFLICT (name) DO NOTHING;

-- auth
insert into api_capability (http_method,  route, capability) values ('POST', 'user/login', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'user/login/token', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'user/logout', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'user/reset_password', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'user/current', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'user/current', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'user/current/update', 'auth') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- api endpoints
insert into api_capability (http_method, route, capability) values ('GET', 'api_capabilities', 'api-endpoints-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'api_capabilities/*', 'api-endpoints-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'api_capabilities', 'api-endpoints-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'api_capabilities/*', 'api-endpoints-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'api_capabilities/*', 'api-endpoints-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- asns
insert into api_capability (http_method, route, capability) values ('GET', 'asns', 'asns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'asns/*', 'asns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'asns', 'asns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'asns/*', 'asns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'asns/*', 'asns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- cache config files
insert into api_capability (http_method, route, capability) values ('GET', 'servers/*/configfiles/ats', 'cache-config-files-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/*/configfiles/ats/*', 'cache-config-files-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/*/configfiles/ats/*', 'cache-config-files-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/configfiles/ats/*', 'cache-config-files-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- cache groups
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups/trimmed', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups/*', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cachegroups', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'cachegroups/*', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cachegroups/*', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cachegroups/*/queue_update', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cachegroups/*/deliveryservices', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups/*/parameters', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups/*/unassigned_parameters', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroup/*/parameter', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroupparameters', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cachegroupparameters', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cachegroupparameters/*/*', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroups/*/parameter/available', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cachegroup_fallbacks', 'cache-groups-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'cachegroup_fallbacks', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cachegroup_fallbacks', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cachegroup_fallbacks', 'cache-groups-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- capabilities
insert into api_capability (http_method, route, capability) values ('GET', 'capabilities', 'capabilities-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'capabilities/*', 'capabilities-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'capabilities', 'capabilities-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'capabilities/*', 'capabilities-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'capabilities/*', 'capabilities-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- cdns
insert into api_capability (http_method, route, capability) values ('GET', 'cdns', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/name/*', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cdns', 'cdns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'cdns/*', 'cdns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cdns/*', 'cdns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cdns/name/*', 'cdns-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cdns/*/queue_update', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/snapshot', 'cdns-snapshot') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/snapshot/new', 'cdns-snapshot') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'cdns/*/snapshot', 'cdns-snapshot') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'snapshot/*', 'cdns-snapshot') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/configs', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/configs/routing', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/configs/monitoring', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/domains', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/health', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/health', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/capacity', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/routing', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/name/*/sslkeys', 'cdn-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/usage/overview', 'cdns-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/name/*/dnsseckeys', 'cdn-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cdns/dnsseckeys/generate', 'cdn-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/name/*/dnsseckeys/delete', 'cdn-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- change logs
insert into api_capability (http_method, route, capability) values ('GET', 'logs', 'change-logs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'logs/*/days', 'change-logs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'logs/newcount', 'change-logs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- coordinates
insert into api_capability (http_method, route, capability) values ('GET', 'coordinates', 'coordinates-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'coordinates', 'coordinates-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'coordinates', 'coordinates-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'coordinates', 'coordinates-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- delivery services
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservices/*', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservices/*/safe', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservices/*', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/health', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/capacity', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/routing', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/state', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservice_stats', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/xmlId/*/sslkeys', 'delivery-service-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/hostname/#hostname/sslkeys', 'delivery-service-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/sslkeys/generate', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/sslkeys/add', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/xmlId/*/sslkeys/delete', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/urlkeys', 'delivery-service-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/xmlId/*/urlkeys', 'delivery-service-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/xmlId/*/urlkeys/generate', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/xmlId/*/urlkeys/copyFromXmlId/*', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/urisignkeys', 'delivery-service-security-keys-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/*/urisignkeys', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservices/*/urisignkeys', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservices/*/urisignkeys', 'delivery-service-security-keys-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- delivery service regexes
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservice_matches', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices_regexes', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/regexes', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/regexes/*', 'delivery-services-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/*/regexes', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservices/*/regexes/*', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservices/*/regexes/*', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- delivery service requests
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservice_requests', 'delivery-service-requests-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservice_requests', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservice_requests', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservice_requests', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservice_requests/*/assign', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservice_requests/*/status', 'delivery-services-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/request', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservice_request_comments', 'delivery-service-requests-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservice_request_comments', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'deliveryservice_request_comments', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservice_request_comments', 'delivery-service-requests-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- delivery service servers
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryserviceserver', 'delivery-service-servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryserviceserver', 'delivery-service-servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservices/*/servers', 'delivery-service-servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservice_server/*/*', 'delivery-service-servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- divisions
insert into api_capability (http_method, route, capability) values ('GET', 'divisions', 'divisions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'divisions/*', 'divisions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'divisions/name/*', 'divisions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'divisions', 'divisions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'divisions/*', 'divisions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'divisions/*', 'divisions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'divisions/name/*', 'divisions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- extensions
insert into api_capability (http_method, route, capability) values ('GET', 'to_extensions', 'to-extensions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'to_extensions', 'to-extensions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'to_extensions/*/delete', 'to-extensions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- federations
insert into api_capability (http_method, route, capability) values ('GET', 'federations', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'federations', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'federations', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'federations', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/federations', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'cdns/*/federations/*', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'cdns/*/federations', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'cdns/*/federations/*', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'cdns/*/federations/*', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'federations/*/users', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'federations/*/users', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'federations/*/users/*', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'federations/*/deliveryservices', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'federations/*/deliveryservices', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'federations/*/deliveryservices/*', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'federations/*/federation_resolvers', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'federations/*/federation_resolvers', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'federation_resolvers', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'federation_resolvers/*', 'federations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- hardware info
insert into api_capability (http_method, route, capability) values ('GET', 'hwinfo', 'hwinfo-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- iso
insert into api_capability (http_method, route, capability) values ('GET', 'osversions', 'iso-generate') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'isos', 'iso-generate') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- jobs
insert into api_capability (http_method, route, capability) values ('GET', 'jobs', 'jobs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'jobs/*', 'jobs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'user/current/jobs', 'jobs-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'user/current/jobs', 'jobs-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- misc
insert into api_capability (http_method, route, capability) values ('GET', 'dbdump', 'db-dump') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- origins
insert into api_capability (http_method, route, capability) values ('GET', 'origins', 'origins-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'origins', 'origins-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'origins', 'origins-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'origins', 'origins-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- parameters
insert into api_capability (http_method, route, capability) values ('GET', 'parameters', 'parameters-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'parameters/*', 'parameters-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'parameters', 'parameters-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'parameterprofile', 'parameters-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'parameters/*', 'parameters-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'parameters/*', 'parameters-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'parameters/*/validate', 'parameters-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'parameters/*/profiles', 'parameters-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'parameters/*/unassigned_profiles', 'parameters-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- phys locations
insert into api_capability (http_method, route, capability) values ('GET', 'phys_locations', 'phys-locations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'phys_locations/trimmed', 'phys-locations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'phys_locations/*', 'phys-locations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'phys_locations', 'phys-locations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'regions/*/phys_locations', 'phys-locations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'phys_locations/*', 'phys-locations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'phys_locations/*', 'phys-locations-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- profiles
insert into api_capability (http_method, route, capability) values ('GET', 'profiles', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/trimmed', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/*', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profiles', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'profiles/*', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'profiles/*', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profiles/name/*/copy/*', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profiles/import', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/*/export', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/*/parameters', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/*/unassigned_parameters', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profiles/name/*/parameters', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'parameters/profile/*', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profiles/name/*/parameters', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profiles/*/parameters', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'profileparameters', 'profiles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profileparameters', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'profileparameter', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'profileparameters/*/*', 'profiles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- regions
insert into api_capability (http_method, route, capability) values ('GET', 'regions', 'regions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'regions/*', 'regions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'regions/name/*', 'regions-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'regions', 'regions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'divisions/*/regions', 'regions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'regions/*', 'regions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'regions/*', 'regions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'regions/name/*', 'regions-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- riak
insert into api_capability (http_method, route, capability) values ('GET', 'riak/ping', 'riak') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'keys/ping', 'riak') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'riak/bucket/*/key/*/values', 'riak') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'riak/stats', 'riak') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- roles
insert into api_capability (http_method, route, capability) values ('GET', 'roles', 'roles-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'roles', 'roles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'roles', 'roles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'roles', 'roles-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- servers
insert into api_capability (http_method, route, capability) values ('GET', 'servers', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/*', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'servers', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'servers/*', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'servers/*', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/servers', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/unassigned_servers', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'deliveryservices/*/servers/eligible', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/details', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/hostname/*/details', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/totals', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/status', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'servers/*/queue_update', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'servers/*/status', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'servers/*/update_status', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'servers/checks', 'servers-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'servercheck', 'servers-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- stats
insert into api_capability (http_method, route, capability) values ('GET', 'caches/stats', 'stats-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'stats_summary', 'stats-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'stats_summary/create', 'stats-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'traffic_monitor/stats', 'stats-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- statuses
insert into api_capability (http_method, route, capability) values ('GET', 'statuses', 'statuses-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'statuses/*', 'statuses-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'statuses', 'statuses-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'statuses/*', 'statuses-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'statuses/*', 'statuses-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- static dns entries
insert into api_capability (http_method, route, capability) values ('GET', 'staticdnsentries', 'static-dns-entries-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'staticdnsentries', 'static-dns-entries-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'staticdnsentries', 'static-dns-entries-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'staticdnsentries', 'static-dns-entries-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- steering targets
insert into api_capability (http_method, route, capability) values ('GET', 'steering/*/targets', 'steering-targets-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'steering/*/targets/*', 'steering-targets-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'steering/*/targets', 'steering-targets-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'steering/*/targets/*', 'steering-targets-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'steering/*/targets/*', 'steering-targets-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- system info
insert into api_capability (http_method, route, capability) values ('GET', 'system/info', 'system-info-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- tenants
insert into api_capability (http_method, route, capability) values ('GET', 'tenants', 'tenants-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'tenants/*', 'tenants-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'tenants', 'tenants-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'tenants/*', 'tenants-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'tenants/*', 'tenants-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- types
insert into api_capability (http_method, route, capability) values ('GET', 'types', 'types-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'types/trimmed', 'types-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'types/*', 'types-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'types', 'types-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'types/*', 'types-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'types/*', 'types-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
-- users
insert into api_capability (http_method, route, capability) values ('GET', 'users', 'users-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'users/*', 'users-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'users', 'users-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('PUT', 'users/*', 'users-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'users/register', 'users-register') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'users/*/deliveryservices', 'users-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'users/*/deliveryservices/available', 'users-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('POST', 'deliveryservice_user', 'users-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('DELETE', 'deliveryservice_user/*/*', 'users-write') ON CONFLICT (http_method, route, capability) DO NOTHING;
insert into api_capability (http_method, route, capability) values ('GET', 'about', 'about-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
`
	err := execSQL(db, sqlStmt, "capability")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

func SetupRoleCapabilities(db *sql.DB) error {
	sqlStmt := `
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'auth') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'api-endpoints-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'api-endpoints-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'asns-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'asns-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cache-config-files-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cache-groups-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cache-groups-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'capabilities-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'capabilities-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cdns-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cdns-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cdns-snapshot') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cdn-security-keys-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'cdn-security-keys-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'change-logs-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'coordinates-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'coordinates-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-services-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-services-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-security-keys-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-security-keys-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-requests-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-requests-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-servers-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'delivery-service-servers-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'divisions-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'divisions-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'to-extensions-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'to-extensions-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'federations-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'federations-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'hwinfo-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'jobs-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'jobs-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'iso-generate') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'db-dump') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'origins-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'origins-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'parameters-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'parameters-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'phys-locations-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'phys-locations-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'profiles-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'profiles-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'regions-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'regions-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'riak') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'roles-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'roles-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'servers-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'servers-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'stats-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'stats-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'statuses-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'statuses-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'static-dns-entries-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'static-dns-entries-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'steering-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'steering-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'steering-targets-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'steering-targets-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'system-info-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'tenants-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'tenants-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'types-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'types-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'users-register') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'users-read') ON CONFLICT (role_id, cap_name) DO NOTHING;
insert into role_capability (role_id, cap_name) values ((select id from role where name='admin'), 'users-write') ON CONFLICT (role_id, cap_name) DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'about-read') ON CONFLICT (role_id, cap_name) DO NOTHING;

INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'parameters-read-secure') ON CONFLICT (role_id, cap_name) DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'servers-read-secure') ON CONFLICT (role_id, cap_name) DO NOTHING;
`
	err := execSQL(db, sqlStmt, "role_capability")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupTmusers ...
func SetupTmusers(db *sql.DB) error {

	var err error
	encryptedPassword, err := auth.DerivePassword(Config.TrafficOps.UserPassword)
	if err != nil {
		return fmt.Errorf("password encryption failed %v", err)
	}

	// Creates users in different tenants
	sqlStmt := `
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.Disallowed + `','` + encryptedPassword + `','` + encryptedPassword + `', 1, 1);
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.ReadOnly + `','` + encryptedPassword + `','` + encryptedPassword + `', 2, 1);
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.Operations + `','` + encryptedPassword + `','` + encryptedPassword + `', 3, 1);
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.Admin + `','` + encryptedPassword + `','` + encryptedPassword + `', 4, 1);
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.Portal + `','` + encryptedPassword + `','` + encryptedPassword + `', 5, 1);
INSERT INTO tm_user (username, local_passwd, confirm_local_passwd, role, tenant_id) VALUES ('` + Config.TrafficOps.Users.Federation + `','` + encryptedPassword + `','` + encryptedPassword + `', 6, 1);
`
	err = execSQL(db, sqlStmt, "tm_user")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupTenants ...
func SetupTenants(db *sql.DB) error {

	// TODO: root tenant must be present in initial database.  "badtenant" is needed for now so tests can be done
	// with a tenant outside the user's tenant.  That should be removed once User API tests are in place rather than the SetupUsers defined above.
	sqlStmt := `
INSERT INTO tenant (name, active, parent_id, last_updated) VALUES ('root', true, null, '2018-01-19 19:01:21.327262');
INSERT INTO tenant (name, active, parent_id, last_updated) VALUES ('badtenant', true, 1, '2018-01-19 19:01:21.327262');
`
	err := execSQL(db, sqlStmt, "tenant")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupDeliveryServiceTmUsers ...
func SetupDeliveryServiceTmUsers(db *sql.DB) error {

	sqlStmt := `
INSERT INTO deliveryservice_tmuser (deliveryservice, tm_user_id, last_updated) VALUES (100, (SELECT id FROM tm_user where username = 'admin') , '2018-01-19 21:19:32.372969');
`
	err := execSQL(db, sqlStmt, "deliveryservice_tmuser")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupJobStatuses ...
func SetupJobStatuses(db *sql.DB) error {

	sqlStmt := `
INSERT INTO job_status (id, name, description, last_updated) VALUES (1, 'PENDING', 'Job is queued, but has not been picked up by any agents yet', '2018-01-19 21:19:32.444857');
`
	err := execSQL(db, sqlStmt, "job_status")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupJobAgents ...
func SetupJobAgents(db *sql.DB) error {

	sqlStmt := `
INSERT INTO job_agent (id, name, description, active, last_updated) VALUES (1, 'agent1', 'Test Agent1', 0, '2018-01-19 21:19:32.448076');
`
	err := execSQL(db, sqlStmt, "job_agent")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// SetupJobs ...
func SetupJobs(db *sql.DB) error {

	sqlStmt := `
INSERT INTO job (id, agent, object_type, object_name, keyword, parameters, asset_url, asset_type, status, start_time, entered_time, job_user, last_updated, job_deliveryservice) VALUES (100, 1, null, null, 'PURGE', 'TTL:48h', 'http://cdn2.edge/job1/.*', 'file', 1, '2018-01-19 21:01:14.000000', '2018-01-19 21:01:14.000000', (SELECT id FROM tm_user where username = 'admin'), '2018-01-19 21:19:32.468643', 100);
INSERT INTO job (id, agent, object_type, object_name, keyword, parameters, asset_url, asset_type, status, start_time, entered_time, job_user, last_updated, job_deliveryservice) VALUES (200, 1, null, null, 'PURGE', 'TTL:48h', 'http://cdn2.edge/job2/.*', 'file', 1, '2018-01-19 21:09:34.000000', '2018-01-19 21:09:34.000000', (SELECT id FROM tm_user where username = 'admin'), '2018-01-19 21:19:32.450915', 200);
INSERT INTO job (id, agent, object_type, object_name, keyword, parameters, asset_url, asset_type, status, start_time, entered_time, job_user, last_updated, job_deliveryservice) VALUES (300, 1, null, null, 'PURGE', 'TTL:48h', 'http://cdn2.edge/job3/.*', 'file', 1, '2018-01-19 21:14:34.000000', '2018-01-19 21:14:34.000000', (SELECT id FROM tm_user where username = 'admin'), '2018-01-19 21:19:32.460870', 100);
`
	err := execSQL(db, sqlStmt, "job")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return nil
}

// Teardown - ensures that the data is cleaned up for a fresh run
func Teardown(db *sql.DB) error {

	sqlStmt := `
	DELETE FROM to_extension;
	DELETE FROM staticdnsentry;
	DELETE FROM job;
	DELETE FROM job_agent;
	DELETE FROM job_status;
	DELETE FROM log;
	DELETE FROM asn;
	DELETE FROM deliveryservice_tmuser;
	DELETE FROM tm_user;
	DELETE FROM role;
	ALTER SEQUENCE role_id_seq RESTART WITH 1;
	DELETE FROM deliveryservice_regex;
	DELETE FROM regex;
	DELETE FROM deliveryservice_server;
	DELETE FROM deliveryservice;
	DELETE FROM origin;
	DELETE FROM server;
	DELETE FROM phys_location;
	DELETE FROM region;
	DELETE FROM division;
	DELETE FROM profile;
	DELETE FROM parameter;
	DELETE FROM profile_parameter;
	DELETE FROM cachegroup;
	DELETE FROM coordinate;
	DELETE FROM type;
	DELETE FROM status;
	DELETE FROM snapshot;
	DELETE FROM cdn;
	DELETE FROM tenant;
	ALTER SEQUENCE tenant_id_seq RESTART WITH 1;
`
	err := execSQL(db, sqlStmt, "Tearing down")
	if err != nil {
		return fmt.Errorf("exec failed %v", err)
	}
	return err
}

// execSQL ...
func execSQL(db *sql.DB, sqlStmt string, dbTable string) error {

	log.Debugln(dbTable + " data")
	var err error

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("transaction begin failed %v %v ", err, tx)
	}

	res, err := tx.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("exec failed %v %v", err, res)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit failed %v %v", err, res)
	}
	return nil
}
