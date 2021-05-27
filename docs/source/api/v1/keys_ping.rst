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
.. _to-api-keys-ping:

*************
``keys/ping``
*************
Checks whether :ref:`tv-overview` is online.

.. deprecated:: ATCv4

``GET``
=======
:Auth. Required: Yes
:Roles Required: None
:Response Type:  Object

Request Structure
-----------------
No parameters available.

.. code-block:: http
	:caption: Request Example

	GET /api/1.1/keys/ping HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
:server: The hostname and port of :ref:`tv-overview`.
:status: The `reason phrase <https://www.w3.org/Protocols/rfc2616/rfc2616-sec6.html#sec6.1.1>`_ of the response that :ref:`to-overview` received from :ref:`tv-overview`.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 24 Feb 2020 21:09:31 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: Y/Br43Y5SXXBIneAgHANBXDP0hqO4Lkguk0vmuTU7xktZq3EldK5SX9OkEm9gzRkPKjQVUy0hhldsq6Ax46k7A==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 24 Feb 2020 20:09:31 GMT
	Content-Length: 166

	{ "alerts": [{
			"level": "warning",
			"text": "This endpoint is deprecated, please use /vault/ping instead"
		}],
		"response": {
			"status": "OK",
			"server": "trafficvault.infra.ciab.test:8087"
		}
	}
