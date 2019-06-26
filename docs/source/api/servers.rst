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

.. _to-api-servers:

***********
``servers``
***********

``GET``
=======
Retrieves properties of all servers across all CDNs.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|    Name    | Required |                                                Description                                                        |
	+============+==========+===================================================================================================================+
	| cachegroup | no       | Return only those servers within the :term:`Cache Group` identified by this integral, unique identifier           |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|    dsId    | no       | Return only those servers assigned to the :term:`Delivery Service` identified by this integral, unique identifier |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|  hostName  | no       | Return only those servers that have this (short) hostname                                                         |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|     id     | no       | Return only the server with this integral, unique identifier                                                      |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|  profileId | no       | Return only those servers that are using the profile identified by this integral, unique identifier               |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|   status   | no       | Return only those servers with this status - see :ref:`health-proto`                                              |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+
	|    type    | no       | Return only servers of this 'type'                                                                                |
	+------------+----------+-------------------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/servers?hostName=mid HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:cachegroup:     The name of the Cache Group to which this server belongs
:cachegroupId:   The integral, unique identifier of the Cache Group to which this server belongs
:cdnId:          The integral, unique identifier of the CDN to which the server belongs
:cdnName:        Name of the CDN to which the server belongs
:domainName:     The domain part of the server's Fully Qualified Domain Name (FQDN)
:guid:           An identifier used to uniquely identify the server

	.. deprecated:: 1.1
		This is a legacy key which only still exists for compatibility reasons - it should always be ``null``

:hostName:       The (short) hostname of the server
:httpsPort:      The port on which the server listens for incoming HTTPS connections/requests
:id:             An integral, unique identifier for this server
:iloIpAddress:   The IPv4 address of the server's Integrated Lights-Out (ILO) service\ [1]_
:iloIpGateway:   The IPv4 gateway address of the server's ILO service\ [1]_
:iloIpNetmask:   The IPv4 subnet mask of the server's ILO service\ [1]_
:iloPassword:    The password of the of the server's ILO service user\ [1]_ - displays as simply ``******`` if the currently logged-in user does not have the 'admin' or 'operations' role(s)
:iloUsername:    The user name for the server's ILO service\ [1]_
:interfaceMtu:   The Maximum Transmission Unit (MTU) to configured on ``interfaceName``
:interfaceName:  The name of the primary network interface used by the server
:ip6Address:     The IPv6 address and subnet mask of ``interfaceName``
:ip6Gateway:     The IPv6 address of the gateway used by ``interfaceName``
:ipAddress:      The IPv4 address of ``interfaceName``
:ipGateway:      The IPv4 address of the gateway used by ``interfaceName``
:ipNetmask:      The IPv4 subnet mask used by ``interfaceName``
:lastUpdated:    The date and time at which this server description was last modified
:mgmtIpAddress:  The IPv4 address of some network interface on the server used for 'management'
:mgmtIpGateway:  The IPv4 address of a gateway used by some network interface on the server used for 'management'
:mgmtIpNetmask:  The IPv4 subnet mask used by some network interface on the server used for 'management'
:offlineReason:  A user-entered reason why the server is in ADMIN_DOWN or OFFLINE status
:physLocation:   The name of the physical location where the server resides
:physLocationId: An integral, unique identifier for the physical location where the server resides
:profile:        The name of the profile this server uses
:profileDesc:    A description of the profile this server uses
:profileId:      An integral, unique identifier for the profile used by this server
:revalPending:   A boolean value which, if ``true`` indicates that this server has pending content invalidation/revalidation
:rack:           A string indicating "server rack" location
:routerHostName: The human-readable name of the router responsible for reaching this server
:routerPortName: The human-readable name of the port used by the router responsible for reaching this server
:status:         The status of the server

	.. seealso:: :ref:`health-proto`

:statusId: The integral, unique identifier of the status of this server

	.. seealso:: :ref:`health-proto`

:tcpPort: The port on which this server listens for incoming TCP connections

	.. note:: This is typically thought of as synonymous with "HTTP port", as the port specified by ``httpsPort`` may also be used for incoming TCP connections.

:type:       The name of the 'type' of this server
:typeId:     The integral, unique identifier of the 'type' of this server
:updPending: A boolean value which, if ``true``, indicates that the server has updates of some kind pending, typically to be acted upon by Traffic Ops ORT
:xmppId:     An identifier to be used in XMPP communications with the server - in nearly all cases this will be the same as ``hostName``
:xmppPasswd: The password used in XMPP communications with the server

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: WyapQctUIhjzEALka5QbBiZRZ58Mlc6MJSwjBeGyJS2UzbL3W6lN/4kvAZtPrP4qMWQBWz6JjbF7Y5lNRASUmQ==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 10 Dec 2018 16:13:31 GMT
	Content-Length: 939

	{ "response": [
		{
			"cachegroup": "CDN_in_a_Box_Mid",
			"cachegroupId": 6,
			"cdnId": 2,
			"cdnName": "CDN-in-a-Box",
			"domainName": "infra.ciab.test",
			"guid": null,
			"hostName": "mid",
			"httpsPort": 443,
			"id": 10,
			"iloIpAddress": "",
			"iloIpGateway": "",
			"iloIpNetmask": "",
			"iloPassword": "",
			"iloUsername": "",
			"interfaceMtu": 1500,
			"interfaceName": "eth0",
			"ip6Address": "fc01:9400:1000:8::120",
			"ip6Gateway": "fc01:9400:1000:8::1",
			"ipAddress": "172.16.239.120",
			"ipGateway": "172.16.239.1",
			"ipNetmask": "255.255.255.0",
			"lastUpdated": "2018-12-05 18:45:05+00",
			"mgmtIpAddress": "",
			"mgmtIpGateway": "",
			"mgmtIpNetmask": "",
			"offlineReason": "",
			"physLocation": "Apachecon North America 2018",
			"physLocationId": 1,
			"profile": "ATS_MID_TIER_CACHE",
			"profileDesc": "Mid Cache - Apache Traffic Server",
			"profileId": 10,
			"rack": "",
			"revalPending": false,
			"routerHostName": "",
			"routerPortName": "",
			"status": "REPORTED",
			"statusId": 3,
			"tcpPort": 80,
			"type": "MID",
			"typeId": 12,
			"updPending": false,
			"xmppId": "mid",
			"xmppPasswd": ""
		}
	]}

.. [1] For more information see the `Wikipedia page on Lights-Out management <https://en.wikipedia.org/wiki/Out-of-band_management>`_\ .

``POST``
========
Allows a user to create a new server.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:cachegroupId: The integral, unique identifier of the Cache Group to which this server shall belong
:cdnId:        The integral, unique identifier of the CDN to which the server shall belong
:domainName:   The domain part of the server's Fully Qualified Domain Name (FQDN)
:hostName:     The (short) hostname of the server
:httpsPort:    An optional port number on which the server listens for incoming HTTPS connections/requests
:iloIpAddress: An optional IPv4 address of the server's Integrated Lights-Out (ILO) service\ [1]_
:iloIpGateway: An optional IPv4 gateway address of the server's ILO service\ [1]_
:iloIpNetmask: An optional IPv4 subnet mask of the server's ILO service\ [1]_
:iloPassword:  An optional string containing the password of the of the server's ILO service user\ [1]_ - displays as simply ``******`` if the currently logged-in user does not have the 'admin' or 'operations' role(s)
:iloUsername:  An optional string containing the user name for the server's ILO service\ [1]_
:interfaceMtu: The Maximum Transmission Unit (MTU) to configured on ``interfaceName``

	.. note:: In virtually all cases this ought to be 1500. Further note that the only acceptable values are 1500 and 9000.

:interfaceName:  The name of the primary network interface used by the server
:ip6Address:     An optional IPv6 address and subnet mask of ``interfaceName``
:ip6Gateway:     An optional IPv6 address of the gateway used by ``interfaceName``
:ipAddress:      The IPv4 address of ``interfaceName``
:ipGateway:      The IPv4 address of the gateway used by ``interfaceName``
:ipNetmask:      The IPv4 subnet mask used by ``interfaceName``
:mgmtIpAddress:  An optional IPv4 address of some network interface on the server used for 'management'
:mgmtIpGateway:  An optional IPv4 address of a gateway used by some network interface on the server used for 'management'
:mgmtIpNetmask:  An optional IPv4 subnet mask used by some network interface on the server used for 'management'
:physLocationId: An integral, unique identifier for the physical location where the server resides
:profileId:      An integral, unique identifier for the profile used by this server
:revalPending:   A boolean value which, if ``true`` indicates that this server has pending content invalidation/revalidation
:rack:           An optional string indicating "server rack" location
:routerHostName: An optional string containing the human-readable name of the router responsible for reaching this server
:routerPortName: An optional string containing the human-readable name of the port used by the router responsible for reaching this server
:statusId:       The integral, unique identifier of the status of this server

	.. seealso:: :ref:`health-proto`

:tcpPort: An optional port number on which this server listens for incoming TCP connections

	.. note:: This is typically thought of as synonymous with "HTTP port", as the port specified by ``httpsPort`` may also be used for incoming TCP connections.

:typeId:     The integral, unique identifier of the 'type' of this server
:updPending: A boolean value which, if ``true``, indicates that the server has updates of some kind pending, typically to be acted upon by Traffic Ops ORT
:xmppId:     An optional identifier to be used in XMPP communications with the server - in nearly all cases this should be the same as ``hostName``
:xmppPasswd: An optional password used in XMPP communications with the server

.. code-block:: http
	:caption: Request Example

	POST /api/1.4/servers HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 599
	Content-Type: application/json

	{
		"cachegroupId": 6,
		"cdnId": 2,
		"domainName": "infra.ciab.test",
		"hostName": "test",
		"httpsPort": 443,
		"iloIpAddress": "",
		"iloIpGateway": "",
		"iloIpNetmask": "",
		"iloPassword": "",
		"iloUsername": "",
		"interfaceMtu": 1500,
		"interfaceName": "eth0",
		"ip6Address": "::1",
		"ip6Gateway": "::2",
		"ipAddress": "0.0.0.1",
		"ipGateway": "0.0.0.2",
		"ipNetmask": "255.255.255.0",
		"mgmtIpAddress": "",
		"mgmtIpGateway": "",
		"mgmtIpNetmask": "",
		"offlineReason": "",
		"physLocationId": 1,
		"profileId": 10,
		"routerHostName": "",
		"routerPortName": "",
		"statusId": 3,
		"tcpPort": 80,
		"typeId": 12,
		"updPending": false
	}

Response Structure
------------------
:cachegroup:     The name of the Cache Group to which this server belongs
:cachegroupId:   The integral, unique identifier of the Cache Group to which this server belongs
:cdnId:          The integral, unique identifier of the CDN to which the server belongs
:cdnName:        Name of the CDN to which the server belongs
:domainName:     The domain part of the server's Fully Qualified Domain Name (FQDN)
:guid:           An identifier used to uniquely identify the server

	.. deprecated:: 1.1
		This is a legacy key which only still exists for compatibility reasons - it should always be ``null``

:hostName:       The (short) hostname of the server
:httpsPort:      The port on which the server listens for incoming HTTPS connections/requests
:id:             An integral, unique identifier for this server
:iloIpAddress:   The IPv4 address of the server's Integrated Lights-Out (ILO) service\ [1]_
:iloIpGateway:   The IPv4 gateway address of the server's ILO service\ [1]_
:iloIpNetmask:   The IPv4 subnet mask of the server's ILO service\ [1]_
:iloPassword:    The password of the of the server's ILO service user\ [1]_ - displays as simply ``******`` if the currently logged-in user does not have the 'admin' or 'operations' role(s)
:iloUsername:    The user name for the server's ILO service\ [1]_
:interfaceMtu:   The Maximum Transmission Unit (MTU) to configured on ``interfaceName``
:interfaceName:  The name of the primary network interface used by the server
:ip6Address:     The IPv6 address and subnet mask of ``interfaceName``
:ip6Gateway:     The IPv6 address of the gateway used by ``interfaceName``
:ipAddress:      The IPv4 address of ``interfaceName``
:ipGateway:      The IPv4 address of the gateway used by ``interfaceName``
:ipNetmask:      The IPv4 subnet mask used by ``interfaceName``
:lastUpdated:    The date and time at which this server description was last modified
:mgmtIpAddress:  The IPv4 address of some network interface on the server used for 'management'
:mgmtIpGateway:  The IPv4 address of a gateway used by some network interface on the server used for 'management'
:mgmtIpNetmask:  The IPv4 subnet mask used by some network interface on the server used for 'management'
:offlineReason:  A user-entered reason why the server is in ADMIN_DOWN or OFFLINE status
:physLocation:   The name of the physical location where the server resides
:physLocationId: An integral, unique identifier for the physical location where the server resides
:profile:        The name of the profile this server uses
:profileDesc:    A description of the profile this server uses
:profileId:      An integral, unique identifier for the profile used by this server
:revalPending:   A boolean value which, if ``true`` indicates that this server has pending content invalidation/revalidation
:rack:           A string indicating "server rack" location
:routerHostName: The human-readable name of the router responsible for reaching this server
:routerPortName: The human-readable name of the port used by the router responsible for reaching this server
:status:         The status of the server

	.. seealso:: :ref:`health-proto`

:statusId: The integral, unique identifier of the status of this server

	.. seealso:: :ref:`health-proto`

:tcpPort: The port on which this server listens for incoming TCP connections

	.. note:: This is typically thought of as synonymous with "HTTP port", as the port specified by ``httpsPort`` may also be used for incoming TCP connections.

:type:       The name of the 'type' of this server
:typeId:     The integral, unique identifier of the 'type' of this server
:updPending: A boolean value which, if ``true``, indicates that the server has updates of some kind pending, typically to be acted upon by Traffic Ops ORT
:xmppId:     An identifier to be used in XMPP communications with the server - in nearly all cases this will be the same as ``hostName``
:xmppPasswd: The password used in XMPP communications with the server

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: mcGmmu5ONDg3jmvlkItcw6jxiT1ecmePYujZfmKiZrn5ThKjsSadeJIynaeOK0XVUjHuYHdtdynSqxr2rdzEyA==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 10 Dec 2018 17:44:04 GMT
	Content-Length: 850

	{ "alerts": [
		{
			"text": "server was created.",
			"level": "success"
		}
	],
	"response": {
		"cachegroup": "CDN_in_a_Box_Edge",
		"cachegroupId": 6,
		"cdnId": 2,
		"cdnName": "ALL",
		"domainName": "infra.ciab.test",
		"guid": null,
		"hostName": "test",
		"httpsPort": 443,
		"id": 13,
		"iloIpAddress": "",
		"iloIpGateway": "",
		"iloIpNetmask": "",
		"iloPassword": "",
		"iloUsername": "",
		"interfaceMtu": 1500,
		"interfaceName": "eth0",
		"ip6Address": "::1",
		"ip6Gateway": "::2",
		"ipAddress": "0.0.0.1",
		"ipGateway": "0.0.0.2",
		"ipNetmask": "255.255.255.0",
		"lastUpdated": "2018-12-10 17:44:04+00",
		"mgmtIpAddress": "",
		"mgmtIpGateway": "",
		"mgmtIpNetmask": "",
		"offlineReason": "",
		"physLocation": "Apachecon North America 2018",
		"physLocationId": 1,
		"profile": "ENROLLER_ALL",
		"profileDesc": "Enroller Service Profile for all CDNs",
		"profileId": 10,
		"rack": null,
		"revalPending": null,
		"routerHostName": "",
		"routerPortName": "",
		"status": "ONLINE",
		"statusId": 3,
		"tcpPort": 80,
		"type": "MID",
		"typeId": 12,
		"updPending": false,
		"xmppId": "test",
		"xmppPasswd": null
	}}
