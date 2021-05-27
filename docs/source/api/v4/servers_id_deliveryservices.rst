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

.. _to-api-servers-id-deliveryservices:

***********************************
``servers/{{ID}}/deliveryservices``
***********************************

``GET``
=======
Retrieves all :term:`Delivery Services` assigned to a specific server.

:Auth. Required: Yes
:Roles Required: None\ [#tenancy]_
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+------------------------------------------------------------------------------------------------------------+
	| Name | Description                                                                                                |
	+======+============================================================================================================+
	| ID   | The integral, unique identifier of the server for which assigned :term:`Delivery Services` shall be listed |
	+------+------------------------------------------------------------------------------------------------------------+

.. table:: Request Query Parameters

	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Required | Description                                                                                                   |
	+===========+==========+===============================================================================================================+
	| orderby   | no       | Choose the ordering of the results - must be the name of one of the fields of the objects in the ``response`` |
	|           |          | array                                                                                                         |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| sortOrder | no       | Changes the order of sorting. Either ascending (default or "asc") or descending ("desc")                      |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| limit     | no       | Choose the maximum number of results to return                                                                |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| offset    | no       | The number of results to skip before beginning to return results. Must use in conjunction with limit          |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| page      | no       | Return the n\ :sup:`th` page of results, where "n" is the value of this parameter, pages are ``limit`` long   |
	|           |          | and the first page is 1. If ``offset`` was defined, this query parameter has no effect. ``limit`` must be     |
	|           |          | defined to make use of ``page``.                                                                              |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/servers/9/deliveryservices HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:active: A string that defines :ref:`ds-active`.

	.. versionchanged:: 4.0
		Prior to API version 4.0, this was a boolean field.

:anonymousBlockingEnabled:  A boolean that defines :ref:`ds-anonymous-blocking`
:ccrDnsTtl:                 The :ref:`ds-dns-ttl` - named "ccrDnsTtl" for legacy reasons
:cdnId:                     The integral, unique identifier of the :ref:`ds-cdn` to which the :term:`Delivery Service` belongs
:cdnName:                   Name of the :ref:`ds-cdn` to which the :term:`Delivery Service` belongs
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
:exampleURLs:               An array of :ref:`ds-example-urls`
:firstHeaderRewrite:        A set of :ref:`ds-first-header-rw-rules`
:fqPacingRate:              The :ref:`ds-fqpr`
:geoLimit:                  An integer that defines the :ref:`ds-geo-limit`
:geoLimitCountries:         A string containing a comma-separated list defining the :ref:`ds-geo-limit-countries`
:geoLimitRedirectUrl:       A :ref:`ds-geo-limit-redirect-url`
:geoProvider:               The :ref:`ds-geo-provider`
:globalMaxMbps:             The :ref:`ds-global-max-mbps`
:globalMaxTps:              The :ref:`ds-global-max-tps`
:httpBypassFqdn:            A :ref:`ds-http-bypass-fqdn`
:id:                        An integral, unique identifier for this :term:`Delivery Service`
:infoUrl:                   An :ref:`ds-info-url`
:initialDispersion:         The :ref:`ds-initial-dispersion`
:innerHeaderRewrite:        A set of :ref:`ds-inner-header-rw-rules`
:ipv6RoutingEnabled:        A boolean that defines the :ref:`ds-ipv6-routing` setting on this :term:`Delivery Service`
:lastHeaderRewrite:         A set of :ref:`ds-last-header-rw-rules`
:lastUpdated:               The date and time at which this :term:`Delivery Service` was last updated, in :rfc:`3339` format

	.. versionchanged:: 4.0
		In API versions earlier than 4.0, this string is in a legacy format - ``YYYY-MM-DD hh:mm:ss+ZZ`` (no sub-second precision) - instead of :rfc:`3339` format.

:logsEnabled: A boolean that defines the :ref:`ds-logs-enabled` setting on this :term:`Delivery Service`
:longDesc:    The :ref:`ds-longdesc` of this :term:`Delivery Service`
:longDesc1:   The :ref:`ds-longdesc2` of this :term:`Delivery Service`
:longDesc2:   The :ref:`ds-longdesc3` of this :term:`Delivery Service`
:matchList:   The :term:`Delivery Service`'s :ref:`ds-matchlist`

	:pattern:   A regular expression - the use of this pattern is dependent on the ``type`` field (backslashes are escaped)
	:setNumber: An integer that provides explicit ordering of :ref:`ds-matchlist` items - this is used as a priority ranking by Traffic Router, and is not guaranteed to correspond to the ordering of items in the array.
	:type:      The type of match performed using ``pattern``.

:maxDnsAnswers:        The :ref:`ds-max-dns-answers` allowed for this :term:`Delivery Service`
:maxOriginConnections: The :ref:`ds-max-origin-connections`
:midHeaderRewrite:     A set of :ref:`ds-mid-header-rw-rules`
:missLat:              The :ref:`ds-geo-miss-default-latitude` used by this :term:`Delivery Service`
:missLong:             The :ref:`ds-geo-miss-default-longitude` used by this :term:`Delivery Service`
:multiSiteOrigin:      A boolean that defines the use of :ref:`ds-multi-site-origin` by this :term:`Delivery Service`
:orgServerFqdn:        The :ref:`ds-origin-url`
:originShield:         A :ref:`ds-origin-shield` string
:profileDescription:   The :ref:`profile-description` of the :ref:`ds-profile` with which this :term:`Delivery Service` is associated
:profileId:            The :ref:`profile-id` of the :ref:`ds-profile` with which this :term:`Delivery Service` is associated
:profileName:          The :ref:`profile-name` of the :ref:`ds-profile` with which this :term:`Delivery Service` is associated
:protocol:             An integral, unique identifier that corresponds to the :ref:`ds-protocol` used by this :term:`Delivery Service`
:qstringIgnore:        An integral, unique identifier that corresponds to the :ref:`ds-qstring-handling` setting on this :term:`Delivery Service`
:rangeRequestHandling: An integral, unique identifier that corresponds to the :ref:`ds-range-request-handling` setting on this :term:`Delivery Service`
:regexRemap:           A :ref:`ds-regex-remap`
:regionalGeoBlocking:  A boolean defining the :ref:`ds-regionalgeo` setting on this :term:`Delivery Service`
:remapText:            :ref:`ds-raw-remap`
:signed:               ``true`` if  and only if ``signingAlgorithm`` is not ``null``, ``false`` otherwise
:signingAlgorithm:     Either a :ref:`ds-signing-algorithm` or ``null`` to indicate URL/URI signing is not implemented on this :term:`Delivery Service`
:rangeSliceBlockSize: An integer that defines the byte block size for the ATS Slice Plugin. It can only and must be set if ``rangeRequestHandling`` is set to 3.
:sslKeyVersion:        This integer indicates the :ref:`ds-ssl-key-version`
:tenantId:             The integral, unique identifier of the :ref:`ds-tenant` who owns this :term:`Delivery Service`
:topology:             The unique name of the :term:`Topology` that this :term:`Delivery Service` is assigned to
:trRequestHeaders:     If defined, this defines the :ref:`ds-tr-req-headers` used by Traffic Router for this :term:`Delivery Service`
:trResponseHeaders:    If defined, this defines the :ref:`ds-tr-resp-headers` used by Traffic Router for this :term:`Delivery Service`
:type:                 The :ref:`ds-types` of this :term:`Delivery Service`
:typeId:               The integral, unique identifier of the :ref:`ds-types` of this :term:`Delivery Service`
:xmlId:                This :term:`Delivery Service`'s :ref:`ds-xmlid`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Permissions-Policy: interest-cohort=()
	Set-Cookie: mojolicious=...; Path=/; Expires=Wed, 19 May 2021 15:51:04 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 19 May 2021 14:51:04 GMT
	Content-Length: 839

	{ "response": [{
		"active": "ACTIVE",
		"anonymousBlockingEnabled": false,
		"ccrDnsTtl": null,
		"cdnId": 2,
		"cdnName": "CDN-in-a-Box",
		"checkPath": null,
		"consistentHashQueryParams": [
			"abc",
			"pdq",
			"xxx",
			"zyx"
		],
		"consistentHashRegex": null,
		"deepCachingType": "NEVER",
		"displayName": "Demo 1",
		"dnsBypassCname": null,
		"dnsBypassIp": null,
		"dnsBypassIp6": null,
		"dnsBypassTtl": null,
		"dscp": 0,
		"ecsEnabled": false,
		"edgeHeaderRewrite": null,
		"exampleURLs": [
			"http://video.demo1.mycdn.ciab.test",
			"https://video.demo1.mycdn.ciab.test"
		],
		"firstHeaderRewrite": null,
		"fqPacingRate": null,
		"geoLimit": 0,
		"geoLimitCountries": null,
		"geoLimitRedirectURL": null,
		"geoProvider": 0,
		"globalMaxMbps": null,
		"globalMaxTps": null,
		"httpBypassFqdn": null,
		"id": 2,
		"infoUrl": null,
		"initialDispersion": 1,
		"innerHeaderRewrite": null,
		"ipv6RoutingEnabled": true,
		"lastHeaderRewrite": null,
		"lastUpdated": "2021-05-19T14:49:04.279784Z",
		"logsEnabled": true,
		"longDesc": "Apachecon North America 2018",
		"longDesc1": null,
		"longDesc2": null,
		"matchList": [
			{
				"type": "HOST_REGEXP",
				"setNumber": 0,
				"pattern": ".*\\.demo1\\..*"
			}
		],
		"maxDnsAnswers": null,
		"maxOriginConnections": 0,
		"maxRequestHeaderBytes": 0,
		"midHeaderRewrite": null,
		"missLat": 42,
		"missLong": -88,
		"multiSiteOrigin": false,
		"originShield": null,
		"orgServerFqdn": "http://origin.infra.ciab.test",
		"profileDescription": null,
		"profileId": null,
		"profileName": null,
		"protocol": 2,
		"qstringIgnore": 0,
		"rangeRequestHandling": 0,
		"rangeSliceBlockSize": null,
		"regexRemap": null,
		"regionalGeoBlocking": false,
		"remapText": null,
		"routingName": "video",
		"serviceCategory": null,
		"signed": false,
		"signingAlgorithm": null,
		"sslKeyVersion": 1,
		"tenant": "root",
		"tenantId": 1,
		"topology": null,
		"trResponseHeaders": null,
		"trRequestHeaders": null,
		"type": "HTTP",
		"typeId": 1,
		"xmlId": "demo1"
	}]}


``POST``
========
Assign an arbitrary number of :term:`Delivery Services` to a single server.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+----------+---------------------------------------------------------------------------------------------+
	| Name | Required | Description                                                                                 |
	+------+----------+---------------------------------------------------------------------------------------------+
	| ID   | Yes      | The integral, unique identifier of the server that you want to assign delivery services to. |
	+------+----------+---------------------------------------------------------------------------------------------+

.. table:: Request Query Parameters

	+---------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------------------------+
	| Name    | Required | Description                                                                                                                                                           |
	+---------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------------------------+
	| replace | Yes      | Whether the list of :term:`Delivery Services` you provide should replace the existing list or be merged with the existing list. Must be a 1, or true, or 0, or false. |
	+---------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------------------------+

The request body is an array of IDs of :term:`Delivery Services` that you want to assign to the server. The array can be empty, but it must be provided.

.. code-block:: http
	:caption: Request Example

	POST /api/4.0/servers/6/deliveryservices?replace=1 HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 3

	[
		1
	]

Response Structure
------------------
:dsIds:         An array of integral, unique identifiers for :term:`Delivery Services` which the request added to server. If ``:replace:`` is ``false``, :term:`Delivery Services` that are already assigned will remain, though they are not listed by ``:dsIds:``.
:replace:       The ``replace`` value you provided in the body of the request, or ``null`` if none was provided.
:serverId:      The server's integral, unique identifier

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Tue, 25 Feb 2020 09:08:32 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: iV+JzAZSsmlxRZsNtIRg3oA9470hAwrMpq5xhcYVi0Y831Trx2YRlsyhYpOPqHg5+QPoXHGF0nx8uso0fuNarw==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 25 Feb 2020 08:08:32 GMT
	Content-Length: 129

	{
		"alerts": [
			{
				"text": "successfully assigned dses to server",
				"level": "success"
			}
		],
		"response": {
			"serverId": 6,
			"dsIds": [
				1
			],
			"replace": true
		}
	}


.. [#tenancy] Only the :term:`Delivery Services` visible to the requesting user's :term:`Tenant` will appear, regardless of their :term:`Role` or the :term:`Delivery Services`' actual 'server assignment' status.
