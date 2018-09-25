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


-- THIS FILE INCLUDES STATIC DATA REQUIRED OF TRAFFIC OPS

-- cdns
insert into cdn (name, dnssec_enabled, domain_name) values ('ALL', false, '-') ON CONFLICT (name) DO NOTHING;

-- job agents
insert into job_agent (name, description, active) values ('dummy', 'Description of Purge Agent', 1) ON CONFLICT (name) DO NOTHING;

-- job statuses
insert into job_status (name, description) values ('PENDING', 'Job is queued, but has not been picked up by any agents yet') ON CONFLICT (name) DO NOTHING;

-- parameters
-- Moved into postinstall global parameters
insert into profile (name, description, type, cdn) values ('GLOBAL', 'Global Traffic Ops profile, DO NOT DELETE', 'UNK_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;

---------------------------------

-- global parameters (settings)
---------------------------------
DO
$do$
BEGIN
        IF NOT EXISTS (SELECT id FROM PARAMETER WHERE name = 'tm.instance_name' AND config_file = 'global') THEN
                insert into parameter (name, config_file, value) values ('tm.instance_name', 'global', 'Traffic Ops CDN');
                insert into profile_parameter (profile, parameter) values ( (select id from profile where name = 'GLOBAL'), (select id from parameter where name = 'tm.instance_name' and config_file = 'global' and value = 'Traffic Ops CDN') ) ON CONFLICT (profile, parameter) DO NOTHING;
        END IF;
        IF NOT EXISTS (SELECT id FROM PARAMETER WHERE name = 'tm.toolname' AND config_file = 'global') THEN
                insert into parameter (name, config_file, value) values ('tm.toolname', 'global', 'Traffic Ops');
                insert into profile_parameter (profile, parameter) values ( (select id from profile where name = 'GLOBAL'), (select id from parameter where name = 'tm.toolname' and config_file = 'global' and value = 'Traffic Ops') ) ON CONFLICT (profile, parameter) DO NOTHING;
        END IF;
        IF NOT EXISTS (SELECT id FROM PARAMETER WHERE name = 'use_tenancy' AND config_file = 'global') THEN
                insert into parameter (name, config_file, value) values ('use_tenancy', 'global', '1');
                insert into profile_parameter (profile, parameter) values ( (select id from profile where name = 'GLOBAL'), (select id from parameter where name = 'use_tenancy' and config_file = 'global' and value = '1') ) ON CONFLICT (profile, parameter) DO NOTHING;
        END IF;
        IF NOT EXISTS (SELECT id FROM PARAMETER WHERE name = 'maxRevalDurationDays' AND config_file = 'regex_revalidate.config') THEN
                insert into parameter (name, config_file, value) values ('maxRevalDurationDays', 'regex_revalidate.config', '90');
                insert into profile_parameter (profile, parameter) values ( (select id from profile where name = 'GLOBAL'), (select id from parameter where name = 'maxRevalDurationDays' and config_file = 'regex_revalidate.config' and value = '90') ) ON CONFLICT (profile, parameter) DO NOTHING;
        END IF;
END
$do$;

-- profiles
---------------------------------
insert into profile (name, description, type, cdn) values ('TRAFFIC_ANALYTICS', 'Traffic Analytics profile', 'UNK_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('TRAFFIC_OPS', 'Traffic Ops profile', 'UNK_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('TRAFFIC_OPS_DB', 'Traffic Ops DB profile', 'UNK_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('TRAFFIC_PORTAL', 'Traffic Portal profile', 'TP_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('TRAFFIC_STATS', 'Traffic Stats profile', 'TS_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('INFLUXDB', 'InfluxDb profile', 'INFLUXDB_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;
insert into profile (name, description, type, cdn) values ('RIAK_ALL', 'Riak profile for all CDNs', 'RIAK_PROFILE', (SELECT id FROM cdn WHERE name='ALL')) ON CONFLICT (name) DO NOTHING;

-- statuses
insert into status (name, description) values ('OFFLINE', 'Server is Offline. Not active in any configuration.') ON CONFLICT (name) DO NOTHING;
insert into status (name, description) values ('ONLINE', 'Server is online.') ON CONFLICT (name) DO NOTHING;
insert into status (name, description) values ('REPORTED', 'Server is online and reported in the health protocol.') ON CONFLICT (name) DO NOTHING;
insert into status (name, description) values ('ADMIN_DOWN', 'Sever is administrative down and does not receive traffic.') ON CONFLICT (name) DO NOTHING;
insert into status (name, description) values ('CCR_IGNORE', 'Server is ignored by traffic router.') ON CONFLICT (name) DO NOTHING;
insert into status (name, description) values ('PRE_PROD', 'Pre Production. Not active in any configuration.') ON CONFLICT (name) DO NOTHING;

-- tenants
insert into tenant (name, active, parent_id) values ('root', true, null) ON CONFLICT DO NOTHING;
insert into tenant (name, active, parent_id) values ('unassigned', true, (select id from tenant where name='root')) ON CONFLICT DO NOTHING;

-- roles
-- out of the box, only 3 roles are defined. Other roles can be created by the admin as needed.
insert into role (name, description, priv_level) values ('admin', 'Has access to everything.', 30) ON CONFLICT (name) DO NOTHING;
insert into role (name, description, priv_level) values ('operations', 'Has all reads and most write capabilities', 20) ON CONFLICT (name) DO NOTHING;
insert into role (name, description, priv_level) values ('read-only', 'Has access to all read capabilities', 10) ON CONFLICT (name) DO NOTHING;
insert into role (name, description, priv_level) values ('disallowed', 'Block all access', 0) ON CONFLICT (name) DO NOTHING;

-- capabilities
-- auth
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

-- secure
INSERT INTO capability (name, description) values ('parameters-read-secure', 'Ability to view secure parameter values') ON CONFLICT (name) DO NOTHING;
INSERT INTO capability (name, description) values ('servers-read-secure', 'Ability to view secure server values') ON CONFLICT (name) DO NOTHING;

-- about
INSERT INTO capability (name, description) values ('about-read', 'Ability to read the server about information') ON CONFLICT (name) DO NOTHING;

-- roles_capabilities
-- out of the box, the admin role has ALL capabilities
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

INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'parameters-read-secure') ON CONFLICT (role_id, cap_name) DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'servers-read-secure') ON CONFLICT (role_id, cap_name) DO NOTHING;

INSERT INTO role_capability (role_id, cap_name) VALUES ((SELECT id FROM role WHERE name = 'admin'), 'about-read') ON CONFLICT (role_id, cap_name) DO NOTHING;

-- Using role 'read-only'

INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'auth' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;

INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'api-endpoints-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'asns-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'cache-config-files-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'cache-groups-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'capabilities-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'cdns-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'cdn-security-keys-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'change-logs-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'coordinates-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'delivery-services-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'delivery-service-security-keys-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'delivery-service-requests-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'delivery-service-servers-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'divisions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'to-extensions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'federations-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'hwinfo-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'jobs-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'origins-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'parameters-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'phys-locations-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'profiles-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'regions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'roles-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'servers-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'stats-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'statuses-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'static-dns-entries-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'steering-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'steering-targets-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'system-info-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'tenants-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'types-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'users-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'read-only'), 'about-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'read-only') ON CONFLICT DO NOTHING;

-- Using role 'operations'

INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'auth' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

-- Includes 'readonly' endpoints:
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'api-endpoints-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'asns-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cache-config-files-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cache-groups-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'capabilities-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cdns-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cdn-security-keys-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'change-logs-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'coordinates-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-services-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-security-keys-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-requests-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-servers-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'divisions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'to-extensions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'federations-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'hwinfo-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'jobs-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'origins-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'parameters-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'phys-locations-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'profiles-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'regions-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'roles-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'servers-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'stats-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'statuses-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'static-dns-entries-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'steering-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'steering-targets-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'system-info-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'tenants-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'types-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'users-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

-- Explicitly require 'operations'
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'api-endpoints-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'asns-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cache-groups-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'capabilities-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cdns-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'cdns-snapshot' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-services-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-security-keys-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-servers-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'divisions-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'iso-generate' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
-- For 'parameters-write': related endpoints will check manually for 'parameters-secure-write', which currently doesn't exist yet.
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'parameters-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'phys-locations-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'profiles-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'regions-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'roles-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'servers-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'stats-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'statuses-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'tenants-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'types-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'users-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

-- Inherited endpoint from the 'privilege hierarchy' (really, just federations)
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'federations-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

-- Outstanding capabilities that had to be thought about
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'coordinates-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'delivery-service-requests-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'to-extensions-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'jobs-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'steering-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'steering-targets-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'users-register' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;
INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'static-dns-entries-write' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

INSERT INTO role_capability (role_id, cap_name) SELECT (SELECT id FROM role WHERE name = 'operations'), 'about-read' WHERE EXISTS (SELECT id FROM role WHERE name = 'operations') ON CONFLICT DO NOTHING;

-- api_capabilities

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
insert into api_capability (http_method, route, capability) values ('GET', 'federation_resolvers', 'federations-read') ON CONFLICT (http_method, route, capability) DO NOTHING;
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
-- about
INSERT INTO api_capability (http_method, route, capability) VALUES ('GET', 'about', 'about-read') ON CONFLICT (http_method, route, capability) DO NOTHING;

-- types

-- delivery service types
insert into type (name, description, use_in_table) values ('HTTP', 'HTTP Content Routing', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('HTTP_NO_CACHE', 'HTTP Content Routing, no caching', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('HTTP_LIVE', 'HTTP Content routing cache in RAM', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('HTTP_LIVE_NATNL', 'HTTP Content routing, RAM cache, National', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('DNS', 'DNS Content Routing', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('DNS_LIVE', 'DNS Content routing, RAM cache, Local', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('DNS_LIVE_NATNL', 'DNS Content routing, RAM cache, National', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('ANY_MAP', 'No Content Routing - arbitrary remap at the edge, no Traffic Router config', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STEERING', 'Steering Delivery Service', 'deliveryservice') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CLIENT_STEERING', 'Client-Controlled Steering Delivery Service', 'deliveryservice') ON CONFLICT (name) DO NOTHING;

-- server types
insert into type (name, description, use_in_table) values ('EDGE', 'Edge Cache', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('MID', 'Mid Tier Cache', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('ORG', 'Origin', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CCR', 'Traffic Router', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('RASCAL', 'Traffic Monitor', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('RIAK', 'Riak keystore', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('INFLUXDB', 'influxDb server', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TRAFFIC_ANALYTICS', 'traffic analytics server', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TRAFFIC_OPS', 'traffic ops server', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TRAFFIC_OPS_DB', 'traffic ops DB server', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TRAFFIC_PORTAL', 'traffic portal server', 'server') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TRAFFIC_STATS', 'traffic stats server', 'server') ON CONFLICT (name) DO NOTHING;

-- cachegroup types
insert into type (name, description, use_in_table) values ('EDGE_LOC', 'Edge Logical Location', 'cachegroup') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('MID_LOC', 'Mid Logical Location', 'cachegroup') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('ORG_LOC', 'Origin Logical Site', 'cachegroup') ON CONFLICT (name) DO NOTHING;

-- to_extension types
insert into type (name, description, use_in_table) values ('CHECK_EXTENSION_BOOL', 'Extension for checkmark in Server Check', 'to_extension') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CHECK_EXTENSION_NUM', 'Extension for int value in Server Check', 'to_extension') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CHECK_EXTENSION_OPEN_SLOT', 'Open slot for check in Server Status', 'to_extension') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CONFIG_EXTENSION', 'Extension for additional configuration file', 'to_extension') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STATISTIC_EXTENSION', 'Extension source for 12M graphs', 'to_extension') ON CONFLICT (name) DO NOTHING;

-- regex types
insert into type (name, description, use_in_table) values ('HOST_REGEXP', 'Host header regular expression', 'regex') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('HEADER_REGEXP', 'HTTP header regular expression', 'regex') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('PATH_REGEXP', 'URL path regular expression', 'regex') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STEERING_REGEXP', 'Steering target filter regular expression', 'regex') ON CONFLICT (name) DO NOTHING;

-- federation types
insert into type (name, description, use_in_table) values ('RESOLVE4', 'federation type resolve4', 'federation') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('RESOLVE6', 'federation type resolve6', 'federation') ON CONFLICT (name) DO NOTHING;

-- static dns entry types
insert into type (name, description, use_in_table) values ('A_RECORD', 'Static DNS A entry', 'staticdnsentry') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('AAAA_RECORD', 'Static DNS AAAA entry', 'staticdnsentry') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('CNAME_RECORD', 'Static DNS CNAME entry', 'staticdnsentry') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('TXT_RECORD', 'Static DNS TXT entry', 'staticdnsentry') ON CONFLICT (name) DO NOTHING;

--steering_target types
insert into type (name, description, use_in_table) values ('STEERING_WEIGHT', 'Weighted steering target', 'steering_target') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STEERING_ORDER', 'Ordered steering target', 'steering_target') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STEERING_GEO_ORDER', 'Geo-ordered steering target', 'steering_target') ON CONFLICT (name) DO NOTHING;
insert into type (name, description, use_in_table) values ('STEERING_GEO_WEIGHT', 'Geo-weighted steering target', 'steering_target') ON CONFLICT (name) DO NOTHING;

-- users
insert into tm_user (username, role, full_name, token, tenant_id) values ('extension',
    (select id from role where name = 'operations'), 'Extension User, DO NOT DELETE', '91504CE6-8E4A-46B2-9F9F-FE7C15228498',
    (select id from tenant where name = 'root')) ON CONFLICT DO NOTHING;

-- to extensions
-- some of the old ones do not get a new place, and there will be 'gaps' in the column usage.... New to_extension add will have to take care of that.
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (1, 'ILO_PING', 'ILO', 'aa', '1.0.0', '-', 'ToPingCheck.pl', '1', '{ check_name: "ILO", "base_url": "https://localhost", "select": "ilo_ip_address", "cron": "9 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (2, '10G_PING', '10G', 'ab', '1.0.0', '-', 'ToPingCheck.pl', '1', '{ check_name: "10G", "base_url": "https://localhost", "select": "ip_address", "cron": "18 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (3, 'FQDN_PING', 'FQDN', 'ac', '1.0.0', '-', 'ToPingCheck.pl', '1', '{ check_name: "FQDN", "base_url": "https://localhost", "select": "host_name", "cron": "27 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (4, 'CHECK_DSCP', 'DSCP', 'ad', '1.0.0', '-', 'ToDSCPCheck.pl', '1', '{ "check_name": "DSCP", "base_url": "https://localhost", "cron": "36 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
-- open EF
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (5, 'OPEN', '', 'ae', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (6, 'OPEN', '', 'af', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
--
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (7, 'IPV6_PING', '10G6', 'ag', '1.0.0', '-', 'ToPingCheck.pl', '1', '{ "select": "ip6_address", "cron": "0 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
-- upd_pending H -> open
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (8, 'OPEN', '', 'ah', '1.0.0', '', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
-- open IJ
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (9, 'OPEN', '', 'ai', '1.0.0', '', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (10, 'OPEN', '', 'aj', '1.0.0', '', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
--
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (11, 'CHECK_MTU', 'MTU', 'ak', '1.0.0', '-', 'ToMtuCheck.pl', '1', '{ "check_name": "MTU", "base_url": "https://localhost", "cron": "45 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (12, 'CHECK_TRAFFIC_ROUTER_STATUS', 'RTR', 'al', '1.0.0', '-', 'ToRTRCheck.pl', '1', '{  "check_name": "RTR", "base_url": "https://localhost", "cron": "10 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_BOOL') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (13, 'OPEN', '', 'am', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (14, 'CACHE_HIT_RATIO_LAST_15', 'CHR', 'an', '1.0.0', '-', 'ToCHRCheck.pl', '1', '{ check_name: "CHR", "base_url": "https://localhost", cron": "0,15,30,45 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_NUM') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (15, 'DISK_UTILIZATION', 'CDU', 'ao', '1.0.0', '-', 'ToCDUCheck.pl', '1', '{ check_name: "CDU", "base_url": "https://localhost", cron": "20 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_NUM') ) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (16, 'ORT_ERROR_COUNT', 'ORT', 'ap', '1.0.0', '-', 'ToORTCheck.pl', '1', '{ check_name: "ORT", "base_url": "https://localhost", "cron": "40 * * * *" }',
        (select id from type where name='CHECK_EXTENSION_NUM') ) ON CONFLICT DO NOTHING;
-- rest open
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (17, 'OPEN', '', 'aq', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (18, 'OPEN', '', 'ar', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (19, 'OPEN', '', 'bf', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (20, 'OPEN', '', 'at', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (21, 'OPEN', '', 'au', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (22, 'OPEN', '', 'av', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (23, 'OPEN', '', 'aw', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (24, 'OPEN', '', 'ax', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (25, 'OPEN', '', 'ay', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (26, 'OPEN', '', 'az', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (27, 'OPEN', '', 'ba', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (28, 'OPEN', '', 'bb', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (29, 'OPEN', '', 'bc', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (30, 'OPEN', '', 'bd', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
insert into to_extension (id, name, servercheck_short_name, servercheck_column_name, version, info_url, script_file, isactive, additional_config_json, type)
values (31, 'OPEN', '', 'be', '1.0.0', '-', '', '0', '', (select id from type where name='CHECK_EXTENSION_OPEN_SLOT')) ON CONFLICT DO NOTHING;
