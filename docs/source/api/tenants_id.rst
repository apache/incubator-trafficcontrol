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

.. _to-api-tenants-id:

******************
``tenants/{{ID}}``
******************

``GET``
=======
.. deprecated:: 1.1
	Use the  ``id`` query parameter of a ``GET`` request to :ref:`to-api-tenants` instead.

Get a specific tenant.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+----------------------------------------------------------------+
	| Name |                 Description                                    |
	+======+================================================================+
	|  ID  | The integral, unique identifier for the tenant being inspected |
	+------+----------------------------------------------------------------+

.. code-block:: http
	:caption: Request Structure

	GET /api/1.4/tenants/1 HTTP/1.1
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
:parentName:  The name of the parent tenant

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
	Date: Tue, 11 Dec 2018 20:00:28 GMT
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

``PUT``
=======
.. warning:: Inactive tenants cannot be deleted or modified in any way by this request method, so be careful whenever setting ``active`` to ``false``. This is a known bug in Traffic Ops, and is tracked by `GitHub Issue #2732 <https://github.com/apache/trafficcontrol/issues/2732>`_.

Updates a specific tenant.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+---------------------------------------------------------------+
	| Name |                 Description                                   |
	+======+===============================================================+
	|  ID  | The integral, unique identifier for the tenant being modified |
	+------+---------------------------------------------------------------+

:active:   An optional boolean - default: ``false`` - which indicates whether or not the tenant shall be immediately active
:name:     The name of the tenant
:parentId: The integral, unique identifier of the parent of this tenant

.. code-block:: http
	:caption: Request Example

	PUT /api/1.4/tenants/9 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 59
	Content-Type: application/json

	{
		"active": true,
		"name": "quest",
		"parentId": 3
	}

Response Structure
------------------
:active:      A boolean which indicates whether or not the tenant is active
:id:          The integral, unique identifier of this tenant
:name:        This tenant's name
:parentId:    The integral, unique identifier of this tenant's parent
:parentName:  The name of the parent tenant

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: 5soYQFrG2x5ZJ1e5UZIOLUv/928qyd2Bfgw21Wv85rqjLpyeT3djkfRVD1/xpKConulNrZs2czJKrrwZA7X61w==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 11 Dec 2018 20:30:54 GMT
	Content-Length: 163

	{ "alerts": [
		{
			"text": "tenant was updated.",
			"level": "success"
		}
	],
	"response": {
		"id": 9,
		"name": "quest",
		"active": true,
		"lastUpdated": "2018-12-11 20:30:54+00",
		"parentId": 3,
		"parentName": "tenant-01"
	}}

``DELETE``
==========
.. warning:: Inactive tenants cannot be deleted or modified in any way by this request method, so be careful whenever setting ``active`` to ``false``. This is a known bug in Traffic Ops, and is tracked by `GitHub Issue #2732 <https://github.com/apache/trafficcontrol/issues/2732>`_.

Deletes a specific tenant.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``


Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------+
	| Name |                 Description                                  |
	+======+==============================================================+
	|  ID  | The integral, unique identifier for the tenant being deleted |
	+------+--------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/1.4/tenants/9 HTTP/1.1
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
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: KU0XIbFoD0Cy06kzH2Gl59pBqie/TEFJgh33mssGNwXJZlRkTLaSTHT8Df4X+pOs7UauZH10akGvaA0UTiN/vg==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 11 Dec 2018 20:40:31 GMT
	Content-Length: 61

	{ "alerts": [
		{
			"text": "tenant was deleted.",
			"level": "success"
		}
	]}
