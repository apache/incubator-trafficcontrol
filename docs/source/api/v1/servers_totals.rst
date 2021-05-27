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

.. _to-api-v1-servers-totals:

******************
``servers/totals``
******************
.. deprecated:: 1.1

``GET``
=======
Retrieves a count of each :term:`Type` of server across all CDNs.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
No parameters available.

Response Structure
------------------
:count: The number of servers of this type configured in this instance of Traffic Ops
:type:  The name of the :term:`Type` servers herein counted

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Mon, 10 Dec 2018 17:02:02 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: J4wy8zf+LX44/qWIbvziWHCcDZpUJ9GOpOVUVqPbVHUCh1V19o8FnE7T+V0639n9Xyw9k10NcaGIqASA+O9Rzg==
	Content-Length: 305

	{ "alerts": [
		{
			"level": "warning",
			"text": "This endpoint is deprecated"
		}],
		"response": [
		{
			"count": 1,
			"type": "EDGE"
		},
		{
			"count": 1,
			"type": "MID"
		},
		{
			"count": 1,
			"type": "CCR"
		},
		{
			"count": 1,
			"type": "RASCAL"
		},
		{
			"count": 1,
			"type": "RIAK"
		},
		{
			"count": 2,
			"type": "TRAFFIC_OPS"
		},
		{
			"count": 1,
			"type": "TRAFFIC_OPS_DB"
		},
		{
			"count": 1,
			"type": "TRAFFIC_PORTAL"
		},
		{
			"count": 1,
			"type": "BIND"
		},
		{
			"count": 1,
			"type": "ENROLLER"
		}
	]}
