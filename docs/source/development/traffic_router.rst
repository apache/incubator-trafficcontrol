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

.. _dev-traffic-router:

**************
Traffic Router
**************
Introduction
============
Traffic Router is a Java Tomcat application that routes clients to the closest available cache on the CDN using both HTTP and DNS. Cache server availability is determined by Traffic Monitor; consequently Traffic Router polls Traffic Monitor for its configuration and :term:`cache server` health state information, and uses this data to make routing decisions.  HTTP routing is performed by localizing the client based on the request's source IP address (IPv4 or IPv6), and issues an HTTP 302 response to redirect to the nearest :term:`cache server`. HTTP routing utilizes consistent hashing on request URLs to optimize cache performance and request distribution. DNS routing is performed by localizing clients, resolvers in most cases, requesting ``A`` and ``AAAA`` records for a configurable name such as ``foo.deliveryservice.somecdn.net``. Traffic Router is comprised of seven separate Maven modules:

* shared - A reusable utility JAR for defining :term:`Delivery Service` Certificates
* configuration - A reusable JAR defining the ConfigurationListener interface
* connector - A JAR that overrides Tomcat's standard Http11Protocol Connector class and allows Traffic Router to delay opening listen sockets until it is in a state suitable for routing traffic
* geolocation - Submodule for defining geolocation services
* neustar - A JAR that provides a bean "neustarGeolocationService" that implements the GeolocationService interface defined in the geolocation maven submodule, which can optionally be added to the build of Traffic Router
* core - Services DNS and HTTP requests, performs localization on routing requests, and is deployed as a WAR to a Service (read: connector/listen port) within Tomcat which is separate from the API
* build - A simple Maven project which gathers the artifacts from the modules and builds an RPM

Software Requirements
=====================
To work on Traffic Router you need a \*nix (MacOS and Linux are most commonly used) environment that has the following installed:

* Eclipse >= Kepler SR2 (or another Java IDE)
* Maven >= 3.3.1
* JDK >= 8.0 (OpenJDK suggested, but not required)
* OpenSSL >= 1.0.2
* :abbr:`APR (Apache Portable Runtime)` >= 1.4.8-3
* Tomcat Native >= 1.2.23
* Not Tomcat - You do not need a Tomcat installation for development. An embedded version is launched for development testing instead.

.. _dev-tr-mac-jdk:

Get OpenJDK 8 on macOS
--------------------------
If you are on macOS, OpenJDK 8 is not available from Homebrew by default, but it can still be installed using Homebrew with little effort.

Using Homebrew, |AdoptOpenJDK instructions|_

.. code-block:: shell
	:caption: Install OpenJDK 8 on macOS

        brew tap AdoptOpenJDK/openjdk
        brew cask install adoptopenjdk8

Next, set the JAVA_HOME environment variable. Add this line to your ``~/.bash_profile``:

.. code-block:: shell
        :caption: Set JAVA_HOME environment variable

        export JAVA_HOME=$(/usr/libexec/java_home -v1.8)

.. |AdoptOpenJDK instructions| replace:: add the AdoptOpenJDK tap and install the ``adoptopenjdk8`` cask
.. _AdoptOpenJDK instructions: https://github.com/AdoptOpenJDK/homebrew-openjdk#other-versions

Traffic Router Project Tree Overview
====================================
* ``traffic_control/traffic_traffic_router/`` - base directory for Traffic Router

	* ``connector/`` - Source code for Traffic Router Connector;

		* ``src/main/java`` - Java source directory for Traffic Router Connector

	* ``core/`` - Source code for Traffic Router Core, which is built as its own deployable WAR file and communicates with :ref:`tr-api` using JMX

		* ``src/main`` - Main source directory for Traffic Router Core

			* ``lib/systemd/system/traffic_router.service`` - Unit script for launching the Traffic Router with Tomcat
			* ``conf/`` - All of the required configuration files for running the traffic_router web application, including those needed for Tomcat
			* ``java/`` - Java source code for Traffic Router Core
			* ``resources/`` - Resources pulled in during an RPM build
			* ``scripts/`` - Scripts used by the RPM build process
			* ``webapp/`` - Java "webapp" resources
			* ``var/log/`` - location of all the Traffic Router runtime logs

		* ``src/test`` - Test source directory for Traffic Router Core

			* ``conf/`` - Minimal Configuration files that make it possible to run JUnit tests
			* ``db/`` - Files downloaded by unit tests
			* ``java/`` - JUnit-based unit tests for Traffic Router Core
			* ``resources/`` - Example data files used by junit tests

				* ``var/auto-zones`` - BIND formatted zone files generated by Traffic Router Core during unit testing

Java Formatting Conventions
===========================
None at this time. The codebase will eventually be formatted per Java standards.

Installing The Developer Environment
====================================
To install the Traffic Router Developer environment:

#. Clone the traffic_control repository using Git.
#. Change directories into ``traffic_control/traffic_router``.
#. Set the environment variable TRAFFIC_MONITOR_HOSTS to be a semicolon delimited list of Traffic Monitors that can be accessed during integration tests OR install the :file:`traffic_monitor.properties` file.
#. Additional configuration is set using the below files:

  * copy :file:`core/src/main/conf/dns.properties` to :file:`core/src/test/conf/`
  * copy :file:`core/src/main/conf/http.properties` to :file:`core/src/test/conf/`
  * copy :file:`core/src/main/conf/log4j.properties` to :file:`core/src/test/conf/`
  * copy :file:`core/src/main/conf/traffic_monitor.properties` to :file:`core/src/test/conf/` and then edit the ``traffic_monitor.bootstrap.hosts`` property
  * copy :file:`core/src/main/conf/traffic_ops.properties` to :file:`core/src/test/conf/` and then edit the credentials as appropriate for the Traffic Ops instance you will be using.
  * Default configuration values now reside in :file:`core/src/main/webapp/WEB-INF/applicationContext.xml`

	.. note:: These values may be overridden by creating and/or modifying the property files listed in :file:`core/src/main/resources/applicationProperties.xml`

	.. note:: Pre-existing properties files are still honored by Traffic Router. For example :file:`traffic_monitor.properties` may contain the :abbr:`FQDN (Fully Qualified Domain Name)` and port of the Traffic Monitor instance(s), separated by semicolons as necessary (do not include scheme e.g. ``http://``)


#. Import the existing git repository as projects into your IDE (Eclipse):

	a. :menuselection:`File --> Import --> Git --> Projects from Git`; :guilabel:`Next`
	#. :guilabel:`Existing local repository`; :guilabel:`Next`
	#. :guilabel:`Add` - browse to find ``traffic_control``; :guilabel:`Open`
	#. Select ``traffic_control``; :guilabel:`Next`
	#. Ensure :guilabel:`Import existing projects` is selected, expand ``traffic_control``, select ``traffic_router``; :guilabel:`Next`
	#. Ensure ``traffic_router_api``, ``traffic_router_connector``, and ``traffic_router_core`` are checked; :guilabel:`Finish` (this step can take several minutes to complete)
	#. Ensure ``traffic_router_api``, ``traffic_router_connector``, and ``traffic_router_core`` have been opened by Eclipse after importing

#. From the terminal or your IDE, run ``mvn clean verify`` from the ``traffic_router`` directory. This will run a series of integration tests and will temporarily start and embedded version of Traffic Router and a 'fake' simulated instance of Traffic Monitor.

#. Start the embedded Tomcat instance for Core from within your IDE by following these steps:

	a. In the package explorer, expand ``traffic_router_core``
	#. Expand ``src/test/java``
	#. Expand the package ``com.comcast.cdn.traffic_control.traffic_router.core``
	#. Open and run ``TrafficRouterStart.java``

		..  Note:: If an error is displayed in the Console, run ``mvn clean verify`` from the ``traffic_router`` directory

Once running, the :ref:`tr-api` is available over HTTP at http://localhost:3333 and over HTTPS at https://localhost:3443,  the HTTP routing interface is available on http://localhost:8888 and HTTPS is available on http://localhost:8443. The DNS server and routing interface is available on localhost:53 via TCP and UDP.

Manual Testing
==============
Look up the URL for a test HTTP :term:`Delivery Service` in Traffic Ops and then make a request. When Traffic Router is running and used as a resolver for the host in the :term:`Delivery Service` URL, the requested origin content should be found through an Edge-tier :term:`cache server`.

.. code-block:: console
	:caption: Example Test for an HTTP :term:`Delivery Service`

	root@enroller:/shared/enroller# curl -skvL http://video.demo1.mycdn.ciab.test/
	*   Trying fc01:9400:1000:8::60...
	* TCP_NODELAY set
	* Connected to video.demo1.mycdn.ciab.test (fc01:9400:1000:8::60) port 80 (#0)
	> GET / HTTP/1.1
	> Host: video.demo1.mycdn.ciab.test
	> User-Agent: curl/7.52.1
	> Accept: */*
	>
	< HTTP/1.1 302 Found
	< Location: http://edge.demo1.mycdn.ciab.test/
	< Content-Length: 0
	< Date: Wed, 16 Jan 2019 21:52:14 GMT
	<
	* Curl_http_done: called premature == 0
	* Connection #0 to host video.demo1.mycdn.ciab.test left intact
	* Issue another request to this URL: 'http://edge.demo1.mycdn.ciab.test/'
	*   Trying fc01:9400:1000:8::100...
	* TCP_NODELAY set
	* Connected to edge.demo1.mycdn.ciab.test (fc01:9400:1000:8::100) port 80 (#1)
	> GET / HTTP/1.1
	> Host: edge.demo1.mycdn.ciab.test
	> User-Agent: curl/7.52.1
	> Accept: */*
	>
	< HTTP/1.1 200 OK
	< Content-Type: text/html
	< Accept-Ranges: bytes
	< ETag: "1473249267"
	< Last-Modified: Wed, 07 Nov 2018 13:53:57 GMT
	< Cache-Control: public, max-age=300
	< Access-Control-Allow-Origin: *
	< Access-Control-Allow-Headers: Accept, Origin, Content-Type
	< Access-Control-Allow-Methods: GET, POST, PUT, OPTIONS
	< Content-Length: 1881
	< Date: Wed, 16 Jan 2019 21:52:15 GMT
	< Server: ATS/7.1.4
	< Age: 1
	< Via: http/1.1 mid.infra.ciab.test (ApacheTrafficServer/7.1.4 [uScMsSfWpSeN:t cCMi p sS]), http/1.1 edge.infra.ciab.test (ApacheTrafficServer/7.1.4 [uScMsSfWpSeN:t cCMi pSs ])
	< Connection: keep-alive
	<
	<!DOCTYPE html>
	<!-- Licensed to the Apache Software Foundation (ASF) under one
	or more contributor license agreements.  See the NOTICE file
	distributed with this work for additional information
	regarding copyright ownership.  The ASF licenses this file
	to you under the Apache License, Version 2.0 (the
	"License"); you may not use this file except in compliance
	with the License.  You may obtain a copy of the License at

	  http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing,
	software distributed under the License is distributed on an
	"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
	KIND, either express or implied.  See the License for the
	specific language governing permissions and limitations
	under the License. -->
	<html lang="en">
	<head>
		<title>CDN In a Box</title>
		<meta charset="utf-8"/>
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width; height=device-height; initial-scale=1"/>
		<link rel="shortcut-icon" href="/tc_logo.svg"/>
		<meta name="author" content="Apache"/>
		<meta name="creator" content="Apache"/>
		<meta name="publisher" content="Apache"/>
		<meta name="description" content="A simple test origin for Apache Traffic Control"/>
		<style type="text/css">
			html {
				height: 100vh;
				width: 100vw;
			}

			body {
				text-align: center;
				background-image: url(/tc_logo.svg);
				background-color: black;
				background-position: center;
				background-repeat: no-repeat;
				background-size: 25%;
				font-family: "Ubuntu Mono","Consolas",sans-serif;
				color: white;
				margin: 0;
				padding-top: 0.67em;
				max-width: 100%;
			}

			h1 {
				margin-top: 0.67em;
			}

			p {
				text-align: left;
				width: 80vw;
				min-width: 320px;
				margin: auto;
			}
		</style>
	</head>
	<body>
		<h1>Test Origin</h1>
		<p>This is a test "origin" server for Apache Traffic Control</p>
	</body>
	</html>
	* Curl_http_done: called premature == 0
	* Connection #1 to host edge.demo1.mycdn.ciab.test left intact

Test Cases
==========
* Unit tests can be executed using Maven by running ``mvn test`` at the root of the ``traffic_router`` project.
* Unit and Integration tests can be executed using Maven by running ``mvn verify`` at the root of the ``traffic_router`` project.

.. _dev-debugging-unit-tests:

Debugging Unit Tests
--------------------
In order to write tests or understand why a test is failing, a developer may want to debug the unit tests. In order to stop at breakpoints, you should run the tests without forking processes. Even once you have specified for Surefire not to fork tests, the debugging connection will disconnect several times while the tests are running, so you should
- Have JVM act as the debugging client and have your IDE act as the debugging server, even if you are remote debugging
- In your IDE's debugging configuration for the Traffic Router unit tests, enable "auto restart", meaning that, once the debugging client running on JVM has disconnected because a particular test is over, your IDE's debugging server will exit (unavoidable) but will immediately restart in time for the next test.

.. note:: If you run the tests with debugging enabled and with JVM acting as the debugging client but your IDE is not actively listening for debugging connections, the unit tests *will* fail.

Command for running the tests with debugging enabled:

.. code-block:: shell
	:caption: Run the Traffic Router unit tests with debugging enabled

	mvn '-Dmaven.surefire.debug=-agentlib:jdwp=transport=dt_socket,server=n,suspend=n,address=localhost:8000 -DforkCount=0 -DreuseForks=false' test -Djava.library.path=/usr/share/java

Debugging Unit Tests in Docker
------------------------------
In order to run the unit tests in a controlled, well-defined environment, you may prefer to run them from within Docker. A Docker environment for running the Traffic Router unit tests exists in the repository at `/traffic_router/tests <https://github.com/apache/trafficcontrol/tree/master/traffic_router/tests>`_, and it supports debugging. In order to enable debugging, set ``DEBUG_ENABLE`` to ``'true'`` in `docker-compose.yml <https://github.com/apache/trafficcontrol/blob/master/traffic_router/tests/docker-compose.yml>`_. As mentioned in :ref:`dev-debugging-unit-tests`,
- Have your IDE act as the debugging server. In Intellij, this debug mode is called *Listen to remote JVM*.
- Enable *auto-restart* in your IDE's debugging configuration for the Traffic Router unit tests so your IDE doesn't stop listening for connections after the first test ends
- Set the port to 8000 (debugging port is specified in the Dockerfile)

Additionally, you will need to make sure that host.docker.internal resolves to `the Docker host <https://docs.docker.com/docker-for-mac/networking#i-want-to-connect-from-a-container-to-a-service-on-the-host>`_'s IP address (NOT the Docker container's IP address). If you are using Docker Desktop for Mac or Docker Desktop for Windows, this is already set up for you. If you are on Linux, you will need to either figure out how to make host.docker.internal resolve to the ``docker0`` network interface's IP address, or, in ``docker-compose.yml``, change the value of the ``DEBUG_HOST`` environment variable to the IP address of the ``docker0`` interface.

.. note:: If you run the tests with debugging enabled and with JVM (in Docker) acting as the debugging client but your IDE is not actively listening for debugging connections, the unit tests *will* fail.

Once your IDE is listening for debugging connections, start the unit tests:

.. code-block:: shell
	:caption: Run the Traffic Router unit tests in Docker, with or without debugging enabled

	docker-compose up

RPM Packaging
=============
Running ``mvn package`` on a Linux-based distribution will trigger the build process to create the Traffic Router RPM and the Traffic Router ``.war`` file, but will not run the integration tests, so it is a good way to update those artifacts quickly during development. But the preferred way to build the Traffic Router RPMs is by following the instructions in :ref:`dev-building`

API
===

:ref:`tr-api`

.. toctree::
  :hidden:
  :maxdepth: 1

  traffic_router/traffic_router_api
