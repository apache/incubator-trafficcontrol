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
.. _to-api-v2-logs:

********
``logs``
********

.. note:: This endpoint's responses will contain a cookie (``last_seen_log``) that is used by :ref:`to-api-logs-newcount` to determine the time of last access. Be sure your client uses cookies properly if you intend to use :ref:`to-api-logs-newcount` in concert with this endpoint!

``GET``
=======
Fetches a list of changes that have been made to the Traffic Control system

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+-------+----------+------------------------------------------------------+
	| Name  | Required | Description                                          |
	+=======+==========+======================================================+
	| days  | no       | An integer number of days of change logs to return   |
	+-------+----------+------------------------------------------------------+
	| limit | no       | The number of records to which to limit the response |
	+-------+----------+------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/2.0/logs?days=1&limit=2 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:id:          Integral, unique identifier for the Log entry
:lastUpdated: Date and time at which the change was made, in an ISO-like format
:level:       Log categories for each entry, e.g. 'UICHANGE', 'OPER', 'APICHANGE'
:message:     Log detail about what occurred
:ticketNum:   Optional field to cross reference with any bug tracking systems
:user:        Name of the user who made the change

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Thu, 15 Nov 2018 15:11:38 GMT
	X-Server-Name: traffic_ops_golang/
	Set-Cookie: last_seen_log="2018-11-15% 15:11:38"; path=/; Max-Age=604800
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: 40dV+azaZ3b6F30y6YHVbV3H2a3ekZrdoxICupwaxQnj62pwYfb7YCM7Qhe3OAItmB77Tbg9INy27ymaz3hr9A==
	Content-Length: 357

	{ "response": [
		{
			"ticketNum": null,
			"level": "APICHANGE",
			"lastUpdated": "2018-11-14 21:40:06.493975+00",
			"user": "admin",
			"id": 444,
			"message": "User [ test ] unlinked from deliveryservice [ 1 | demo1 ]."
		},
		{
			"ticketNum": null,
			"level": "APICHANGE",
			"lastUpdated": "2018-11-14 21:37:30.707571+00",
			"user": "admin",
			"id": 443,
			"message": "1 delivery services were assigned to test"
		}
	]}
