# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

## [Unreleased]
### Added
- Traffic Router: TR now generates a self-signed certificate at startup and uses it as the default TLS cert.
  The default certificate is used whenever a client attempts an SSL handshake for an SNI host which does not match
  any of the other certificates.
- Client Steering Forced Diversity: force Traffic Router to return more unique edge caches in CLIENT_STEERING results instead of the default behavior which can sometimes return a result of multiple targets using the same edge cache. In the case of edge cache failures, this feature will give clients a chance to retry a different edge cache. This can be enabled with the new "client.steering.forced.diversity" Traffic Router profile parameter.
- Traffic Ops Golang Endpoints
  - /api/1.4/deliveryservices `(GET,POST,PUT)`
  - /api/1.4/users `(GET,POST,PUT)`
  - /api/1.1/deliveryservices/xmlId/:xmlid/sslkeys `GET`
  - /api/1.1/deliveryservices/hostname/:hostname/sslkeys `GET`
  - /api/1.1/deliveryservices/sslkeys/add `POST`
  - /api/1.1/deliveryservices/xmlId/:xmlid/sslkeys/delete `GET`
  - /api/1.4/deliveryservices_required_capabilities `(GET,POST,DELETE)`
  - /api/1.1/servers/status `GET`
  - /api/1.4/cdns/dnsseckeys/refresh `GET`
  - /api/1.1/cdns/name/:name/dnsseckeys `GET`
  - /api/1.1/roles `GET`
  - /api/1.4/cdns/name/:name/dnsseckeys `GET`
  - /api/1.4/user/login/oauth `POST`
  - /api/1.1/servers/:name/configfiles/ats `GET`
  - /api/1.1/profiles/:name/configfiles/ats/* `GET`
  - /api/1.1/servers/:name/configfiles/ats/* `GET`
  - /api/1.1/cdns/:name/configfiles/ats/* `GET`
  - /api/1.1/servers/:id/status `PUT`
  - /api/1.1/dbdump `GET`
  - /api/1.1/servers/:name/configfiles/ats/parent.config
  - /api/1.1/servers/:name/configfiles/ats/remap.config
  - /api/1.1/user/login/token `POST`
  - /api/1.4/deliveryservice_stats `GET`
  - /api/1.1/deliveryservices/request
  - /api/1.1/federations/:id/users
  - /api/1.1/federations/:id/users/:userID

- To support reusing a single riak cluster connection, an optional parameter is added to riak.conf: "HealthCheckInterval". This options takes a 'Duration' value (ie: 10s, 5m) which affects how often the riak cluster is health checked.  Default is currently set to: "HealthCheckInterval": "5s".
- Added a new Go db/admin binary to replace the Perl db/admin.pl script which is now deprecated and will be removed in a future release. The new db/admin binary is essentially a drop-in replacement for db/admin.pl since it supports all of the same commands and options; therefore, it should be used in place of db/admin.pl for all the same tasks.
- Added an API 1.4 endpoint, /api/1.4/cdns/dnsseckeys/refresh, to perform necessary behavior previously served outside the API under `/internal`.
- Added the DS Record text to the cdn dnsseckeys endpoint in 1.4.
- Added monitoring.json snapshotting. This stores the monitoring json in the same table as the crconfig snapshot. Snapshotting is now required in order to push out monitoring changes.
- To traffic_ops_ort.pl added the ability to handle ##OVERRIDE## delivery service ANY_MAP raw remap text to replace and comment out a base delivery service remap rules. THIS IS A TEMPORARY HACK until versioned delivery services are implemented.
- Snapshotting the CRConfig now deletes HTTPS certificates in Riak for delivery services which have been deleted in Traffic Ops.
- Added a context menu in place of the "Actions" column from the following tables in Traffic Portal: cache group tables, CDN tables, delivery service tables, parameter tables, profile tables, server tables.
- Traffic Portal standalone Dockerfile
- In Traffic Portal, removes the need to specify line breaks using `__RETURN__` in delivery service edge/mid header rewrite rules, regex remap expressions, raw remap text and traffic router additional request/response headers.
- In Traffic Portal, provides the ability to clone delivery service assignments from one cache to another cache of the same type. Issue #2963.
- Added an API 1.4 endpoint, /api/1.4/server_capabilities, to create, read, and delete server capabilities.
- Traffic Ops now allows each delivery service to have a set of query parameter keys to be retained for consistent hash generation by Traffic Router.
- In Traffic Portal, delivery service table columns can now be rearranged and their visibility toggled on/off as desired by the user. Hidden table columns are excluded from the table search. These settings are persisted in the browser.
- Added an API 1.4 endpoint, /api/1.4/user/login/oauth to handle SSO login using OAuth.
- Added /#!/sso page to Traffic Portal to catch redirects back from OAuth provider and POST token into the API.
- In Traffic Portal, server table columns can now be rearranged and their visibility toggled on/off as desired by the user. Hidden table columns are excluded from the table search. These settings are persisted in the browser.
- Added pagination support to some Traffic Ops endpoints via three new query parameters, limit and offset/page
- Traffic Ops /servers and /deliveryservices endpoints now support If-Modified-Since and If-None-Match requests.
- Traffic Ops now supports a "sortOrder" query parameter on some endpoints to return API responses in descending order
- Traffic Ops now uses a consistent format for audit logs across all Go endpoints
- Added cache-side config generator, atstccfg, installed with ORT. Includes all configs.
- In Traffic Portal, all tables now include a 'CSV' link to enable the export of table data in CSV format.
- Pylint configuration now enforced (present in [a file in the Python client directory](./traffic_control/clients/python/pylint.rc))
- Added an optional SMTP server configuration to the TO configuration file, api now has unused abilitiy to send emails
- Traffic Monitor now has "gbps" calculated stat, allowing operators to monitor bandwidth in Gbps.
- Added an API 1.4 endpoint, /api/1.4/deliveryservices_required_capabilities, to create, read, and delete associations between a delivery service and a required capability.
- Added ATS config generation omitting parents without Delivery Service Required Capabilities.
- In Traffic Portal, added the ability to create, view and delete server capabilities and associate those server capabilities with servers and delivery services. See [blueprint](./blueprints/server-capabilitites.md)
- Added validation to prevent assigning servers to delivery services without required capabilities.
- Added deep coverage zone routing percentage to the Traffic Portal dashboard.

### Changed
- Traffic Router:  TR will now allow steering DSs and steering target DSs to have RGB enabled. (fixes #3910)
- Traffic Portal:  Traffic Portal now allows Regional Geo Blocking to be enabled for a Steering Delivery Service.
- Traffic Router, added TLS certificate validation on certificates imported from Traffic Ops
  - validates modulus of private and public keys
  - validates current timestamp falls within the certificate date bracket
  - validates certificate subjects against the DS URL
- Traffic Ops Golang Endpoints
  - Updated /api/1.1/cachegroups: Cache Group Fallbacks are included
  - Updated /api/1.1/cachegroups: fixed so fallbackToClosest can be set through API
    - Warning:  a PUT of an old Cache Group JSON without the fallbackToClosest field will result in a `null` value for that field
- Traffic Router: fixed a bug which would cause `REFUSED` DNS answers if the zone priming execution did not complete within the configured `zonemanager.init.timeout` period.
- Issue 2821: Fixed "Traffic Router may choose wrong certificate when SNI names overlap"
- traffic_ops/app/bin/checks/ToDnssecRefresh.pl now requires "user" and "pass" parameters of an operations-level user! Update your scripts accordingly! This was necessary to move to an API endpoint with proper authentication, which may be safely exposed.
- Traffic Monitor UI updated to support HTTP or HTTPS traffic.
- Traffic Monitor health/stat time now includes full body download (like prior TM <=2.1 version)
- Modified Traffic Router logging format to include an additional field for DNS log entries, namely `rhi`. This defaults to '-' and is only used when EDNS0 client subnet extensions are enabled and a client subnet is present in the request. When enabled and a subnet is present, the subnet appears in the `chi` field and the resolver address is in the `rhi` field.
- Changed traffic_ops_ort.pl so that hdr_rw-<ds>.config files are compared with strict ordering and line duplication when detecting configuration changes.
- Traffic Ops (golang), Traffic Monitor, Traffic Stats are now compiled using Go version 1.11. Grove was already being compiled with this version which improves performance for TLS when RSA certificates are used.
- Fixed issue #3497: TO API clients that don't specify the latest minor version will overwrite/default any fields introduced in later versions
- Fixed permissions on DELETE /api/$version/deliveryservice_server/{dsid}/{serverid} endpoint
- Issue 3476: Traffic Router returns partial result for CLIENT_STEERING Delivery Services when Regional Geoblocking or Anonymous Blocking is enabled.
- Upgraded Traffic Portal to AngularJS 1.7.8
- Issue 3275: Improved the snapshot diff performance and experience.
- Issue 3550: Fixed TC golang client setting for cache control max age
- Issue #3605: Fixed Traffic Monitor custom ports in health polling URL.
- Issue 3587: Fixed Traffic Ops Golang reverse proxy and Riak logs to be consistent with the format of other error logs.
- Database migrations have been collapsed. Rollbacks to migrations that previously existed are no longer possible.
- Issue #3750: Fixed Grove access log fractional seconds.
- Issue #3646: Fixed Traffic Monitor Thresholds.
- Modified Traffic Router API to be available via HTTPS.
- Added fields to traffic_portal_properties.json to configure SSO through OAuth.
- Added field to cdn.conf to configure whitelisted URLs for Json Key Set URL returned from OAuth provider.
- Improved [profile comparison view in Traffic Portal](https://github.com/apache/trafficcontrol/blob/master/blueprints/profile-param-compare-manage.md).
- Issue #3871 - provides users with a specified role the ability to mark any delivery service request as complete.
- Fixed Traffic Ops Golang POST servers/id/deliveryservice continuing erroneously after a database error.
- Fixed Traffic Ops Golang POST servers/id/deliveryservice double-logging errors.

### Deprecated/Removed
- The TO API `cachegroup_fallbacks` endpoint is now deprecated

## [3.0.0] - 2018-10-30
### Added
- Removed MySQL-to-Postgres migration tools.  This tool is supported for 1.x to 2.x upgrades only and should not be used with 3.x.
- Backup Edge Cache group: If the matched group in the CZF is not available, this list of backup edge cache group configured via Traffic Ops API can be used as backup. In the event of all backup edge cache groups not available, GEO location can be optionally used as further backup. APIs detailed [here](http://traffic-control-cdn.readthedocs.io/en/latest/development/traffic_ops_api/v12/cachegroup_fallbacks.html)
- Traffic Ops Golang Proxy Endpoints
  - /api/1.4/users `(GET,POST,PUT)`
  - /api/1.3/origins `(GET,POST,PUT,DELETE)`
  - /api/1.3/coordinates `(GET,POST,PUT,DELETE)`
  - /api/1.3/staticdnsentries `(GET,POST,PUT,DELETE)`
  - /api/1.1/deliveryservices/xmlId/:xmlid/sslkeys `GET`
  - /api/1.1/deliveryservices/hostname/:hostname/sslkeys `GET`
  - /api/1.1/deliveryservices/sslkeys/add `POST`
  - /api/1.1/deliveryservices/xmlId/:xmlid/sslkeys/delete `GET`
- Delivery Service Origins Refactor: The Delivery Service API now creates/updates an Origin entity on Delivery Service creates/updates, and the `org_server_fqdn` column in the `deliveryservice` table has been removed. The `org_server_fqdn` data is now computed from the Delivery Service's primary origin (note: the name of the primary origin is the `xml_id` of its delivery service).
- Cachegroup-Coordinate Refactor: The Cachegroup API now creates/updates a Coordinate entity on Cachegroup creates/updates, and the `latitude` and `longitude` columns in the `cachegroup` table have been replaced with `coordinate` (a foreign key to Coordinate). Coordinates created from Cachegroups are given the name `from_cachegroup_\<cachegroup name\>`.
- Geolocation-based Client Steering: two new steering target types are available to use for `CLIENT_STEERING` delivery services: `STEERING_GEO_ORDER` and `STEERING_GEO_WEIGHT`. When targets of these types have an Origin with a Coordinate, Traffic Router will order and prioritize them based upon the shortest total distance from client -> edge -> origin. Co-located targets are grouped together and can be weighted or ordered within the same location using `STEERING_GEO_WEIGHT` or `STEERING_GEO_ORDER`, respectively.
- Tenancy is now the default behavior in Traffic Ops.  All database entries that reference a tenant now have a default of the root tenant.  This eliminates the need for the `use_tenancy` global parameter and will allow for code to be simplified as a result. If all user and delivery services reference the root tenant, then there will be no difference from having `use_tenancy` set to 0.
- Cachegroup Localization Methods: The Cachegroup API now supports an optional `localizationMethods` field which specifies the localization methods allowed for that cachegroup (currently 'DEEP_CZ', 'CZ', and 'GEO'). By default if this field is null/empty, all localization methods are enabled. After Traffic Router has localized a client, it will only route that client to cachegroups that have enabled the localization method used. For example, this can be used to prevent GEO-localized traffic (i.e. most likely from off-net/internet clients) to cachegroups that aren't optimal for internet traffic.
- Traffic Monitor Client Update: Traffic Monitor is updated to use the Traffic Ops v13 client.
- Removed previously deprecated `traffic_monitor_java`
- Added `infrastructure/cdn-in-a-box` for Apachecon 2018 demonstration
- The CacheURL Delivery service field is deprecated.  If you still need this functionality, you can create the configuration explicitly via the raw remap field.

## [2.2.0] - 2018-06-07
### Added
- Per-DeliveryService Routing Names: you can now choose a Delivery Service's Routing Name (rather than a hardcoded "tr" or "edge" name). This might require a few pre-upgrade steps detailed [here](http://traffic-control-cdn.readthedocs.io/en/latest/admin/traffic_ops/migration_from_20_to_22.html#per-deliveryservice-routing-names)
- [Delivery Service Requests](http://traffic-control-cdn.readthedocs.io/en/latest/admin/quick_howto/ds_requests.html#ds-requests): When enabled, delivery service requests are created when ALL users attempt to create, update or delete a delivery service. This allows users with higher level permissions to review delivery service changes for completeness and accuracy before deploying the changes.
- Traffic Ops Golang Proxy Endpoints
  - /api/1.3/about `(GET)`
  - /api/1.3/asns `(GET,POST,PUT,DELETE)`
  - /api/1.3/cachegroups `(GET,POST,PUT,DELETE)`
  - /api/1.3/cdns `(GET,POST,PUT,DELETE)`
  - /api/1.3/cdns/capacity `(GET)`
  - /api/1.3/cdns/configs `(GET)`
  - /api/1.3/cdns/dnsseckeys `(GET)`
  - /api/1.3/cdns/domain `(GET)`
  - /api/1.3/cdns/monitoring `(GET)`
  - /api/1.3/cdns/health `(GET)`
  - /api/1.3/cdns/routing `(GET)`
  - /api/1.3/deliveryservice_requests `(GET,POST,PUT,DELETE)`
  - /api/1.3/divisions `(GET,POST,PUT,DELETE)`
  - /api/1.3/hwinfos `(GET)`
  - /api/1.3/login `(POST)`
  - /api/1.3/parameters `(GET,POST,PUT,DELETE)`
  - /api/1.3/profileparameters `(GET,POST,PUT,DELETE)`
  - /api/1.3/phys_locations `(GET,POST,PUT,DELETE)`
  - /api/1.3/ping `(GET)`
  - /api/1.3/profiles `(GET,POST,PUT,DELETE)`
  - /api/1.3/regions `(GET,POST,PUT,DELETE)`
  - /api/1.3/servers `(GET,POST,PUT,DELETE)`
  - /api/1.3/servers/checks `(GET)`
  - /api/1.3/servers/details `(GET)`
  - /api/1.3/servers/status `(GET)`
  - /api/1.3/servers/totals `(GET)`
  - /api/1.3/statuses `(GET,POST,PUT,DELETE)`
  - /api/1.3/system/info `(GET)`
  - /api/1.3/types `(GET,POST,PUT,DELETE)`
- Fair Queuing Pacing: Using the FQ Pacing Rate parameter in Delivery Services allows operators to limit the rate of individual sessions to the edge cache. This feature requires a Trafficserver RPM containing the fq_pacing experimental plugin AND setting 'fq' as the default Linux qdisc in sysctl.
- Traffic Ops rpm changed to remove world-read permission from configuration files.

### Changed
- Reformatted this CHANGELOG file to the keep-a-changelog format

[Unreleased]: https://github.com/apache/trafficcontrol/compare/RELEASE-3.0.0...HEAD
[3.0.0]: https://github.com/apache/trafficcontrol/compare/RELEASE-2.2.0...RELEASE-3.0.0
[2.2.0]: https://github.com/apache/trafficcontrol/compare/RELEASE-2.1.0...RELEASE-2.2.0
