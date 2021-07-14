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

.. role:: bash(code)
	:language: bash

***********
Traffic Ops
***********
Traffic Ops is quite possible the single most complex and most important Traffic Control component. It has many different configuration options that affect a wide range of other components and their interactions.

.. _to-install:

Installing
==========

System Requirements
-------------------
The user must have the following for a successful minimal install:

- CentOS 7 or later
- Two machines - physical or virtual -, each with at least two (v)CPUs, 4GB of RAM, and 20 GB of disk space
- Access to CentOS Base and EPEL :manpage:`yum(8)` repositories
- Access to `The Comprehensive Perl Archive Network (CPAN) <http://www.cpan.org/>`_

Guide
-----
#. Install PostgreSQL Database. For a production install it is best to install PostgreSQL on its own server/virtual machine.

	.. seealso:: For more information on installing PostgreSQL, see `their documentation <https://www.postgresql.org/docs/>`_.

	.. code-block:: shell
		:caption: Example PostgreSQL Install Procedure

		yum update -y
		yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm
		yum install -y postgresql13-server
		su - postgres -c '/usr/pgsql-13/bin/initdb -A md5 -W' #-W forces the user to provide a superuser (postgres) password


#. Edit :file:`/var/lib/pgsql/13/data/pg_hba.conf` to allow the Traffic Ops instance to access the PostgreSQL server. For example, if the IP address of the machine to be used as the Traffic Ops host is ``192.0.2.1`` add the line ``host  all   all     192.0.2.1/32 md5`` to the appropriate section of this file.

#. Edit the :file:`/var/lib/pgsql/13/data/postgresql.conf` file to add the appropriate listen_addresses or ``listen_addresses = '*'``, set ``timezone = 'UTC'``, and start the database

	.. code-block:: shell
		:caption: Starting PostgreSQL with :manpage:`systemd(1)`

		systemctl enable postgresql-13
		systemctl start postgresql-13
		systemctl status postgresql-13 # Prints the status of the PostgreSQL service, to prove it's running


#. Build a :file:`traffic_ops-{version string}.rpm` file using the instructions under the :ref:`dev-building` page - or download a pre-built release from `the Apache Continuous Integration server <https://builds.apache.org/view/S-Z/view/TrafficControl/>`_.

#. Install a PostgreSQL client on the Traffic Ops host

	.. code-block:: shell
		:caption: Installing PostgreSQL Client from a Hosted Source

		yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm

#. Install the Traffic Ops RPM. The Traffic Ops RPM file should have been built in an earlier step.

	.. code-block:: shell
		:caption: Installing a Generated Traffic Ops RPM

		yum install -y ./dist/traffic_ops-3.0.0-xxxx.yyyyyyy.el7.x86_64.rpm

	.. note:: This will install the PostgreSQL client, ``psql`` as a dependency.

#. Login to the Database from the Traffic Ops machine. At this point you should be able to login from the Traffic Ops (hostname ``to`` in the example) host to the PostgreSQL (hostname ``pg`` in the example) host

	.. code-block:: psql
		:caption: Example Login to Traffic Ops Database from Traffic Ops Server

		to-# psql -h pg -U postgres
		Password for user postgres:
		psql (13.2)
		Type "help" for help.

		postgres=#


#. Create the user and database. By default, Traffic Ops will expect to connect as the ``traffic_ops`` user to the ``traffic_ops`` database.

	.. code-block:: console
		:caption: Creating the Traffic Ops User and Database

		to-# psql -U postgres -h pg -c "CREATE USER traffic_ops WITH ENCRYPTED PASSWORD 'tcr0cks';"
		Password for user postgres:
		CREATE ROLE
		to-# createdb traffic_ops --owner traffic_ops -U postgres -h pg
		Password:
		to-#

#. Now, run the following command as the root user (or with :manpage:`sudo(8)`): :file:`/opt/traffic_ops/install/bin/postinstall`. Some additional files will be installed, and then it will proceed with the next phase of the install, where it will ask you about the local environment for your CDN. Please make sure you remember all your answers and verify that the database answers match the information previously used to create the database.

	.. code-block:: console
		:caption: Example Output

		to-# /opt/traffic_ops/install/bin/postinstall
		...

		===========/opt/traffic_ops/app/conf/production/database.conf===========
		Database type [Pg]:
		Database type: Pg
		Database name [traffic_ops]:
		Database name: traffic_ops
		Database server hostname IP or FQDN [localhost]: pg
		Database server hostname IP or FQDN: pg
		Database port number [5432]:
		Database port number: 5432
		Traffic Ops database user [traffic_ops]:
		Traffic Ops database user: traffic_ops
		Password for Traffic Ops database user:
		Re-Enter Password for Traffic Ops database user:
		Writing json to /opt/traffic_ops/app/conf/production/database.conf
		Database configuration has been saved
		===========/opt/traffic_ops/app/db/dbconf.yml===========
		Database server root (admin) user [postgres]:
		Database server root (admin) user: postgres
		Password for database server admin:
		Re-Enter Password for database server admin:
		===========/opt/traffic_ops/app/conf/cdn.conf===========
		Generate a new secret? [yes]:
		Generate a new secret?: yes
		Number of secrets to keep? [10]:
		Number of secrets to keep?: 10
		Not setting up ldap
		===========/opt/traffic_ops/install/data/json/users.json===========
		Administration username for Traffic Ops [admin]:
		Administration username for Traffic Ops: admin
		Password for the admin user:
		Re-Enter Password for the admin user:
		Writing json to /opt/traffic_ops/install/data/json/users.json
		===========/opt/traffic_ops/install/data/json/openssl_configuration.json===========
		Do you want to generate a certificate? [yes]:
		Country Name (2 letter code): US
		State or Province Name (full name): CO
		Locality Name (eg, city): Denver
		Organization Name (eg, company): Super CDN, Inc
		Organizational Unit Name (eg, section):
		Common Name (eg, your name or your server's hostname):
		RSA Passphrase:
		Re-Enter RSA Passphrase:
		===========/opt/traffic_ops/install/data/json/profiles.json===========
		Traffic Ops url [https://localhost]:
		Traffic Ops url: https://localhost
		Human-readable CDN Name.  (No whitespace, please) [kabletown_cdn]: blue_cdn
		Human-readable CDN Name.  (No whitespace, please): blue_cdn
		DNS sub-domain for which your CDN is authoritative [cdn1.kabletown.net]: blue-cdn.supercdn.net
		DNS sub-domain for which your CDN is authoritative: blue-cdn.supercdn.net
		Writing json to /opt/traffic_ops/install/data/json/profiles.json

		... much SQL output skipped

		Starting Traffic Ops
		Restarting traffic_ops (via systemctl):                    [  OK  ]
		Waiting for Traffic Ops to restart
		Success! Postinstall complete.



	.. table:: Explanation of the information that needs to be provided:

		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Field                                              | Description                                                                                    |
		+====================================================+================================================================================================+
		| Database type                                      | This requests the type of database to be used. Answer with the default - 'Pg' to indicate a    |
		|                                                    | PostgreSQL database.                                                                           |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Database name                                      | The name of the database Traffic Ops uses to store the configuration information.              |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Database server hostname IP or FQDN                | The hostname of the database server (``pg`` in the example).                                   |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Database port number                               | The database port number. The default value, 5432, should be correct unless you changed it     |
		|                                                    | during the setup.                                                                              |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Traffic Ops database user                          | The username Traffic Ops will use to read/write from the database.                             |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Password for Traffic Ops                           | The password for the database user that Traffic Ops uses.                                      |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Database server root (admin) user name             | Privileged database user that has permission to create the database and user for Traffic Ops...|
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Database server root (admin) user password         | The password for the privileged database user.                                                 |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Traffic Ops URL                                    | The URL to connect to this instance of Traffic Ops, usually :samp:`https://{Traffic Ops host}` |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Human-readable CDN Name                            | The name of the first CDN which Traffic Ops will be manage.                                    |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| DNS sub-domain for which your CDN is authoritative | The DNS domain that will be delegated to this Traffic Control CDN.                             |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Administration username for Traffic Ops            | The Administration (highest privilege) Traffic Ops user to create. Use this user to login      |
		|                                                    | for the first time and create other users.                                                     |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+
		| Password for the admin user                        | The password for the administrative Traffic Ops user.                                          |
		+----------------------------------------------------+------------------------------------------------------------------------------------------------+

.. deprecated:: ATCv6
	The postinstall script is now written in Python. If you run into issues with the postinstall script, you are encouraged to file an issue at https://github.com/apache/trafficcontrol/issues/new/choose. The original Perl postinstall script is deprecated and will be removed in a future ATC release. To use the deprecated version anyway, run ``/opt/traffic_ops/install/bin/_postinstall.pl`` directly instead of ``/opt/traffic_ops/install/bin/postinstall``.

The postinstall script can also be run non-interactively using :atc-file:`traffic_ops/install/bin/input.json`. To use it, first change the values to match your environment, then pass it to the ``postinstall`` script:
	.. code-block:: console
		:caption: Postinstall in Automatic (-a) mode

		/opt/traffic_ops/install/bin/postinstall -a --cfile /opt/traffic_ops/install/bin/input.json

.. deprecated:: ATCv6
	Once the Perl script is removed, the values in ``input.json`` for the ``"hidden"`` properties will be changed from ``"1"`` and ``"0"`` to ``true`` and ``false``.

.. _to-upgrading:

Upgrading
=========
To upgrade from older Traffic Ops versions, stop the service, use :manpage:`yum(8)` to upgrade to the latest available Traffic Ops package, and use the :ref:`admin <database-management>` tool to perform the database upgrade.

.. tip:: In order to upgrade to the latest version of Traffic Ops, please be sure that you have first upgraded to the latest available minor or patch version of your current release. For example, if your current Traffic Ops version is 3.0.0 and version 3.1.0 is available, you must first upgrade to 3.1.0 before proceeding to upgrade to 4.0.0. (Specifically, this means running all migrations, :file:`traffic_ops/app/db/seeds.sql`, and :file:`traffic_ops/app/db/patches.sql` for the latest of your current major version - which should be handled by the :program:`admin` tool). The latest migration available before the release of 4.0.0 (pending at the time of this writing) was :file:`traffic_ops/app/db/migrations/20180814000625_remove_capabilities_for_reseed.sql`, so be sure that migrations up to this point have been run before attempting to upgrade Traffic Ops.

.. seealso:: :ref:`database-management` for more details about :program:`admin`.

.. code-block:: shell
	:caption: Sample Script for Upgrading Traffic Ops

	systemctl stop traffic_ops
	yum upgrade traffic_ops
	pushd /opt/traffic_ops/app/
	./db/admin --env production upgrade
	popd

After this completes, see Guide_ for instructions on running the :program:`postinstall` script. Once the :program:`postinstall` script, has finished, run the following command as the root user (or with :manpage:`sudo(8)`): ``systemctl start traffic_ops`` to start the service.

Upgrading to 6.0
----------------

As of Apache Traffic Control 6.0, Traffic Ops supports PostgreSQL version 13.2. In order to migrate from the prior PostgreSQL version 9.6, it is recommended to use the `pg_upgrade <https://www.postgresql.org/docs/13/pgupgrade.html>`_ tool.

.. _to-running:

Running
=======
While this section contains instructions for running Traffic Ops manually, the only truly supported method is via :manpage:`systemd(8)`, e.g. :bash:`systemctl start traffic_ops` (this method starts the program properly and uses its default configuration file locations).

.. program:: traffic_ops

traffic_ops_golang
------------------
``traffic_ops_golang [--version] [--plugins] [--api-routes] --cfg CONFIG_PATH --dbcfg DB_CONFIG_PATH [--riakcfg RIAK_CONFIG_PATH]``

.. option:: --cfg CONFIG_PATH

	This **mandatory** command line flag specifies the absolute or relative path to the configuration file to be used by Traffic Ops - `cdn.conf`_.

.. option:: --dbcfg DB_CONFIG_PATH

	This **mandatory** command line flag specifies the absolute or relative path to a configuration file used by Traffic Ops to establish connections to the PostgreSQL database - `database.conf`_

.. option:: --plugins

	List the installed plugins and exit.

.. option:: --api-routes

	Print information about all API routes and exit. If also used with the :option:`--cfg` option, also print out the configured routing blacklist information from `cdn.conf`_.

.. option:: --riakcfg RIAK_CONFIG_PATH

	.. deprecated:: 6.0
		This optional command line flag specifies the absolute or relative path to a configuration file used by Traffic Ops to establish connections to Riak when used as the Traffic Vault backend - `riak.conf`_. Please use ``"traffic_vault_backend": "riak"`` and ``"traffic_vault_config": {...}`` (with the contents of `riak.conf`_) instead.

	.. impl-detail:: The name of this flag is derived from the current database used in the implementation of Traffic Vault - `Riak KV <https://riak.com/products/riak-kv/index.html>`_.

.. option:: --version

	Print version information and exit.

Configuring
===========
:program:`traffic_ops_golang` uses several configuration files, but the most important of these is `cdn.conf`_.

Configuration Files
-------------------

.. _cdn.conf:

cdn.conf
""""""""
This file deals with the configuration parameters of running Traffic Ops itself. It is a JSON-format set of options and their respective values. `traffic_ops_golang`_ will use whatever file is specified by its :option:`--cfg` option. The keys of the file are described below.

:acme_accounts: This is an optional array of objects to define ref:`external_account_binding` information to an existing :abbr:`ACME (Automatic Certificate Management Environment)` account.  The `acme_provider` and `user_email` combination must be unique.

	.. versionadded:: 5.1

	:acme_provider: The certificate provider. This field needs to correlate to the AuthType field for each certificate so the renewal functionality knows which provider to use.
	:user_email:    The email used to set up the account with the provider.
	:acme_url:      The URL for the :abbr:`ACME (Automatic Certificate Management Environment)`.
	:kid:           The key ID provided by the :abbr:`ACME (Automatic Certificate Management Environment)` provider for ref:`external_account_binding`.
	:hmac_encoded:  The :abbr:`HMAC (Hashed Message Authentication Code)` key provided by the :abbr:`ACME (Automatic Certificate Management Environment)` provider for ref:`external_account_binding`. This should be in Base64 URL encoded.

:acme_renewal: This object contains the information for the automatic renewal script for certificates.

	.. versionadded:: 5.1

	:renew_days_before_expiration: Set the number of days before expiration date to renew certificates.
	:summary_email: The email address to use for summarizing certificate expiration and renewal status. If it is blank, no email will be sent.

:geniso: This object contains configuration options for system ISO generation.

	:iso_root_path: Sets the filesystem path to the root of the ISO generation directory. For default installations, this should usually be set to :file:`/opt/traffic_ops/app/public`.

	.. deprecated:: ATCv6
		The ``geniso.iso_root_path`` configuration option is unused now that Traffic Ops is rewritten from Perl to Golang and will be removed in a future ATC release.

	.. seealso:: :ref:`tp-tools-generate-iso`

:hypnotoad: This is a group of options that mainly no longer have any meaning..

	:group:             Serves no known purpose anymore.
	:heartbeat_timeout: Serves no known purpose anymore.
	:listen:            This must always be an array containing a single string. This very odd string is apparently a URL. The scheme of the URL doesn't matter, as `traffic_ops_golang`_ ignores that and always uses HTTPS. The host (and optionally port) of the URL used to have a purpose, but no longer does. The "cert" query parameter sets the location of the SSL certificate to use for encrypting connections, while the "key" query parameter is the certificate's corresponding private key. The default configuration file also has the "verify" query parameter which serves no known purpose anymore.
	:pid_file:          Serves no known purpose anymore.
	:user:              Serves no known purpose anymore.
	:workers:           Serves no known purpose anymore.

:inactivity_timeout: Serves no known purpose anymore.
:influxdb_conf_path: An optional field which gives `traffic_ops_golang`_ the absolute or relative path to an `influxdb.conf`_ file. Default if not specified is to first check if the :envvar:`MOJO_MODE` environment variable is set. If it is, then Traffic Ops will look in the current working directory for a subdirectory named ``conf/``, then inside that for a subdirectory with the name that is the value of the :envvar:`MOJO_MODE` variable, and inside that directory for a file named ``influxdb.conf``. If :envvar:`MOJO_MODE` is *not* set, then Traffic Ops will look for a file named ``influxdb.conf`` in the same directory as this ``cdn.conf`` file.

	.. versionadded:: 4.0

	.. warning:: While relative paths are allowed, they are discouraged, as the path will be relative to the working directory of the `traffic_ops_golang`_ process itself, not relative to the ``cdn.conf`` configuration file, which can be confusing.

:ldap_conf_location: An optional field which gives `traffic_ops_golang`_ the absolute or relative path to an `ldap.conf`_ file. Default if not specified is a file named ``ldap.conf`` in the same directory as this ``cdn.conf`` file.

	.. warning:: While relative paths are allowed, they are discouraged, as the path will be relative to the working directory of the `traffic_ops_golang`_ process itself, not relative to the ``cdn.conf`` configuration file, which can be confusing.

:lets_encrypt:

	.. versionadded:: 4.1

	:user_email: A required email address to create an account with Let's Encrypt or to receive expiration updates. If this is not included then `rate limits <https://letsencrypt.org/docs/rate-limits>`_ may apply for the number of certificates.
	:send_expiration_email: A boolean option to send email summarizing certificate expiration status

		.. deprecated:: 5.1
			Future versions of Traffic Ops will not support this legacy configuration option, see acme_renewal: { summary_email: <string> } instead.

	:convert_self_signed: A boolean option to convert self signed to Let's Encrypt certificates as they expire. This only works for certificates labeled as Self Signed in the Certificate Source field.
	:renew_days_before_expiration: Set the number of days before expiration date to renew certificates.

		.. deprecated:: 5.1
			Future versions of Traffic Ops will not support this legacy configuration option, see acme_renewal: { renew_days_before_expiration: <int> } instead.

	:environment: This specifies which Let's Encrypt environment to use: 'staging' or 'production'. It defaults to 'production'.

:portal: This section provides information regarding a connected UI with which users interact, so that emails can include links to it.

	:base_url: This URL should be the root and/or landing page of the UI. For Traffic Portal instances, this should include the fragment part of the URL, e.g. ``https://trafficportal.infra.ciab.test/#!/``.
	:docs_url: The actual use of this URL is unknown, but supposedly it ought to point to the documentation for the Traffic Control instance. It's hard to imagine a fantastic reason this shouldn't just always be https://traffic-control-cdn.readthedocs.io
	:email_from: Most emails sent from the Traffic Ops server will use ``to.email_from``, but specifically password reset requests (which contain a link to a fragment under ``portal.base_url``) will instead use this as the value of their :mailheader:`From` field.
	:pass_reset_path: A path to be added to ``base_url`` that is the URL of the UI's password reset interface. For Traffic Portal instances, this should always be set to "user".
	:user_register_path: A path to be added to ``base_url`` that is the URL of the UI's new user registration interface. For Traffic Portal instances, this should always be set to "user".

:secrets: This is an array of strings, which cannot be empty. The first secret in the array is used to encrypt Traffic Ops authentication cookies - multiple Traffic Ops instances serving the same CDN need to share secrets in order for users logged into one to be able to use their cookie as authentication with other instances.
:smtp:    This optional section contains options for connecting to and authenticating with an :abbr:`SMTP (Simple Mail Transfer Protocol)` server for sending emails. If this section is undefined (or if ``enabled`` is explicitly ``false``), Traffic Ops will not be able to send emails and certain :ref:`to-api` endpoints that depend on that functionality will fail to operate.

	.. versionadded:: 4.0

	:address:  This is the address of the :abbr:`SMTP (Simple Mail Transfer Protocol)` which will be used to send emails. Should include the port number, e.g. ``"localhost:25"`` for :manpage:`sendmail(8)` on the Traffic Ops server.
	:enabled:  A boolean flag that determines whether or not connection to an :abbr:`SMTP (Simple Mail Transfer Protocol)` ought to be allowed. Whatever the settings of the other fields in the ``smtp`` object, email cannot and will not be sent if this is ``false``.
	:password: The password to be used when authenticating with the :abbr:`SMTP (Simple Mail Transfer Protocol)` server.
	:user:     The name of the user to be used when authenticating with the :abbr:`SMTP (Simple Mail Transfer Protocol)` server.

	.. Note:: The SMTP integration currently only supports Login Auth.

:to: Contains information to identify Traffic Ops in a network sense.

	:base_url:             This field is used to identify the location for the now-removed Traffic Ops UI. It no longer serves any purpose.
	:email_from:           Sets the address that will appear in the :mailheader:`From` field of Emails sent by Traffic Ops.
	:no_account_found_msg: When a password reset is requested for an email address not registered to any known user, this is the message that will be sent to that email address.

:traffic_ops_golang: This group configuration options is used exclusively by `traffic_ops_golang`_.

	:crconfig_emulate_old_path: An optional boolean that controls the value of a part of :term:`Snapshots` that report what :ref:`to-api` endpoint is used to generate :term:`Snapshots`. If this is ``true``, it forces Traffic Ops to report that a legacy, deprecated endpoint is used, whereas if it's ``false`` Traffic Ops will report the actual, current endpoint. Default if not specified is ``false``.

		.. deprecated:: 3.0
			Future versions of Traffic Ops will not support this legacy configuration option, and will always report the current endpoint.

	:crconfig_snapshot_use_client_request_host: An optional boolean which controls the value of the Traffic Ops server's URL as inserted into :term:`Snapshots`. If this is ``true``, then the value used will be taken from the :mailheader:`Host` header of the request that generated the :term:`Snapshot`. If it's ``false``, then it will instead use the value of the global "tm.url" :term:`Parameter`. Default if not specified is ``false``.

		.. deprecated:: 3.0
			Future versions of Traffic Ops will not support this legacy configuration option, and will always use the global "tm.url" :term:`Parameter`.

	:db_conn_max_lifetime_seconds: An optional field that sets the maximum lifetime in seconds of any given connection to the Traffic Ops Database. If set to zero, connections are held open until explicitly closed. Default if not specified is the value of `DBConnMaxLifetimeSecondsDefault <https://pkg.go.dev/github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config#pkg-constants>`_.
	:db_max_idle_connections: An optional limit on the number of connections to the Traffic Ops Database to keep alive while idle. If this is less than ``max_db_connections``, that number will be used instead - *even if this field is unset and using its default*. Default if not specified is the value of `DBMaxIdleConnectionsDefault <https://pkg.go.dev/github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config#pkg-constants>`_.
	:db_query_timeout_seconds: An optional field specifying a timeout on database *transactions* (not actually single queries in most cases) within API route handlers. Effectively this is a timeout on a single handler's ability to interact with the Traffic Ops Database. Default if not specified is the value of `DefaultDBQueryTimeoutSecs <https://pkg.go.dev/github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config#pkg-constants>`_.
	:idle_timeout: An optional timeout in seconds for idle client connections to Traffic Ops. If set to zero, the value of ``read_timeout`` will be used instead. If both are zero, then the value of ``read_header_timeout`` will be used. If all three fields are zero, there is no timeout and connections will be kept alive indefinitely - **not** recommended. Default if not specified is zero.
	:insecure: An optional boolean which, if set to ``true`` will cause Traffic Ops to skip verification of client certificates whenever necessary/possible. If set to ``false``, the normal verification behavior is exhibited. Default if not specified is ``false``.

		.. deprecated:: 5.0
			Future versions of Traffic Ops will not support this legacy configuration option, see tls_config: { InsecureSkipVerify: <bool> } instead

	:log_location_debug: This optional field, if specified, should either be the location of a file to which debug-level output will be logged, or one of the special strings ``"stdout"`` which indicates that STDOUT should be used, ``"stderr"`` which indicates that STDERR should be used or ``"null"`` which indicates that no output of this level should be generated. An empty string (``""``) and literally ``null`` are equivalent to ``"null"``. Default if not specified is ``"null"``.
	:log_location_error: This optional field, if specified, should either be the location of a file to which error-level output will be logged, or one of the special strings ``"stdout"`` which indicates that STDOUT should be used, ``"stderr"`` which indicates that STDERR should be used or ``"null"`` which indicates that no output of this level should be generated. An empty string (``""``) and literally ``null`` are equivalent to ``"null"``. Default if not specified is ``"null"``. This field is also used to determine where server profiling statistics are written. Assuming ``profiling_enabled`` is ``true`` and ``profiling_location`` is unset, if this field's value is given as a path to a regular file, a file named :file:`profiling` will be written to the same directory containing the profiling information - overwriting any existing files by that name.
	:log_location_event: This optional field, if specified, should either be the location of a file to which event-level output will be logged, or one of the special strings ``"stdout"`` which indicates that STDOUT should be used, ``"stderr"`` which indicates that STDERR should be used or ``"null"`` which indicates that no output of this level should be generated. An empty string (``""``) and literally ``null`` are equivalent to ``"null"``. Default if not specified is ``"null"``.
	:log_location_info: This optional field, if specified, should either be the location of a file to which informational-level output will be logged, or one of the special strings ``"stdout"`` which indicates that STDOUT should be used, ``"stderr"`` which indicates that STDERR should be used or ``"null"`` which indicates that no output of this level should be generated. An empty string (``""``) and literally ``null`` are equivalent to ``"null"``. Default if not specified is ``"null"``.
	:log_location_warning: This optional field, if specified, should either be the location of a file to which warning-level output will be logged, or one of the special strings ``"stdout"`` which indicates that STDOUT should be used, ``"stderr"`` which indicates that STDERR should be used or ``"null"`` which indicates that no output of this level should be generated. An empty string (``""``) and literally ``null`` are equivalent to ``"null"``. Default if not specified is ``"null"``.
	:cache_ms: Milliseconds to cache requests from the database in memory, for performance. This should generally be small, values above 5 seconds are discouraged. Set 0 to disable the cache. Default if not set is 1000ms.
	:disable_read_while_writer: Disable read-while-writer functionality. Should generally only be set if issues are observed around read-while-write or request caching. Default if not set is false.
	:max_db_connections: An optional limit on the number of allowed concurrent connections to the Traffic Ops Database. If it is less than or equal to zero, there is no limit. Default if not specified is zero.
	:oauth_client_secret: An optional secret string to be shared with OAuth-capable clients attempting to authenticate via OAuth. The default behavior if this is not defined - or is an empty string (``""``) or ``null`` is to disallow authentication via OAuth.

		.. warning:: OAuth support in Traffic Ops is still in its infancy, so most users are advised to avoid defining this field without good cause.

	:plugins: An optional array of enabled plugin names. These names must be unique. Note that a plugin that is installed will not be used unless its name appears in this list - thus "enabling" it. If not specified no plugins will be enabled.
	:plugin_config: This optional object maps plugin names - which **must** appear in the ``plugins`` array - to arbitrary JSON configurations for said plugins. It is up to the plugins themselves to parse these configurations. The default if not specified is no configuration information, somewhat obviously.
	:plugin_shared_config: This optional object is just an arbitrary JSON object that is converted into a native object and made available to any and all loaded and enabled plugins. A typical use-case for this field is avoiding repetition of identical configuration in ``plugin_config``. The default if not specified is ``null``.
	:port: Sets the port on which Traffic Ops will listen for incoming connections.
	:profiling_enabled: An optional boolean which, if ``true`` will enable the gathering of profiling statistics on the Traffic Ops server. Default if not specified is ``false``.
	:profiling_location: An optional string which, if set, should be the absolute path (relative paths are allowed but not recommended) to a file where profiling statistics for the Traffic Ops server will be written. If ``profiling_enabled`` is ``true`` but this is not specified, or is an empty string (``""``) or ``null``, then a file named "profiling" will be created or overwritten in the same directory as the file specified in ``log_location_error``. If that file is not a regular file, then Traffic ops will instead create a temporary directory and write profiling statistics to a file named "profiling" within that directory.
	:proxy_keep_alive: Serves no known purpose anymore.
	:proxy_read_handler_timeout: Serves no known purpose anymore.
	:proxy_timeout: Serves no known purpose anymore.
	:proxy_tls_timeout: Serves no known purpose anymore.
	:read_header_timeout: An optional timeout in seconds before which Traffic Ops must be able to finish reading the headers of an incoming request or it will drop the connection. If set to zero, there is no timeout. Default if not specified is zero.
	:read_timeout: An optional timeout in seconds before which Traffic Ops must be able to finish reading an entire incoming request (including body) or it will drop the connection. If set to zero, there is no timeout. Default if not specified is zero.
	:request_timeout: An optional timeout in seconds that serves as the maximum time each Traffic Ops middleware can take to execute. If it is exceeded, the text "server timed out" is served in place of a response. If set to :code:`0`, :code:`60` is used instead. Default if not specified is :code:`60`.
	:riak_port: An optional field that sets the port on which Traffic Ops will try to contact Traffic Vault for storage and retrieval of sensitive encryption keys.

		.. deprecated:: 6.0
			Please use a ``"port"`` field in ``traffic_vault_config`` instead when using ``"traffic_vault_backend": "riak"``.

		.. impl-detail:: The name of this field is derived from the current database used in the implementation of Traffic Vault - `Riak KV <https://riak.com/products/riak-kv/index.html>`_.


	:whitelisted_oauth_url: An optional array of URLs which are allowed to authenticate Traffic Ops users via OAuth. The default behavior if this field is not defined is to not allow OAuth authentication.

		.. warning:: OAuth support in Traffic Ops is still in its infancy, so most users are advised to avoid defining this field without good cause.

	:write_timeout: An optional timeout in seconds set on handlers. After reading a request's header, the server will have this long to send back a response. If set to zero, there is no timeout. Default if not specified is zero.

	:traffic_vault_backend:

		.. versionadded:: 6.0
			Optional. The name of which backend to use for Traffic Vault. Currently, the only supported backend is "riak".

	:traffic_vault_config:

		.. versionadded:: 6.0
			Optional. The JSON configuration which is unique to the chosen Traffic Vault backend. See :ref:`traffic_vault_admin` for the configuration options for each supported backend.

	.. _admin-routing-blacklist:

	:routing_blacklist: Optional configuration for explicitly disabling any routes via ``disabled_routes``.

		.. versionadded:: 4.0

		:perl_routes: Serves no known purpose anymore.

			.. deprecated:: 6.0
				This was used back when Traffic Ops was still in the process of being rewritten from Perl. It serves no purpose anymore, and will be removed in the future.

		:disabled_routes: A list of API route IDs to disable. Requests matching these routes will receive a 503 response. To find the route ID for a given path you would like to disable, run ``./traffic_ops_golang`` using the :option:`--api-routes` option to view all the route information, including route IDs and paths.
		:ignore_unknown_routes: If ``false`` (default) return an error and prevent startup if unknown route IDs are found. Otherwise, log a warning and continue startup.

	:tls_config: An optional stanza for TLS configuration. The values of which conform to the :godoc:`crypto/tls.Config` structure.

:use_ims:

	.. versionadded:: 5.0
		This is an optional boolean value to enable the handling of the "If-Modified-Since" HTTP request header. Default: false

Example cdn.conf
''''''''''''''''
.. include:: ../../../traffic_ops/app/conf/cdn.conf
	:code: json
	:tab-width: 4

database.conf
"""""""""""""
This file deals with configuration of the Traffic Ops Database; in particular it tells Traffic Ops how to connect with the database for its current environment. `traffic_ops_golang`_ will read this file in from the path pointed to by its :option:`--dbcfg` flag. ``database.conf`` is encoded as a JSON object, and its keys are described below.

:dbname: The name of the PostgreSQL database used. Typically different databases are used for different environments, e.g. "trafficops_test", "trafficops", etc. Many environments choose to use ``traffic_ops``.
:description: An optional, human friendly description of the database. Generally this should just describe the purpose of the database e.g. "This database is used for integration testing with our toolset".
:hostname: The hostname (:abbr:`FQDN (Fully Qualified Domain Name)`) of the server that runs the Traffic Ops Database.
:password: The password to use when authenticating with the Traffic Ops database. In a typical install process, the ``postinstall`` script will ask for a password to use for this connection, and this should match that.
:port: The port number (as a string) on which the Traffic Ops Database is listening for incoming connections. `traffic_ops_golang`_ ignores this and always uses the default PostgreSQL port (5432).
:ssl: A boolean that sets whether or not the Traffic Ops Database encrypts its connections with SSL.
:type: A string that gives the "type" of database pointed to by all the other options. Once upon a time it was possible for this to either be "mysql" or "postgres", but the only valid value anymore is "postgres" - and `traffic_ops_golang`_ ignores this field entirely (and in fact doesn't even care if it's defined at all) and only supports "postgres" databases.
:user: The name of the user as whom to connect to the database. In a typical install process, the ``postinstall`` script will ask for the name of a user to set up for the Traffic Ops Database, and this should match that. Many environments choose to use ``traffic_ops``.

Example database.conf
'''''''''''''''''''''
.. include:: ../../../traffic_ops/app/conf/production/database.conf
	:code: json
	:tab-width: 4

influxdb.conf
"""""""""""""
This file deals with configuration of the InfluxDB cluster that serves Traffic Stats; specifically it tells Traffic Ops how to authenticate with the InfluxDB cluster and which measurements to check. `traffic_ops_golang`_ will look for this file at the path given by the value of ``influx_db_conf_path`` in `cdn.conf`_. This file is encoded as a JSON object, and its keys are described below.

.. seealso:: For more information about InfluxDB, see `the InfluxDB documentation <https://docs.influxdata.com/influxdb/v1.7/>`_.

:cache_stats_db_name: This field sets the name of the "database" (measurement) used to query for :term:`Cache Group` statistics. `traffic_ops_golang`_ will default to ``"cache_stats"`` if this field is not defined. It is recommended that this field not be defined.

	.. danger:: The **only** valid value for this is ``"cache_stats"``, if it is anything else Traffic Stats data for :term:`Cache Group` statistics will be inaccessible through the :ref:`to-api`.

:deliveryservice_stats_db_name: This field sets the name of the "database" (measurement) used to query for :term:`Delivery Service` statistics. `traffic_ops_golang`_ will default to ``"deliveryservice_stats"`` if this field is not defined. It is recommended that this field not be defined.

	.. danger:: The **only** valid value for this is ``"deliveryservice_stats"``, if it is anything else Traffic Stats data for :term:`Delivery Service` statistics will be inaccessible through the :ref:`to-api`.

:password: Sets the password to use when authenticating with InfluxDB clusters.
:secure: An optional boolean that sets whether or not to use SSL encrypted connections to the InfluxDB cluster (the InfluxDB servers would need to be configured to use SSL). Default if not specified is ``false``.
:user: Sets the user name as whom to authenticate with InfluxDB clusters.

Example influxdb.conf
'''''''''''''''''''''
.. include:: ../../../traffic_ops/app/conf/production/influxdb.conf
	:code: json
	:tab-width: 4

ldap.conf
"""""""""
This file defines methods of connection to an :abbr:`LDAP (Lightweight Directory Access Protocol)` server and semantics for searching for users on it for the purpose of authentication. `traffic_ops_golang`_ will look for this file at the path given by the value of ``ldap_conf_location`` in `cdn.conf`_. ``ldap.conf``'s contents are a JSON-encoded object, the keys of which are detailed below.

.. seealso:: For more information on :abbr:`LDAP (Lightweight Directory Access Protocol)` see `the LDAP Wikipedia page <https://en.wikipedia.org/wiki/Lightweight_Directory_Access_Protocol>`_ and :rfc:`4511`.

:admin_dn: The :abbr:`LDAP (Lightweight Directory Access Protocol)` :abbr:`DN (Distinguished Name)` of the administrative user.
:admin_pass: The password of the administrative user for the :abbr:`LDAP (Lightweight Directory Access Protocol)`.
:host: The full hostname of the LDAP server, preceded by a scheme (only ``ldap://`` and ``ldaps://`` are supported), optionally including port number.
:insecure: A boolean that tells Traffic Ops whether or not to verify the certificate chain of the :abbr:`LDAP (Lightweight Directory Access Protocol)` server if it uses TLS-encrypted communications.
:ldap_timeout_secs: Sets a timeout in seconds for connections to the :abbr:`LDAP (Lightweight Directory Access Protocol)`.
:search_base: The directory relative to which searches for users should be conducted.
:search_query: A query to be used to search for users. The string ``%s`` should appear exactly once in this string, where user names will be inserted procedurally by the handler for :abbr:`LDAP (Lightweight Directory Access Protocol)` logins.

Example ldap.conf
'''''''''''''''''
.. include:: ../../../traffic_ops/app/conf/example-ldap.conf
	:code: json
	:tab-width: 4

riak.conf
"""""""""
.. deprecated:: 6.0
	The ``riak.conf`` configuration file and associated :option:`--riakcfg` flag have been deprecated and will be removed from Traffic Control in the future. Please use ``"traffic_vault_backend": "riak"`` and put the existing contents of ``riak.conf`` into ``"traffic_vault_config": {...}`` in `cdn.conf`_ instead.

This file sets authentication options for connections to Riak when used as the Traffic Vault backend. `traffic_ops_golang`_ will look for this file at the path given by the value of the :option:`--riakcfg` flag as passed on startup. The contents of ``riak.conf`` are encoded as a JSON object, the keys of which are described in :ref:`traffic_vault_riak_backend`.

.. impl-detail:: The name of this file is derived from the current database used in the implementation of Traffic Vault - `Riak KV <https://riak.com/products/riak-kv/index.html>`_.

Installing the SSL Certificate
------------------------------
By default, Traffic Ops runs as an SSL web server (that is, over HTTPS), and a certificate needs to be installed.

Self-signed Certificate (Development)
"""""""""""""""""""""""""""""""""""""
.. code-block:: console
	:caption: Example Procedure

	$ openssl genrsa -des3 -passout pass:x -out localhost.pass.key 2048
	Generating RSA private key, 2048 bit long modulus
	...
	$ openssl rsa -passin pass:x -in localhost.pass.key -out localhost.key
	writing RSA key
	$ rm localhost.pass.key

	$ openssl req -new -key localhost.key -out localhost.csr
	You are about to be asked to enter information that will be incorporated
	into your certificate request.
	What you are about to enter is what is called a Distinguished Name or a DN.
	There are quite a few fields but you can leave some blank
	For some fields there will be a default value,
	If you enter '.', the field will be left blank.
	-----
	Country Name (2 letter code) [XX]:US<enter>
	State or Province Name (full name) []:CO<enter>
	Locality Name (eg, city) [Default City]:Denver<enter>
	Organization Name (eg, company) [Default Company Ltd]: <enter>
	Organizational Unit Name (eg, section) []: <enter>
	Common Name (eg, your name or your server's hostname) []: <enter>
	Email Address []: <enter>

	Please enter the following 'extra' attributes
	to be sent with your certificate request
	A challenge password []: pass<enter>
	An optional company name []: <enter>
	$ openssl x509 -req -sha256 -days 365 -in localhost.csr -signkey localhost.key -out localhost.crt
	Signature ok
	subject=/C=US/ST=CO/L=Denver/O=Default Company Ltd
	Getting Private key
	$ sudo cp localhost.crt /etc/pki/tls/certs
	$ sudo cp localhost.key /etc/pki/tls/private
	$ sudo chown trafops:trafops /etc/pki/tls/certs/localhost.crt
	$ sudo chown trafops:trafops /etc/pki/tls/private/localhost.key

Certificate from Certificate Authority (Production)
"""""""""""""""""""""""""""""""""""""""""""""""""""

.. Note:: You will need to know the appropriate answers when generating the certificate request file :file:`trafficopss.csr`.

.. code-block:: console
	:caption: Example Procedure

	$ openssl genrsa -des3 -passout pass:x -out trafficops.pass.key 2048
	Generating RSA private key, 2048 bit long modulus
	...
	$ openssl rsa -passin pass:x -in trafficops.pass.key -out trafficops.key
	writing RSA key
	$ rm localhost.pass.key

Generate the :abbr:`CSR (Certificate Signing Request)` file needed for :abbr:`CA (Certificate Authority)` request

.. code-block:: console
	:caption: Example Certificate Signing Request File Generation

	$ openssl req -new -key trafficops.key -out trafficops.csr
	You are about to be asked to enter information that will be incorporated
	into your certificate request.
	What you are about to enter is what is called a Distinguished Name or a DN.
	There are quite a few fields but you can leave some blank
	For some fields there will be a default value,
	If you enter '.', the field will be left blank.
	-----
	Country Name (2 letter code) [XX]: <enter country code>
	State or Province Name (full name) []: <enter state or province>
	Locality Name (eg, city) [Default City]: <enter locality name>
	Organization Name (eg, company) [Default Company Ltd]: <enter organization name>
	Organizational Unit Name (eg, section) []: <enter organizational unit name>
	Common Name (eg, your name or your server's hostname) []: <enter server's hostname name>
	Email Address []: <enter e-mail address>

	Please enter the following 'extra' attributes
	to be sent with your certificate request
	A challenge password []: <enter challenge password>
	An optional company name []: <enter>
	$ sudo cp trafficops.key /etc/pki/tls/private
	$ sudo chown trafops:trafops /etc/pki/tls/private/trafficops.key

You must then take the output file :file:`trafficops.csr` and submit a request to your :abbr:`CA (Certificate Authority)`. Once you get approved and receive your :file:`trafficops.crt` file

.. code-block:: shell
	:caption: Certificate Installation

	sudo cp trafficops.crt /etc/pki/tls/certs
	sudo chown trafops:trafops /etc/pki/tls/certs/trafficops.crt

If necessary, install the :abbr:`CA (Certificate Authority) certificate's ``.pem`` and ``.crt`` files in ``/etc/pki/tls/certs``.

You will need to update `cdn.conf`_ with any necessary changes.

.. code-block:: text
	:caption: Sample 'listen' Line When Path to ``trafficops.crt`` and ``trafficops.key`` are Known

	'hypnotoad' => ...
		'listen' => 'https://[::]:443?cert=/etc/pki/tls/certs/trafficops.crt&key=/etc/pki/tls/private/trafficops.key&ca=/etc/pki/tls/certs/localhost.ca&verify=0x00&ciphers=AES128-GCM-SHA256:HIGH:!RC4:!MD5:!aNULL:!EDH:!ED'
		...

.. _admin-to-ext-script:

Managing Traffic Ops Extensions
===============================
Traffic Ops supports "`Check Extensions`_", which are analytics scripts that collect and display information as columns in the table under :menuselection:`Monitor --> Cache Checks` in Traffic Portal.

.. seealso:: Traffic Ops also supports a more involved type of extension in the form of :ref:`to_go_plugins`.

.. |checkmark| image:: images/good.png
.. |X| image:: images/bad.png

.. _to-check-ext:

Check Extensions
----------------
Check Extensions are scripts that, after registering with Traffic Ops, have a column reserved in the :menuselection:`Monitor --> Cache Checks` view and usually run periodically using :manpage:`cron(8)`. Each extension is a separate executable located in :file:`{$TO_HOME}/bin/checks/` on the Traffic Ops server (though all of the default extensions are written in Perl, this is in *no way* a requirement; they can be any valid executable). The currently registered extensions can be listed by running ``/opt/traffic_ops/app/bin/extensions -a``. Some extensions automatically registered with the Traffic Ops database (``to_extension`` table) at install time (see :atc-file:`traffic_ops/app/db/seeds.sql`). However, :manpage:`cron(8)` must still be configured to run these checks periodically. The extensions are called like so:

.. code-block:: shell
	:caption: Example Check Extension Call

	$TO_HOME/bin/checks/<name>  -c "{\"base_url\": \",https://\"<traffic_ops_ip>\", \"check_name\": \"<check_name>\"}" -l <log level>

:name: The basename of the extension executable
:traffic_ops_ip: The IP address or :abbr:`FQDN (Fully Qualified Domain Name)` of the Traffic Ops server
:check_name: The name of the check e.g. ``CDU``, ``CHR``, ``DSCP``, ``MTU``, etc...
:log_level: A whole number between 1 and 4 (inclusive), with 4 being the most verbose. Implementation of this field is optional

It is the responsibility of the check extension script to iterate over the servers it wants to check and post the results. An example script might proceed by logging into the Traffic Ops server using the HTTPS ``base_url`` provided on the command line. The script is hard-coded with an authentication token that is also provisioned in the Traffic Ops User database. This token allows the script to obtain a cookie used in later communications with the Traffic Ops API. The script then obtains a list of all :term:`cache server`\ s to be polled by accessing :ref:`to-api-servers`. This list is then iterated, running a command to gather the stats from each server. For some extensions, an HTTP ``GET`` request might be made to the :abbr:`ATS (Apache Traffic Server)` ``astats`` plugin, while for others the server might be pinged, or a command might run over :manpage:`ssh(1)`. The results are then compiled into a numeric or boolean result and the script submits a ``POST`` request containing the result back to Traffic Ops using :ref:`to-api-servercheck`. A check extension can have a column of |checkmark|'s and |X|'s (CHECK_EXTENSION_BOOL) or a column that shows a number (CHECK_EXTENSION_NUM).

Check Extensions Installed by Default
"""""""""""""""""""""""""""""""""""""
:abbr:`CDU (Cache Disk Usage)`
	This check shows how much of the available total cache disk is in use. A "warm" :term:`cache server` should show 100.00.
:abbr:`CHR (Cache Hit Ratio)`
	The cache hit ratio for the cache in the last 15 minutes (the interval is determined by the :manpage:`cron(8)` entry).
:abbr:`DSCP (Differential Services CodePoint)`
	Checks if the returning traffic from the cache has the correct :abbr:`DSCP (Differential Services CodePoint Check)` value as assigned in the :term:`Delivery Service`. (Some routers will overwrite :abbr:`DSCP (Differential Services CodePoint)`)
:abbr:`MTU (Maximum Transmission Unit)`
	Checks if the Traffic Ops host (if that is the one running the check) can send and receive 8192B packets to the ``ip_address`` of the server in the server table.
:abbr:`ORT (Operational Readiness Test)`
	The ORT column shows how many changes the :term:`ORT` script would apply if it was run. The number in this column should be 0 for :term:`cache servers` that do not have updates pending.
10G
	Is the ``ip_address`` (the main IPv4 address) from the server table ping-able?
:abbr:`ILO (Integrated Lights-Out)`
	Is the ``ilo_ip_address`` (the lights-out-management IPv4 address) from the server table ping-able?
10G6
	Is the ``ip6_address`` (the main IPv6 address) from the server table ping-able?
:abbr:`FQDN (Fully Qualified Domain Name)`
	Is the :abbr:`FQDN (Fully Qualified Domain Name)` (the concatenation of ``host_name`` and ``.`` and ``domain_name`` from the server table) ping-able?
:abbr:`RTR (Responds to Traffic Router)`
	Checks the state of each :term:`cache server` as perceived by all Traffic Monitors (via Traffic Router). This extension asks each Traffic Router for the state of the :term:`cache server`. A check failure is indicated if one or more monitors report an error for a :term:`cache server`. A :term:`cache server` is only marked as good if all reports are positive.

	.. note:: This is a pessimistic approach, opposite of how Traffic Monitor marks a :term:`cache server` as up, i.e. "the optimistic approach".

Example Cron File
-----------------
The :manpage:`cron(8)` file should be edited by running  :manpage:`crontab(1)` as the ``traffops`` user, or with :manpage:`sudo(8)`. You may need to adjust the path to your ``$TO_HOME`` to match your system.
Edit $TO_USER and $TO_PASS to match ORT input parameters.

.. code-block:: shell
	:caption: Example Cron File

	PERL5LIB=/opt/traffic_ops/app/local/lib/perl5:/opt/traffic_ops/app/lib

	# IPv4 ping examples - The 'select: ["hostName","domainName"]' works but, if you want to check DNS resolution use FQDN.
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"10G\", \"select\": [\"hostName\",\"domainName\"]}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"10G\", \"select\": \"ipAddress\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"10G\", \"name\": \"IPv4 Ping\", \"select\": \"ipAddress\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1

	# IPv6 ping examples
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"10G6\", \"name\": \"IPv6 Ping\", \"select\": \"ip6Address\", \"syslog_facility\": \"local0\"}" >/dev/null 2>&1
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"10G6\", \"select\": \"ip6Address\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1

	# iLO ping
	18 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"ILO\", \"select\": \"iloIpAddress\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	18 * * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"ILO\", \"name\": \"ILO ping\", \"select\": \"iloIpAddress\", \"syslog_facility\": \"local0\"}" >/dev/null 2>&1

	# MTU ping
	45 0 * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"MTU\", \"select\": \"ipAddress\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	45 0 * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"MTU\", \"select\": \"ip6Address\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	45 0 * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"MTU\", \"name\": \"Max Trans Unit\", \"select\": \"ipAddress\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1
	45 0 * * * root /opt/traffic_ops/app/bin/checks/ToPingCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"MTU\", \"name\": \"Max Trans Unit\", \"select\": \"ip6Address\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1

	# FQDN
	27 * * * * root /opt/traffic_ops/app/bin/checks/ToFQDNCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"FQDN\""  >> /var/log/traffic_ops/extensionCheck.log 2>&1
	27 * * * * root /opt/traffic_ops/app/bin/checks/ToFQDNCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"FQDN\", \"name\": \"DNS Lookup\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1

	# DSCP
	36 * * * * root /opt/traffic_ops/app/bin/checks/ToDSCPCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"DSCP\", \"cms_interface\": \"eth0\"}" >> /var/log/traffic_ops/extensionCheck.log 2>&1
	36 * * * * root /opt/traffic_ops/app/bin/checks/ToDSCPCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"DSCP\", \"name\": \:term:`Delivery Service`\", \"cms_interface\": \"eth0\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1

	# RTR
	10 * * * * root /opt/traffic_ops/app/bin/checks/ToRTRCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"RTR\"}"  >> /var/log/traffic_ops/extensionCheck.log 2>&1
	10 * * * * root /opt/traffic_ops/app/bin/checks/ToRTRCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"RTR\", \"name\": \"Content Router Check\", \"syslog_facility\": \"local0\"}" > /dev/null 2>&1

	# CHR
	*/15 * * * * root /opt/traffic_ops/app/bin/checks/ToCHRCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"CHR\"}"  >> /var/log/traffic_ops/extensionCheck.log 2>&1

	# CDU
	20 * * * * root /opt/traffic_ops/app/bin/checks/ToCDUCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"CDU\"}"  >> /var/log/traffic_ops/extensionCheck.log 2>&1

	# ORT
	40 * * * * ssh_key_edge_user /opt/traffic_ops/app/bin/checks/ToORTCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"ORT\", \"to_user\":\"$TO_USER\", \"to_pass\": \"$TO_PASS\"}"  >> /var/log/traffic_ops/extensionCheck.log 2>&1
	40 * * * * ssh_key_edge_user /opt/traffic_ops/app/bin/checks/ToORTCheck.pl -c "{\"base_url\": \"https://localhost\", \"check_name\": \"ORT\", \"name\": \"Operational Readiness Test\", \"syslog_facility\": \"local0\", \"to_user\":\"$TO_USER\", \"to_pass\": \"$TO_PASS\"}" > /dev/null 2>&1
