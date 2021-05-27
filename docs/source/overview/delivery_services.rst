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

.. _delivery-services:

*****************
Delivery Services
*****************
"Delivery Services" are a very important construct in :abbr:`ATC (Apache Traffic Control)`. At their most basic, they are a source of content and a set of :term:`cache servers` and configuration options used to distribute that content.

Delivery Services are modeled several times over, in the Traffic Ops database, in Traffic Portal forms and tables, and several times for various :ref:`to-api` versions in the new Go Traffic Ops codebase. Go-specific data structures can be found in `the project's GoDoc documentation <https://pkg.go.dev/github.com/apache/trafficcontrol/lib/go-tc#DeliveryServiceV4>`_. Rather than application-specific definitions, what follows is an attempt at consolidating all of the different properties and names of properties of Delivery Service objects throughout the :abbr:`ATC (Apache Traffic Control)` suite. The names of these fields are typically chosen as the most human-readable and/or most commonly-used names for the fields, and when reading please note that in many cases these names will appear camelCased or snake_cased to be machine-readable. Any aliases of these fields that are not merely case transformations of the indicated, canonical names will be noted in a table of aliases.

.. seealso:: The API reference for Delivery Service-related endpoints such as :ref:`to-api-deliveryservices` contains definitions of the Delivery Service object(s) returned and/or accepted by those endpoints.
.. seealso:: :ref:`delivery-service-requests`

.. _ds-active:

Active
------
"Active" defines whether or not this Delivery Service is routed by Traffic Router and whether or not :term:`cache servers` have the necessary configuration to serve its content. In nearly every case, this is a string which may have one of the following values with the associated meanings:

ACTIVE
	This Delivery Service is both routed (appears in CDN :term:`Snapshots`) and served by :term:`cache servers`.
PRIMED
	This Delivery Service is not routed (doesn't appear in CDN :term:`Snapshots`), but is still served by :term:`cache servers`.
INACTIVE
	This Delivery Service is neither routed (doesn't appear in CDN :term:`Snapshots`), nor is its content served by :term:`cache servers`.

Setting a Delivery Service's Active property in a way that changes whether or not it is routed will require that a new :term:`Snapshot` be taken, while setting it in a way that changes whether or not its content is served by :term:`cache servers` will require that a :term:`Queue Server Updates` is performed on the affected :term:`cache servers`.

.. versionchanged:: ATCv6/APIv4
	Changed from a strict boolean to an enumeration of the herein listed values. In versions of :abbr:`ATC (Apache Traffic Control)` later than 6.0, :ref:`to-api` versions earlier than 4.0 will report ACTIVE Delivery Services with Active properties set to ``true``, and PRIMED or INACTIVE Delivery Services with Active properties set to ``false``.

.. _ds-anonymous-blocking:

Anonymous Blocking
------------------
Enables/Disables blocking of anonymized IP address - proxies, :abbr:`TOR (The Onion Ring)` exit nodes, etc - for this Delivery Service. Set to true to enable blocking of anonymous IPs for this Delivery Service.

.. table:: Aliases

	+--------------------------+-----------------------------------------------------------------------------+-----------------------------------------------------------------------------------------+
	| Name                     | Use(s)                                                                      | Type(s)                                                                                 |
	+==========================+=============================================================================+=========================================================================================+
	| anonymousBlockingEnabled | Traffic Ops client and server Go code, :ref:`to-api` requests and responses | usually unchanged (boolean), but sometimes as a string containing a boolean e.g. in the |
	|                          |                                                                             | response of a ``GET`` request to :ref:`to-api-cdns-name-snapshot`                       |
	+--------------------------+-----------------------------------------------------------------------------+-----------------------------------------------------------------------------------------+

.. note:: Anonymous Blocking requires an anonymous IP address database from the Delivery Service's Geolocation Provider. E.g. `MaxMind's Anonymous IP Database <https://www.maxmind.com/en/solutions/geoip2-enterprise-product-suite/anonymous-ip-database>`_ when MaxMind is used as the Geolocation Provider.

.. seealso:: The :ref:`anonymous_blocking-qht` "Quick-How-To" guide.

.. _ds-cacheurl:

Cache URL Expression
--------------------
.. deprecated:: 3.0
	This feature is no longer supported by :abbr:`ATS (Apache Traffic Server)` and consequently it will be removed from Traffic Control in the future. Current plans are to remove after ATC 5.X is no longer supported.

Manipulates the cache key of the incoming requests. Normally, the cache key is the :term:`Origin` domain. This can be changed so that multiple services can share a cache key, can also be used to preserve cached content if service origin is changed.

.. warning:: This field provides access to a feature that was only present in :abbr:`ATS (Apache Traffic Server)` 6.X and earlier. As :term:`cache servers` must now use :abbr:`ATS (Apache Traffic Server)` 7.1.X, this field **must** be blank unless all :term:`cache servers` can be guaranteed to use that older :abbr:`ATS (Apache Traffic Server)` version (**NOT** recommended).

.. _ds-cdn:

CDN
---
A CDN to which this Delivery Service belongs. Only :term:`cache servers` within this CDN are available to route content for this Delivery Service. Additionally, only Traffic Routers assigned to this CDN will perform said routing. Most often ``cdn``/``CDN`` refers to the *name* of the CDN to which the Delivery Service belongs, but occasionally (most notably in the payloads and/or query parameters of certain :ref:`to-api` endpoints) it actually refers to the *integral, unique identifier* of said CDN.

.. _ds-check-path:

Check Path
----------
A request path on the :term:`origin server` which is used to by certain :ref:`Traffic Ops Extensions <admin-to-ext-script>` to indicate the "health" of the :term:`Origin`.

.. _ds-consistent-hashing-regex:

Consistent Hashing Regular Expression
-------------------------------------
When Traffic Router performs :ref:`consistent-hashing` on a client request to find an :term:`Edge-tier cache server` to which to redirect them, it can optionally first modify the request path by extracting the pieces that match this regular expression.

.. seealso:: :ref:`pattern-based-consistenthash`

.. table:: Aliases

	+----------------------------------+---------------------------------------------------------+----------------------------------------------------------------------------------------------------+
	| Name                             | Use(s)                                                  | Type(s)                                                                                            |
	+==================================+=========================================================+====================================================================================================+
	| consistentHashRegex              | In source code and :ref:`to-api` requests and responses | unchanged (regular expression)                                                                     |
	+----------------------------------+---------------------------------------------------------+----------------------------------------------------------------------------------------------------+
	| pattern-based consistent hashing | documentation and the Traffic Portal UI                 | unchanged (regular expression), but usually used when discussing the concept rather than the field |
	+----------------------------------+---------------------------------------------------------+----------------------------------------------------------------------------------------------------+

.. _ds-consistent-hashing-qparams:

Consistent Hashing Query Parameters
-----------------------------------
When Traffic Router performs :ref:`consistent-hashing` on a client request to find an :term:`Edge-tier cache server` to which to redirect them, it can optionally take into account any number of query parameters. This field defines them, formally as a Set but often represented as an Array/List due to encoding limitations. That is, if the Consistent Hashing Query Parameters on a Delivery Service are ``{test}`` and a client makes a request for ``/?test=something`` they will be directed to a different :term:`cache server` than a different client that requests ``/?test=somethingElse``, but the *same* :term:`cache server` as a client that requests ``/?test=something&quest=somethingToo``.

.. table:: Aliases

	+---------------------------+--------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+
	| Name                      | Use(s)                                                                   | Type(s)                                                                                        |
	+===========================+==========================================================================+================================================================================================+
	| consistentHashQueryParams | In source code, Traffic Portal, and :ref:`to-api` requests and responses | unchanged (Array of strings - should ALWAYS be unique, thus treated as a Set in most contexts) |
	+---------------------------+--------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+

.. _ds-deep-caching:

Deep Caching
------------
Controls the :ref:`deep-cache` feature of Traffic Router when serving content for this Delivery Service. This should always be represented by one of two values:

ALWAYS
	This Delivery Service will always use :ref:`deep-cache`
NEVER
	This Delivery Service will never use :ref:`deep-cache`

.. impl-detail:: Traffic Ops and Traffic Ops client Go code use an empty string as the name of the enumeration member that represents "NEVER".

.. _ds-display-name:

Display Name
------------
The "name" of the Delivery Service. Since nearly any use of a string-based identification method for Delivery Services (e.g. in Traffic Portal tables) uses xml_id_, this is of limited use. For that reason and for consistency's sake it is suggested that this be the same as the xml_id_. However, unlike the xml_id_, this can contain any UTF-8 characters without restriction.

.. _ds-dns-bypass-cname:

DNS Bypass CNAME
----------------
When the limits placed on this Delivery Service by the `Global Max Mbps`_ and/or `Global Max Tps`_ are exceeded, a DNS-:ref:`Routed <ds-types>` Delivery Service will direct excess traffic to the host referred to by this :abbr:`CNAME (Canonical Name)` record.

.. note:: IPv6 traffic will be redirected if and only if `IPv6 Routing Enabled`_ is "true" for this Delivery Service.

.. _ds-dns-bypass-ip:

DNS Bypass IP
-------------
When the limits placed on this Delivery Service by the `Global Max Mbps`_ and/or `Global Max Tps`_ are exceeded, a DNS-:ref:`Routed <ds-types>` Delivery Service will direct excess IPv4 traffic to this IPv4 address.

.. _ds-dns-bypass-ipv6:

DNS Bypass IPv6
---------------
When the limits placed on this Delivery Service by the `Global Max Mbps`_ and/or `Global Max Tps`_ are exceeded, a DNS-:ref:`Routed <ds-types>` Delivery Service will direct excess IPv6 traffic to this IPv6 address.

.. note:: This requires an accompanying configuration of `IPv6 Routing Enabled`_ such that IPv6 traffic is allowed at all.

.. _ds-dns-bypass-ttl:

DNS Bypass TTL
--------------
When the limits placed on this Delivery Service by the `Global Max Mbps`_ and/or `Global Max Tps`_ are exceeded, a DNS-:ref:`Routed <ds-types>` Delivery Service will direct excess traffic to their `DNS Bypass IP`_, `DNS Bypass IPv6`_, or `DNS Bypass CNAME`_.

.. _ds-dns-ttl:

DNS TTL
-------
The :abbr:`TTL (Time To Live)` on the DNS record for the Traffic Router A and AAAA records. DNS-:ref:`Routed <ds-types>` Delivery Services will send this :abbr:`TTL (Time To Live)` along with their record responses to clients requesting access to this Delivery Service. Setting too high or too low will result in poor caching performance.

.. table:: Aliases

	+-------------+--------------------------------------------------------------------------------------+---------------------------------------------+
	| Name        | Use(s)                                                                               | Type(s)                                     |
	+=============+======================================================================================+=============================================+
	| CCR DNS TTL | In Delivery Service objects returned by the :ref:`to-api`                            | unchanged (``int``, ``integer`` etc.)       |
	+-------------+--------------------------------------------------------------------------------------+---------------------------------------------+
	| CCR TTL     | Legacy Traffic Ops UI, documentation for older Traffic Control versions              | unchanged (``int``, ``integer`` etc.)       |
	+-------------+--------------------------------------------------------------------------------------+---------------------------------------------+
	| ttl         | In CDN :term:`Snapshot` structures, where it is displayed on a per-record-type-basis | map of record type names to integral values |
	+-------------+--------------------------------------------------------------------------------------+---------------------------------------------+

.. _ds-dscp:

DSCP
----
The :abbr:`DSCP (Differentiated Services Code Point)` which will be used to mark IP packets as they are sent out of the CDN to the client.

.. seealso:: `The Differentiated Services Wikipedia article <https://en.wikipedia.org/wiki/Differentiated_services>`_.

.. warning:: The :abbr:`DSCP (Differentiated Services Code Point)` setting in Traffic Portal is *only* for setting traffic towards the client, and gets applied *after* the initial TCP handshake is complete and the HTTP request has been received. Before that the cache can't determine what Delivery Service is being requested, and consequently can't know what :abbr:`DSCP (Differentiated Services Code Point)` to apply. Therefore, the :abbr:`DSCP (Differentiated Services Code Point)` feature can not be used for security settings; the IP packets that form the TCP handshake are not going to be :abbr:`DSCP (Differentiated Services Code Point)`-marked.

.. impl-detail:: DSCP settings only apply on :term:`cache servers` that run :abbr:`Apache Traffic Server`. The implementation uses the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ to create a rule that will mark traffic bound outward from the CDN to the client.

.. _ds-edge-header-rw-rules:

Edge Header Rewrite Rules
-------------------------
This field in general contains the contents of the a configuration file used by the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ when serving content for this Delivery Service - on :term:`Edge-tier cache servers`.

.. tip:: Because this ultimately is the contents of an :abbr:`ATS (Apache Traffic Server)` configuration file, it can make use of the :ref:`ort-special-strings`.

.. note:: This field cannot be used if the Delivery Service is assigned to a :term:`Topology`.

.. _ds-ecs:

EDNS0 Client Subnet Enabled
---------------------------
A boolean value that controls whether or not EDNS0 client subnet is enabled on this Delivery Service by Traffic Router. When creating a Delivery Service in Traffic Portal, this will default to "false".

.. _ds-example-urls:

Example URLs
------------
The Example URLs of a Delivery Service are the scheme/host specifications that clients can use to request content through it. These are determined by Traffic Ops from the Delivery Service's configuration, and are read-only in virtually every context. The only reason a Delivery Service should ever have no Example URLs is if it is an ANY_MAP-`Type`_ Delivery Service (since they are not routed). For example, a Delivery Service that can deliver HTTP and HTTPS content, has a `Routing Name`_ of "cdn", an `xml_id`_ of "demo1", and belonging to a `CDN`_ that is authoritative for the `mycdn.ciab.test` domain would have two Example URLs:

- `https://cdn.demo1.mycdn.ciab.test`
- `http://cdn.demo1.mycdn.ciab.test`

Note that these are irrespective of request path; meaning a client can request e.g. `https://cdn.demo1.mycdn.ciab.test/index.html` through this Delivery Service.

.. warning:: This list does not consider any `Static DNS Entries`_ configured on the Delivery Service, those are

.. table:: Aliases

	+-----------------------+----------------------+-----------------------------+
	| Name                  | Use(s)               | Type(s)                     |
	+=======================+======================+=============================+
	| Delivery Service URLs | Traffic Portal forms | unchanged (list of strings) |
	+-----------------------+----------------------+-----------------------------+

.. _ds-fqpr:

Fair-Queuing Pacing Rate Bps
----------------------------
The maximum bytes per second a :term:`cache server` will deliver on any single TCP connection. This uses the Linux kernel’s Fair-Queuing :manpage:`setsockopt(2)` (``SO_MAX_PACING_RATE``) to limit the rate of delivery. Traffic exceeding this speed will only be rate-limited and not diverted. This option requires extra configuration on all :term:`cache servers` assigned to this Delivery Service - specifically, the line ``net.core.default_qdisc = fq`` must exist in :file:`/etc/sysctl.conf`.

.. seealso:: :manpage:`tc-fq_codel(8)`

.. seealso:: This is implemented using the `ATS fq_pacing plign <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/fq_pacing.en.html>`_.

.. table:: Aliases

	+--------------+---------------------------------------------------------------------------------+---------------------------------------+
	| Name         | Use(s)                                                                          | Type(s)                               |
	+==============+=================================================================================+=======================================+
	| FQPacingRate | Traffic Ops source code, Delivery Service objects returned by the :ref:`to-api` | unchanged (``int``, ``integer`` etc.) |
	+--------------+---------------------------------------------------------------------------------+---------------------------------------+

.. _ds-first-header-rw-rules:

First Header Rewrite Rules
--------------------------
This field in general contains the contents of the a configuration file used by the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ when serving content for this Delivery Service - on :term:`First-tier cache servers`.

.. tip:: Because this ultimately is the contents of an :abbr:`ATS (Apache Traffic Server)` configuration file, it can make use of the :ref:`ort-special-strings`.

.. note:: This field can only be used if the Delivery Service is assigned to a :term:`Topology`.

.. _ds-geo-limit:

Geo Limit
---------
Limits access to a Delivery Service by geographic location. The only practical difference between this and `Regional Geoblocking`_ is the configuration method; as opposed to `Regional Geoblocking`_, GeoLimit configuration is handled by country-wide codes and the :term:`Coverage Zone File`. When a client is denied access to a requested resource on an HTTP-:ref:`Routed <ds-types>` Delivery Service, they will receive a ``503 Service Unavailable`` instead of the usual ``302 Found`` response - unless `Geo Limit Redirect URL`_ is defined, in which case a ``302 Found`` response pointing to that URL will be returned by Traffic Router. If the Delivery Service is a DNS-:ref:`Routed <ds-types>` Delivery Service, the IP address of the *resolver* for the client DNS request is what is checked. If the IP address of this resolver is found to be in a restricted location, the Traffic Router will respond with an ``NXDOMAIN`` response, causing the name resolution to fail. This is nearly always an integral, unique identifier for a behavior set to be followed by Traffic Router. The defined values are:

0
	Geographic access limiting is not enabled, and content served by this Delivery Service will be accessible regardless of the clients geographic location. (Aliased as "0 - None" in Traffic Portal forms)
1
	A client will be allowed to request content if and only if their IP address is found by Traffic Router within the :term:`Coverage Zone File`. Otherwise, access will be denied. (Aliased as "1 - CZF Only" in Traffic Portal forms)
2
	A client will be allowed to request content if their IP address is found by Traffic Router within the :term:`Coverage Zone File`, or if looking up the client's IP address in the Geographic IP mapping database provided by `Geolocation Provider`_ indicates the client resides in a country that is found in the `Geo Limit Countries`_ array. (Aliased as "2 - CZF + Country Code(s)" in Traffic Portal forms - formerly was known as "CZF + US" when only the US country code was supported)

.. warning:: The definitions of each integral, unique identifier are hidden in implementations in each :abbr:`ATC (Apache Traffic Control)` component. Different components will handle invalid values differently, and there's no actual enforcement that the stored integral, unique identifier actually be within the representable range.

.. table:: Aliases

	+------------------+---------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+
	| Name             | Use(s)                                                                    | Type(s)                                                                                        |
	+==================+===========================================================================+================================================================================================+
	| coverageZoneOnly | In CDN :term:`Snapshot` structures, especially in :ref:`to-api` responses | A boolean which, if ``true``, tells Traffic Router to only service requests when the client IP |
	|                  |                                                                           | address is found in the :term:`Coverage Zone File`                                             |
	+------------------+---------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+

.. danger:: Geographic access limiting is **not** sufficient to guarantee access is properly restricted. The limiting is implemented by Traffic Router, which means that direct requests to :term:`Edge-tier cache servers` will bypass it entirely.

.. _ds-geo-limit-countries:

Geo Limit Countries
-------------------
When `Geo Limit`_ is being used with this Delivery Service (and is set to exactly ``2``), this is optionally a list of country codes to which access to content provided by the Delivery Service will be restricted. Normally, this is a comma-delimited string of said country codes. When creating a Delivery Service with this field or modifying the Geo Limit Countries field on an existing Delivery Service, any amount of whitespace between country codes is permissible, as it will be removed on submission, but responses from the :ref:`to-api` should never include such whitespace.

.. table:: Aliases

	+------------------+---------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+
	| Name             | Use(s)                                                                    | Type(s)                                                                                        |
	+==================+===========================================================================+================================================================================================+
	| geoEnabled       | In CDN :term:`Snapshot` structures, especially in :ref:`to-api` responses | An array of objects each having the key "countryCode" that is a string containing an allowed   |
	|                  |                                                                           | country code - one should exist for each allowed country code                                  |
	+------------------+---------------------------------------------------------------------------+------------------------------------------------------------------------------------------------+

.. _ds-geo-limit-redirect-url:

Geo Limit Redirect URL
----------------------
If `Geo Limit`_ is being used with this Delivery Service, this is optionally a URL to which clients will be redirected when Traffic Router determines that they are not in a geographic zone that permits their access to the Delivery Service content. This changes the response from Traffic Router from ``503 Service Unavailable`` to ``302 Found`` with a provided location that will be this URL. There is no restriction on the provided URL; it may even be the path to a resource served by this Delivery Service. In fact, this field need not even be a full URL, it can be a relative path. Both of these cases are handled specially by Traffic Router.

- If the provided URL is a resource served by the Delivery Service (e.g. if the client requests ``http://cdn.dsXMLID.somedomain.example.com/index.html`` but are denied access by `Geo Limit`_ and the Geo Limit Redirect URL is something like ``http://cdn.dsXMLID.somedomain.example.com/help.php``), Traffic Router will find an appropriate :term:`Edge-tier cache server` and redirect the client, ignoring Geo Limit restrictions *for this request only*.
- If the provided "URL" is actually a relative path, it will be considered *relative to the requested Delivery Service :abbr:`FQDN (Fully Qualified Domain Name)`*. This means that e.g. if the client requests ``http://cdn.dsXMLID.somedomain.example.com/index.html`` but are denied access by `Geo Limit`_ and the Geo Limit Redirect URL is something like ``/help.php``, Traffic Router will find an appropriate :term:`Edge-tier cache server` and redirect the client to it as though they had requested ``http://cdn.dsXMLID.somedomain.example.com/help.php``, ignoring `Geo Limit`_ restrictions *for this request only*.

.. table:: Aliases

	+---------------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------------------------+
	| Name                            | Use(s)                                                         | Type(s)                                                                                         |
	+=================================+================================================================+=================================================================================================+
	| :abbr:`NGB (National GeoBlock)` | Older documentation, in Traffic Router comments and error logs | unchanged (``string``, ``String`` etc.)                                                         |
	+---------------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------------------------+
	| geoRedirectURLType              | Internally in Traffic Router                                   | A ``String`` that describes whether or not the actual Geo Limit Redirect URL is relative to the |
	|                                 |                                                                | Delivery Service base :abbr:`FQDN (Fully Qualified Domain Name)`. Should be one of:             |
	|                                 |                                                                |                                                                                                 |
	|                                 |                                                                | INVALID_URL                                                                                     |
	|                                 |                                                                |     The Geo Limit Redirect URL has not yet been parsed, or an error occurred during parsing     |
	|                                 |                                                                | DS_URL                                                                                          |
	|                                 |                                                                |     The Geo Limit Redirect URL is served by this Delivery Service                               |
	|                                 |                                                                | NOT_DS_URL                                                                                      |
	|                                 |                                                                |     The Geo Limit Redirect URL is external to this Delivery Service                             |
	+---------------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------------------------+

.. note:: The use of a redirect URL relies on the ability of Traffic Router to redirect the client using HTTP ``302 Found`` responses. As such, this field has no effect on DNS-:ref:`Routed <ds-types>` Delivery Services.

.. _ds-geo-provider:

Geolocation Provider
--------------------
This is nearly always the integral, unique identifier of a provider for a database that maps IP addresses to geographic locations. Less frequently, this may be accompanied by the actual name of the provider. Only two values are possible at the time of this writing:

0: MaxMind
	IP address to geographic location mapping will be provided by a `MaxMind GeoIP2 database <https://www.maxmind.com/en/geoip2-databases>`_.
1: Neustar
	IP address to geographic location mapping will be provided by a `Neustar GeoPoint IP address database <https://www.security.neustar/digital-performance/ip-intelligence/ip-address-data>`_.

	.. warning:: It's not clear whether Neustar databases are actually supported; this is an old option and compatibility may have been broken over time.

.. table:: Aliases

	+-------------+-------------------------------------------------------------------------------+-----------------------------------------+
	| Name        | Use(s)                                                                        | Type(s)                                 |
	+=============+===============================================================================+=========================================+
	| geoProvider | Traffic Ops and Traffic Ops client code, :ref:`to-api` requests and responses | unchanged (integral, unique identifier) |
	+-------------+-------------------------------------------------------------------------------+-----------------------------------------+

.. _ds-geo-miss-default-latitude:

Geo Miss Default Latitude
-------------------------
Default Latitude for this Delivery Service. When the geographic location of the client cannot be determined, they will be routed as if they were at this latitude.

.. table:: Aliases

	+---------+--------------------------------------------------------+---------------------+
	| Name    | Use(s)                                                 | Type(s)             |
	+=========+========================================================+=====================+
	| missLat | In :ref:`to-api` responses and Traffic Ops source code | unchanged (numeric) |
	+---------+--------------------------------------------------------+---------------------+

.. _ds-geo-miss-default-longitude:

Geo Miss Default Longitude
--------------------------
Default Longitude for this Delivery Service. When the geographic location of the client cannot be determined, they will be routed as if they were at this longitude.

.. table:: Aliases

	+----------+--------------------------------------------------------+---------------------+
	| Name     | Use(s)                                                 | Type(s)             |
	+==========+========================================================+=====================+
	| missLong | In :ref:`to-api` responses and Traffic Ops source code | unchanged (numeric) |
	+----------+--------------------------------------------------------+---------------------+

.. _ds-global-max-mbps:

Global Max Mbps
---------------
The maximum :abbr:`Mbps (Megabits per second)` this Delivery Service can serve across all :term:`Edge-tier cache servers` before traffic will be diverted to the bypass destination. For a DNS-:ref:`Routed <ds-types>` Delivery Service, the `DNS Bypass IP`_ or `DNS Bypass IPv6`_ will be used (depending on whether this was a A or AAAA request), and for HTTP-:ref:`Routed <ds-types>` Delivery Services the `HTTP Bypass FQDN`_ will be used.

.. table:: Aliases

	+--------------------+--------------------------------------------------------------------------------------+------------------------------------------------------------------------------------------------------------------+
	| Name               | Use(s)                                                                               | Type(s)                                                                                                          |
	+====================+======================================================================================+==================================================================================================================+
	| totalKbpsThreshold | In :ref:`to-api` responses - most notably :ref:`to-api-cdns-name-configs-monitoring` | unchanged (numeric), but converted from :abbr:`Mbps (Megabits per second)` to :abbr:`Kbps (kilobits per second)` |
	+--------------------+--------------------------------------------------------------------------------------+------------------------------------------------------------------------------------------------------------------+

.. _ds-global-max-tps:

Global Max TPS
--------------
The maximum :abbr:`TPS (Transactions per Second)` this Delivery Service can serve across all :term:`Edge-tier cache servers` before traffic will be diverted to the bypass destination. For a DNS-:ref:`Routed <ds-types>` Delivery Service, the `DNS Bypass IP`_ or `DNS Bypass IPv6`_ will be used (depending on whether this was a A or AAAA request), and for HTTP-:ref:`Routed <ds-types>` Delivery Services the `HTTP Bypass FQDN`_ will be used.

.. table:: Aliases

	+-------------------+--------------------------------------------------------------------------------------+---------------------+
	| Name              | Use(s)                                                                               | Type(s)             |
	+===================+======================================================================================+=====================+
	| totalTpsThreshold | In :ref:`to-api` responses - most notably :ref:`to-api-cdns-name-configs-monitoring` | unchanged (numeric) |
	+-------------------+--------------------------------------------------------------------------------------+---------------------+

.. _ds-http-bypass-fqdn:

HTTP Bypass FQDN
----------------
When the limits placed on this Delivery Service by the `Global Max Mbps`_ and/or `Global Max Tps`_ are exceeded, an HTTP-:ref:`Routed <ds-types>` Delivery Service will direct excess traffic to this :abbr:`Fully Qualified Domain Name`.

.. _ds-ipv6-routing:

IPv6 Routing Enabled
--------------------
A boolean value that controls whether or not clients using IPv6 can be routed to this Delivery Service by Traffic Router. When creating a Delivery Service in Traffic Portal, this will default to "true".

.. _ds-info-url:

Info URL
--------
This should be a URL (though neither the :ref:`to-api` nor the Traffic Ops Database in any way enforce the validity of said URL) to which administrators or others may refer for further information regarding a Delivery Service - e.g. a related JIRA ticket.

.. _ds-initial-dispersion:

Initial Dispersion
------------------
The number of :term:`Edge-tier cache servers` across which a particular asset will be distributed within each :term:`Cache Group`. For most use-cases, this should be 1, meaning that all clients requesting a particular asset will be directed to 1 :term:`cache server` per :term:`Cache Group`. Depending on the popularity and size of assets, consider increasing this number in order to spread the request load across more than 1 :term:`cache server`. The larger this number, the more copies of a particular asset are stored in a :term:`Cache Group`, which can "pollute" caches (if load distribution is unnecessary) and decreases caching efficiency (due to cache misses if the asset is not requested enough to stay "fresh" in all the caches).

.. _ds-inner-header-rw-rules:

Inner Header Rewrite Rules
--------------------------
This field in general contains the contents of the a configuration file used by the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ when serving content for this Delivery Service - on :term:`Inner-tier cache servers`.

.. tip:: Because this ultimately is the contents of an :abbr:`ATS (Apache Traffic Server)` configuration file, it can make use of the :ref:`ort-special-strings`.

.. note:: This field can only be used if the Delivery Service is assigned to a :term:`Topology`.

.. _ds-last-header-rw-rules:

Last Header Rewrite Rules
-------------------------
This field in general contains the contents of the a configuration file used by the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ when serving content for this Delivery Service - on :term:`Last-tier cache servers`.

.. tip:: Because this ultimately is the contents of an :abbr:`ATS (Apache Traffic Server)` configuration file, it can make use of the :ref:`ort-special-strings`.

.. note:: This field can only be used if the Delivery Service is assigned to a :term:`Topology`.

.. _ds-logs-enabled:

Logs Enabled
------------
A boolean switch that can be toggled to enable/disable logging for a Delivery Service.

.. note:: This doesn't actually do anything. It was part of the functionality for a planned Traffic Control component named "Traffic Logs" - which was never created.

.. _ds-longdesc:

Long Description
----------------
Free text field that has no strictly defined purpose, but it is suggested that it contain a short description of the Delivery Service and its purpose.

.. table::

	+----------+---------------------------------------------------------+-----------------------------------------+
	| Name     | Use(s)                                                  | Type(s)                                 |
	+==========+=========================================================+=========================================+
	| longDesc | Traffic Control source code and :ref:`to-api` responses | unchanged (``string``, ``String`` etc.) |
	+----------+---------------------------------------------------------+-----------------------------------------+

.. _ds-longdesc2:

Long Description 2
------------------
Free text field that has no strictly defined purpose.

.. table::

	+----------------------------+---------------------------------------------------------+-----------------------------------------+
	| Name                       | Use(s)                                                  | Type(s)                                 |
	+============================+=========================================================+=========================================+
	| longDesc1\ [#cardinality]_ | Traffic Control source code and :ref:`to-api` responses | unchanged (``string``, ``String`` etc.) |
	+----------------------------+---------------------------------------------------------+-----------------------------------------+

.. _ds-longdesc3:

Long Description 3
------------------
Free text field that has no strictly defined purpose.

.. table::

	+----------------------------+---------------------------------------------------------+-----------------------------------------+
	| Name                       | Use(s)                                                  | Type(s)                                 |
	+============================+=========================================================+=========================================+
	| longDesc2\ [#cardinality]_ | Traffic Control source code and :ref:`to-api` responses | unchanged (``string``, ``String`` etc.) |
	+----------------------------+---------------------------------------------------------+-----------------------------------------+

.. _ds-matchlist:

Match List
----------
A Match List is a set of regular expressions used by Traffic Router to determine whether a given request from a client should be served by this Delivery Service. Under normal circumstances this field should only ever be read-only as its contents should be generated by Traffic Ops based on the Delivery Service's configuration. These regular expressions can each be one of the following types:

HEADER_REGEXP
	This Delivery Service will be used if an HTTP Header/Value pair can be found in the clients request matching this regular expression.\ [#httpOnlyRegex]_
HOST_REGEXP
	This Delivery Service will be used if the requested host matches this regular expression. The host can be found using the ``Host`` HTTP Header, or as the requested name in a DNS request, depending on the `Type`_ of the Delivery Service.
PATH_REGEXP
	This Delivery Service will be used if the request path matches this regular expression.\ [#httpOnlyRegex]_

.. _ds-steering-regexp:

STEERING_REGEXP
	This Delivery Service will be used if this regular expression matches the xml_id_ of one of this Delivery Service's "targets"

		.. note:: This regular expression type can only exist in the Match List of STEERING-`Type`_ Delivery Services - and **not** CLIENT_STEERING.

.. table:: Aliases

	+-----------------------+----------------------+------------------------------------------------------+
	| Name                  | Use(s)               | Type(s)                                              |
	+=======================+======================+======================================================+
	| deliveryservice_regex | Traffic Ops database | unique, integral identifier for a regular expression |
	+-----------------------+----------------------+------------------------------------------------------+

.. _ds-max-dns-answers:

Max DNS Answers
---------------

DNS-routed Delivery Service
  The maximum number of :term:`Edge-tier cache server` IP addresses that the Traffic Router will include in responses to DNS requests. When provided, the :term:`cache server` IP addresses included are rotated in each response to spread traffic evenly. This number should scale according to the amount of traffic the Delivery Service is expected to serve.

HTTP-routed Delivery Service
  If the Traffic Router profile parameter "edge.http.limit" is set, setting this to a non-zero value will override that parameter for this delivery service, limiting the number of Traffic Router IP addresses (A records) that are included in responses to DNS requests for this delivery service.

.. _ds-max-origin-connections:

Max Origin Connections
----------------------
The maximum number of TCP connections individual :term:`Mid-tier cache servers` are allowed to make to the `Origin Server Base URL`. A value of ``0`` in this field indicates that there is no maximum.

.. _ds-max-request-header-bytes:

Max Request Header Bytes
------------------------
The maximum size(in bytes) of the request header that is allowed for this Delivery Service.

.. _ds-mid-header-rw-rules:

Mid Header Rewrite Rules
------------------------
This field in general contains the contents of the a configuration file used by the `ATS Header Rewrite Plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html>`_ when serving content for this Delivery Service - on :term:`Mid-tier cache servers`.

.. tip:: Because this ultimately is the contents of an :abbr:`ATS (Apache Traffic Server)` configuration file, it can make use of the :ref:`ort-special-strings`.

.. note:: This field cannot be used if the Delivery Service is assigned to a :term:`Topology`.

.. _ds-origin-url:

Origin Server Base URL
----------------------
The Origin Server’s base URL which includes the protocol (http or https). Example: ``http://movies.origin.com``. Must not include paths, query parameters, document fragment identifiers, or username/password URL fields.

.. table:: Aliases

	+---------------+------------------------------------------------------------+----------------------------------------------+
	| Name          | Use(s)                                                     | Type(s)                                      |
	+===============+============================================================+==============================================+
	| orgServerFqdn | :ref:`to-api` responses and in Traffic Control source code | unchanged (usually ``str``, ``string`` etc.) |
	+---------------+------------------------------------------------------------+----------------------------------------------+

.. _ds-origin-shield:

Origin Shield
-------------
An experimental feature that allows administrators to list additional forward proxies that sit between the :term:`Mid-tier` and the :term:`Origin`. In most scenarios, this is represented (and required to be input) as a pipe (``|``)-delimited string.

.. _ds-profile:

Profile
-------
Either the :ref:`profile-name` of a :term:`Profile` used by this Delivery Service, or the :ref:`profile-id` of said :term:`Profile`.

.. table:: Aliases

	+-------------+------------------------------------------------------------------------------------------------+----------------------------------------------------------------------------------------+
	| Name        | Use(s)                                                                                         | Type(s)                                                                                |
	+=============+================================================================================================+========================================================================================+
	| profileId   | In Traffic Control source code and some :ref:`to-api` responses dealing with Delivery Services | Unlike the more general "Profile", this is *always* an integral, unique identifier     |
	+-------------+------------------------------------------------------------------------------------------------+----------------------------------------------------------------------------------------+
	| profileName | In Traffic Control source code and some :ref:`to-api` responses dealing with Delivery Services | Unlike the more general "Profile", this is *always* a name (``str``, ``string``, etc.) |
	+-------------+------------------------------------------------------------------------------------------------+----------------------------------------------------------------------------------------+

.. _ds-protocol:

Protocol
--------
The protocol with which to serve content from this Delivery Service. This defines the way the Delivery Service will handle client requests that are either HTTP or HTTPS, which is distinct from what protocols are used to direct traffic. For example, this can be used to direct clients to only request content using HTTP, or to allow clients to use either HTTP or HTTPS, etc. Normally, this will be the name of the protocol handling, but occasionally this will appear as the integral, unique identifier of the protocol handling instead. The integral, unique identifiers and their associated names and meanings are:

0: HTTP
	This Delivery Service will only accept unsecured HTTP requests. Requests made with HTTPS will fail.
1: HTTPS
	This Delivery Service will only accept secured HTTPS requests. Requests made with HTTP will fail.
2: HTTP AND HTTPS
	This Delivery Service will accept both unsecured HTTP requests and secured HTTPS requests.
3: HTTP TO HTTPS
	When this Delivery Service is using HTTP :ref:`Content Routing <ds-types>` unsecured HTTP requests will be met with a response that indicates to the client that further requests must use HTTPS.

	.. note:: If any other type of :ref:`Content Routing <ds-types>` is used, this functionality cannot be used. In those cases, a protocol setting of ``3``/"HTTP TO HTTPS" will result in the same behavior as ``1``/"HTTPS". This behavior is tracked by `GitHub Issue #3221 <https://github.com/apache/trafficcontrol/issues/3221>`_


.. warning:: The definitions of each integral, unique identifier are hidden in implementations in each :abbr:`ATC (Apache Traffic Control)` component. Different components will handle invalid values differently, and there's no actual enforcement that the stored integral, unique identifier actually be within the representable range.

.. table:: Aliases

	+----------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------------------------------------+
	| Name     | Use(s)                  | Type(s)                                                                                                                                                             |
	+==========+=========================+=====================================================================================================================================================================+
	| Protocol | CDN :term:`Snapshots` | An object containing the key ``"acceptHttps"`` that is a string containing a boolean that expresses whether Traffic Router should accept HTTPS requests for this      |
	|          |                         | Delivery Service, and the key ``"redirectToHttps"`` that is also a string containing a boolean which expresses whether or not Traffic Router should redirect HTTP   |
	|          |                         | requests to HTTPS URLs. Optionally, the key ``"acceptHttp"`` may also appear, once again a string containing a boolean that expresses whether or not Traffic Router |
	|          |                         | should accept unsecured HTTP requests - this is implicitly treated as ``"true"`` by Traffic Router when it is not present.                                          |
	+----------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------------------------------------+

.. _ds-qstring-handling:

Query String Handling
---------------------
Describes how query strings should be handled by the :term:`Edge-tier cache servers` when serving content for this Delivery Service. This is nearly always expressed as an integral, unique identifier for each behavior, though in Traffic Portal a more descriptive value is typically used, or at least provided in addition to the integral, unique identifier. The allowed values and their meanings are:

0
	For the purposes of caching, :term:`Edge-tier cache servers` will consider URLs unique if and only if they are unique up to and including any and all query parameters. They will also pass the query parameters in their own requests to :term:`Mid-tier cache servers` (which in turn will exhibit the same caching behavior and pass the query parameters in requests to the :term:`Origin`). (Aliased as "USE" in Traffic Portal tables, and "0 - use qstring in cache key, and pass up" in Traffic Portal forms)
1
	For the purposes of caching, neither :term:`Edge-tier` nor :term:`Mid-tier cache servers` will consider the query parameter string when determining if a URL is stored in cache. However, the query string will still be passed in upstream requests to :term:`Mid-tier cache servers` and in turn the :term:`Origin`. (Aliased as "IGNORE" in Traffic Portal tables and "1 - ignore in cache key, and pass up" in Traffic Portal forms)
2
	The query parameter string will be stripped from URLs immediately when the request is received by an :term:`Edge-tier cache server`. This means it is never considered for the purposes of caching unique URLs and will not be passed in upstream requests. (Aliased as "DROP" in Traffic Portal tables and "2 - drop at edge" in Traffic Portal forms)

	.. warning:: The implementation of dropping query parameter strings at the :term:`Edge-tier` uses a `Regex Remap Expression`_ and thus Delivery Services with this type of query string handling cannot make use of `Regex Remap Expression`_\ s.

.. table:: Aliases

	+------------------+------------------------------------------------------------+-----------------------------------------------------------------------------------------+
	| Name             | Use(s)                                                     | Type(s)                                                                                 |
	+==================+============================================================+=========================================================================================+
	| Qstring Handling | Traffic Portal tables                                      | One of the Traffic Portal value aliases "USE" (``0``), "IGNORE" (``1``), "DROP" (``2``) |
	+------------------+------------------------------------------------------------+-----------------------------------------------------------------------------------------+
	| qstringIgnore    | Traffic Ops code, :ref:`to-api` requests/responses         | unchanged (integral, unique identifier)                                                 |
	+------------------+------------------------------------------------------------+-----------------------------------------------------------------------------------------+

The Delivery Service's Query String Handling can be set directly as a field on the Delivery Service object itself, or it can be overridden by a :term:`Parameter` on a Profile_ used by this Delivery Service. The special :term:`Parameter` named ``psel.qstring_handling`` and configuration file ``parent.config`` will have it's contents directly inserted into the ``parent.config`` file on all :term:`cache servers` assigned to this Delivery Service.

.. danger:: Using the ``psel.qstring_handling`` :term:`Parameter` is **strongly** discouraged for several reasons. Firstly, at a Delivery Service level it will **NOT** change the configuration of that Delivery Service's own Query String Handling - which will cause it to appear in Traffic Portal and in :ref:`to-api` responses as though it were configured one way while actually behaving a different way altogether. Also, no validation is performed on the value given to it. Because it's inserted verbatim into the ``qstring`` field of a line in :abbr:`ATS (Apache Traffic Server)` `parent.config configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_, a typo or an ignorant user can easily cause :abbr:`ATS (Apache Traffic Server)` instances on all :term:`cache servers` assigned to that Delivery Service to fail to reload their configuration, possibly grinding entire CDNs to a halt.


.. seealso:: When implemented as a :term:`Parameter` (``psel.qstring_handling``), its value must be a valid value for the ``qstring`` field of a line in the :abbr:`ATS (Apache Traffic Server)` ``parent.config`` configuration file. For a description of valid values, see the `documentation for parent.config <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_

.. _ds-range-request-handling:

Range Request Handling
----------------------
Describes how HTTP "Range Requests" should be handled by the Delivery Service at the :term:`Edge-tier`. This is nearly always an integral, unique identifier for the behavior set required of the :term:`Edge-tier cache servers`. The valid values and their respective meanings are:

0
	Do not cache Range Requests at all. (Aliased as "0 - Don't cache" in Traffic Portal forms)

		.. note:: This is not retroactive - when modifying an existing Delivery Services to have this value for "Range Request Handling", ranges requested from files that are already cached due to a non-range request will be served out of cache for as long as the Cache-Control headers allow.

1
	Use the `background_fetch <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/background_fetch.en.html>`_ plugin to service the range request while caching the whole object. (Aliased as "1 - Use background_fetch plugin" in Traffic Portal forms)
2
	Use the `cache_range_requests <https://github.com/apache/trafficserver/tree/7.1.x/plugins/experimental/cache_range_requests>`_ plugin to cache ranges as unique objects. (Aliased as "2 - Use cache_range_requests plugin" in Traffic Portal forms)
3
	Use the `slice <https://github.com/apache/trafficserver/tree/master/plugins/experimental/slice>`_ plugin to slice range based requests into deterministic chunks. (Aliased as "3 - Use slice plugin" in Traffic Portal forms)

		.. versionadded:: ATCv4.1

.. note:: Range Request Handling can only be implemented on :term:`cache servers` using :abbr:`ATS (Apache Traffic Server)` because of its dependence on :abbr:`ATS (Apache Traffic Server)` plugins. The value may be set on any Delivery Service, but will have no effect when the :term:`cache servers` that ultimately end up serving the content are e.g. Grove, Nginx, etc.

.. warning:: The definitions of each integral, unique identifier are hidden in implementations in each :abbr:`ATC (Apache Traffic Control)` component. Different components will handle invalid values differently, and there's no actual enforcement that the stored integral, unique identifier actually be within the representable range.

.. _ds-slice-block-size:

Range Slice Request Block Size
-------------------------------------
The block size in bytes that is used for `slice <https://github.com/apache/trafficserver/tree/master/plugins/experimental/slice>`_ plugin.

This can only and must be set if the :ref:`ds-range-request-handling` is set to ``3``.

.. _ds-raw-remap:

Raw Remap Text
--------------
For HTTP and DNS-:ref:`Routed <ds-types>` Delivery Services, this will be added to the end of a line in the `remap.config ATS configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/remap.config.en.html>`_ line on the cache verbatim. For ANY_MAP-:ref:`Type <ds-types>` Delivery Services this must be defined.

.. tip:: Because this ultimately is a raw line of content in a configuration file, it can make use of the :ref:`ort-special-strings`. Of particular note is the :ref:`ort-remap-override` template string.

.. note:: This field **must** be defined on ANY_MAP-`Type`_ Delivery Services, but is otherwise optional.

.. seealso:: `The Apache Trafficserver documentation for the Regex Remap plugin <https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/regex_remap.en.html>`_

.. table:: Aliases

	+-----------+-----------------------------------------------------------------+---------------------------------------+
	| Name      | Use(s)                                                          | Type(s)                               |
	+===========+=================================================================+=======================================+
	| remapText | In Traffic Ops source code and :ref:`to-api` requests/responses | unchanged (``text``, ``string`` etc.) |
	+-----------+-----------------------------------------------------------------+---------------------------------------+

Directives
"""

The Raw Remap text is ordinarily added at the end of the line, after everything else. However, it may be necessary to add Range Request Handling after the Raw Remap. For example, if you have a plugin which manipulates the Range header. In this case, you can insert the text ``__RANGE_DIRECTIVE__`` in the Raw Remap text, and the range request handling directives will be added at that point.

For example, if you have an Apache Traffic Server lua plugin which manipulates the range, and are using Slice Range Request Handling which needs to run after your plugin, you can set a Raw Remap, ``@plugin=tslua.so @pparam=range.lua __RANGE_DIRECTIVE__``, and the ``@plugin=slice.so`` range directive will be inserted after your plugin.

.. _ds-regex-remap:

Regex Remap Expression
----------------------
Allows remapping of incoming requests URL using regular expressions to search and replace text. In a more literal sense, this is the raw contents of a configuration file used by the `ATS regex_remap plugin  <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/regex_remap.en.html>`_. At its most basic, the contents of this field should consist of ``map`` followed by a regular expression and then a "template URL" - all space-separated. The regular expression matches a client's request *path* (i.e. not a full URL - ``/path/to/content`` **not** ``https://origin.example.com/path/to/content``) and when such a match occurs, the request is transformed into a request for the template URL. The most basic usage of the template URL is to use ``$1``-``$9`` to insert the corresponding regular expression capture group. For example, a regular expression of :regexp:`^/a/(.*)` and a template URL of ``https://origin.example.com/b/$1`` maps requests for :term:`Origin` content under path ``/a/`` to the same sub-paths under path ``b``. Note that since it's a full URL, this mapping can be made to another server entirely.

.. seealso:: The `documentation for the regex_remap plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/regex_remap.en.html>`_ for :abbr:`ATS (Apache Traffic Server)`

.. caution:: This field is not validated by Traffic Ops to be correct syntactically, and can cause Traffic Server to not start if invalid. Please use with caution.

.. warning:: Regex remap expressions are incompatible with `Query String Handling`_ being set to ``2``. The behavior of a :term:`cache server` under that configuration is undefined.

	.. tip:: It is, of course, entirely possible to write a Regex Remap Expression that reproduces the desired `Query String Handling`_ as well as any other desired behavior.

.. seealso:: `The Apache Trafficserver documentation for the Regex Remap plugin <https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/regex_remap.en.html>`_

.. table:: Aliases

	+------------+----------------------------------------------------------------------------+-----------------------------+
	| Name       | Use(s)                                                                     | Type(s)                     |
	+============+============================================================================+=============================+
	| regexRemap | Traffic Ops source code and database, and :ref:`to-api` requests/responses | unchanged (``string`` etc.) |
	+------------+----------------------------------------------------------------------------+-----------------------------+

.. _ds-regionalgeo:

Regional Geoblocking
--------------------
A boolean value that defines whether or not :ref:`Regional Geoblocking <regionalgeo-qht>` is active on this Delivery Service. The actual configuration of :ref:`Regional Geoblocking <regionalgeo-qht>` is done in the :term:`Profile` used by the Traffic Router serving the Delivery Service. Rules for this Delivery Service may exist, but they will not actually be used unless this field is ``true``.

.. tip:: :ref:`Regional Geoblocking <regionalgeo-qht>` is configured primarily with respect to Canadian postal codes, so unless specifically Canadian regions should be allowed/disallowed to access content, `Geo Limit`_ is probably a better setting for controlling access to content according to geographic location.

.. _ds-required-capabilities:

Required Capabilities
---------------------
.. versionadded:: ATCv4

A Delivery Service can be associated with :term:`Server Capabilities` that it requires :term:`cache servers` serving its content to have. When one or more :term:`Server Capability` is required by a Delivery Service, it will block the assignment of :term:`cache servers` to it that do not have those :term:`Server Capabilities`. Additionally, the :term:`Edge-tier cache servers` assigned to a Delivery Service that requires a :term:`Server Capability` will only request content they do not have cached from :term:`Mid-tier cache servers` which also have this :term:`Server Capability`.

Typically, a required :term:`Server Capability` is represented merely by the name of said :term:`Server Capability`. In fact, there's nothing more to a :term:`Server Capability` than its name; it's the responsibility of CDN operators to ensure that they are assigned and required properly. There is no mechanism to detect whether or not a :term:`cache server` has a given :term:`Server Capability`, it must be assigned manually.

.. _ds-routing-name:

Routing Name
------------
A DNS label in the Delivery Service's domain that forms the :abbr:`FQDN (Fully Qualified Domain Name)` that is used by clients to request content. All together, the constructed :abbr:`FQDN (Fully Qualified Domain Name)` looks like: :file:`{Delivery Service Routing Name}.{Delivery Service xml_id}.{CDN Subdomain}.{CDN Domain}.{Top-Level Domain}`\ [#xmlValid]_.

.. _ds-servers:

Servers
-------
Servers can be assigned to Delivery Services using the :ref:`tp-configure-servers` and :ref:`tp-services-delivery-service` Traffic Portal sections, or by directly using the :ref:`to-api-deliveryserviceserver` endpoint. Only :term:`Edge-tier cache servers` can be assigned to a Delivery Service, and once they are so assigned they will begin to serve content for the Delivery Service (after updates are queued and then applied). Any servers assigned to a Delivery Service must also belong to the same CDN_ as the Delivery Service itself. At least one server must be assigned to a Delivery Service in order for it to serve any content.

.. _ds-service-category:

Service Category
----------------
A service category is a tag that describes the type of content being delivered by the Delivery Service. Some example values are: "Linear" and "VOD"

.. _ds-signing-algorithm:

Signing Algorithm
-----------------
URLs/URIs may be signed using one of two algorithms before a request for the content to which they refer is sent to the :term:`Origin` (which in practice can be any upstream network). At the time of this writing, this field is restricted within the Traffic Ops Database to one of two values (or ``NULL``/"None", to indicate no signing should be done).

.. seealso:: The url_sig `README <https://github.com/apache/trafficserver/blob/master/plugins/experimental/url_sig/README>`_.

.. seealso:: `The draft RFC for uri_signing <https://tools.ietf.org/html/draft-ietf-cdni-uri-signing-16>`_ - note, however that the current implementation of uri_signing uses Draft 12 of that RFC document, **NOT** the latest.

url_sig
	URL signing will be implemented in this Delivery Service using the `url_sig Apache Traffic Server plugin <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/url_sig.en.html>`_. (Aliased as "URL Signature Keys" in Traffic Portal forms)
uri_signing
	URL signing will be implemented in this Delivery Service using an algorithm based on a work-in-progress RFC specification draft. (Aliased as "URI Signing Keys" in Traffic Portal forms)

.. table:: Aliases

	+--------+------------------------------------------------------------------------------------------+---------------------------------------------------------------------------------------------+
	| Name   | Use(s)                                                                                   | Type(s)                                                                                     |
	+========+==========================================================================================+=============================================================================================+
	| Signed | In all components prior to Traffic Control v2.2. Some endpoints in early versions of the | A boolean value where ``true`` was the same as "url_sig" in current versions, and ``false`` |
	|        | :ref:`to-api` will still return this field instead of "signingAlgorithm".                | indicated URL signing would not be done for the Delivery Service.                           |
	+--------+------------------------------------------------------------------------------------------+---------------------------------------------------------------------------------------------+

Keys for either algorithm can be generated within :ref:`Traffic Portal <tp-services-delivery-service>`.

.. _ds-ssl-key-version:

SSL Key Version
---------------
An integer that describes the version of the SSL key(s) - if any - used by this Delivery Service. This is incremented whenever Traffic Portal generates new SSL keys for the Delivery Service.

.. warning:: This number will not be correct if keys are manually replaced using the API, as the key generation API does not increment it!

.. _ds-static-dns-entries:

Static DNS Entries
------------------
Static DNS Entries can be added *under* a Delivery Service's domain. These DNS records can be configured in the :ref:`tp-services-delivery-service` section of Traffic Portal, and can be any valid CNAME, A or AAAA DNS record - provided the associated hostname falls within the DNS domain for the Delivery Service. For example, a Delivery Service with xml_id_ "demo1" and belonging to a CDN_ with domain "mycdn.ciab.test" could have Static DNS Entries for hostnames "foo.demo1.mycdn.ciab.test" or "foo.bar.demo1.mycdn.ciab.test" but not "foo.bar.mycdn.ciab.test" or "foo.bar.test".

.. note:: The `Routing Name`_ of a Delivery Service is not part of the :abbr:`SOA (Start of Authority)` record for the Delivery Service's domain, and so there is no need to place Static DNS Entries below a domain containing it.

.. _ds-tenant:

Tenant
------
The :term:`Tenant` who owns this Delivery Service. They (and their parents, if any) are the only ones allowed to make changes to this Delivery Service. Typically, ``tenant``/``Tenant`` refers to the *name* of the owning :term:`Tenant`, but occasionally (most notably in the payloads and/or query parameters of certain :ref:`to-api` requests) it actually refers to the *integral, unique identifier* of said :term:`Tenant`.

.. table:: Aliases

	+----------+----------------------------------------------+--------------------------------------------------------+
	| Name     | Use(s)                                       | Type(s)                                                |
	+==========+==============================================+========================================================+
	| TenantID | Go code and :ref:`to-api` requests/responses | Integral, unique identifier (``bigint``, ``int`` etc.) |
	+----------+----------------------------------------------+--------------------------------------------------------+

.. _ds-topology:

Topology
--------
A structure composed of :term:`Cache Groups` and parent relationships, which is assignable to one or more :term:`Delivery Services`.

.. _ds-tr-resp-headers:

Traffic Router Additional Response Headers
------------------------------------------
List of HTTP header ``{{name}}:{{value}}`` pairs separated by ``__RETURN__`` or simply on separate lines. Listed pairs will be included in all HTTP responses from Traffic Router for HTTP-:ref:`Routed <ds-types>` Delivery Services.

.. deprecated:: 4.0
	The use of ``__RETURN__`` as a substitute for a real newline is unnecessary and the ability to do so will be removed in the future.

.. table:: Aliases

	+-------------------+----------------------------------------------------------------------------------------+-----------------------------+
	| Name              | Use(s)                                                                                 | Type(s)                     |
	+===================+========================================================================================+=============================+
	| trResponseHeaders | Traffic Control source code and Delivery Service objects returned by the :ref:`to-api` | unchanged (``string`` etc.) |
	+-------------------+----------------------------------------------------------------------------------------+-----------------------------+

.. _ds-tr-req-headers:

Traffic Router Log Request Headers
----------------------------------
List of HTTP header names separated by ``__RETURN__`` or simply on separate lines. Listed pairs will be logged for all HTTP requests to Traffic Router for HTTP-:ref:`Routed <ds-types>` Delivery Services.

.. deprecated:: 4.0
	The use of ``__RETURN__`` as a substitute for a real newline is unnecessary and the ability to do so will be removed in the future.

.. table:: Aliases

	+------------------+----------------------------------------------------------------------------------------+-----------------------------+
	| Name             | Use(s)                                                                                 | Type(s)                     |
	+==================+========================================================================================+=============================+
	| trRequestHeaders | Traffic Control source code and Delivery Service objects returned by the :ref:`to-api` | unchanged (``string`` etc.) |
	+------------------+----------------------------------------------------------------------------------------+-----------------------------+

.. _ds-types:

Type
----
Defines the content routing method used by the Delivery Service. In most cases this is an integral, unique identifier that corresponds to an enumeration of the Delivery Service Types. In other cases, this the actual name of said type.

The "Type" of a Delivery Service can mean several things. First, it can be used to refer to the "routing type" of Delivery Service. This is one of:

.. tip:: The only way to get the integral, unique identifier of a :term:`Type` of Delivery Service is to look at the database after it has been generated; these are non-deterministic and cannot be guaranteed to have any particular value, or even consistent values. This can be done directly or, preferably, using the :ref:`to-api-types` endpoint. Unfortunately, knowing the name of the :term:`Type` is rarely enough for many applications. The ``useInColumn`` values of these :term:`Types` will be ``deliveryservice``.

DNS
	Delivery Services of this routing type are routed by Traffic Router by providing DNS records that provide the IP addresses of :term:`cache servers` when clients look up the full Delivery Service :abbr:`FQDN (Fully Qualified Domain Name)`.
HTTP
	The Traffic Router(s) responsible for routing this Delivery Service will still answer DNS requests for the Delivery Service :abbr:`FQDN (Fully Qualified Domain Name)`, but will provide its own IP address. The client then directs its HTTP request to the Traffic Router, which will use an `HTTP redirection response <https://developer.mozilla.org/en-US/docs/Web/HTTP/Status#Redirection_messages>`_ to direct the client to a :term:`cache server`.

More generally, though, Delivery Services have a Type that defines not only how traffic is routed, but also how content is cached and semantically defines what "content" means in the context of a given Delivery Service.

ANY_MAP
	This is a special kind of Delivery Service that should only be used when control over the clients is guaranteed, and very fine control over the :abbr:`ATS (Apache Traffic Server)` `remap.config  <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/remap.config.en.html>`_ line for this Delivery Service is required. ANY_MAP is not known to Traffic Router. It is not routed in any way. For Delivery Services of this type, the "Raw Remap Text" field **must** be defined, as it is the only configuration generated by Traffic Control. The only way for a client to utilize delivery through an ANY_MAP service is by knowing in advance the IP address of one or more :term:`Edge-tier cache servers` and make the appropriate request(s).
DNS
	Uses DNS content routing. Delivers content normally. This is the recommended Type for delivering smaller objects like web page images.
DNS_LIVE\ [#dupOrigin]_
	Uses DNS Content routing, but optimizes caching for live video streaming. Specifically, the configuration generated for :term:`cache servers` responsible for serving content for this Delivery Service will not cache that content on storage disks. Instead, they will make use of RAM block devices dedicated to ATS - as specified by the special ``RAM_Drive_Prefix`` and ``RAM_Drive_Letters`` :term:`Parameters`. Also, any :term:`Mid-tier` of caching is bypassed.
DNS_LIVE_NATNL
	Works exactly the same as DNS_LIVE, but is optimized for delivery of live video content across a wide physical area. What this means is that the :term:`Mid-tier` of caching is **not** bypassed, unlike DNS_LIVE. The :term:`Mid-tier` will also use block RAM devices.
HTTP
	Uses HTTP content routing, delivers content normally. This is the recommended Type for delivering larger objects like video streams.
HTTP_LIVE\ [#dupOrigin]_
	Uses HTTP Content routing, but optimizes caching for live video streaming. Specifically, the configuration generated for :term:`cache servers` responsible for serving content for this Delivery Service will not cache that content on storage disks. Instead, they will make use of RAM block devices dedicated to ATS - as specified by the special ``RAM_Drive_Prefix`` and ``RAM_Drive_Letters`` :term:`Parameters`. Also, any :term:`Mid-tier` of caching is bypassed.
HTTP_LIVE_NATNL
	Works exactly the same as HTTP_LIVE, but is optimized for delivery of live video content across a wide physical area. What this means is that the :term:`Mid-tier` of caching is **not** bypassed, unlike HTTP_LIVE. The :term:`Mid-tier` will also use block RAM devices.
HTTP_NO_CACHE\ [#dupOrigin]_
	Uses HTTP Content Routing, but :term:`cache servers` will not actually cache the delivered content - they act as just proxies. This will bypass any existing :term:`Mid-tier` entirely (as it's totally useless when content is not being cached).

.. _ds-steering:

STEERING
	This is a sort of "meta" Delivery Service. It is used for directing clients to one of a set of Delivery Services, rather than delivering content directly itself. The Delivery Services to which a STEERING Delivery Service routes clients are referred to as "targets". Targets in general have an associated "value" and can be of several :term:`Types` that define the meaning of the value - these being:

.. _ds-steering-order:

	STEERING_ORDER
		The value of a STEERING_ORDER target sets a strict order of preference. In cases where a response to a client contains multiple Delivery Services, those targets with a lower "value" appear earlier than those with a higher "value". In cases where two or more targets share the same value, they each have an equal chance of being presented to the client - effectively spreading traffic evenly across them.

.. _ds-steering-weight:

	STEERING_WEIGHT
		The values of STEERING_WEIGHT targets are interpreted as "weights", which define how likely it is that any given client will be routed to a specific Delivery Service - effectively this determines the spread of traffic across each target.

	The targets of a Delivery Service may be set using :ref:`the appropriate section of Traffic Portal <tp-services-delivery-service>` or via the :ref:`to-api-steering-id-targets` and :ref:`to-api-steering-id-targets-targetID` :ref:`to-api` endpoints.

	.. seealso:: For more information on setting up a STEERING (or CLIENT_STEERING) Delivery Service, see :ref:`steering-qht`.

	.. seealso:: For implementation details about how Traffic Router routes STEERING (and CLIENT_STEERING) Delivery Services, see :ref:`tr-steering`.

.. _ds-client-steering:

CLIENT_STEERING
	A CLIENT_STEERING Delivery Service is exactly like STEERING except that it provides clients with methods of bypassing the weights, orders, and localizations of targets in order to choose any arbitrary target at will. When utilizing these methods, the client will either directly choose a target immediately or request a list of all available targets from Traffic Router and then choose one to which to send a subsequent request for actual content. CLIENT_STEERING also supports two additional target types:

	STEERING_GEO_ORDER
		These targets behave exactly like STEERING_ORDER targets, but Delivery Services are grouped according to the "locations" of their :term:`Origins`. Before choosing a Delivery Service to which to direct the client, Traffic Router will first create subsets of choices according to these groupings, and order them by physical distance from the client (closest to farthest). Within these subsets, the values of the targets establish a strict precedence ordering, just like STEERING_ORDER targets.
	STEERING_GEO_WEIGHT
		These targets behave exactly like STEERING_WEIGHT targets, but Delivery Services are grouped according to the "locations" of their :term:`Origins`. Before choosing a Delivery Service to which to direct the client, Traffic Router will first create subsets of choices according to these groupings, and order them by physical distance from the client (closest to farthest). Within these subsets, the values of the targets establish the likelihood that any given target within the subset will be chosen for the client - effectively determining the spread of traffic across targets within that subset.

	.. important:: To make use of the STEERING_GEO_ORDER and/or STEERING_GEO_WEIGHT target types, it is first necessary to ensure that at least the "primary" :term:`Origin` of the :term:`Delivery Service` has an associated geographic coordinate pair. This can be done either from the :ref:`tp-configure-origins` page in Traffic Portal, or using the :ref:`to-api-origins` :ref:`to-api` endpoint.

.. note:: "Steering" is also commonly used to collectively refer to either of the kinds of Delivery Services that can participate in steering behavior (STEERING and CLIENT_STEERING).

.. table:: Aliases

	+----------------------+-------------------------------------------------+-----------------------------------------------------------------+
	| Name                 | Use(s)                                          | Type(s)                                                         |
	+======================+=================================================+=================================================================+
	| Content Routing Type | Traffic Portal forms                            | The name of any of the Delivery Service `Type`_\ s (``string``) |
	+----------------------+-------------------------------------------------+-----------------------------------------------------------------+
	| TypeID               | In Go code and :ref:`to-api` requests/responses | Integral, unique identifier (``bigint``, ``int`` etc.)          |
	+----------------------+-------------------------------------------------+-----------------------------------------------------------------+

.. _ds-multi-site-origin:

Use Multi-Site Origin Feature
-----------------------------
A boolean value that indicates whether or not this Delivery Service serves content for an :term:`Origin` that provides content from two or more redundant servers. There are very few good reasons for this to not be ``false``. When ``true``, Traffic Ops will configure :term:`Mid-tier cache servers` to perform load-balancing and other optimizations for redundant :term:`origin servers`.

Naturally, this assumes that each redundant server is exactly identical, from request paths to actual content. If Multi-Site Origin is configured for servers that are *not* identical, the client's experience is undefined. Furthermore, the :term:`origin servers` may have differing IP addresses, but **must** serve content for a single :abbr:`FQDN (Fully Qualified Domain Name)` - as defined by the Delivery Service's `Origin Server Base URL`_. These redundant servers **must** be configured as servers (server :term:`Type` ``ORG``) in Traffic Ops - either using the :ref:`appropriate section of Traffic Portal <tp-configure-servers>` or the :ref:`to-api-servers` endpoint.

.. important:: In order for a given :term:`Mid-tier cache server` to support Multi-Site Origins, the value of a :term:`Parameter` named ``http.parent_proxy_routing_enable`` in configuration file ``records.config`` must be set to ``1`` on that server's :term:`Profile`. If using an optional secondary grouping of Multi-Site Origins, the :term:`Parameter` named ``url_remap.remap_required`` in configuration file ``records.config`` must also be set to ``1`` on that :term:`Profile`. These settings must be applied to all :term:`Mid-tier cache servers`' that are the :term:`parents` of any :term:`Edge-tier cache server` assigned to this Delivery Service.

	.. seealso:: These parameters are described in the :abbr:`ATS (Apache Traffic Server)` documentation sections for `Parent Proxy Configuration <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/records.config.en.html#proxy-config-http-parent-proxy-routing-enable>`_ and `URL Remap Rules <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/records.config.en.html#proxy-config-url-remap-remap-required>`_, respectively.

.. table:: Aliases

	+---------------------------------+-----------------------------------------------------------------------------+---------------------------------------------------------+
	| Name                            | Use(s)                                                                      | Type(s)                                                 |
	+=================================+=============================================================================+=========================================================+
	| multiSiteOrigin                 | In Go code and :ref:`to-api` requests/responses                             | unchanged (``bool``, ``boolean`` etc.)                  |
	+---------------------------------+-----------------------------------------------------------------------------+---------------------------------------------------------+
	| :abbr:`MSO (Multi-Site Origin)` | In documentation and used heavily in discussion in Slack, mailing list etc. | unchanged (usually only used where implicitly ``true``) |
	+---------------------------------+-----------------------------------------------------------------------------+---------------------------------------------------------+

A Delivery Service Profile_ can have :term:`Parameters` that affect Multi-Site Origin configuration. These are detailed in the :ref:`ds-mso-parameters` table. All of these :term:`Parameters` should have their Configuration File set to ``parent.config``. Each :term:`Parameter` directly corresponds to a field in a line of the :abbr:`ATS (Apache Traffic Server)` `parent.config file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>` (usually by almost the same name), and documentation for these fields is provided in the form of links to their entries in the :abbr:`ATS (Apache Traffic Server)` documentation.

.. _round_robin: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html#parent-config-format-round-robin
.. _max_simple_retries: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html#parent-config-format-max-simple-retries
.. _max_unavailable_server_retries: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html#parent-config-format-max-unavailable-server-retries
.. _parent_retry: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html#parent-config-format-parent-retry
.. _unavailable_server_retry_responses: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html#parent-config-format-unavailable-server-retry-responses
.. _parent.config: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html

.. _ds-mso-parameters:

.. table:: :term:`Parameters` of a Delivery Service Profile_ that Affect :abbr:`MSO (Multi-Site-Origin)` Configuration

	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| Name                                        | :abbr:`ATS (Apache Traffic Server)` `parent.config`_ field                 | Effect                                                                              |
	+=============================================+============================================================================+=====================================================================================+
	| mso.algorithm                               | `round_robin`_                                                             | Sets the algorithm used to determine from which :term:`origin server` content will  |
	|                                             |                                                                            | be requested.                                                                       |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| mso.max_simple_retries                      | `max_simple_retries`_                                                      | Sets a strict limit on the number of "simple retries" allowed before giving up      |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| mso.max_unavailable_server_retries          | `max_unavailable_server_retries`_                                          | Sets a strict limit on the number of times the :term:`cache server` will attempt to |
	|                                             |                                                                            | request content from an :term:`origin server` that has previously been considered   |
	|                                             |                                                                            | "unavailable".                                                                      |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| mso.parent_retry                            | `parent_retry`_                                                            | Sets whether the :term:`cache servers` will use "simple retries",                   |
	|                                             |                                                                            | "unavailable server retries", or both.                                              |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| mso.simple_retry_response_codes             | **UNKNOWN**                                                                | **UNKNOWN** - supposedly defines HTTP response codes from an :term:`origin server`  |
	|                                             |                                                                            | that necessitate a "simple retry".                                                  |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+
	| mso.unavailable_server_retry_response_codes | `unavailable_server_retry_responses`_                                      | Defines HTTP response codes from an :term:`origin server` that indicate it is       |
	|                                             |                                                                            | currently "unavailable".                                                            |
	+---------------------------------------------+----------------------------------------------------------------------------+-------------------------------------------------------------------------------------+

.. warning:: The ``mso.simple_retry_response_codes`` :term:`Parameter` has no apparent, possible use according to the :abbr:`ATS (Apache Traffic Server)` `parent.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_. Whether or not it has any effect - let alone the *intended* effect - is not known, and its use is therefore strongly discouraged.

.. seealso:: A quick guide on setting up Multi-Site Origins is given in :ref:`multi-site-origin-qht`.

.. seealso:: See the `Apache Traffic Server documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_ for more information on its implementation of Multi-Site Origins.

.. _ds-xmlid:

xml_id
------
A text-based unique identifier for a Delivery Service. Many :ref:`to-api` endpoints and internal :abbr:`ATC (Apache Traffic Control)` functions use this to uniquely identify a Delivery Service as opposed to the historically favored "ID". This string will become a part of the CDN service domain, which all together looks like: :file:`{Delivery Service Routing Name}.{Delivery Service xml_id}.{CDN Subdomain}.{CDN Domain}.{Top-Level Domain}`. Must be all lowercase, no spaces or special characters, but may contain dashes/hyphens\ [#xmlValid]_.

.. table:: Aliases

	+------+---------------------------------+------------------------+
	| Name | Use(s)                          | Type(s)                |
	+======+=================================+========================+
	| Key  | Traffic Portal tables and forms | unchanged (``string``) |
	+------+---------------------------------+------------------------+

.. [#xmlValid] Some things to consider when choosing an xml_id and routing name: the name should be descriptive and unique, but as brief as possible to avoid creating a monstrous :abbr:`FQDN (Fully Qualified Domain Name)`. Also, because these are combined to form an :abbr:`FQDN (Fully Qualified Domain Name)`, they should not contain any characters that are illegal for a DNS subdomain, e.g. ``.`` (period/dot). Finally, the restrictions on what characters are allowable (especially in xml_id) are, in general, **NOT** enforced by the :ref:`to-api`, so take care that the name is appropriate. See :rfc:`1035` for exact guidelines.
.. [#cardinality] In source code and :ref:`to-api` responses, the "Long Description" fields of a Delivery Service are "0-indexed" - hence the names differing slightly from the ones displayed in user-friendly UIs.
.. [#dupOrigin] These Delivery Services Types are vulnerable to what this writer likes to call the "Duplicate Origin Problem". This problem is tracked by :issue:`3537`.
.. [#httpOnlyRegex] These regular expression types can only appear in the Match List of HTTP-:ref:`Routed <ds-types>` Delivery Services.


.. _ds-parameters:

Delivery Service Parameters
---------------------------

Features which are new, experimental, or not significant enough to be first-class Delivery Service fields are often added as Parameters. To use these, add a Profile to the Delivery Service, with the given Parameter assigned.

parent.config
'''''''''''''
The following Parameters must have the Config File ``parent.config`` to take effect.

- ``try_all_primaries_before_secondary`` - on a Delivery Service Profile, if this exists, try all primary parents before failing over to secondary parents, which may be ideal if objects are unlikely to be in cache. The default behavior is to immediately fail to a secondary, which is ideal if objects are likely to be in cache, as the first consistent-hashed secondary parent will be the primary parent in its own cachegroup and therefore receive requests for that object from clients near its own cachegroup.
- ``enable_h2`` - on a Delivery Service Profile, if this is ``true``, enable HTTP/2 for client requests. Note ATS must also be listening for HTTP/2 in records.config, or this will have no effect. Note this setting is not in the parent.config config file, but either ssl_server_name.yaml in ATS 8 or sni.yaml in ATS 9. The Parameter has the "parent.config" config file for consistency.
- ``tls_versions`` - on a Delivery Service Profile, if this exists, enable the given comma-delimited TLS versions for client requests. For example, ``1.1,1.2,1.3``. Note ATS must also be accepting those TLS versions in records.config, or this will have no effect. Note this setting is not in the parent.config config file, but either ssl_server_name.yaml in ATS 8 or sni.yaml in ATS 9. The Parameter has the "parent.config" config file for consistency.

Additionally, :term:`Delivery Service` :ref:`Profiles <ds-profile>` can have special Parameters with the :ref:`parameter-name` "mso.parent_retry" to :ref:`multi-site-origin-qht`.

.. seealso:: To see how the :ref:`Values <parameter-value>` of these Parameters are interpreted, refer to the `Apache Traffic Server documentation on the parent.config configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_
