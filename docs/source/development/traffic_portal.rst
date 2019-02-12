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

**************
Traffic Portal
**************

Introduction
============
Traffic Portal is an `AngularJS 1.x <https://angularjs.org/>`_ client served from a `Node.js <https://nodejs.org/en/>`_ web server designed to consume the Traffic Ops API. Traffic Portal is the official replacement for the legacy Traffic Ops UI.

Software Requirements
=====================
To work on Traffic Portal you need a \*nix (MacOS and Linux are most commonly used) environment that has the following installed:

	* `Ruby Devel 2.0.x or above <https://www.rpmfind.net/linux/rpm2html/search.php?query=ruby-devel>`_
	* `Compass 1.0.x or above <http://compass-style.org/>`_
	* `Node.js 6.0.x or above <https://nodejs.org/en/>`_
	* `Grunt CLI 1.2.0 or above <https://github.com/gruntjs/grunt-cli>`_
	* Access to a working instance of Traffic Ops

.. note:: The Traffic Portal consumes the Traffic Ops API. By default, Traffic Portal assumes Traffic Ops is running on https://localhost:8444. Temporarily modify traffic_portal/conf/config.js if you need to change the location of Traffic Ops.

Traffic Portal Project Tree Overview
=====================================
	* **traffic_control/traffic_portal/app/src** - contains HTML, JavaScript and Sass source files.

Installing The Traffic Portal Developer Environment
===================================================
#. Clone the `Traffic Control Repository <https://github.com/apache/trafficcontrol>`_
#. Navigate to the ``traffic_portal`` subdirectory of your cloned repository.
#. Run ``npm install`` to install client-side and application dependencies into ``traffic_portal/node_modules``. Only needs to be done the first time unless ``traffic_portal/package.json`` changes.
#. Run ``grunt`` to package the application into ``traffic_portal/app/dist``, start a local HTTPS server (Express), and start a file watcher.
#. Valid SSL certificates and keys are needed for Traffic Portal to run. Generate these (e.g. using `this SuperUser answer <https://superuser.com/questions/226192/avoid-password-prompt-for-keys-and-prompts-for-dn-information#answer-226229>`_) and edit the ``traffic_portal/conf/config.json`` file to reflect their names and locations
#. Navigate to https://localhost:8443 (default listening address/port)
