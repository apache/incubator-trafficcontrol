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

.. _glossary:

********
Glossary
********

.. glossary::
	:sorted:

	302 content routing
		:ref:`http-cr`.

	ACME Account
		An account previously created with an :abbr:`ACME (Automatic Certificate Management Environment)` provider.

	astats (stats_over_http)
		An :abbr:`ATS (Apache Traffic Server)` plugin that allows you to monitor vitals of the :abbr:`ATS (Apache Traffic Server)` server. See :ref:`astats`.

	Cache Server
	cache server
	cache servers
		The main function of a CDN is to proxy requests from clients to :term:`origin servers` and cache the results. To proxy, in the CDN context, is to obtain content using HTTP from an :term:`origin server` on behalf of a client. To cache is to store the results so they can be reused when other clients are requesting the same content. There are three types of proxies in use on the Internet today:

		- :term:`reverse proxy`: Used by Traffic Control for Edge-tier :dfn:`cache servers`.
		- :term:`forward proxy`: Used by Traffic Control for Mid-tier :dfn:`cache servers`.
		- transparent proxy: These are not used by Traffic Control. If you are interested you can learn more about transparent proxies on `wikipedia <http://en.wikipedia.org/wiki/Proxy_server#Transparent_proxy>`_.

	Cache Group
	Cache Groups
		A group of caching HTTP proxy servers that together create a combined larger cache using consistent hashing. Traffic Router treats all servers in a :dfn:`Cache Group` as though they are in the  same geographic location, though they are in fact only in the same general area. A :dfn:`Cache Group` has one single set of geographical coordinates even if the :term:`cache servers` that make up the :dfn:`Cache Group` are actually in :term:`Physical Locations`. The :term:`cache servers` in a :dfn:`Cache Group` are not aware of the other :term:`cache servers` in the group - there is no clustering software or communications between :term:`cache servers` in a :dfn:`Cache Group`.

		There are two basic types of :dfn:`Cache Groups`: EDGE_LOC and MID_LOC ("LOC" being short for "location" - a holdover from when :dfn:`Cache Groups` were called "Cache Locations). Traffic Control is a two-tiered system, where the clients get directed to the Edge-tier (EDGE_LOC) :dfn:`Cache Group`. On cache miss, the :term:`cache server` in the Edge-tier :dfn:`Cache Group` obtains content from a Mid-tier (MID_LOC) :dfn:`Cache Group`, rather than the origin, which is shared with multiple Edge-tier :dfn:`Cache Groups`. Edge-tier :dfn:`Cache Groups` are usually configured to have a single "parent" :dfn:`Cache Group`, but in general Mid-tier :dfn:`Cache Groups` have many "children".

		..  Note:: Often the Edge-tier to Mid-tier relationship is based on network distance, and does not necessarily match the geographic distance.

		.. seealso:: A :dfn:`Cache Group` serves a particular part of the network as defined in the :term:`Coverage Zone File` (or :term:`Deep Coverage Zone File`, when applicable).

		.. seealso:: For a more complete description of Cache Groups, see the :ref:`cache-groups` overview section.

	content routing
		Directing clients (or client systems) to a particular location or device in a location for optimal delivery of content See also :ref:`http-cr` and :ref:`dns-cr`.

	Coverage Zone File
	Coverage Zone Map
		The :abbr:`CZM (Coverage Zone Map)` or :abbr:`CZF (Coverage Zone File)` is a file that maps network prefixes to :term:`Cache Groups`. Traffic Router uses the :abbr:`CZM (Coverage Zone Map)` to determine what :term:`Cache Group` is closest to the client. If the client IP address is not in this :abbr:`CZM (Coverage Zone Map)`, it falls back to geographic mapping, using a `MaxMind GeoIP2 database <https://www.maxmind.com/en/geoip2-databases>`_ to find the client's location, and the geographic coordinates from Traffic Ops for the :term:`Cache Group`. Traffic Router is inserted into the HTTP retrieval process by making it the authoritative DNS server for the domain of the CDN :term:`Delivery Service`. In the example of the :term:`reverse proxy`, the client was given the ``http://www-origin-cache.cdn.com/foo/bar/fun.html`` URL. In a Traffic Control CDN, URLs start with a routing name, which is configurable per-:term:`Delivery Service`, e.g. ``http://foo.mydeliveryservice.cdn.com/fun/example.html`` with the chosen routing name ``foo``.

		.. code-block:: json
			:caption: Example Coverage Zone File

			{ "coverageZones": {
				"cache-group-01": {
					"network6": [
						"1234:5678::/64",
						"1234:5679::/64"
					],
					"network": [
						"192.168.8.0/24",
						"192.168.9.0/24"
					]
				}
			}}


	Deep Coverage Zone File
	Deep Coverage Zone Map
		The :abbr:`DCZF (Deep Coverage Zone File)` or :abbr:`DCZM (Deep Coverage Zone Map)` maps network prefixes to "locations" - almost like the :term:`Coverage Zone File`. Location names must be unique, and within the file are simply used to group :term:`Edge-tier cache servers`. When a mapping is performed by Traffic Router, it will only look in the :abbr:`DCZF (Deep Coverage Zone File)` if the :term:`Delivery Service` to which a client is being directed makes use of :ref:`ds-deep-caching`. If the client's IP address cannot be matched by entries in this file, Traffic Router will first fall back to the regular :term:`Coverage Zone File`. Then, failing that, it will perform geographic mapping using a database provided by the :term:`Delivery Service`'s :ref:`ds-geo-provider`.

		.. code-block:: json
			:caption: Example Deep Coverage Zone File

			{ "deepCoverageZones": {
				"location-01": {
					"network6": [
						"1234:5678::/64",
						"1234:5679::/64"
					],
					"network": [
						"192.168.8.0/24",
						"192.168.9.0/24"
					],
					"caches": [
						"edge"
					]
				}
			}}

	Delivery Service
	Delivery Services
		:dfn:`Delivery Services` are often referred to as a :term:`reverse proxy` "remap rule" that exists on Edge-tier :term:`cache servers`. In most cases, a :dfn:`Delivery Service` is a one-to-one mapping to an :abbr:`FQDN (Fully Qualified Domain Name)` that is used as a hostname to deliver the content. Many options and settings regarding how to optimize the content delivery exist, which are configurable on a :dfn:`Delivery Service` basis. Some examples of these :dfn:`Delivery Service` settings are:

		* Cache in RAM, cache on disk, or do not cache at all.
		* Use DNS or HTTP Content routing.
		* Limits on transactions per second and bandwidth.
		* Protocol (HTTP or HTTPS).
		* Token-based authentication settings.
		* Header rewrite rules.

		Since Traffic Control version 2.1, :dfn:`Delivery Services` can optionally be linked to a :term:`Profile`, and have :term:`Parameters` associated with them. One example of a feature that uses :dfn:`Delivery Service` :term:`Parameters` is the :ref:`ds-multi-site-origin` configuration. :dfn:`Delivery Services` are also for use in allowing multiple :term:`Tenants` to coexist in a Traffic Control CDN without interfering with each other, and to keep information about their content separated.

		.. seealso:: See :ref:`delivery-services` for a more in-depth explanation of :dfn:`Delivery Services`.

	Delivery Service Request
	Delivery Service Requests
	DSR
	DSRs
		A :dfn:`Delivery Service Request` is the result of attempting to modify a :term:`Delivery Service` when ``dsRequests.enabled`` is set to ``true`` in ``traffic_portal_properties.json``. See :ref:`ds_requests` for more information.

		.. seealso:: See :ref:`delivery-service-requests` for a more in-depth explanation of :dfn:`Delivery Service Requests`, including their data model. See :ref:`ds_requests` for more information on how to use :dfn:`Delivery Service Requests` in Traffic Portal.

	Delivery Service required capabilities
		:dfn:`Delivery Services required capabilities` are capabilities, which correlate to server capabilities, that are required in order to assign a server to a delivery service.`

	Division
	Divisions
		A group of :term:`Regions`.

	Edge
	Edge-tier
	Edge-Tier
	Edge-tier cache
	Edge-tier caches
	Edge-tier cache server
	Edge-tier cache servers
		Closest to the client or end-user. The edge tier is the tier that serves the client, edge caches are caches in the edge tier. In a Traffic Control CDN the basic function of the edge cache is that of a :term:`reverse proxy`.

	Federation
	Federations
		:dfn:`Federations` allow for other ("federated") CDNs (e.g. at a different :abbr:`ISP (Internet Service Provider)`) to add a list of DNS resolvers and an :abbr:`FQDN (Fully Qualified Domain Name)` to be used in a DNS CNAME record for a :term:`Delivery Service`. When a request is made from one of the federated CDN's clients, Traffic Router will return the CNAME record configured from the federation mapping. This allows the federated CDN to serve the content without the content provider changing the URL, or having to manage multiple URLs. For example, if the external CDN was actually another :abbr:`ATC (Apache Traffic Control)`-managed CDN, then a federation mapping to direct clients toward it should use the :abbr:`FQDN (Fully Qualified Domain Name)` of a :term:`Delivery Service` on the external CDN.

		Federations only have meaning to DNS-routed :term:`Delivery Services` - HTTP-routed Delivery services should instead treat the external :abbr:`FQDN (Fully Qualified Domain Name)` as an :term:`Origin` to achieve the same effect.

		.. seealso:: Federations are currently only manageable by directly using the :ref:`to-api`. The endpoints related to federations are :ref:`to-api-federations`, :ref:`to-api-federation_resolvers`, :ref:`to-api-federations-id-deliveryservices`, :ref:`to-api-federations-id-deliveryservices-id`, :ref:`to-api-federations-id-federation_resolvers`, :ref:`to-api-federations-id-users`, and :ref:`to-api-federations-id-users-id`.

	First-tier
	First-tier cache
	First-tier caches
	First-tier cache server
	First-tier cache servers
		Closest to the client or end-user. The first tier in a :term:`Topology` is the tier that serves the client, similar to the :term:`Edge-tier`.

	forward proxy
	forward proxies
		A forward proxy acts on behalf of the client such that the :term:`origin server` is (potentially) unaware of the proxy's existence. All Mid-tier :term:`cache servers` in a Traffic Control based CDN are :dfn:`forward proxies`. In a :dfn:`forward proxy` scenario, the client is explicitly configured to use the the proxy's IP address and port as a :dfn:`forward proxy`. The client always connects to the :dfn:`forward proxy` for content. The content provider does not have to change the URL the client obtains, and is (potentially) unaware of the proxy in the middle.

		..  seealso:: `ATS documentation on forward proxy <https://docs.trafficserver.apache.org/en/latest/admin/forward-proxy.en.html>`_.

		If a client uses a :dfn:`forward proxy` to request the URL ``http://www.origin.com/foo/bar/fun.html`` the resulting chain of events follows.

		#. To retrieve ``http://www.origin.com/foo/bar/fun.html``, the client sends an HTTP request to the :dfn:`forward proxy`.

			.. code-block:: http
				:caption: Client Requests Content from its :dfn:`Forward Proxy`

				GET http://www.origin.com/foo/bar/fun.html HTTP/1.1
				Host: www.origin.com

			..  Note:: In this case, the client requests the entire URL instead of just the path as is the case when using a :term:`reverse proxy` or when requesting content directly from the :term:`origin server`.

		#. The proxy verifies whether the response for ``http://www-origin-cache.cdn.com/foo/bar/fun.html`` is already in the cache. If it is not in the cache:

			#. The proxy sends the HTTP request to the :term:`Origin`.

				.. code-block:: http
					:caption: The :dfn:`Forward Proxy` Requests Content from the :term:`origin server`

					GET /foo/bar/fun.html HTTP/1.1
					Host: www.origin.com

			#. The :term:`origin server` responds with the requested content.

				.. code-block:: http
					:caption: The :term:`origin server`'s Response

					HTTP/1.1 200 OK
					Date: Sun, 14 Dec 2014 23:22:44 GMT
					Server: Apache/2.2.15 (Red Hat)
					Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
					ETag: "1aa008f-2d-50a3559482cc0"
					Content-Length: 45
					Connection: close
					Content-Type: text/html; charset=UTF-8

					<!DOCTYPE html><html><body>This is a fun file</body></html>


			#. The proxy sends this on to the client, optionally adding a ``Via:`` header to indicate that the request was serviced by proxy.

				.. code-block:: http
					:caption: The :dfn:`Forward Proxy`'s Response to the Client

					HTTP/1.1 200 OK
					Date: Sun, 14 Dec 2014 23:22:44 GMT
					Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
					ETag: "1aa008f-2d-50a3559482cc0"
					Content-Length: 45
					Connection: close
					Content-Type: text/html; charset=UTF-8
					Age: 0
					Via: http/1.1 cache01.cdn.kabletown.net (ApacheTrafficServer/4.2.1 [uScSsSfUpSeN:t cCSi p sS])
					Server: ATS/4.2.1

					<!DOCTYPE html><html><body>This is a fun file</body></html>


			If, however, the requested content *was* in the cache the proxy responds to the client with the previously retrieved result

			.. code-block:: http
				:caption: The :dfn:`Forward Proxy` Sends the Cached Response

				HTTP/1.1 200 OK
				Date: Sun, 14 Dec 2014 23:22:44 GMT
				Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
				ETag: "1aa008f-2d-50a3559482cc0"
				Content-Length: 45
				Connection: close
				Content-Type: text/html; charset=UTF-8
				Age: 99711
				Via: http/1.1 cache01.cdn.kabletown.net (ApacheTrafficServer/4.2.1 [uScSsSfUpSeN:t cCSi p sS])
				Server: ATS/4.2.1

				<!DOCTYPE html><html><body>This is a fun file</body></html>

	geo localization or geo routing
		Localizing clients to the nearest caches using a geo database like the one from Maxmind.

	Health Protocol
		The protocol to monitor the health of all the caches. See :ref:`health-proto`.

	Inner-tier
	Inner-tier cache
	Inner-tier caches
	Inner-tier cache server
	Inner-tier cache servers
		The tier between the First tier and the Last tier. The inner tier in a :term:`Topology` is the tier that forwards requests from other caches to other caches, i.e. caches in this tier do not directly serve the end-user and do not make requests to :term:`Origins`.

	Last-tier
	Last-tier cache
	Last-tier caches
	Last-tier cache server
	Last-tier cache servers
		The tier above the First and Inner tiers. The last tier in a :term:`Topology` is the tier that forwards requests from other caches to :term:`Origins`.

	localization
		Finding location on the network, or on planet earth

	Mid
	Mid-tier
	Mid-Tier
	Mid-tier cache
	Mid-tier caches
	Mid-tier cache server
	Mid-tier cache servers
		The tier above the edge tier. The mid tier does not directly serves the end-user and is used as an additional layer between the edge and the :term:`Origin`. In a Traffic Control CDN the basic function of the mid cache is that of a :term:`forward proxy`.

	Origin
	Origins
	origin server
	origin servers
	Origin Servers
		The source of content for the CDN. Usually a redundant HTTP/1.1 webserver.

	ORT
		The previous Traffic Control cache config app, replaced by :term:`t3c`.

	Parameter
	Parameters
		Typically refers to a line in a configuration file, but in practice can represent any arbitrary configuration option.

		.. seealso:: The :ref:`profiles-and-parameters` overview section.

	parent
	parents
		The :dfn:`parent(s)` of a :term:`cache server` is/are the :term:`cache server`\ (s) belonging to either the "parent" or "secondary parent" :term:`Cache Group`\ (s) of the :term:`Cache Group` to which the :term:`cache server` belongs. For example, in general it is true that an :term:`Edge-tier cache server` has one or more :dfn:`parents` which are :term:`Mid-tier cache servers`.

	Physical Location
	Physical Locations
		A pair of geographic coordinates (latitude and longitude) that is used by :term:`Cache Groups` to define their location. This information is used by Traffic Router to route client traffic to the geographically nearest :term:`Cache Group`.

	Profile
	Profiles
		A :dfn:`Profile` is, most generally, a group of :term:`Parameters` that will be applied to a server. :dfn:`Profiles` are typically re-used by all :term:`Edge-tier cache servers` within a CDN or :term:`Cache Group`. A :dfn:`Profile` will, in addition to configuration :term:`Parameters`, define the CDN to which a server belongs and the :ref:`"Type" <profile-type>` of the Profile - which determines some behaviors of Traffic Control components. The allowed :ref:`"Types" <profile-type>` of :dfn:`Profiles` are **not** the same as :term:`Types`, and are maintained as a PostgreSQL "Enum" in :atc-file:`traffic_ops/app/db/create_tables.sql`.

		.. tip:: A :dfn:`Profile` of the wrong type assigned to a Traffic Control component *will* (in general) cause it to function incorrectly, regardless of the :term:`Parameters` assigned to it.

		.. seealso:: The :ref:`profiles-and-parameters` overview section.

	Queue
	Queue Updates
	Queue Server Updates
		:dfn:`Queuing Updates` is an action that signals to various ATC components - most notably :term:`cache servers` - that any configuration changes that are pending are to be applied now. Specifically, Traffic Monitor and Traffic Router are updated through a CDN :term:`Snapshot`, and *not* :dfn:`Queued Updates`. In particular, :term:`ORT` will notice that the server on which it's running has new configuration, and will request the new configuration from Traffic Ops.

		Updates may be queued on a server-by-server basis (in Traffic Portal's :ref:`tp-configure-servers` view), a Cache Group-wide basis (in Traffic Portal's :ref:`tp-configure-cache-groups` view), or on a CDN-wide basis (in Traffic Portal's :ref:`tp-cdns` view). Usually using the CDN-wide version is easiest, and unless there are special circumstances, and/or the user really knows what he or she is doing, it is recommended that the full CDN-wide :dfn:`Queue Updates` be used.

		This is similar to taking a CDN :term:`Snapshot`, but this configuration change affects only servers, and not routing.

		That seems like a vague difference because it is - in general the rule to follow is that changes to :term:`Profiles` and :term:`Parameters` requires only updates be queued, changes to the assignments of :term:`cache servers` to :term:`Delivery Services` requires both a :term:`Snapshot` *and* a :dfn:`Queue Updates`, and changes to only a :term:`Delivery Service` itself (usually) entails a :term:`Snapshot` only. These aren't exhaustive rules, and a grasp of what changes require which action(s) will take time to form. In general, when doing both :dfn:`Queuing Updates` as well as taking a CDN :term:`Snapshot`, it is advisable to first :dfn:`Queue Updates` and *then* take the :term:`Snapshot`, as otherwise Traffic Router may route clients to :term:`Edge-tier cache servers` that are not equipped to service their request(s). However, when modifying the assignment(s) of :term:`cache servers` to one or more :term:`Delivery Services`, a :term:`Snapshot` ought to be taken before updates are queued.

		.. warning:: Updates to :term:`Parameters` with certain :ref:`parameter-config-file` values may require running :term:`ORT` in a different mode, occasionally manually. Though the server may appear to no longer have pending updates in these cases, until this manual intervention is performed the configuration *will* **not** *be correct*.

	Region
	Regions
		A group of :term:`Physical Locations`.

	reverse proxy
	reverse proxies
		A :dfn:`reverse proxy` acts on behalf of the :term:`origin server` such that the client is (potentially) unaware it is not communicating directly with the :term:`Origin`. All Edge-tier :term:`cache servers` in a Traffic Control CDN are :dfn:`reverse proxies`. To the end user a Traffic Control-based CDN appears as a :dfn:`reverse proxy` since it retrieves content from the :term:`origin server`, acting on behalf of that :term:`origin server`. The client requests a URL that has a hostname which resolves to the :dfn:`reverse proxy`'s IP address and, in compliance with the HTTP 1.1 specification (:rfc:`2616`), the client sends a ``Host:`` header to the :dfn:`reverse proxy` that matches the hostname in the URL. The proxy looks up this hostname in a list of mappings to find the :term:`Origin` hostname; if the hostname of the ``Host:`` header is not found in the list, the proxy will send an error (usually either ``404 Not Found`` or ``503 Service Unavailable`` as appropriate) to the client. If the supplied hostname is found in this list of mappings, the proxy checks its cache, and when the content is not already present, connects to the :term:`Origin` to which the requested ``Host:`` maps requests the path of the original URL, providing the :term:`Origin` hostname in the ``Host`` header. The proxy then stores the URL in its cache and serves the contents to the client. When there are subsequent requests for the same URL, a caching proxy serves the content out of its cache - provided :ref:`cache-revalidation` are satisfied - thereby reducing latency and network traffic.

		.. seealso:: `The Apache Traffic Server documentation on reverse proxy <https://docs.trafficserver.apache.org/en/latest/admin/reverse-proxy-http-redirects.en.html#http-reverse-proxy>`_.

		To insert a :dfn:`reverse proxy` into a typical HTTP 1.1 request and response flow, the :dfn:`reverse proxy` needs to be told where the :term:`origin server` can be reached (and which :term:`Origin` to use for a given request when it's configured to proxy requests for multiple :term:`Origins`). In :abbr:`ATS (Apache Traffic Server)` this is handled by adding rules to `the remap.config configuration file <https://docs.trafficserver.apache.org/en/latest/admin-guide/files/remap.config.en.html>`_. The content owner must inform the clients, by updating the URL, to receive the content from the cache and not from the :term:`origin server` directly. For example, clients might be instructed to request content from ``http://www-origin-cache.cdn.com`` which points to a :dfn:`reverse proxy` for the actual :term:`Origin` located at ``http://www.origin.com``.

		Now, if the client requests ``/foo/bar/fun.html`` from the :dfn:`reverse proxy` the sequence of events is as follows. is given the URL ``http://www-origin-cache.cdn.com/foo/bar/fun.html`` (note the different hostname) and when attempting to obtain that URL, the following occurs:

		#. The client sends a DNS request to the :abbr:`LDNS (Local Domain Name Server)` to resolve the name ``www-origin-cache.cdn.com`` to an IP address.
		#. The :abbr:`LDNS (Local Domain Name Server)` finds an IP address for ``www-origin-cache.cdn.com`` e.g. ``55.44.33.22``.
		#. The client sends an HTTP request for ``/foo/bar/fun.html`` to the IP address.

			.. code-block:: http
				:caption: Client Requests Content from the :dfn:`Reverse Proxy`

				GET /foo/bar/fun.html HTTP/1.1
				Host: www-origin-cache.cdn.com

		#. The :dfn:`reverse proxy` finds out the URL of the true :term:`Origin` - in the case of :abbr:`ATS (Apache Traffic Server)` this is done by looking up ``www-origin-cache.cdn.com`` in its remap rules - and finds that it is ``www.origin.com``.
		#. The proxy checks its cache to see if the response for ``GET /foo/bar/fun.html HTTP/1.1`` from ``www.origin.com`` is already in the cache.
		#. If the response is not in the cache:

			#. The proxy sends the request to the actual :term:`Origin`

				.. code-block:: http
					:caption: :dfn:`Reverse Proxy` Requests Content from the :term:`origin server`

					GET /foo/bar/fun.html HTTP/1.1
					Host: www.origin.com

			#. The :term:`origin server` responds with the requested content

				.. code-block:: http
					:caption: Response from the :term:`origin server`

					HTTP/1.1 200 OK
					Date: Sun, 14 Dec 2014 23:22:44 GMT
					Server: Apache/2.2.15 (Red Hat)
					Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
					ETag: "1aa008f-2d-50a3559482cc0"
					Content-Length: 45
					Connection: close
					Content-Type: text/html; charset=UTF-8

					<!DOCTYPE html><html><body>This is a fun file</body></html>

			#. The proxy sends the response on to the client, optionally adding a ``Via:`` header to indicate that the request was serviced by proxy.

				.. code-block:: http
					:caption: Resulting Response from the :dfn:`Reverse Proxy` to the Client

					HTTP/1.1 200 OK
					Date: Sun, 14 Dec 2014 23:22:44 GMT
					Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
					ETag: "1aa008f-2d-50a3559482cc0"
					Content-Length: 45
					Connection: close
					Content-Type: text/html; charset=UTF-8
					Age: 0
					Via: http/1.1 cache01.cdn.kabletown.net (ApacheTrafficServer/4.2.1 [uScSsSfUpSeN:t cCSi p sS])
					Server: ATS/4.2.1

					<!DOCTYPE html><html><body>This is a fun file</body></html>

			If, however, the response *was* already in the cache - and still valid according to the :ref:`cache-revalidation` - the proxy responds to the client with the previously retrieved result.

			.. code-block:: http
				:caption: The :dfn:`Reverse Proxy` Provides a Cached Response

				HTTP/1.1 200 OK
				Date: Sun, 14 Dec 2014 23:22:44 GMT
				Last-Modified: Sun, 14 Dec 2014 23:18:51 GMT
				ETag: "1aa008f-2d-50a3559482cc0"
				Content-Length: 45
				Connection: close
				Content-Type: text/html; charset=UTF-8
				Age: 39711
				Via: http/1.1 cache01.cdn.kabletown.net (ApacheTrafficServer/4.2.1 [uScSsSfUpSeN:t cCSi p sS])
				Server: ATS/4.2.1

				<!DOCTYPE html><html><body>This is a fun file</body></html>

	Role
	Roles
		Permissions :dfn:`Roles` define the operations a user is allowed to perform, and are currently an ordered list of permission levels.

	Server Capability
	Server Capabilities
		A :dfn:`Server Capability` (not to be confused with a "Capability") expresses the capacity of a :term:`cache server` to serve a particular kind of traffic. For example, a :dfn:`Server Capability` could be created named "RAM" to be assigned to :term:`cache servers` that have RAM-disks allocated for content caching. :dfn:`Server Capabilities` can also be required by :term:`Delivery Services`, which will prevent :term:`cache servers` without that :dfn:`Server Capability` from being assigned to them. It also prevents :term:`Mid-tier cache servers` without said :term:`Server Capability` from being selected to serve upstream requests from those :term:`Edge-tier cache servers` assigned to the requiring :term:`Delivery Services`.

	Service Category
	Service Categories
		A :dfn:`Service Category` defines the type of content being delivered by a :dfn:`Delivery Service`. For example, a :dfn:`Service Category` could be created named "linear" and assigned to a :dfn:`Delivery Service` that delivers linear content.

	Snapshot
	Snapshots
	CDN Snapshot
	CDN Snapshots
		Previously called a "CRConfig" or "CRConfig.json" (and still called such in many places), this is a rather large set of routing information generated from a CDN's configuration and topology.

	Status
	Statuses
		A :dfn:`Status` represents the current operating state of a server. The default :dfn:`Statuses` made available on initial startup of Traffic Ops are related to the :ref:`health-proto` and are explained in that section.

	t3c
		The Traffic Control cache config app, used to generate and apply cache configuration files.

	Tenant
	Tenants
	Tenancy
	Tenancies
		Users are grouped into :dfn:`Tenants` (or :dfn:`Tenancies`) to segregate ownership of and permissions over :term:`Delivery Services` and their resources. To be clear, the notion of :dfn:`Tenancy` **only** applies within the context of :term:`Delivery Services` and does **not** apply permissions restrictions to any other aspect of Traffic Control.

	Topology Node
	Topology Nodes
	Parent Topology Node
	Parent Topology Nodes
	Child Topology Node
	Child Topology Nodes
		Each :dfn:`Topology Node` is associated with a particular :term:`Cache Group`. In addition, the Topology Node has 0, 1, or 2 Parent Topology Nodes and has 0, 1, or 2 Child Topology Nodes, according to your configuration.

	Topology
	Topologies
		A structure composed of :term:`Cache Groups` and parent relationships, which is assignable to one or more :term:`Delivery Services`.

	Type
	Types
		A :dfn:`Type` defines a type of some kind of object configured in Traffic Ops. Unfortunately, that is exactly as specific as this definition can be.
