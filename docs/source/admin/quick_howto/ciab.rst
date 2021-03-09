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

.. _ciab:

************
CDN in a Box
************
"CDN in a Box" is a name given to the time-honored tradition of new Traffic Control developers/potential users attempting to set up their own, miniature CDN to just see how it all fits together. Historically, this has been a nightmare of digging through leftover ``virsh`` scripts and manually configuring pretty hefty networking changes (don't even get me started on DNS) and just generally having a bad time. For a few years now, different people had made it to various stages of merging the project into Docker for ease of networking, but certain constraints hampered progress - until now. The project has finally reached a working state, and now getting a mock/test CDN running can be a very simple task (albeit rather time-consuming).

Getting Started
===============
Because it runs in Docker, the only true prerequisites are:

* Docker version >= 17.05.0-ce
* Docker Compose\ [1]_ version >= 1.9.0

Building
--------
The CDN in a Box directory is found within the Traffic Control repository at :file:`infrastructure/cdn-in-a-box/`. CDN in a Box relies on the presence of pre-built :file:`{component}.rpm` files for the following Traffic Control components:

* Traffic Monitor - at :file:`infrastructure/cdn-in-a-box/traffic_monitor/traffic_monitor.rpm`
* Traffic Ops - at :file:`infrastructure/cdn-in-a-box/traffic_ops/traffic_ops.rpm`
* Traffic Portal - at :file:`infrastructure/cdn-in-a-box/traffic_portal/traffic_portal.rpm`
* Traffic Router - at :file:`infrastructure/cdn-in-a-box/traffic_router/traffic_router.rpm` - also requires an Apache Tomcat RPM at :file:`infrastructure/cdn-in-a-box/traffic_router/tomcat.rpm`

.. note:: These can also be specified via the ``RPM`` variable to a direct Docker build of the component - with the exception of Traffic Router, which instead accepts ``JDK8_RPM`` to specify a Java Development Kit RPM,  ``TRAFFIC_ROUTER_RPM`` to specify a Traffic Router RPM, and  ``TOMCAT_RPM`` to specify an Apache Tomcat RPM.

These can all be supplied manually via the steps in :ref:`dev-building` (for Traffic Control component RPMs) or via some external source. Alternatively, the :file:`infrastructure/cdn-in-a-box/Makefile` file contains recipes to build all of these - simply run :manpage:`make(1)` from the :file:`infrastructure/cdn-in-a-box/` directory. Once all RPM dependencies have been satisfied, run ``docker-compose build --parallel`` from the :file:`infrastructure/cdn-in-a-box/` directory to construct the images needed to run CDN in a Box.

.. tip:: If you have gone through the steps to :ref:`dev-building-natively`, you can run ``make native`` instead of ``make`` to build the RPMs quickly. Another option is running ``make -j4`` to build 4 components at once, if your computer can handle it.

.. tip:: When updating CDN-in-a-Box, there is no need to remove old images before building new ones. Docker detects which files are updated and only reuses cached layers that have not changed.

By default, CDN in a Box will be based on CentOS 8. To base CDN in a Box on CentOS 7, set the ``RHEL_VERSION`` environment variable to ``7`` (for CDN in a Box, it defaults to ``8``):

.. code-block:: shell
	:caption: Building CDN in a Box to run CentOS 7 instead of CentOS 8

	export RHEL_VERSION=7
	make # Builds RPMs for CentOS 7
	docker-compose build --parallel # Builds CentOS 7 CDN in a Box images

Usage
-----
In a typical scenario, if the steps in `Building`_ have been followed, all that's required to start the CDN in a Box is to run ``docker-compose up`` - optionally with the ``-d`` flag to run without binding to the terminal - from the :file:`infrastructure/cdn-in-a-box/` directory. This will start up the entire stack and should take care of any needed initial configuration. The services within the environment are by default not exposed locally to the host. If this is the desired behavior when bringing up CDN in a Box the command ``docker-compose -f docker-compose.yml -f docker-compose.expose-ports.yml up`` should be run. The ports are configured within the :file:`infrastructure/cdn-in-a-box/docker-compose.expose-ports.yml` file, but the default ports are shown in :ref:`ciab-service-info`. Some services have credentials associated, which are totally configurable in `variables.env`_.

.. _ciab-service-info:
.. table:: Service Info

	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Service                         | Ports exposed and their usage                                  | Username                              | Password                                  |
	+=================================+================================================================+=======================================+===========================================+
	| DNS                             | DNS name resolution on 9353                                    | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Edge Tier Cache                 | Apache Trafficserver HTTP caching reverse proxy on port 9000   | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Mid Tier Cache                  | Apache Trafficserver HTTP caching forward proxy on port 9100   | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Second Mid-Tier Cache (parent   | Apache Trafficserver HTTP caching forward proxy on port 9100   | N/A                                   | N/A                                       |
	| of the first Mid-Tier Cache)    |                                                                |                                       |                                           |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Mock Origin Server              | Example web page served on port 9200                           | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| SMTP Server                     | Passwordless, cleartext SMTP server on port 25 (no relay)      | N/A                                   | N/A                                       |
	|                                 | Web interface exposed on port 4443 (port 443 in the container) |                                       |                                           |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Monitor                 | Web interface and API on port 80                               | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Ops                     | API endpoints on port 6443                                     | ``TO_ADMIN_USER`` in `variables.env`_ | ``TO_ADMIN_PASSWORD`` in `variables.env`_ |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Ops PostgresQL Database | PostgresQL connections accepted on port 5432 (database name:   | ``DB_USER`` in `variables.env`_       | ``DB_USER_PASS`` in `variables.env`_      |
	|                                 | ``DB_NAME`` in `variables.env`_)                               |                                       |                                           |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Portal                  | Web interface on 443 (Javascript required)                     | ``TO_ADMIN_USER`` in `variables.env`_ | ``TO_ADMIN_PASSWORD`` in `variables.env`_ |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Router                  | Web interfaces on ports 3080 (HTTP) and 3443 (HTTPS), with a   | N/A                                   | N/A                                       |
	|                                 | DNS service on 53 and an API on 3333 (HTTP) and 2222 (HTTPS)   |                                       |                                           |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Vault                   | Riak key-value store on port 8010                              | ``TV_ADMIN_USER`` in `variables.env`_ | ``TV_ADMIN_PASSWORD`` in `variables.env`_ |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Stats                   | N/A                                                            | N/A                                   | N/A                                       |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+
	| Traffic Stats Influxdb          | Influxdbd connections accepted on port 8086 (database name:    | ``INFLUXDB_ADMIN_USER`` in            | ``INFLUXDB_ADMIN_PASSWORD`` in            |
	|                                 | ``cache_stats``, ``daily_stats`` and                           | `variables.env`_                      | `variables.env`_                          |
	|                                 | ``deliveryservice_stats``)                                     |                                       |                                           |
	+---------------------------------+----------------------------------------------------------------+---------------------------------------+-------------------------------------------+

.. seealso:: :ref:`tr-api` and :ref:`tm-api`

While the components may be interacted with by the host using these ports, the true operation of the CDN can only truly be seen from within the Docker network. To see the CDN in action, connect to a container within the CDN in a Box project and use cURL to request the URL ``http://video.demo1.mycdn.ciab.test`` which will be resolved by the DNS container to the IP of the Traffic Router, which will provide a ``302 FOUND`` response pointing to the Edge-Tier cache. A typical choice for this is the "enroller" service, which has a very nuanced purpose not discussed here but already has the :manpage:`curl(1)` command line tool installed. For a more user-friendly interface into the CDN network, see `VNC Server`_.

.. code-block:: shell
	:caption: Example Command to See the CDN in Action

	sudo docker-compose exec enroller /usr/bin/curl -L "http://video.demo1.mycdn.ciab.test"

When the CDN is to be shut down, it is often best to do so using ``sudo docker-compose down -v`` due to the use of shared volumes in the system which might interfere with a proper initialization upon the next run.

.. note:: Although :abbr:`DNSSEC (DNS Security Extensions)` is enabled for the CDN (sending DNS queries directly to Traffic Router), DNSSEC is not enabled for the ``dns`` service.

Readiness Check
"""""""""""""""

In order to check the "readiness" of your CDN, you can optionally start the Readiness Container, which will continually :manpage:`curl(1)` the :term:`Delivery Services` in your CDN until they all return successful responses before exiting successfully.

.. code-block:: shell
	:caption: Example Command to Run the Readiness Container

	sudo docker-compose -f docker-compose.readiness.yml up

Integration Tests
"""""""""""""""""

There also exist TP and TO integration tests containers. Both of these containers assume that CDN in a Box is already running on the local system.

.. code-block:: shell
	:caption: Running TP Integration Tests

	sudo docker-compose -f docker-compose.traffic-portal-test.yml up

.. code-block:: shell
	:caption: Running TO Integration Tests

	sudo docker-compose -f docker-compose.traffic-ops-test.yml up

.. note:: If all CDN in a Box containers are started at once (example: ``docker-compose -f docker-compose.yml -f docker-compose.traffic-ops-test.yml up integration``), the :ref:`Enroller <ciab-enroller>` initial data load is skipped to prevent data conflicts with the :ref:`Traffic Ops API tests fixtures <dev-traffic-ops-fixtures>`.

variables.env
"""""""""""""
.. literalinclude:: ../../../../infrastructure/cdn-in-a-box/variables.env
	:language: shell
	:lines: 17-
	:tab-width: 4

.. note:: While these port settings can be changed without hampering the function of the CDN in a Box system, note that changing a port without also changing the matching port-mapping in :file:`infrastructure/cdn-in-a-box/docker-compose.yml` for the affected service *will* make it unreachable from the host.

.. [1] It is perfectly possible to build and run all containers without Docker Compose, but it's not recommended and not covered in this guide.

X.509 SSL/TLS Certificates
==========================
All components in Apache Traffic Control utilize SSL/TLS secure communications by default. For SSL/TLS connections to properly validate within the "CDN in a Box" container network a shared self-signed X.509 Root :abbr:`CA (Certificate Authority)` is generated at the first initial startup. An X.509 Intermediate :abbr:`CA (Certificate Authority)` is also generated and signed by the Root :abbr:`CA (Certificate Authority)`. Additional "wildcard" certificates are generated/signed by the Intermediate :abbr:`CA (Certificate Authority)` for each container service and demo1, demo2, and demo3 :term:`Delivery Services`. All certificates and keys are stored in the ``ca`` host volume which is located at :file:`infrastruture/cdn-in-a-box/traffic_ops/ca`\ [4]_.

.. _ciab-x509-certificate-list:
.. table:: Self-Signed X.509 Certificate List

	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| Filename                  | Description                                                              | X.509 CN/SAN                           |
	+===========================+==========================================================================+========================================+
	| CIAB-CA-root.crt          | Shared Root :abbr:`CA (Certificate Authority)` Certificate               | N/A                                    |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| CIAB-CA-intr.crt          | Shared Intermediate :abbr:`CA (Certificate Authority)` Certificate       | N/A                                    |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| CIAB-CA-fullchain.crt     | Shared :abbr:`CA (Certificate Authority)` Certificate Chain Bundle\ [5]_ | N/A                                    |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| infra.ciab.test.crt       | Infrastruture Certificate                                                | :file:`{prefix}.infra.ciab.test`       |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| demo1.mycdn.ciab.test.crt | Demo1 :term:`Delivery Service` Certificate                               | :file:`{prefix}.demo1.mycdn.ciab.test` |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| demo2.mycdn.ciab.test.crt | Demo2 :term:`Delivery Service` Certificate                               | :file:`{prefix}.demo2.mycdn.ciab.test` |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+
	| demo3.mycdn.ciab.test.crt | Demo3 :term:`Delivery Service` Certificate                               | :file:`{prefix}.demo3.mycdn.ciab.test` |
	+---------------------------+--------------------------------------------------------------------------+----------------------------------------+

.. [4] The ``ca`` volume is not purged with normal ``docker volume`` commands. This feature is by design to allow the existing shared SSL certificate to be trusted at the system level across restarts. To re-generate all SSL certificates and keys, remove the ``infrastructure/cdn-in-a-box/traffic_ops/ca`` directory before startup.
.. [5] The full chain bundle is a file that contains both the Root and Intermediate :abbr:`CA (Certificate Authority)` certificates.

Trusting the Certificate Authority
----------------------------------
For developer and testing use-cases, it may be necessary to have full x509 :abbr:`CA (Certificate Authority)` validation by HTTPS clients\ [6]_\ [7]_. For x509 validation to work properly, the self-signed x509 :abbr:`CA (Certificate Authority)` certificate must be trusted either at the system level or by the client application itself.

.. note:: HTTP Client applications such as Google Chrome, Firefox, :manpage:`curl(1)`, and :manpage:`wget(1)` can also be individually configured to trust the :abbr:`CA (Certificate Authority)` certificate. Review each program's respective documentation for instructions.

Importing the :abbr:`CA (Certificate Authority)` Certificate on OSX
"""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""
#. Copy the CIAB root and intermediate :abbr:`CA (Certificate Authority)` certificates from :file:`infrastructure/cdn-in-a-box/traffic_ops/ca/` to the Mac.
#. Double-click the CIAB root :abbr:`CA (Certificate Authority)` certificate to open it in Keychain Access.
#. The CIAB root :abbr:`CA (Certificate Authority)` certificate appears in login.
#. Copy the CIAB root :abbr:`CA (Certificate Authority)` certificate to System.
#. Open the CIAB root :abbr:`CA (Certificate Authority)` certificate, expand :guilabel:`Trust`, select :guilabel:`Use System Defaults`, and save your changes.
#. Reopen the CIAB root :abbr:`CA (Certificate Authority)` certificate, expand :guilabel:`Trust, select :guilabel:`Always Trust`, and save your changes.
#. Delete the CIAB root :abbr:`CA (Certificate Authority)` certificate from login.
#. Repeat the previous steps with the Intermediate :abbr:`CA (Certificate Authority)` certificate to import it as well
#. Restart all HTTPS clients (browsers, etc).

Importing the :abbr:`CA (Certificate Authority)` certificate on Windows
-----------------------------------------------------------------------
#. Copy the CIAB root :abbr:`CA (Certificate Authority)` and intermediate :abbr:`CA (Certificate Authority)` certificates from :file:`infrastructure/cdn-in-a-box/traffic_ops/ca/` to Windows filesystem.
#. As Administrator, start the Microsoft Management Console.
#. Add the Certificates snap-in for the computer account and manage certificates for the local computer.
#. Import the CIAB root :abbr:`CA (Certificate Authority)` certificate into :menuselection:`Trusted Root Certification Authorities --> Certificates`.
#. Import the CIAB intermediate :abbr:`CA (Certificate Authority)` certificate into :menuselection:`Trusted Root Certification Authorities --> Certificates`.
#. Restart all HTTPS clients (browsers, etc).

Importing the :abbr:`CA (Certificate Authority)` certificate on CentOS 8 (Linux)
--------------------------------------------------------------------------------
#. Copy the CIAB full chain :abbr:`CA (Certificate Authority)` certificate bundle from :file:`infrastructure/cdn-in-a-box/traffic_ops/ca/CIAB-CA-fullchain.crt` to path :file:`/etc/pki/ca-trust/source/anchors/`.
#. Run ``update-ca-trust-extract`` as the root user or with :manpage:`sudo(8)`.
#. Restart all HTTPS clients (browsers, etc).

Importing the :abbr:`CA (Certificate Authority)` certificate on Ubuntu (Linux)
------------------------------------------------------------------------------
#. Copy the CIAB full chain :abbr:`CA (Certificate Authority)` certificate bundle from :file:`infrastructure/cdn-in-a-box/traffic_ops/ca/CIAB-CA-fullchain.crt` to path :file:`/usr/local/share/ca-certificates/`.
#. Run ``update-ca-certificates`` as the root user or with :manpage:`sudo(8)`.
#. Restart all HTTPS clients (browsers, etc).

.. [6] All containers within CDN-in-a-Box start up with the self-signed :abbr:`CA (Certificate Authority)` already trusted.
.. [7] The 'demo1' :term:`Delivery Service` X509 certificate is automatically imported into Traffic Vault on startup.

Advanced Usage
==============
This section will be amended as functionality is added to the CDN in a Box project.

.. _ciab-enroller:

The Enroller
------------
The "enroller" began as an efficient way for Traffic Ops to be populated with data as CDN in a Box starts up. It connects to Traffic Ops as the "admin" user and processes files places in the docker volume shared between the containers. The enroller watches each directory within the ``/shared/enroller`` directory for new :file:`{filename}.json` files to be created there. These files must follow the format outlined in the API guide for the ``POST`` method for each data type,  (e.g. for a ``region``, follow the guidelines for ``POST api/2.0/regions``). Of note, the ``enroller`` does not require fields that reference database ids for other objects within the database.

.. program::enroller

.. option:: --dir directory

	Base directory to watch for data. Mutually exclusive with :option:`--http`\ .

.. option:: --http port

	Act as an HTTP server for ``POST`` requests on this port. Mutually exclusive with :option:`--dir`\ .

.. option:: --started filename

	The name of a file which will be created in the :option:`--dir` directory when given, indicating service was started (default: "enroller-started").


The enroller runs within CDN in a Box using :option:`--dir` which provides the above behavior. It can also be run using :option:`--http` to instead have it listen on the indicated port. In this case, it accepts only ``POST`` requests with the JSON provided in the request payload, e.g. ``curl -X POST https://enroller/api/2.0/regions -d @newregion.json``. CDN in a Box does not currently use this method, but may be modified in the future to avoid using the shared volume approach.

Auto Snapshot/Queue-Updates
---------------------------
An automatic :term:`Snapshot` of the current Traffic Ops CDN configuration/topology will be performed once the "enroller" has finished loading all of the data and a minimum number of servers have been enrolled. To enable this feature, set the boolean ``AUTO_SNAPQUEUE_ENABLED`` to ``true`` [8]_. The :term:`Snapshot` and :term:`Queue Updates` actions will not be performed until all servers in ``AUTO_SNAPQUEUE_SERVERS`` (comma-delimited string) have been enrolled. The current enrolled servers will be polled every ``AUTO_SNAPQUEUE_POLL_INTERVAL`` seconds, and each action (:term:`Snapshot` and :term:`Queue Updates`) will be delayed ``AUTO_SNAPQUEUE_ACTION_WAIT`` seconds [9]_.

.. [8] Automatic :term:`Snapshot`/:term:`Queue Updates` is enabled by default in `variables.env`_.
.. [9] Server poll interval and delay action wait are defaulted to a value of 2 seconds.

Mock Origin Service
-------------------
The default "origin" service container provides a basic static file HTTP server as the central repository for content. Additional files can be added to the origin root content directory located at :file:`infrastructure/cdn-in-a-box/origin/content`. To request content directly from the origin directly and bypass the CDN:

* Origin Service URL: http://origin.infra.ciab.test/index.html
* Docker Host: http://localhost:9200/index.html

.. _ciab-optional-containers:

Optional Containers
===================

All optional containers that are not part of the core CDN-in-a-Box stack are located in the ``infrastructure/cdn-in-a-box/optional`` directory.

- :file:`infrastructure/cdn-in-a-box/optional/docker-compose.{NAME}.yml`
- :file:`infrastructure/cdn-in-a-box/optional/{NAME}/Dockerfile`

Multiple optional containers may be combined by using a shell alias:

.. code-block:: shell
	:caption: Starting Optional Containers with an Alias

	# From the infrastructure/cdn-in-a-box directory
	# (Assuming the names of the optional services are stored in the `NAME1` and `NAME2` environment variables)
	alias mydc="docker-compose -f $PWD/docker-compose.yml -f $PWD/optional/docker-compose.$NAME1.yml -f  $PWD/optional/docker-compose.$NAME2.yml"
	docker volume prune -f
	mydc build
	mydc up

VNC Server
----------
The TightVNC optional container provides a basic lightweight window manager (fluxbox), Firefox browser, xterm, and a few other utilities within the CDN-In-A-Box "tcnet" bridge network. This can be very helpful for quick demonstrations of CDN-in-a-Box that require the use of real container network :abbr:`FQDN (Fully Qualified Domain Name)`\ s and full X.509 validation.

#. Download and install a VNC client. TightVNC client is preferred as it supports window resizing, host-to-vnc copy/pasting, and optimized frame buffer compression.
#. Set your VNC console password by adding the ``VNC_PASSWD`` environment variable to :file:`infrastructure/cdn-in-a-box/varibles.env`. The password needs to be at least six characters long. The default password is randomized for security.
#. Start up CDN-in-a-Box stack. It is recommended that this be done using a custom bash alias

	.. code-block:: shell
		:caption: CIAB Startup Using Bash Alias

		# From infrastructure/cdn-in-a-box
		alias mydc="docker-compose "` \
			`"-f $PWD/docker-compose.yml "` \
			`"-f $PWD/docker-compose.expose-ports.yml "` \
			`"-f $PWD/optional/docker-compose.vnc.yml "` \
			`"-f $PWD/optional/docker-compose.vnc.expose-ports.yml"
		docker volume prune -f
		mydc build
		mydc kill
		mydc rm -fv
		mydc up


#. Connect with a VNC client to localhost port 5909.
#. When Traffic Portal becomes available, the Firefox within the VNC instance will subsequently be started.
#. An xterm with bash shell is also automatically spawned and minimized for convenience.

Socks Proxy
-----------
Dante's socks proxy is an optional container that can be used to provide browsers and other clients the ability to resolve DNS queries and network connectivity directly on the tcnet bridged interface. This is very helpful when running the CDN-In-A-Box stack on OSX/Windows docker host that lacks network bridge/IP-forward support. Below is the basic procedure to enable the Socks Proxy support for CDN-in-a-Box:

#. Start the CDN-in-a-Box stack at least once so that the x.509 self-signed :abbr:`CA (Certificate Authority)` is created.
#. On the host, import and Trust the :abbr:`CA (Certificate Authority)` for your target Operating System. See `Trusting the Certificate Authority`_
#. On the host, using either Firefox or Chrome, download the `FoxyProxy browser plugin <https://getfoxyproxy.org/>`_ which enables dynamic proxy support via URL regular expression
#. Once FoxyProxy is installed, click the Fox icon on the upper right hand of the browser window, select :guilabel:`Options`
#. Once in Options Dialog, Click :guilabel:`Add New Proxy` and navigate to the General tab:
#. Fill in the General tab according to the table

	.. table:: General Tab Values

		+------------+---------+
		| Name       |   Value |
		+============+=========+
		| Proxy Name |    CIAB |
		+------------+---------+
		| Color      |   Green |
		+------------+---------+

#. Fill in the Proxy Details tab according to the table

	.. table:: Proxy Details Tab Values

		+----------------------------+-----------+
		| Name                       |     Value |
		+============================+===========+
		| Manual Proxy Configuration |      CIAB |
		+----------------------------+-----------+
		| Host or IP Address         | localhost |
		+----------------------------+-----------+
		| Port                       |      9080 |
		+----------------------------+-----------+
		| Socks Proxy                |   checked |
		+----------------------------+-----------+
		| Socks V5                   |  selected |
		+----------------------------+-----------+

#. Go to URL Patterns tab, click Add New Pattern, and fill out form according to

	.. table:: URL Patters Tab Values

		+--------------+-----------------------+
		| Name         |                 Value |
		+==============+=======================+
		| Pattern Name |          CIAB Pattern |
		+--------------+-----------------------+
		| URL Pattern  |  :regexp:`\*.test/\*` |
		+--------------+-----------------------+
		| Whitelist    |              selected |
		+--------------+-----------------------+
		| Wildcards    |              selected |
		+--------------+-----------------------+

#. Enable dynamic 'pre-defined and patterns' mode by clicking the fox icon in the upper right of the browser. This mode only forwards URLs that match the wildcard :regexp:`\*.test/\*` to the Socks V5 proxy.

#. On the docker host start up CDN-in-a-Box stack. It is recommended that this be done using a custom bash alias

	.. code-block:: shell
		:caption: CIAB Startup Using Bash Alias

		# From infrastructure/cdn-in-a-box
		alias mydc="docker-compose -f $PWD/docker-compose.yml -f $PWD/optional/docker-compose.socksproxy.yml"
		docker volume prune -f
		mydc build
		mydc kill
		mydc rm -fv
		mydc up

#. Once the CDN-in-a-box stack has started, use the aforementioned browser to access Traffic Portal via the socks proxy on the docker host.

.. seealso:: `The official Docker Compose documentation CLI reference <https://docs.docker.com/compose/reference/>`_ for complete instructions on how to pass service definition files to the ``docker-compose`` executable.

Static Subnet
-------------
Since ``docker-compose`` will randomly create a subnet and it has a chance to conflict with your network environment, using static subnet is a good choice.

.. code-block:: shell
	:caption: CIAB Startup with Static Subnet

	# From the infrastructure/cdn-in-a-box directory
	alias mydc="docker-compose -f $PWD/docker-compose.yml -f $PWD/optional/docker-compose.static-subnet.yml"
	docker volume prune -f
	mydc build
	mydc up

VPN Server
----------
This container provides an OpenVPN service. It's primary use is to allow users and developers to easily access CIAB network.

How to use it
"""""""""""""
#. It is recommended that this be done using a custom bash alias.

	.. code-block:: shell
		:caption: CIAB Startup with VPN

		# From infrastructure/cdn-in-a-box
		alias mydc="docker-compose -f $PWD/docker-compose.yml -f $PWD/docker-compose.expose-ports.yml -f $PWD/optional/docker-compose.vpn.yml -f $PWD/optional/docker-compose.vpn.expose-ports.yml"
		mydc down -v
		mydc build
		mydc up

#. All certificates, keys, and client configuration are stored at ``infrastruture/cdn-in-a-box/optional/vpn/vpnca``. You just simply change ``REALHOSTIP`` and ``REALPORT`` of ``client.ovpn`` to fit your environment, and then you can use it to connect to this OpenVPN server.

The proposed VPN client
"""""""""""""""""""""""
On Linux, we suggest ``openvpn``. On most Linux distributions, this will also be the name of the package that provides it.

.. code-block:: shell
	:caption: Install openvpn on ubuntu/debian

	apt-get update && apt-get install -y openvpn

On OSX, it only works with brew installed openvpn client, not the *OpenVPN GUI client*.

.. code-block:: shell
	:caption: Install openvpn on OSX

	brew install openvpn

If you want a GUI version of VPN client, we recommend `Tunnelblick <https://tunnelblick.net/>`_.

Private Subnet for Routing
""""""""""""""""""""""""""
Since ``docker-compose`` randomly creates a subnet, this container prepares 2 default private subnets for routing:

* 172.16.127.0/255.255.240.0
* 10.16.127.0/255.255.240.0

The subnet that will be used is determined automatically based on the subnet prefix. If the subnet prefix which ``docker-compose`` selected is ``192.`` or ``10.``, this container will select 172.16.127.0/255.255.240.0 for its routing subnet. Otherwise, it selects 10.16.127.0/255.255.240.0.

Of course, you can decide which routing subnet subnet by supplying the environment variables ``PRIVATE_NETWORK`` and ``PRIVATE_NETMASK``.

Pushed Settings
"""""""""""""""
Pushed settings are shown as follows:

* DNS
* A routing rule for the ``CIAB`` subnet

.. note:: It will not change your default gateway. That means apart from CDN in a Box traffic and DNS requests, all other traffic will use the standard interface bound to the default gateway.

Grafana
-------
This container provides a Grafana service. It's an open platform for analytics and monitoring. This container has prepared necessary *datasources* and *scripted dashboards*. Please refer to :ref:`grafana-config` for detailed Settings.

How to start it
"""""""""""""""
It is recommended that this be done using a custom bash alias.

.. code-block:: shell
	:caption: CIAB Startup with Grafana

	# From infrastructure/cdn-in-a-box
	alias mydc="docker-compose -f $PWD/docker-compose.yml -f $PWD/optional/docker-compose.grafana.yml -f $PWD/optional/docker-compose.grafana.expose-ports.yml"
	mydc down -v
	mydc build
	mydc up

Apart from start Grafana, the above commands also expose port 3000 for it.

Check the charts
""""""""""""""""
There are some *scripted dashboards* can show beautiful charts. You can display different charts by passing in different *query string*

* ``https://<grafanaHost>/dashboard/script/traffic_ops_cachegroup.js?which=``. The query parameter `which` in this particular URL should be the **cachegroup**. Take CIAB as an example, it can be filled in with **CDN_in_a_Box_Edge** or **CDN_in_a_Box_Edge**.
* ``https://<grafanaHost>/dashboard/script/traffic_ops_deliveryservice.js?which=``. The query parameter `which` in this particular URL should be the **xml_id** of the desired Delivery Service.
* ``https://<grafanaHost>/dashboard/script/traffic_ops_server.js?which=``. The query parameter `which` in this particular URL should be the **hostname** (not **FQDN**). It can be filled in with **edge** or **mid** in CIAB.

Debugging
---------

See :ref:`dev-debugging-ciab`.
