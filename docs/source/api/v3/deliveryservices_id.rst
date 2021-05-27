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

.. _to-api-v3-deliveryservices-id:

***************************
``deliveryservices/{{ID}}``
***************************

``PUT``
=======
Allows users to edit an existing :term:`Delivery Service`.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"\ [#tenancy]_
:Response Type:  **NOT PRESENT** - Despite returning a ``200 OK`` response (rather than e.g. a ``204 NO CONTENT`` response), this endpoint does **not** return a representation of the modified resource in its payload, and instead returns nothing - not even a success message.

Request Structure
-----------------
:active:                   A boolean that defines :ref:`ds-active`. ``true`` means that the Delivery Service is ACTIVE, while ``false`` means that it is either PRIMED or INACTIVE (API versions earlier than 4 are incapable of distinguishing between the two).
:anonymousBlockingEnabled: A boolean that defines :ref:`ds-anonymous-blocking`
:cacheurl:                 A :ref:`ds-cacheurl`

	.. deprecated:: ATCv3.0
		This field has been deprecated in Traffic Control 3.x and is subject to removal in Traffic Control 4.x or later

:ccrDnsTtl:                 The :ref:`ds-dns-ttl` - named "ccrDnsTtl" for legacy reasons
:cdnId:                     The integral, unique identifier of the :ref:`ds-cdn` to which the :term:`Delivery Service` belongs

		.. note:: If the Delivery Service has SSL Keys, then cdnId is not allowed to change as that would invalidate the SSL Key

:checkPath:                 A :ref:`ds-check-path`
:consistentHashRegex:       A :ref:`ds-consistent-hashing-regex`
:consistentHashQueryParams: An array of :ref:`ds-consistent-hashing-qparams`
:deepCachingType:           The :ref:`ds-deep-caching` setting for this :term:`Delivery Service`
:displayName:               The :ref:`ds-display-name`
:dnsBypassCname:            A :ref:`ds-dns-bypass-cname`
:dnsBypassIp:               A :ref:`ds-dns-bypass-ip`
:dnsBypassIp6:              A :ref:`ds-dns-bypass-ipv6`
:dnsBypassTtl:              The :ref:`ds-dns-bypass-ttl`
:dscp:                      A :ref:`ds-dscp` to be used within the :term:`Delivery Service`
:ecsEnabled:                A boolean that defines the :ref:`ds-ecs` setting on this :term:`Delivery Service`
:edgeHeaderRewrite:         A set of :ref:`ds-edge-header-rw-rules`
:firstHeaderRewrite:        A set of :ref:`ds-first-header-rw-rules`
:fqPacingRate:              The :ref:`ds-fqpr`
:geoLimit:                  An integer that defines the :ref:`ds-geo-limit`
:geoLimitCountries:         A string containing a comma-separated list defining the :ref:`ds-geo-limit-countries`\ [#geolimit]_
:geoLimitRedirectUrl:       A :ref:`ds-geo-limit-redirect-url`\ [#geolimit]_
:geoProvider:               The :ref:`ds-geo-provider`
:globalMaxMbps:             The :ref:`ds-global-max-mbps`
:globalMaxTps:              The :ref:`ds-global-max-tps`
:httpBypassFqdn:            A :ref:`ds-http-bypass-fqdn`
:infoUrl:                   An :ref:`ds-info-url`
:initialDispersion:         The :ref:`ds-initial-dispersion`
:innerHeaderRewrite:        A set of :ref:`ds-inner-header-rw-rules`
:ipv6RoutingEnabled:        A boolean that defines the :ref:`ds-ipv6-routing` setting on this :term:`Delivery Service`
:lastHeaderRewrite:         A set of :ref:`ds-last-header-rw-rules`
:logsEnabled:               A boolean that defines the :ref:`ds-logs-enabled` setting on this :term:`Delivery Service`
:longDesc:                  The :ref:`ds-longdesc` of this :term:`Delivery Service`
:longDesc1:                 An optional field containing the :ref:`ds-longdesc2` of this :term:`Delivery Service`
:longDesc2:                 An optional field containing the :ref:`ds-longdesc3` of this :term:`Delivery Service`
:maxDnsAnswers:             The :ref:`ds-max-dns-answers` allowed for this :term:`Delivery Service`
:maxOriginConnections:      The :ref:`ds-max-origin-connections`
:midHeaderRewrite:          A set of :ref:`ds-mid-header-rw-rules`
:missLat:                   The :ref:`ds-geo-miss-default-latitude` used by this :term:`Delivery Service`
:missLong:                  The :ref:`ds-geo-miss-default-longitude` used by this :term:`Delivery Service`
:multiSiteOrigin:           A boolean that defines the use of :ref:`ds-multi-site-origin` by this :term:`Delivery Service`
:orgServerFqdn:             The :ref:`ds-origin-url`
:originShield:              A :ref:`ds-origin-shield` string
:profileId:                 An optional :ref:`profile-id` of the :ref:`ds-profile` with which this :term:`Delivery Service` will be associated
:protocol:                  An integral, unique identifier that corresponds to the :ref:`ds-protocol` used by this :term:`Delivery Service`
:qstringIgnore:             An integral, unique identifier that corresponds to the :ref:`ds-qstring-handling` setting on this :term:`Delivery Service`
:rangeRequestHandling:      An integral, unique identifier that corresponds to the :ref:`ds-range-request-handling` setting on this :term:`Delivery Service`
:regexRemap:                A :ref:`ds-regex-remap`
:regionalGeoBlocking:       A boolean defining the :ref:`ds-regionalgeo` setting on this :term:`Delivery Service`
:remapText:                 :ref:`ds-raw-remap`
:routingName:               The :ref:`ds-routing-name` of this :term:`Delivery Service`

		.. note:: If the Delivery Service has SSL Keys, then routingName is not allowed to change as that would invalidate the SSL Key

:signed:                    ``true`` if  and only if ``signingAlgorithm`` is not ``null``, ``false`` otherwise
:signingAlgorithm:          Either a :ref:`ds-signing-algorithm` or ``null`` to indicate URL/URI signing is not implemented on this :term:`Delivery Service`
:rangeSliceBlockSize:      An integer that defines the byte block size for the ATS Slice Plugin. It can only and must be set if ``rangeRequestHandling`` is set to 3. It can only be between (inclusive) 262144 (256KB) - 33554432 (32MB).
:sslKeyVersion:             This integer indicates the :ref:`ds-ssl-key-version`
:tenantId:                  The integral, unique identifier of the :ref:`ds-tenant` who owns this :term:`Delivery Service`
:topology:                  The unique name of the :term:`Topology` that this :term:`Delivery Service` is assigned to
:trRequestHeaders:          If defined, this defines the :ref:`ds-tr-req-headers` used by Traffic Router for this :term:`Delivery Service`
:trResponseHeaders:         If defined, this defines the :ref:`ds-tr-resp-headers` used by Traffic Router for this :term:`Delivery Service`
:typeId:                    The integral, unique identifier of the :ref:`ds-types` of this :term:`Delivery Service`
:xmlId:                     This :term:`Delivery Service`'s :ref:`ds-xmlid`

	.. note:: While this field **must** be present, it is **not** allowed to change; this must be the same as the ``xml_id`` the :term:`Delivery Service` already has. This should almost never be different from the :term:`Delivery Service`'s ``displayName``.


.. code-block:: http
	:caption: Request Example

	PUT /api/3.0/deliveryservices/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 761
	Content-Type: application/json

	{
		"active": true,
		"anonymousBlockingEnabled": false,
		"cdnId": 2,
		"cdnName": "CDN-in-a-Box",
		"deepCachingType": "NEVER",
		"displayName": "demo",
		"dscp": 0,
		"ecsEnabled": true,
		"geoLimit": 0,
		"geoProvider": 0,
		"initialDispersion": 1,
		"ipv6RoutingEnabled": false,
		"lastUpdated": "2018-11-14 18:21:17+00",
		"logsEnabled": true,
		"longDesc": "A Delivery Service created expressly for API documentation examples",
		"missLat": -1,
		"missLong": -1,
		"multiSiteOrigin": false,
		"orgServerFqdn": "http://origin.infra.ciab.test",
		"protocol": 0,
		"qstringIgnore": 0,
		"rangeRequestHandling": 0,
		"regionalGeoBlocking": false,
		"routingName": "video",
		"signed": false,
		"tenant": "root",
		"tenantId": 1,
		"typeId": 1,
		"xmlId": "demo1"
	}


Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: z4PhNX7vuL3xVChQ1m2AB9Yg5AULVxXcg/SpIdNs6c5H0NE8XYXysP+DGNKHfuwvY7kxvUdBeoGlODJ6+SfaPg==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 20 Nov 2018 14:12:25 GMT
	Content-Length: 0
	Content-Type: text/plain; charset=utf-8


``DELETE``
==========
Deletes the target :term:`Delivery Service`

:Auth. Required: Yes
:Roles Required: "admin" or "operations"\ [#tenancy]_
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-------------------------------------------------------------------------------+
	| Name | Description                                                                   |
	+======+===============================================================================+
	| ID   | The integral, unique identifier of the :term:`Delivery Service` to be deleted |
	+------+-------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/3.0/deliveryservices/2 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...


Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: w9NlQpJJEl56r6iYq/fk8o5WfAXeUS5XR9yDHvKUgPO8lYEo8YyftaSF0MPFseeOk60dk6kQo+MLYTDIAhhRxw==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 20 Nov 2018 14:56:37 GMT
	Content-Length: 57

	{ "alerts": [
		{
			"text": "ds was deleted.",
			"level": "success"
		}
	]}


.. [#tenancy] Only those :term:`Delivery Services` assigned to :term:`Tenants` that are the requesting user's :term:`Tenant` or children thereof will appear in the output of a ``GET`` request, and the same constraints are placed on the allowed values of the ``tenantId`` field of a ``PUT`` request to update a new :term:`Delivery Service`. Furthermore, the only :term:`Delivery Services` a user may delete are those assigned to a :term:`Tenant` that is either the same :term:`Tenant` as the user's :term:`Tenant`, or a descendant thereof.
.. [#geoLimit] These fields must be defined if and only if ``geoLimit`` is non-zero
