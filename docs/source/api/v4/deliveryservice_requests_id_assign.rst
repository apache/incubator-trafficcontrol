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

.. _to-api-deliveryservice_requests-id-assign:

******************************************
``deliveryservice_requests/{{ID}}/assign``
******************************************
Assign a :term:`Delivery Service Request` to a user.

``GET``
=======
.. versionadded:: 4.0

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object (string)

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-----------------------------------------------------------------------------------------------------------------+
	| Name | Description                                                                                                     |
	+======+=================================================================================================================+
	|  ID  | The integral, unique identifier of the :term:`Delivery Service Request` for which assignment is being retrieved |
	+------+-----------------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/deliveryservice_requests/1/assign HTTP/1.1
	User-Agent: python-requests/2.24.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
The response is the username of the user to whom the :term:`Delivery Service Request` is assigned - or ``null`` if it is unassigned.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Tue, 02 Feb 2021 22:48:48 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 02 Feb 2021 21:48:48 GMT
	Content-Length: 45

	{ "response": "admin" }


``PUT``
=======
:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+----------------------------------------------------------------------------------------+
	| Name | Description                                                                            |
	+======+========================================================================================+
	|  ID  | The integral, unique identifier of the :term:`Delivery Service Request` being assigned |
	+------+----------------------------------------------------------------------------------------+

:assignee: The username of the user to whom the :term:`Delivery Service Request` is assigned

	.. versionadded:: 4.0

:assigneeId: The integral, unique identifier of the user to whom the :term:`Delivery Service Request` is assigned

	.. versionchanged:: 4.0
		Prior to APIv4.0, this was the only property that could be used to change a :term:`Delivery Service Request`'s Assignee - and thus was a required field.

It is not required to send both of these; either property is sufficient to determine an :ref:`dsr-assignee`. In most cases, it's easier to use just `assignee`. If both *are* given, then `assigneeId` will take precedence in the event that the two properties do not refer to the same user. Sending a request that sets the assignee to ``null`` un-assigns the :term:`DSR` from any assignees it previously had\ [#implicit-null]_.

.. code-block:: http
	:caption: Request Example

	PUT /api/4.0/deliveryservice_requests/1/assign HTTP/1.1
	User-Agent: python-requests/2.24.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 20

	{"assignee": "admin"}

Response Structure
------------------
The response contains a full representation of the newly assigned :term:`Delivery Service Request`.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Sun, 23 Feb 2020 14:45:51 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: h7uBZHLQtRYbOSOR5AtQQrZ4uMeEWivWNT74fCf6WtLbAMwGpRrMjNmBYKduv48DEnRqG6WVM/4nBu3AkCUqPw==
	X-Server-Name: traffic_ops_golang/
	Date: Sun, 23 Feb 2020 13:45:51 GMT
	Content-Length: 931

	{ "alerts": [{
		"text": "Changed assignee of 'demo1' Delivery Service Request to 'admin'",
		"level": "success"
	}],
	"response": {
		"assignee": "admin",
		"author": "admin",
		"changeType": "update",
		"createdAt": "2020-09-25T06:52:23.758877Z",
		"id": 6,
		"lastEditedBy": "admin",
		"lastUpdated": "2020-09-25T07:01:24.600029Z",
		"original": {
			"active": "ACTIVE",
			"anonymousBlockingEnabled": false,
			"cacheurl": null,
			"ccrDnsTtl": null,
			"cdnId": 2,
			"cdnName": "CDN-in-a-Box",
			"checkPath": null,
			"displayName": "Demo 1",
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
			"infoUrl": null,
			"initialDispersion": 1,
			"ipv6RoutingEnabled": true,
			"lastUpdated": "2020-09-25T07:13:08.753352Z",
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
			"rangeSliceBlockSize": null,
			"topology": "demo1-top",
			"firstHeaderRewrite": null,
			"innerHeaderRewrite": null,
			"lastHeaderRewrite": null,
			"serviceCategory": null
		},
		"requested": {
			"active": "ACTIVE",
			"anonymousBlockingEnabled": false,
			"cacheurl": null,
			"ccrDnsTtl": 30,
			"cdnId": 2,
			"cdnName": null,
			"checkPath": null,
			"displayName": "Demo 1 but modified by a DSR",
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
			"infoUrl": null,
			"initialDispersion": 3,
			"ipv6RoutingEnabled": null,
			"lastUpdated": null,
			"logsEnabled": false,
			"longDesc": "long desc",
			"longDesc1": null,
			"longDesc2": null,
			"matchList": null,
			"maxDnsAnswers": null,
			"midHeaderRewrite": null,
			"missLat": null,
			"missLong": null,
			"multiSiteOrigin": null,
			"originShield": null,
			"orgServerFqdn": null,
			"profileDescription": null,
			"profileId": null,
			"profileName": null,
			"protocol": null,
			"qstringIgnore": null,
			"rangeRequestHandling": null,
			"regexRemap": null,
			"regionalGeoBlocking": false,
			"remapText": null,
			"routingName": "cdn",
			"signed": false,
			"sslKeyVersion": null,
			"tenantId": 1,
			"type": null,
			"typeId": 8,
			"xmlId": "demo1",
			"exampleURLs": null,
			"deepCachingType": "NEVER",
			"fqPacingRate": null,
			"signingAlgorithm": null,
			"tenant": null,
			"trResponseHeaders": null,
			"trRequestHeaders": null,
			"consistentHashRegex": null,
			"consistentHashQueryParams": null,
			"maxOriginConnections": 0,
			"ecsEnabled": false,
			"rangeSliceBlockSize": null,
			"topology": null,
			"firstHeaderRewrite": null,
			"innerHeaderRewrite": null,
			"lastHeaderRewrite": null,
			"serviceCategory": null
		},
		"status": "draft"
	}}

.. [#implicit-null] Because of how the Traffic Ops API parses requests, there is no distinction between ``null`` and ``undefined``/missing properties. This means that sending the request payload ``{}`` in this case will result in the :term:`DSR` being unassigned.
