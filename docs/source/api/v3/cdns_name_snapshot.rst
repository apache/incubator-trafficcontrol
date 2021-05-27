..
..
.. Licensed under the Apache License, Version 2.0 (the "License");
.. you may not use this file except in compliance with the License.
.. You may obtain a copy of the License at
..
..     http://www.apache.org/licenses/LICENSE-2.0
..
.. Unless required by applicable law or agreed to in writing, software
.. distributed under the License is distributed on an "AS IS" BASIS,
.. WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
.. See the License for the specific language governing permissions and
.. limitations under the License.
..

.. _to-api-v3-cdns-name-snapshot:

**************************
``cdns/{{name}}/snapshot``
**************************
.. caution:: This page is a stub! Much of it may be missing or just downright wrong - it needs a lot of love from people with the domain knowledge required to update it.

``GET``
=======
Retrieves the *current* :term:`Snapshot` for a CDN, which represents the current *operating state* of the CDN, **not** the current *configuration* of the CDN. The contents of this :term:`Snapshot` are currently used by Traffic Monitor and Traffic Router.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------------+
	| Name | Description                                                        |
	+======+====================================================================+
	| name | The name of the CDN for which a :term:`Snapshot` shall be returned |
	+------+--------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/3.0/cdns/CDN-in-a-Box/snapshot HTTP/1.1
	User-Agent: python-requests/2.23.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
:config: An object containing basic configurations on the actual CDN object

	:api.cache-control.max-age:     A string containing an integer which specifies the value of ``max-age`` in the :mailheader:`Cache-Control` header of some HTTP responses, likely the :ref:`tr-api` responses
	:certificates.polling.interval: A string containing an integer which specifies the interval, in seconds, on which other Traffic Control components should check for updated SSL certificates
	:consistent.dns.routing:        A string containing a boolean which indicates whether DNS routing will use a consistent hashing method or "round-robin"

		"false"
			The "round-robin" method will be used to define DNS routing
		"true"
			A consistent hashing method will be used to define DNS routing

	:coveragezone.polling.interval:      A string containing an integer which specifies the interval, in seconds, on which Traffic Routers should check for a new Coverage Zone file
	:coveragezone.polling.url:           The URL where a :term:`Coverage Zone File` may be requested by Traffic Routers
	:dnssec.dynamic.response.expiration: A string containing a number and unit suffix that specifies the length of time for which dynamic responses to DNSSEC lookup queries should remain valid
	:dnssec.dynamic.concurrencylevel:    An integer that defines the size of the concurrency level (threads) of the Guava cache used by ZoneManager to store zone material
	:dnssec.dynamic.initialcapacity:     An integer that defines the initial size of the Guava cache, default is 10000. Too low of a value can lead to expensive resizing
	:dnssec.init.timeout:                An integer that defines the number of minutes to allow for zone generation, this bounds the zone priming activity
	:dnssec.enabled:                     A string that tells whether or not the CDN uses DNSSEC; one of:

		"false"
			DNSSEC is not used within this CDN
		"true"
			DNSSEC is used within this CDN

	:domain_name:                        A string that is the :abbr:`TLD (Top-Level Domain)` served by the CDN
	:federationmapping.polling.interval: A string containing an integer which specifies the interval, in seconds, on which other Traffic Control components should check for new federation mappings
	:federationmapping.polling.url:      The URL where Traffic Control components can request federation mappings
	:geolocation.polling.interval:       A string containing an integer which specifies the interval, in seconds, on which other Traffic Control components should check for new IP-to-geographic-location mapping databases
	:geolocation.polling.url:            The URL where Traffic Control components can request IP-to-geographic-location mapping database files
	:keystore.maintenance.interval:      A string containing an integer which specifies the interval, in seconds, on which Traffic Routers should refresh their zone caches
	:neustar.polling.interval:           A string containing an integer which specifies the interval, in seconds, on which other Traffic Control components should check for new "Neustar" databases
	:neustar.polling.url:                The URL where Traffic Control components can request "Neustar" databases
	:soa:                                An object defining the :abbr:`SOA (Start of Authority)` for the CDN's :abbr:`TLD (Top-Level Domain)` (defined in ``domain_name``)

		:admin: The name of the administrator for this zone - i.e. the RNAME

			.. note:: This rarely represents a proper email address, unfortunately.

		:expire:  A string containing an integer that sets the number of seconds after which secondary name servers should stop answering requests for this zone if the master does not respond
		:minimum: A string containing an integer that sets the :abbr:`TTL (Time To Live)` - in seconds - of the record for the purpose of negative caching
		:refresh: A string containing an integer that sets the number of seconds after which secondary name servers should query the master for the :abbr:`SOA (Start of Authority)` record, to detect zone changes
		:retry:   A string containing an integer that sets the number of seconds after which secondary name servers should retry to request the serial number from the master if the master does not respond

			.. note:: :rfc:`1035` dictates that this should always be less than ``refresh``.

		.. seealso:: `The Wikipedia page on Start of Authority records <https://en.wikipedia.org/wiki/SOA_record>`_.

	:steeringmapping.polling.interval:       A string containing an integer which specifies the interval, in seconds, on which Traffic Control components should check for new steering mappings
	:ttls:                                   An object that contains keys which are types of DNS records that have values which are strings containing integers that specify the time for which a response to the specific type of record request should remain valid
	:zonemanager.cache.maintenance.interval: A configuration option for the ZoneManager Java class of Traffic Router
	:zonemanager.threadpool.scale:           A configuration option for the ZoneManager Java class of Traffic Router

:contentRouters: An object containing keys which are the (short) hostnames of the Traffic Routers that serve requests for :term:`Delivery Services` in this CDN

	:api.port:        A string containing the port number on which the :ref:`tr-api` is served by this Traffic Router via HTTP
	:secure.api.port: An optionally present string containing the port number on which the :ref:`tr-api` is served by this Traffic Router via HTTPS
	:fqdn:            This Traffic Router's :abbr:`FQDN (Fully Qualified Domain Name)`
	:httpsPort:       The port number on which this Traffic Router listens for incoming HTTPS requests
	:ip:              This Traffic Router's IPv4 address
	:ip6:             This Traffic Router's IPv6 address
	:location:        A string which is the :ref:`cache-group-name` of the :term:`Cache Group` to which this Traffic Router belongs
	:port:            The port number on which this Traffic Router listens for incoming HTTP requests
	:profile:         The :ref:`profile-name` of the :term:`Profile` used by this Traffic Router
	:status:          The health status of this Traffic Router

		.. seealso:: :ref:`health-proto`

:contentServers: An object containing keys which are the (short) hostnames of the :term:`Edge-tier cache servers` in the CDN; the values corresponding to those keys are routing information for said servers

	:cacheGroup:       A string that is the :ref:`cache-group-name` of the :term:`Cache Group` to which the server belongs
	:capabilities:     An array of this :term:`Cache Server`'s :term:`Server Capabilities`. If the Cache Server has no Server Capabilities, this field is omitted.
	:deliveryServices: An object containing keys which are the names of :term:`Delivery Services` to which this :term:`cache server` is assigned; the values corresponding to those keys are arrays of :abbr:`FQDNs (Fully Qualified Domain Names)` that resolve to this :term:`cache server`

		.. note:: Only :term:`Edge-tier cache servers` can be assigned to a :term:`Delivery Service`, and therefore this field will only be present when ``type`` is ``"EDGE"``.

	:fqdn:            The server's :abbr:`FQDN (Fully Qualified Domain Name)`
	:hashCount:       The number of servers to be placed into a single "hash ring" in Traffic Router
	:hashId:          A unique string to be used as the key for hashing servers - as of version 3.0.0 of Traffic Control, this is always the same as the server's (short) hostname and only still exists for legacy compatibility reasons
	:httpsPort:       The port on which the :term:`cache server` listens for incoming HTTPS requests
	:interfaceName:   The name of the main network interface device used by this :term:`cache server`
	:ip6:             The server's IPv6 address
	:ip:              The server's IPv4 address
	:locationId:      This field is exactly the same as ``cacheGroup`` and only exists for legacy compatibility reasons
	:port:            The port on which this :term:`cache server` listens for incoming HTTP requests
	:profile:         The :ref:`profile-name` of the :term:`Profile` used by the :term:`cache server`
	:routingDisabled: An integer representing the boolean concept of whether or not Traffic Routers should route client traffic to this :term:`cache server`; one of:

		0
			Do not route traffic to this server
		1
			Route traffic to this server normally

	:status: This :term:`cache server`'s status

		.. seealso:: :ref:`health-proto`

	:type: The :term:`Type` of this :term:`cache server`; which ought to be one of (but in practice need not be in certain special circumstances):

		EDGE
			This is an :term:`Edge-tier cache server`
		MID
			This is a :term:`Mid-tier cache server`

:deliveryServices: An object containing keys which are the :ref:`xml_ids <ds-xmlid>` of all of the :term:`Delivery Services` within the CDN

	:anonymousBlockingEnabled: A string containing a boolean that tells whether or not :ref:`ds-anonymous-blocking` is set on this :term:`Delivery Service`; one of:

		"true"
			Anonymized IP addresses are blocked by this :term:`Delivery Service`
		"false"
			Anonymized IP addresses are not blocked by this :term:`Delivery Service`

		.. seealso:: :ref:`anonymous_blocking-qht`

	:consistentHashQueryParameters: A set of query parameters that Traffic Router should consider when determining a consistent hash for a given client request.

	:consistentHashRegex:           An optional regular expression that will ensure clients are consistently routed to a :term:`cache server` based on matches to it.

	:coverageZoneOnly:              A string containing a boolean that tells whether or not this :term:`Delivery Service` routes traffic based only on its :term:`Coverage Zone File`

		.. seealso:: :ref:`ds-geo-limit`

	:deepCachingType: A string that defines the :ref:`ds-deep-caching` setting of this :term:`Delivery Service`
	:dispersion:      An object describing the "dispersion" - or number of :term:`cache servers` within a single :term:`Cache Group` across which the same content is spread - within the :term:`Delivery Service`

		:limit: The maximum number of :term:`cache servers` in which the response to a single request URL will be stored

			.. note:: If this is greater than the number of :term:`cache servers` in the :term:`Cache Group` chosen to service the request, then content will be spread across all of them. That is, it causes no problems.

		:shuffled: A string containing a boolean that tells whether the :term:`cache servers` chosen for content dispersion are chosen randomly or based on a consistent hash of the request URL; one of:

			"false"
				:term:`cache servers` will be chosen consistently
			"true"
				:term:`cache servers` will be chosen at random

	:domains:             An array of domains served by this :term:`Delivery Service`
	:ecsEnabled:          A string containing a boolean from :ref:`ds-ecs` that tells whether EDNS0 client subnet is enabled on this :term:`Delivery Service`; one of:

		"false"
			EDNS0 client subnet is not enabled on this :term:`Delivery Service`
		"true"
			EDNS0 client subnet is enabled on this :term:`Delivery Service`

	:geolocationProvider: The name of a provider for IP-to-geographic-location mapping services - currently the only valid value is ``"maxmindGeolocationService"``
	:ip6RoutingEnabled:   A string containing a boolean that defines the :ref:`ds-ipv6-routing` setting for this :term:`Delivery Service`; one of:

		"false"
			IPv6 traffic will not be routed by this :term:`Delivery Service`
		"true"
			IPv6 traffic will be routed by this :term:`Delivery Service`

	:matchList: An array of methods used by Traffic Router to determine whether or not a request can be serviced by this :term:`Delivery Service`

		:pattern:   A regular expression - the use of this pattern is dependent on the ``type`` field (backslashes are escaped)
		:setNumber: An integral, unique identifier for the set of types to which the ``type`` field belongs
		:type:      The name of the :term:`Type` of match performed using ``pattern`` to determine whether or not to use this :term:`Delivery Service`

			HOST_REGEXP
				Use the :term:`Delivery Service` if ``pattern`` matches the :mailheader:`Host` HTTP header of an HTTP request, or the name requested for resolution in a DNS request
			HEADER_REGEXP
				Use the :term:`Delivery Service` if ``pattern`` matches an HTTP header (both the name and value) in an HTTP request\ [#httpOnly]_
			PATH_REGEXP
				Use the :term:`Delivery Service` if ``pattern`` matches the request path of this :term:`Delivery Service`'s URL\ [#httpOnly]_
			STEERING_REGEXP
				Use the :term:`Delivery Service` if ``pattern`` matches the :ref:`ds-xmlid` of one of this :term:`Delivery Service`'s "Steering" target :term:`Delivery Services`

	:missLocation: An object representing the default geographic coordinates to use for a client when lookup of their IP has failed in both the :term:`Coverage Zone File` (and/or possibly the :term:`Deep Coverage Zone File`) and the IP-to-geographic-location database

		:lat:  Geographic latitude as a floating point number
		:long: Geographic longitude as a floating point number

	:protocol: An object that describes how the :term:`Delivery Service` ought to handle HTTP requests both with and without TLS encryption

		:acceptHttps: A string containing a boolean that tells whether HTTPS requests should be normally serviced by this :term:`Delivery Service`; one of:

			"false"
				Refuse to service HTTPS requests
			"true"
				Service HTTPS requests normally

		:redirectToHttps: A string containing a boolean that tells whether HTTP requests ought to be re-directed to use HTTPS; one of:

			"false"
				Do not redirect unencrypted traffic; service it normally
			"true"
				Respond to HTTP requests with instructions to use HTTPS instead

		.. seealso:: :ref:`ds-protocol`

	:regionalGeoBlocking: A string containing a boolean that defines the :ref:`ds-regionalgeo` setting of this :term:`Delivery Service`; one of:

		"false"
			Regional Geographic Blocking is not used by this :term:`Delivery Service`
		"true"
			Regional Geographic Blocking is used by this :term:`Delivery Service`

		.. seealso:: :ref:`regionalgeo-qht`
	:requiredCapabilities: An array of this Delivery Service's :term:`required capabilities <Delivery Service required capabilities>`. If there are no required capabilities, this field is omitted.
	:routingName: A string that is this :ref:`Delivery Service's Routing Name <ds-routing-name>`
	:soa:         An object defining the :abbr:`SOA (Start of Authority)` record for the :term:`Delivery Service`'s :abbr:`TLDs (Top-Level Domains)` (defined in ``domains``)

		:admin: The name of the administrator for this zone - i.e. the RNAME

			.. note:: This rarely represents a proper email address, unfortunately.

		:expire:  A string containing an integer that sets the number of seconds after which secondary name servers should stop answering requests for this zone if the master does not respond
		:minimum: A string containing an integer that sets the :abbr:`TTL (Time To Live)` - in seconds - of the record for the purpose of negative caching
		:refresh: A string containing an integer that sets the number of seconds after which secondary name servers should query the master for the :abbr:`SOA (Start of Authority)` record, to detect zone changes
		:retry:   A string containing an integer that sets the number of seconds after which secondary name servers should retry to request the serial number from the master if the master does not respond

			.. note:: :rfc:`1035` dictates that this should always be less than ``refresh``.

		.. seealso:: `The Wikipedia page on Start of Authority records <https://en.wikipedia.org/wiki/SOA_record>`_.

	:sslEnabled: A string containing a boolean that tells whether this :term:`Delivery Service` uses SSL; one of:

		"false"
			SSL is not used by this :term:`Delivery Service`
		"true"
			SSL is used by this :term:`Delivery Service`

		.. seealso:: :ref:`ds-protocol`

	:topology: The name of the :term:`Topology` that this :term:`Delivery Service` is assigned to. If the Delivery Service is not assigned to a topology, this field is omitted.
	:ttls: An object that contains keys which are types of DNS records that have values which are strings containing integers that specify the time for which a response to the specific type of record request should remain valid

		.. note:: This overrides ``config.ttls``.

:edgeLocations: An object containing keys which are the names of Edge-Tier :term:`Cache Groups` within the CDN

	:backupLocations: An object that describes this :ref:`Cache Group's Fallbacks <cache-group-fallbacks>`

		:fallbackToClosest: A string containing a boolean which defines the :ref:`cache-group-fallback-to-closest` behavior of this :term:`Cache Group`; one of:

			"false"
				Do not fall back on the closest available :term:`Cache Group`
			"true"
				Fall back on the closest available :term:`Cache Group`

		:list: If this :term:`Cache Group` has any :ref:`cache-group-fallbacks`, this key will appear and will be an array of those :ref:`Cache Groups' Names <cache-group-name>`

	:latitude:            A floating point number that defines this :ref:`Cache Group's Latitude <cache-group-latitude>`
	:localizationMethods: An array of strings that represents this :ref:`Cache Group's Localization Methods <cache-group-localization-methods>`
	:longitude:           A floating point number that defines this :ref:`Cache Group's Longitude <cache-group-longitude>`

:monitors: An object containing keys which are the (short) hostnames of Traffic Monitors within this CDN

	:fqdn:      The :abbr:`FQDN (Fully Qualified Domain Name)` of this Traffic Monitor
	:httpsPort: The port number on which this Traffic Monitor listens for incoming HTTPS requests
	:ip6:       This Traffic Monitor's IPv6 address
	:ip:        This Traffic Monitor's IPv4 address
	:location:  A string which is the :ref:`cache-group-name` of the :term:`Cache Group` to which this Traffic Monitor belongs
	:port:      The port number on which this Traffic Monitor listens for incoming HTTP requests
	:profile:   A string which is the :ref:`profile-name` of the :term:`Profile` used by this Traffic Monitor

		.. note:: For legacy reasons, this must always start with "RASCAL-".

	:status: The health status of this Traffic Monitor

		.. seealso:: :ref:`health-proto`

:stats: An object containing metadata information regarding the CDN

	:CDN_name:   The name of this CDN
	:date:       The UNIX epoch timestamp date in the Traffic Ops server's own timezone
	:tm_host:    The :abbr:`FQDN (Fully Qualified Domain Name)` of the Traffic Ops server

		.. deprecated:: ATCv6

			This information should never be used; instead all tools and (especially) components **must** use the documented API. This field was removed in APIv4

	:tm_path:    A path relative to the root of the Traffic Ops server where a request may be replaced to have this :term:`Snapshot` overwritten by the current *configured state* of the CDN
	:tm_user:    The username of the currently logged-in user
	:tm_version: The full version number of the Traffic Ops server, including release number, git commit hash, and supported Enterprise Linux version

:topologies: An array of :term:`Topologies` where each key is the name of that Topology.

	:nodes: An array of the names of the :term:`Edge-Tier` :term:`Cache Groups` in this :term:`Topology`. :term:`Mid-Tier` Cache Groups in the topology are not included.

:trafficRouterLocations: An object containing keys which are the :ref:`names of Cache Groups <cache-group-name>` within the CDN which contain Traffic Routers

	:backupLocations: An object that describes this :ref:`Cache Group's Fallbacks <cache-group-fallbacks>`

		:fallbackToClosest: A string containing a boolean which defines this :ref:`Cache Group's Fallback to Closest <cache-group-fallback-to-closest>` setting; one of:

			"false"
				Do not fall back on the closest available :term:`Cache Group`
			"true"
				Fall back on the closest available :term:`Cache Group`

	:latitude:            A floating point number that defines this :ref:`Cache Group's Latitude <cache-group-latitude>`
	:localizationMethods: An array of strings that represents this :ref:`Cache Group's Localization Methods <cache-group-localization-methods>`
	:longitude:           A floating point number that defines this :ref:`Cache Group's Longitude <cache-group-longitude>`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Wed, 27 May 2020 18:33:17 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: B5qdN9URIfu11gQxPZ8YaaMvy2HMrzsnrpt6vF037yv6OQiKCRyrUMX6wYs7QW4YVaeUrvmS2ya5l2YC0kvNAg==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 27 May 2020 17:33:17 GMT
	Content-Length: 1360


	{
		"response": {
		    "config": {
			"api.cache-control.max-age": "10",
			"certificates.polling.interval": "300000",
			"consistent.dns.routing": "true",
			"coveragezone.polling.interval": "3600000",
			"coveragezone.polling.url": "https://trafficops.infra.ciab.test:443/coverage-zone.json",
			"dnssec.dynamic.response.expiration": "300s",
			"dnssec.enabled": "false",
			"domain_name": "mycdn.ciab.test",
			"federationmapping.polling.interval": "60000",
			"federationmapping.polling.url": "https://${toHostname}/api/2.0/federations/all",
			"geolocation.polling.interval": "86400000",
			"geolocation.polling.url": "https://trafficops.infra.ciab.test:443/GeoLite2-City.mmdb.gz",
			"keystore.maintenance.interval": "300",
			"neustar.polling.interval": "86400000",
			"neustar.polling.url": "https://trafficops.infra.ciab.test:443/neustar.tar.gz",
			"soa": {
			    "admin": "twelve_monkeys",
			    "expire": "604800",
			    "minimum": "30",
			    "refresh": "28800",
			    "retry": "7200"
			},
			"steeringmapping.polling.interval": "60000",
			"ttls": {
			    "A": "3600",
			    "AAAA": "3600",
			    "DNSKEY": "30",
			    "DS": "30",
			    "NS": "3600",
			    "SOA": "86400"
			},
			"zonemanager.cache.maintenance.interval": "300",
			"zonemanager.threadpool.scale": "0.50"
		    },
		    "contentRouters": {
			"trafficrouter": {
			    "api.port": "3333",
			    "fqdn": "trafficrouter.infra.ciab.test",
			    "httpsPort": 443,
			    "ip": "172.26.0.15",
			    "ip6": "",
			    "location": "CDN_in_a_Box_Edge",
			    "port": 80,
			    "profile": "CCR_CIAB",
			    "secure.api.port": "3443",
			    "status": "ONLINE"
			}
		    },
		    "contentServers": {
			"edge": {
			    "cacheGroup": "CDN_in_a_Box_Edge",
			    "capabilities": [
				"RAM_DISK_STORAGE"
			    ],
			    "fqdn": "edge.infra.ciab.test",
			    "hashCount": 999,
			    "hashId": "edge",
			    "httpsPort": 443,
			    "interfaceName": "eth0",
			    "ip": "172.26.0.3",
			    "ip6": "",
			    "locationId": "CDN_in_a_Box_Edge",
			    "port": 80,
			    "profile": "ATS_EDGE_TIER_CACHE",
			    "routingDisabled": 0,
			    "status": "REPORTED",
			    "type": "EDGE"
			},
			"mid": {
			    "cacheGroup": "CDN_in_a_Box_Mid",
			    "capabilities": [
				"RAM_DISK_STORAGE"
			    ],
			    "fqdn": "mid.infra.ciab.test",
			    "hashCount": 999,
			    "hashId": "mid",
			    "httpsPort": 443,
			    "interfaceName": "eth0",
			    "ip": "172.26.0.4",
			    "ip6": "",
			    "locationId": "CDN_in_a_Box_Mid",
			    "port": 80,
			    "profile": "ATS_MID_TIER_CACHE",
			    "routingDisabled": 0,
			    "status": "REPORTED",
			    "type": "MID"
			}
		    },
		    "deliveryServices": {
			"demo1": {
			    "anonymousBlockingEnabled": "false",
			    "consistentHashQueryParams": [
				"abc",
				"pdq",
				"xxx",
				"zyx"
			    ],
			    "coverageZoneOnly": "false",
			    "deepCachingType": "NEVER",
			    "dispersion": {
				"limit": 1,
				"shuffled": "true"
			    },
			    "domains": [
				"demo1.mycdn.ciab.test"
			    ],
			    "ecsEnabled": "false",
			    "geolocationProvider": "maxmindGeolocationService",
			    "ip6RoutingEnabled": "true",
			    "matchsets": [
				{
				    "matchlist": [
					{
					    "match-type": "HOST",
					    "regex": ".*\\.demo1\\..*"
					}
				    ],
				    "protocol": "HTTP"
				}
			    ],
			    "missLocation": {
				"lat": 42,
				"long": -88
			    },
			    "protocol": {
				"acceptHttps": "true",
				"redirectToHttps": "false"
			    },
			    "regionalGeoBlocking": "false",
			    "requiredCapabilities": [
				"RAM_DISK_STORAGE"
			    ],
			    "routingName": "video",
			    "soa": {
				"admin": "traffic_ops",
				"expire": "604800",
				"minimum": "30",
				"refresh": "28800",
				"retry": "7200"
			    },
			    "sslEnabled": "true",
			    "topology": "my-topology",
			    "ttls": {
				"A": "",
				"AAAA": "",
				"NS": "3600",
				"SOA": "86400"
			    }
			}
		    },
		    "edgeLocations": {
			"CDN_in_a_Box_Edge": {
			    "backupLocations": {
				"fallbackToClosest": "true"
			    },
			    "latitude": 38.897663,
			    "localizationMethods": [
				"GEO",
				"CZ",
				"DEEP_CZ"
			    ],
			    "longitude": -77.036574
			}
		    },
		    "monitors": {
			"trafficmonitor": {
			    "fqdn": "trafficmonitor.infra.ciab.test",
			    "httpsPort": 443,
			    "ip": "172.26.0.14",
			    "ip6": "",
			    "location": "CDN_in_a_Box_Edge",
			    "port": 80,
			    "profile": "RASCAL-Traffic_Monitor",
			    "status": "ONLINE"
			}
		    },
		    "stats": {
			"CDN_name": "CDN-in-a-Box",
			"date": 1590600715,
			"tm_host": "trafficops.infra.ciab.test:443",
			"tm_path": "/api/3.0/snapshot",
			"tm_user": "admin",
			"tm_version": "development"
		    },
		    "topologies": {
			"my-topology": {
			    "nodes": [
				"CDN_in_a_Box_Edge"
			    ]
			}
		    },
		    "trafficRouterLocations": {
			"CDN_in_a_Box_Edge": {
			    "backupLocations": {
				"fallbackToClosest": "false"
			    },
			    "latitude": 38.897663,
			    "localizationMethods": [
				"GEO",
				"CZ",
				"DEEP_CZ"
			    ],
			    "longitude": -77.036574
			}
		    }
		}
	}


.. [#httpOnly] These only apply to HTTP-:ref:`routed <ds-types>` :term:`Delivery Services`
