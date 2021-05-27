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

.. _to-api-v2-deliveryservices-id-safe:

********************************
``deliveryservices/{{ID}}/safe``
********************************

``PUT``
=======
Allows a user to edit metadata fields of a :term:`Delivery Service`.

:Auth. Required: Yes
:Roles Required: None\ [#tenancy]_
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------------------------+
	| Name | Description                                                                    |
	+======+================================================================================+
	|  ID  | The integral, unique identifier of the :term:`Delivery Service` being modified |
	+------+--------------------------------------------------------------------------------+

:displayName: A string that is the :ref:`ds-display-name`
:infoUrl:     An optional\ [#optional]_ string containing the :ref:`ds-info-url`
:longDesc:    An optional\ [#optional]_ string containing the :ref:`ds-longdesc` of this :term:`Delivery Service`
:longDesc1:   An optional\ [#optional]_ string containing the :ref:`ds-longdesc2` of this :term:`Delivery Service`

.. code-block:: http
	:caption: Request Example

	PUT /api/2.0/deliveryservices/1/safe HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 132

	{
		"displayName": "test",
		"infoUrl": "this is not even a real URL",
		"longDesc": "longDesc1 is implicitly set to null in this example
	}

Response Structure
------------------
:active:                   A boolean that defines :ref:`ds-active`. ``true`` means that the Delivery Service is ACTIVE, while ``false`` means that it is either PRIMED or INACTIVE (API versions earlier than 4 are incapable of distinguishing between the two).
:anonymousBlockingEnabled: A boolean that defines :ref:`ds-anonymous-blocking`
:cacheurl:                 A :ref:`ds-cacheurl`

	.. deprecated:: ATCv3.0
		This field has been deprecated in Traffic Control 3.x and is subject to removal in Traffic Control 4.x or later

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
:ipv6RoutingEnabled:        A boolean that defines the :ref:`ds-ipv6-routing` setting on this :term:`Delivery Service`
:lastUpdated:               The date and time at which this :term:`Delivery Service` was last updated, in an ISO-like format
:logsEnabled:               A boolean that defines the :ref:`ds-logs-enabled` setting on this :term:`Delivery Service`
:longDesc:                  The :ref:`ds-longdesc` of this :term:`Delivery Service`
:longDesc1:                 The :ref:`ds-longdesc2` of this :term:`Delivery Service`
:longDesc2:                 The :ref:`ds-longdesc3` of this :term:`Delivery Service`
:matchList:                 The :term:`Delivery Service`'s :ref:`ds-matchlist`

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
:trRequestHeaders:     If defined, this defines the :ref:`ds-tr-req-headers` used by Traffic Router for this :term:`Delivery Service`
:trResponseHeaders:    If defined, this defines the :ref:`ds-tr-resp-headers` used by Traffic Router for this :term:`Delivery Service`
:type:                 The :ref:`ds-types` of this :term:`Delivery Service`
:typeId:               The integral, unique identifier of the :ref:`ds-types` of this :term:`Delivery Service`
:xmlId:                This :term:`Delivery Service`'s :ref:`ds-xmlid`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 10 Feb 2020 16:33:03 GMT; Max-Age=3600; HttpOnly
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 10 Feb 2020 15:33:03 GMT
	Content-Length: 853

	{ "alerts": [
		{
			"text": "Delivery Service safe update successful.",
			"level": "success"
		}
	],
	"response": [
		{
			"active": true,
			"anonymousBlockingEnabled": false,
			"cacheurl": null,
			"ccrDnsTtl": null,
			"cdnId": 2,
			"cdnName": "CDN-in-a-Box",
			"checkPath": null,
			"displayName": "test",
			"dnsBypassCname": null,
			"dnsBypassIp": null,
			"dnsBypassIp6": null,
			"dnsBypassTtl": null,
			"dscp": 0,
			"edgeHeaderRewrite": null,
			"geoLimit": 0,
			"geoLimitCountries": null,
			"geoLimitRedirectURL": null,
			"geoProvider": 0,
			"globalMaxMbps": null,
			"globalMaxTps": null,
			"httpBypassFqdn": null,
			"id": 1,
			"infoUrl": "this is not even a real URL",
			"initialDispersion": 1,
			"ipv6RoutingEnabled": true,
			"lastUpdated": "2020-02-10 15:33:03+00",
			"logsEnabled": true,
			"longDesc": "longDesc1 is implicitly set to null in this example",
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
			"regexRemap": null,
			"regionalGeoBlocking": false,
			"remapText": null,
			"routingName": "video",
			"signed": false,
			"sslKeyVersion": 1,
			"tenantId": 1,
			"type": "HTTP",
			"typeId": 1,
			"xmlId": "demo1",
			"exampleURLs": [
				"http://video.demo1.mycdn.ciab.test",
				"https://video.demo1.mycdn.ciab.test"
			],
			"deepCachingType": "NEVER",
			"fqPacingRate": null,
			"signingAlgorithm": null,
			"tenant": "root",
			"trResponseHeaders": null,
			"trRequestHeaders": null,
			"consistentHashRegex": null,
			"consistentHashQueryParams": [
				"abc",
				"pdq",
				"xxx",
				"zyx"
			],
			"maxOriginConnections": 0,
			"ecsEnabled": false,
			"rangeSliceBlockSize": null
		}
	]}

.. [#tenancy] Only those :term:`Delivery Services` assigned to :term:`Tenants` that are the requesting user's :term:`Tenant` or children thereof may be modified with this endpoint.
.. [#optional] If these fields are not present in the request body they are *implicitly set to* ``null``.
