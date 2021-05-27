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

.. _to-api-v1-profiles-id-unassigned_parameters:

*****************************************
``profiles/{{ID}}/unassigned_parameters``
*****************************************
.. warning:: There are **very** few good reasons to use this endpoint - be sure not limit said use.

.. deprecated:: ATCv4

``GET``
=======
Retrieves all :term:`Parameters` *not* assigned to the specified :term:`Profile`.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-----------------------------------------------------------------------------------------------------+
	| Name | Description                                                                                         |
	+======+=====================================================================================================+
	|  ID  | The :ref:`profile-id` of the :term:`Profile` for which unassigned :term:`Parameters` will be listed |
	+------+-----------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/profiles/9/unassigned_parameters HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:configFile:  The :term:`Parameter`'s :ref:`parameter-config-file`
:id:          The :term:`Parameter`'s :ref:`parameter-id`
:lastUpdated: The date and time at which this :term:`Parameter` was last updated, in an ISO-like format
:name:        :ref:`parameter-name` of the :term:`Parameter`
:profiles:    An array of :term:`Profile` :ref:`Names <profile-name>` that use this :term:`Parameter`
:secure:      A boolean value that describes whether or not the :term:`Parameter` is :ref:`parameter-secure`
:value:       The :term:`Parameter`'s :ref:`parameter-value`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: iO7YHU+0spCPSaR6oDrVIQwxSS1GoSyi8K6ng4eemuxqOxB9FdfPgBpXN8w+xmxf2ZwRMLXHv5S6cfIoNNDnqw==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 05 Dec 2018 21:37:50 GMT
	Transfer-Encoding: chunked

	{ "alerts": [{
			"level": "warning",
			"text": "This endpoint is deprecated, and will be removed in the future"
		}],
		"response": [
		{
			"configFile": "parent.config",
			"id": 1,
			"lastUpdated": "2018-12-05 17:50:47+00",
			"name": "mso.parent_retry",
			"secure": false,
			"value": "simple_retry"
		},
		{
			"configFile": "parent.config",
			"id": 2,
			"lastUpdated": "2018-12-05 17:50:47+00",
			"name": "mso.parent_retry",
			"secure": false,
			"value": "unavailable_server_retry"
		}]
	}

.. note:: The response example for this endpoint has been truncated to only the first two elements of the resulting array, as the output was hundreds of lines long.
