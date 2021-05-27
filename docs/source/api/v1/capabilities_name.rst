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

.. _to-api-v1-capabilities-name:

*************************
``capabilities/{{name}}``
*************************
.. deprecated:: ATCv4
	As of ATC version 4.0, every method of this endpoint is deprecated. See each method for details.

``GET``
=======
.. deprecated:: ATCv4
	This method of this endpoint is deprecated, please use the 'name' parameter of a ``GET`` request to :ref:`to-api-v1-capabilities` instead.

Get a capability by name.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+----------------------------------------+
	| Name |          Description                   |
	+======+========================================+
	| name | The name of the capability of interest |
	+------+----------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.5/capabilities/testquest HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
:description: Describes the APIs covered by the capability
:lastUpdated: Date and time of the last update made to this capability, in an ISO-like format
:name:        Name of the capability

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Encoding: gzip
	Content-Length: 221
	Content-Type: application/json
	Date: Wed, 29 Jan 2020 20:49:49 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; expires=Thu, 30 Jan 2020 00:49:49 GMT; path=/; HttpOnly
	Vary: Accept-Encoding

	{ "alerts": [
		{
			"level": "warning",
			"text": "This endpoint is deprecated, please use 'GET /capabilities with the 'name' query parameter' instead"
		}
	],
	"response": [
		{
			"lastUpdated": "2020-01-29 20:10:53.821978+00",
			"name": "testquest",
			"description": "A test capability for API examples"
		}
	]}


``PUT``
=======
.. deprecated:: ATCv4
	This method of this endpoint is deprecated. In the future, Capabilities will be immutable, and so no alternative is offered.

Edit a capability.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-------------------------------------------+
	| Name |          Description                      |
	+======+===========================================+
	| name | The name of the capability to be modified |
	+------+-------------------------------------------+

:description: Describes the APIs covered by the capability

.. code-block:: http
	:caption: Request Example

	PUT /api/1.5/capabilities/testquest HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 36

	{"description": "A new description"}

Response Structure
------------------
:description: Describes the APIs covered by the capability.
:lastUpdated: Date and time of the last update made to this capability, in an ISO-like format
:name:        The name of the capability

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Encoding: gzip
	Content-Length: 224
	Content-Type: application/json
	Date: Wed, 29 Jan 2020 21:25:10 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; expires=Thu, 30 Jan 2020 01:25:10 GMT; path=/; HttpOnly
	Vary: Accept-Encoding

	{ "alerts": [
		{
			"level": "success",
			"text": "Capability was updated."
		},
		{
			"level": "warning",
			"text": "This endpoint and its functionality is deprecated, and will be removed in the future"
		}
	],
	"response": {
		"lastUpdated": "2020-01-29 21:24:56.361518+00",
		"name": "testquest",
		"description": "A new description"
	}}


``DELETE``
==========
.. deprecated:: ATCv4
	This method of this endpoint is deprecated. In the future, Capabilities will be immutable, and so no alternative is offered.

Delete a capability.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters**

	+-----------------+----------+------------------------------------------------+
	| Name            | Required | Description                                    |
	+=================+==========+================================================+
	| ``name``        | yes      | Capability name.                               |
	+-----------------+----------+------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/1.5/capabilities/testquest HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 0


Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Encoding: gzip
	Content-Length: 146
	Content-Type: application/json
	Date: Wed, 29 Jan 2020 21:27:57 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; expires=Thu, 30 Jan 2020 01:27:57 GMT; path=/; HttpOnly
	Vary: Accept-Encoding

	{ "alerts": [
		{
			"level": "success",
			"text": "Capability deleted."
		},
		{
			"level": "warning",
			"text": "This endpoint and its functionality is deprecated, and will be removed in the future"
		}
	]}
