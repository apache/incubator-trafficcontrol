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

.. _to-api-cdns-id-queue_update:

****************************
``cdns/{{ID}}/queue_update``
****************************

``POST``
========
:term:`Queue` or "dequeue" updates for all servers assigned to a specific CDN.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+---------------------------------------------------------------------------+
	| Name | Description                                                               |
	+======+===========================================================================+
	| ID   | The integral, unique identifier for the CDN on which to (de)queue updates |
	+------+---------------------------------------------------------------------------+

.. table:: Request Query Parameters

	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Required | Description                                                                                                   |
	+===========+==========+===============================================================================================================+
	| type      | no       | The name of the ``type`` of servers, for which the updates need to be queued or dequeued.                     |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| profile   | no       | The name of the ``profile`` of servers, for which the updates need to be queued or dequeued.                  |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+

:action: One of "queue" or "dequeue" as appropriate

.. code-block:: http
	:caption: Request Example

	POST /api/4.0/cdns/2/queue_update?type=EDGE HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 19
	Content-Type: application/json

	{"action": "queue"}

Response Structure
------------------
:action: The action processed, either ``"queue"`` or ``"dequeue"``
:cdnId:  The integral, unique identifier for the CDN on which :term:`Queue Updates` was performed or cleared

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: rBpFfrrP+9IFkwsRloEM+v+I8MuBZDXqFu+WUTGtRGypnAn2gHooPoNQRyVvJGjyIQrLXLvqjEtve+lH2Tj4uw==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 14 Nov 2018 21:02:07 GMT
	Content-Length: 54

	{ "response": {
		"action": "queue",
		"cdnId": 2,
		"typeID": 11
	}}
