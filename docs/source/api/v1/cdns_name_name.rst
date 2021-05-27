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

.. _to-api-v1-cdns-name-name:

**********************
``cdns/name/{{name}}``
**********************

``GET``
=======
.. deprecated:: ATCv4
	Use the ``GET`` method of :ref:`to-api-v1-cdns` with the query parameter ``name`` instead.

Extract information about a CDN, identified by name.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+---------------------------------------------+
	| Name |                Description                  |
	+======+=============================================+
	| name | The name of the CDN to be inspected         |
	+------+---------------------------------------------+

Response Structure
------------------
:dnssecEnabled: ``true`` if DNSSEC is enabled on this CDN, otherwise ``false``
:domainName:    Top Level Domain name within which this CDN operates
:id:            The integral, unique identifier for the CDN
:lastUpdated:   Date and time when the CDN was last modified in an ISO-like format
:name:          The name of the CDN

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: bTz86xdnGfbKhxnneb4geXohaw3lhG+h5wc21/ncHFATwp1h80h+txxySCIVfa0hgBrJHEdpGZQsH5w5IknsrQ==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 14 Nov 2018 21:22:16 GMT
	Content-Length: 137

	{ "response": [
		{
			"dnssecEnabled": false,
			"domainName": "mycdn.ciab.test",
			"id": 2,
			"lastUpdated": "2018-11-14 18:21:14+00",
			"name": "CDN-in-a-Box"
		}
	],
	"alerts": [
		{
			"text": "This endpoint is deprecated, please use GET /cdns with query parameter name instead",
			"level": "warning"
		}
	]}

``DELETE``
==========
Allows a user to delete a CDN by name

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+---------------------------------------------+
	| Name |                Description                  |
	+======+=============================================+
	| name | The name of the CDN to be deleted           |
	+------+---------------------------------------------+

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
	Whole-Content-Sha512: Zy4cJN6BEct4ltFLN4e296mM8XnzOs0EQ3/jp4TA3L+g8qtkI0WrL+ThcFq4xbJPU+KHVDSi+b0JBav3xsYPqQ==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 14 Nov 2018 20:59:22 GMT
	Content-Length: 58

	{ "alerts": [
		{
			"text": "cdn was deleted.",
			"level": "success"
		}
	]}
