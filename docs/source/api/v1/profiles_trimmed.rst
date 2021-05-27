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

.. _to-api-v1-profiles-trimmed:

********************
``profiles/trimmed``
********************
.. deprecated:: ATCv4
	Use the ``GET`` method of :ref:`to-api-v1-profiles` instead.

``GET``
=======
:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
No parameters available

Response Structure
------------------
:name: The :ref:`profile-name` of the :term:`Profile`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: 1XiReeWNZfrLjdordj5RpZJxJS1eAJ8v3rOulOnmBXEfTe+Sn3cKx3Pa0Rch4TII4ck/93sI+5L1V1m6MvTCaQ==
	X-Server-Name: traffic_ops_golang/
	Date: Fri, 07 Dec 2018 20:51:28 GMT
	Content-Length: 360

	{ "response": [
		{ "name": "GLOBAL" },
		{ "name": "TRAFFIC_ANALYTICS" },
		{ "name": "TRAFFIC_OPS" },
		{ "name": "TRAFFIC_OPS_DB" },
		{ "name": "TRAFFIC_PORTAL" },
		{ "name": "TRAFFIC_STATS" },
		{ "name": "INFLUXDB" },
		{ "name": "RIAK_ALL" },
		{ "name": "ATS_EDGE_TIER_CACHE" },
		{ "name": "ATS_MID_TIER_CACHE" },
		{ "name": "BIND_ALL" },
		{ "name": "CCR_CIAB" },
		{ "name": "ENROLLER_ALL" },
		{ "name": "RASCAL-Traffic_Monitor" }
	],
	"alerts": [
		{
			"text": "This endpoint is deprecated, please use GET /profiles instead",
			"level": "warning"
		}
	]}
