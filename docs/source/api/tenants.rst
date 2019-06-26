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

.. _to-api-tenants:

***********
``tenants``
***********

``GET``
=======
Get all requested tenants.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+--------+------------------------------------------------------------------------------------+
	|  Name  | Description                                                                        |
	+========+====================================================================================+
	| active | If ``true``, return only active tenants; if ``false`` return only inactive tenants |
	+--------+------------------------------------------------------------------------------------+
	|   id   | Return only tenants with this integral, unique identifier                          |
	+--------+------------------------------------------------------------------------------------+
	|  name  | Return only tenants with this name                                                 |
	+--------+------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/tenants?name=root HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:active:      A boolean which indicates whether or not the tenant is active
:id:          The integral, unique identifier of this tenant
:name:        This tenant's name
:parentId:    The integral, unique identifier of this tenant's parent
:parentName:  The name of the parent of this tenant

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: Yzr6TfhxgpZ3pbbrr4TRG4wC3PlnHDDzgs2igtz/1ppLSy2MzugqaGW4y5yzwzl5T3+7q6HWej7GQZt1XIVeZQ==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 11 Dec 2018 19:57:58 GMT
	Content-Length: 106

	{ "response": [
		{
			"id": 1,
			"name": "root",
			"active": true,
			"lastUpdated": "2018-12-10 19:11:17+00",
			"parentId": null
		}
	]}

``POST``
========
.. warning:: Inactive tenants cannot be deleted or modified in any way, so be careful whenever setting ``active`` to ``false``. This is a known bug in Traffic Ops, and is tracked by `GitHub Issue #2732 <https://github.com/apache/trafficcontrol/issues/2732>`_.

Create a new tenant.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:active:   An optional boolean - default: ``false`` - which indicates whether or not the tenant shall be immediately active
:name:     The name of the tenant
:parentId: The integral, unique identifier of the parent of this tenant

.. code-block:: http
	:caption: Request Example

	POST /api/1.4/tenants HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 48
	Content-Type: application/json

	{
		"active": true,
		"name": "test",
		"parentId": 1,
		"parentName": "root"
	}

Response Structure
------------------
:active:      A boolean which indicates whether or not the tenant is active
:id:          The integral, unique identifier of this tenant
:name:        This tenant's name
:parentId:    The integral, unique identifier of this tenant's parent
:parentName:  The name of the parent of this tenant

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: ysdopC//JQI79BRUa61s6M2HzHxYHpo5RdcuauOoqCYxiVOoUhNZfOVydVkv8zDN2qA374XKnym4kWj3VzQIXg==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 11 Dec 2018 19:37:16 GMT
	Content-Length: 162

	{ "alerts": [
		{
			"text": "tenant was created.",
			"level": "success"
		}
	],
	"response": {
		"id": 9,
		"name": "test",
		"active": true,
		"lastUpdated": "2018-12-11 19:37:16+00",
		"parentId": 1,
		"parentName": "root"
	}}
