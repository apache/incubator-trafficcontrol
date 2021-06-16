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

.. _Apache Traffic Server configuration files: https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/index.en.html

.. _profiles-and-parameters:

***********************
Profiles and Parameters
***********************
:dfn:`Profiles` are a collection of configuration options, defined partially by the Profile's Type_ (not to be confused with the more general ":term:`Type`" used by many other things in Traffic Control) and partially by the :dfn:`Parameters` set on them. Mainly, Profiles and Parameters are used to configure :term:`cache servers`, but they can also be used to configure parts of (nearly) any Traffic Control component, and can even be linked with more abstract concepts like :ref:`Delivery Services <ds-profile>` and :term:`Cache Groups`. The vast majority of configuration done within a Traffic Control CDN must be done through Profiles_ and Parameters_, which can be achieved either through the :ref:`to-api` or in the :ref:`tp-configure-profiles` view of Traffic Portal. For ease of use, Traffic Portal allows for duplication, comparison, import and export of Profiles_ including all of their associated Parameters_.

.. seealso:: For Delivery Service Profile Parameters, see :ref:`ds-parameters`.

.. _profiles:

Profiles
========

Properties
----------
Profile objects as represented in e.g. the :ref:`to-api` or in the :ref:`Traffic Portal Profiles view <tp-configure-profiles>` have several properties that describe their general operation. In certain contexts, the Parameters_ assigned to a Profile (and/or the integral, unique identifiers thereof) may appear as properties of the Profile, but that will not appear in this section as a description of Parameters_ is provided in the section of that name.

.. _profile-cdn:

CDN
"""
A Profile is restricted to operate within a single CDN. Often, "CDN" (or "cdn") refers to the integral, unique identifier of the CDN, but occasionally it refers to the *name* of said CDN. It may also appear as e.g. ``cdnId`` or ``cdnName`` in :ref:`to-api` payloads and responses. A Profile may only be assigned to a server, :term:`Delivery Service`, or :term:`Cache Group` within the same CDN as the Profile itself.

.. _profile-description:

Description
"""""""""""
Profiles may have a description provided by the creating user (or Traffic Control itself in the case of the `Default Profiles`_). The :ref:`to-api` does not enforce length requirements on the description (though Traffic Portal does), and so it's possible for Profiles to have empty descriptions, though it is strongly recommended that Profiles have meaningful descriptions.

.. _profile-id:

ID
""
An integral, unique identifier for the Profile.

.. _profile-name:

Name
""""
Ostensibly this is simply the Profile's name. However, the name of a Profile has drastic consequences for how Traffic Control treats it. Particularly, the name of a Profile is heavily conflated with its Type_. These relationships are discussed further in the Type_ section, on a Type-by-Type basis.

.. _profile-routing-disabled:

Routing Disabled
""""""""""""""""
This property can - and in fact *must* - exist on a Profile of any Type_, but it only has any meaning on a Profile that has a name matching the constraints placed on the names of ATS_PROFILE-`Type`_ Profiles. This means that it will also have meaning on Profiles of Type_ UNK_PROFILE that for whatever reason have names beginning with ``EDGE`` or ``MID``. When this field is defined as ``1`` (may be displayed as ``true`` in e.g. Traffic Portal), Traffic Router will not be informed of any :term:`Delivery Services` to which the :term:`cache server` using this Profile may be assigned. Effectively, this means that client traffic cannot be routed to them, although existing connections would be uninterrupted.

.. _profile-type:

Type
""""
A Profile's :dfn:`Type` determines how its configured Parameters_ are treated by various components, and often even determine how the object using the Profile is treated (particularly when it is a server). Unlike the more general ":term:`Type`" employed by Traffic Control, the allowed Types of Profiles are set in stone, and they are as follows.

.. danger:: Nearly all of these Profile Types have strict naming requirements, and it may be noted that some of said requirements are prefixes ending with ``_``, while others are either not prefixes or do not end with ``_``. This is exactly true; some requirements **need** that ``_`` and some may or may not have it. It is our suggestion, therefore, that for the time being all prefixes use the ``_`` notation to separate words, so as to avoid causing headaches remembering when that matters and when it does not.

ATS_PROFILE
	A Profile that can be used with either an Edge-tier or Mid-tier :term:`cache server` (but not both, in general). This is the only Profile type that will ultimately pass its Parameters_ on to :term:`ORT` in the form of generated configuration files. For this reason, it can make use of the :ref:`t3c-special-strings` in the values of some of its Parameters_.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``EDGE`` or ``MID``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise! This includes :ref:`to-api-caches-stats`.

DS_PROFILE
	A Profile that, rather than applying to a server, is instead :ref:`used by a Delivery Service <ds-profile>`.

ES_PROFILE
	A Profile for `ElasticSearch <https://www.elastic.co/products/elasticsearch>`_ servers. This has no known special meaning to any component of Traffic Control, but if ElasticSearch is in use the use of this Profile Type is suggested regardless.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``ELASTICSEARCH``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

GROVE_PROFILE
	A Profile for use with the experimental Grove HTTP caching proxy.

INFLUXDB_PROFILE
	A Profile used with `InfluxDB <https://www.influxdata.com/>`_, which is used by Traffic Stats.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``INFLUXDB``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

KAFKA_PROFILE
	A Profile for `Kafka <https://kafka.apache.org/>`_ servers. This has no known special meaning to any component of Traffic Control, but if Kafka is in use the use of this Profile Type is suggested regardless.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``KAFKA``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

LOGSTASH_PROFILE
	A Profile for `Logstash <https://www.elastic.co/products/logstash>`_ servers. This has no known special meaning to any component of Traffic Control, but if Logstash is in use the use of this Profile Type is suggested regardless.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``LOGSTASH_``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

ORG_PROFILE
	A Profile that may be used by either :term:`origin servers` or :term:`Origins` (no, they aren't the same thing).

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``MSO``, or contain either ``ORG`` or ``ORIGIN`` anywhere in the name. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

RIAK_PROFILE
	A Profile used for each `Riak <http://basho.com/products/riak-kv/>`_ server in a Traffic Stats cluster.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``RIAK``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

SPLUNK_PROFILE
	A Profile meant to be used with `Splunk <https://www.splunk.com/>`_ servers. This has no known special meaning to any component of Traffic Control, but if Splunk is in use the use of this Profile Type is suggested regardless.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``SPLUNK``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

TM_PROFILE
	A Traffic Monitor Profile.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``RASCAL_`` or ``TM_``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

TP_PROFILE
	A Traffic Portal Profile. This has no known special meaning to any Traffic Control component(s) (not even Traffic Portal itself), but its use is suggested for the profiles used by any and all Traffic Portal servers anyway.

TR_PROFILE
	A Traffic Router Profile.

	.. warning:: For legacy reasons, the names of Profiles of this type *must* begin with ``CCR_`` or ``TR_``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise!

	.. seealso:: :ref:`tr-profile`

TS_PROFILE
	A Traffic Stats Profile.

	.. caution:: For legacy reasons, the names of Profiles of this type *must* be ``TRAFFIC_STATS``. This is **not** enforced by the :ref:`to-api` or Traffic Portal, but certain Traffic Control operations/components expect this and will fail to work otherwise! Furthermore, because Profile names must be unique, this means that only one TS_PROFILE-Type Profile can exist at a time.

UNK_PROFILE
	A catch-all type that can be assigned to anything without imbuing it with any special meaning or behavior.

.. tip:: A Profile of the wrong type assigned to a Traffic Control component *will* (in general) cause it to function incorrectly, regardless of the Parameters_ assigned to it.

.. _default-profiles:

Default Profiles
----------------
Traffic Control comes with some pre-installed Profiles for its basic components, but users are free to define their own as needed. Additionally, these default Profiles can be modified or even removed completely. One of these Profiles is `The GLOBAL Profile`_, which has a dedicated section.

INFLUXDB
	A Profile used by InfluxDB servers that store Traffic Stats information. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
RIAK_ALL
	This Profile is used by Traffic Vault, which is, generally speaking, the only instance in Traffic Control as it can store keys for an arbitrary number of CDNs. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
TRAFFIC_ANALYTICS
	A default Profile that was intended for use with the now-unplanned "Traffic Analytics" :abbr:`ATC (Apache Traffic Control)` component. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
TRAFFIC_OPS
	A Profile used by the Traffic Ops server itself. It's suggested that any and all "mirrors" of Traffic Ops for a given Traffic Control instance be recorded separately and all assigned to this Profile for record-keeping purposes. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
TRAFFIC_OPS_DB
	A Profile used by the PostgreSQL database server that stores all of the data needed by Traffic Ops. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
TRAFFIC_PORTAL
	A Profile used by Traffic Portal servers. This profile name has no known special meaning to any Traffic Control components (not even Traffic Portal itself), but its use is suggested for Traffic Portal servers anyway. It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.
TRAFFIC_STATS
	This is the **only** Profile used by Traffic Stats (though InfluxDB servers have their own Profile(s)). It has a Type_ of UNK_PROFILE and is assigned to the special "ALL" CDN_.

In addition to these Profiles, each release of Apache Traffic Control is accompanied by a set of suggested Profiles suitable for import in the :ref:`tp-configure-profiles` view of Traffic Portal. They may be found on `the Profiles Downloads Index page <http://trafficcontrol.apache.org/downloads/profiles/>`_. These Profiles are typically built from production Profiles by a company using Traffic Control, and as such are typically highly specific to the hardware and network infrastructure available to them. **None of the Profiles bundled with a release are suitable for immediate use without modification**, and in fact many of them cannot actually be imported directly into a new Traffic Control environment, because Profiles with the same :ref:`Names <profile-name>` already exist (as above).

Administrators may alternatively wish to consult the Profiles and Parameters_ available in the :ref:`ciab` environment, as they might be more familiar with them. Furthermore, those Profiles are built with a minimum running Traffic Control system in mind, and thus may be easier to look through. The Profiles and their associated Parameters_ may be found within the :atc-file:`infrastructure/cdn-in-a-box/traffic_ops_data/profiles/` directory.

.. _the-global-profile:

The GLOBAL Profile
------------------
There is a special Profile of Type_ UNK_PROFILE that holds global configuration information - its :ref:`profile-name` is "GLOBAL", its Type_ is UNK_PROFILE and it is assigned to the special "ALL" CDN_. The Parameters_ that may be configured on this Profile are laid out in the :ref:`global-profile-parameters` Table.

.. _global-profile-parameters:
.. table:: Global Profile Parameters

	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| :ref:`parameter-name`    | `Config File`_          | Value_                                                                                                                                |
	+==========================+=========================+=======================================================================================================================================+
	| tm.url                   | global                  | The URL at which this Traffic Ops instance services requests.                                                                         |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| tm.rev_proxy.url         | global                  | Not required. The URL where a caching proxy for configuration files generated by Traffic Ops may be found. Requires a minimum         |
	|                          |                         | :term:`ORT` version of 2.1. When configured, :term:`ORT` will request configuration files via this                                    |
	|                          |                         | :abbr:`FQDN (Fully Qualified Domain Name)`, which should be set up as a reverse proxy to the Traffic Ops server(s). The suggested     |
	|                          |                         | cache lifetime for these files is 3 minutes or less. This setting allows for greater scalability of a CDN maintained by Traffic Ops   |
	|                          |                         | by caching configuration files of profile and CDN scope, as generating these is a very computationally expensive process.             |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| tm.toolname              | global                  | The name of the Traffic Ops tool. Usually "Traffic Ops" - this will appear in the comment headers of generated configuration files.   |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| tm.infourl               | global                  | This is the "for more information go here" URL, which used to be visible in the "About" page of the now-deprecated Traffic Ops UI.    |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| tm.instance_name         | global                  | The name of the Traffic Ops instance - typically to distinguish instances when multiple are active.                                   |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| tm.traffic_mon_fwd_proxy | global                  | When collecting stats from Traffic Monitor, Traffic Ops will use this forward proxy instead of the actual Traffic Monitor host.       |
	|                          |                         | Setting this :ref:`Parameter <parameters>` can significantly lighten the load on the Traffic Monitor system and it is therefore       |
	|                          |                         | recommended that this be set on a production  system.                                                                                 |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| use_reval_pending        | global                  | When this Parameter is present and its Value_ is exactly "1", Traffic Ops will separately keep track of :term:`cache servers`'        |
	|                          |                         | updates and pending content invalidation jobs. This behavior should be enabled by default, and disabling it, while still possible, is |
	|                          |                         | **EXTREMELY DISCOURAGED**.                                                                                                            |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| geolocation.polling.url  | CRConfig.json           | The location of a geographic IP mapping database for Traffic Router instances to use.                                                 |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| geolocation6.polling.url | CRConfig.json           | The location of a geographic IPv6 mapping database for Traffic Router instances to use.                                               |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| maxmind.default.override | CRConfig.json           | The destination geographic coordinates to use for client location when the geographic IP mapping database returns a default location  |
	|                          |                         | that matches the country code. This parameter can be specified multiple times with different values to support default overrides for  |
	|                          |                         | multiple countries. The reason for the name "maxmind" is because the default geographic IP mapping database used by Traffic Control   |
	|                          |                         | is MaxMind's GeoIP2 database. The format of this :ref:`Parameter <parameters>`'s Value_ is:                                           |
	|                          |                         | :file:`{Country Code};{Latitude},{Longitude}`, e.g. ``US;37.751,-97.822``                                                             |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+
	| maxRevalDurationDays     | regex_revalidate.config | This :ref:`Parameter <parameters>` sets the maximum duration, in days, for which a content invalidation job may run. This is          |
	|                          |                         | **extremely** important, as there is currently no way to delete a content invalidation job once it has been created. Furthermore,     |
	|                          |                         | while there is no restriction placed on creating multiple Parameters_ with this :ref:`parameter-name` and `Config File`_ -            |
	|                          |                         | potentially with differing :ref:`Values <parameter-value>` - this is **EXTREMELY DISCOURAGED as any** :ref:`Parameter <parameters>`   |
	|                          |                         | **that has both that** :ref:`parameter-name` **and** `Config File`_ **might be used when generating any given**                       |
	|                          |                         | `regex_revalidate.config`_ **file for any given** :term:`cache server` **and whenever such** Parameters_ **exist, the actual maximum  |
	|                          |                         | duration for content invalidation jobs is undefined, and CAN and WILL differ from server to server, and configuration file to         |
	|                          |                         | configuration file.**                                                                                                                 |
	+--------------------------+-------------------------+---------------------------------------------------------------------------------------------------------------------------------------+


Some of these Parameters_ have the `Config File`_ value global_, while others have `CRConfig.json`_. This is not a typo, and the distinction is that those that use global_ are typically configuration options relating to Traffic Control as a whole or to Traffic Ops itself, whereas `CRConfig.json`_ is used by configuration options that are set globally, but pertain mainly to routing and are thus communicated to Traffic Routers through :term:`CDN Snapshots` (which historically were called "CRConfig Snapshots" or simply "the CRConfig").
When a :ref:`Parameter <parameters>` has a `Config File`_ value that *isn't* one of global_ or `CRConfig.json`_, it refers to the global configuration of said `Config File`_ across all servers that use it across all CDNs configured in Traffic Control. This can be used to easily apply extremely common configuration to a great many servers in one place.

.. _parameters:

Parameters
==========
A :dfn:`Parameter` is usually a way to set a line in a configuration file that will appear on the servers using Profiles_ that have said Parameter. More generally, though, a Parameter merely describes some kind of configuration for some aspect of some thing. There are many Parameters that *must* exist for Traffic Control to work properly, such as those on `The GLOBAL Profile`_ or the `Default Profiles`_. Some Traffic Control components can be associated with Profiles_ that only have a few allowed (or actually just meaningful - others are ignored and don't cause problems) but some can have any number of Parameters to describe custom configuration of things of which Traffic Control itself may not even be aware (most notably :term:`cache servers`). For most Parameters, the meaning of each Parameter's various properties are very heavily tied to the allowed contents of `Apache Traffic Server configuration files`_.

Properties
----------
When represented in Traffic Portal (in the :ref:`tp-configure-parameters` view) or in :ref:`to-api` request and/or response payloads, a Parameter has several properties that define it. In some of these contexts, the Profiles_ to which a Parameter is assigned (and/or the integral, unique identifiers thereof) are represented as a property of the Parameter. However, an explanation of this "property" is not provided here, as the Profiles_ section exists for the purpose of explaining those.

.. _parameter-config-file:

Config File
"""""""""""
This (usually) names the configuration file to which the Parameter belongs. Note that it is only the *name of* the file and **not** the *full path to* the file - e.g. ``remap.config`` not ``/opt/trafficserver/etc/trafficserver/remap.config``. To define the full path to any given configuration file, Traffic Ops relies on a reserved :ref:`parameter-name` value: :ref:`"location" <parameter-name-location>`.

.. seealso:: This section is only meant to cover the special handling of Parameters assigned to specific Config File values. It is **not** meant to be a primer on Apache Traffic Server configuration files, nor is it intended to be exhaustive of the manners in which said files may be manipulated by Traffic Control. For more information, consult the documentation for `Apache Traffic Server configuration files`_.

Certain Config Files are handled specially by Traffic Ops's configuration file generation. Specifically, the format of the configuration is tailored to be correct when the syntax of a configuration file is known. However, these configuration files **must** have :ref:`"location" <parameter-name-location>` Parameters on the :ref:`Profile <profiles>` of servers, or they will not be generated. The Config File values that are special in this way are detailed within this section. When a `Config File`_ is none of these special values, each Parameter assigned to given server's :ref:`Profile <profiles>` with the same `Config File`_ value will create a single line in the resulting configuration file (with the possible exception being when the :ref:`parameter-name` is "header")

12M_facts
'''''''''
This legacy file is generated entirely from a :ref:`Profile <profiles>`'s metadata, and cannot be affected by Parameters.

.. tip:: This Config File serves an unknown and likely historical purpose, so most users/administrators/developers don't need to worry about it.

50-ats.rules
''''''''''''
Parameters have no meaning when assigned to this Config File (except :ref:`"location" <parameter-name-location>`), but it *is* affected by Parameters that are on the same :ref:`Profile <profiles>` with the Config File ``storage.config`` - **NOT this Config File**. For each letter in the special "Drive Letters" Parameter, a line will be added of the form :file:`KERNEL=="{Prefix}{Letter}", OWNER="ats"` where ``Prefix`` is the Value_ of the Parameter with the :ref:`parameter-name` "Drive Prefix" and the Config File ``storage.config`` - but with the first instance of ``/dev/`` removed - , and ``Letter`` is the drive letter. Also, if the Parameter with the :ref:`parameter-name` "RAM Drive Prefix" exists on the same Profile assigned to the server, a line will be inserted for each letter in the special "RAM Drive Letters" Parameter of the form :file:`KERNEL=="{Prefix}{Letter}", OWNER="ats"` where ``Prefix`` is the Value_ of the "RAM Drive Prefix" Parameter - but with the first instance of ``/dev/`` removed -, and ``Letter`` is the drive letter.

.. tip:: This Config File serves an unknown and likely historical purpose, so most users/administrators/developers don't need to worry about it.

astats.config
'''''''''''''
This configuration file will be generated with a line for each Parameter with this Config File value on the :term:`cache server`'s :ref:`Profile <profiles>` in the form :file:`{Name}={Value}` where ``Name`` is the Parameter's :ref:`parameter-name` with trailing characters that match :regexp:`__\\d+$` stripped, and ``Value`` is its Value_.

bg_fetch.config
'''''''''''''''
This configuration file always generates static contents besides the header, and cannot be affected by any Parameters (besides its :ref:`"location" <parameter-name-location>` Parameter).

.. seealso:: For an explanation of the contents of this file, consult `the Background Fetch Apache Traffic Server plugin's official documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/background_fetch.en.html>`_.

cache.config
''''''''''''
This configuration is built entirely from :term:`Delivery Service` configuration, and cannot be affected by Parameters.

.. seealso:: `The Apache Traffic Server cache.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/cache.config.en.html>`_

:file:`cacheurl{anything}.config`
'''''''''''''''''''''''''''''''''
Config Files that match this pattern - where ``anything`` is a string of zero or more characters - can only be generated by providing a :ref:`location <parameter-name-location>` and their contents will be fully determined by properties of :term:`Delivery Services`.

.. seealso:: `The official documentation for the Cache URL Apache Traffic Server plugin <https://docs.trafficserver.apache.org/en/6.2.x/admin-guide/plugins/cacheurl.en.html>`_.

.. deprecated:: ATCv3.0
	This configuration file is only used by Apache Traffic Server version 6.x, whose use is deprecated both by that project and Traffic Control. These Config Files will have no special meaning at some point in the future.

chkconfig
'''''''''
This actually isn't a configuration file at all, kind of. Specifically, it is a valid configuration file for the legacy `chkconfig utility <https://linux.die.net/man/8/chkconfig>`_ - but it is never written to disk on any :term:`cache server`. Though all Traffic Control-supported systems are now using :manpage:`systemd(8)`, :term:`ORT` still uses ``chkconfig``-style configuration to set the status of services on its host system(s). This means that any Parameter with this Config File value should have a :ref:`parameter-name` that is the name of a service on the :term:`cache servers` using the :ref:`Profile <profiles>` to which the Parameter is assigned, and it's Value_ should be a valid ``chkconfig`` configuration line for that service.

CRConfig.json
'''''''''''''
In general, the term "CRConfig" refers to :term:`CDN Snapshots`, which historically were called "CRConfig Snapshots" or simply "the CRConfig". Parameters with this Config File should be only be on either `The GLOBAL Profile`_ where they will affect global routing configuration, or on a Traffic Router's :ref:`Profile <profiles>` where they will affect routing configuration for that Traffic Router only.

.. seealso:: For the available configuration Parameters for a Traffic Router Profile, see :ref:`tr-profile`.

drop_qstring.config
'''''''''''''''''''
This configuration file will be generated with a single line that is exactly: :regexp:`/([^?]+) \$s://\$t/\$1\n` **unless** a Parameter exists on the :ref:`Profile <profiles>` with this Config File value, and the :ref:`parameter-name` "content". In the latter case, the contents of the file will be exactly the Parameter's Value_ (with terminating newline appended).

global
''''''
In general, this Config File isn't actually handled specially by Traffic Ops when generating server configuration files. However, this is the Config File value typically used for Parameters assigned to `The GLOBAL Profile`_ for truly "global" configuration options, and it is suggested that this precedent be maintained - i.e. don't create Parameters with this Config File.

:file:`hdr_rw_{anything}.config`
''''''''''''''''''''''''''''''''
Config Files that match this pattern - where ``anything`` is zero or more characters - are written specially by Traffic Ops to accommodate the :ref:`ds-dscp` setting of :term:`Delivery Services`.

.. tip:: The ``anything`` in those file names is typically a :term:`Delivery Service`'s :ref:`ds-xmlid` - though the inability to affect the file's contents is utterly independent of whether or not a :term:`Delivery Service` with that :ref:`ds-xmlid` actually exists.

.. seealso:: For information on the contents of files like this, consult `the Header Rewrite Apache Traffic Server plugin's documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/header_rewrite.en.html#rewriting-rules>`_

hosting.config
''''''''''''''
This configuration file is mainly generated based on the assignments of :term:`cache servers` to :term:`Delivery Services` and the :term:`Cache Group` hierarchy, but there are a couple of Parameter :ref:`Names <parameter-name>` that can affect it when assigned to this Config File. When a Parameter assigned to the ``storage.config`` Config File - **NOT this Config File** - with the :ref:`parameter-name` "RAM_Drive_Prefix" *exists*, it will cause lines to be generated in this configuration file for each :term:`Delivery Service` that is of on of the :ref:`Types <ds-types>` DNS_LIVE (only if the server is an :term:`Edge-tier cache server`), HTTP_LIVE (only if the server is an :term:`Edge-tier cache server`), DNS_LIVE_NATNL, or HTTP_LIVE_NATNL to which the :term:`cache server` to which the :ref:`Profile <profiles>` containing that Parameter belongs is assigned. Specifically, it will cause each of them to use ``volume=1`` **UNLESS** the Parameter with the :ref:`parameter-name` "Drive_Prefix" associated with Config File ``storage.config`` - again, **NOT this Config File** - *also* exists, in which case they will use ``volume=2``.

.. caution:: If a Parameter with Config File ``storage.config`` and :ref:`parameter-name` "RAM_Drive_Prefix" does *not* exist on a :ref:`Profile <profiles>`, then the :term:`cache servers` using that :ref:`Profile <profiles>` will **be incapable of serving traffic for** :term:`Delivery Services` **of the aforementioned** :ref:`Types <ds-types>`, **even when a** :ref:`"location" <parameter-name-location>` **Parameter exists**.

.. seealso:: For an explanation of the syntax of this configuration file, refer to `the Apache Traffic Server hosting.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/hosting.config.en.html>`_.

ip_allow.config
'''''''''''''''
This configuration file is mostly generated from various server data, but can be affected by a Parameter that has a :ref:`parameter-name` of "purge_allow_ip", which will cause the insertion of a line with :file:`src_ip={VALUE} action=ip_allow method=ALL` where ``VALUE`` is the Parameter's Value_. Additionally, Parameters with :ref:`Names <parameter-name>` like :file:`coalesce_{masklen|number}_v{4|6}` cause Traffic Ops to generate coalesced IP ranges in different ways. In the case that ``number`` was used, the Parameter's Value_ sets the the maximum number of IP address that may be coalesced into a single range. If ``masklen`` was used, the lines that are generated are coalesced into :abbr:`CIDR (Classless Inter-Domain Routing)` ranges using mask lengths determined by the Value_ of the parameter (using '4' sets the mask length of IPv4 address coalescing while using '6' sets the mask length to use when coalescing IPv6 addresses). This is not recommended, as the default mask lengths allow for maximum coalescence. Furthermore, if two Parameters on the same :ref:`Profile <profiles>` assigned to a server having Config File values of ``ip_allow.config`` and :ref:`Names <parameter-name>` that are both "coalesce_masklen_v4" but each has a different Value_, then the actual mask length used to coalesce IPv4 addresses is undefined (but will be one of the two). All forms of the "coalescence Parameters" have this problem.

.. impl-detail:: At the time of this writing, coalescence is implemented through the `the NetAddr\:\:IP Perl library <http://search.cpan.org/~miker/NetAddr-IP-4.078/IP.pm>`_.

.. seealso:: `The Apache Traffic Server ip_allow.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/ip_allow.config.en.html>`_ explains the syntax and meaning of lines in that file.

logging.config
''''''''''''''
This configuration file can only be affected by Parameters with specific :ref:`Names <parameter-name>`. Specifically, for each Parameter assigned to this Config File on the :ref:`Profile <profiles>` used by the :term:`cache server` with the name :file:`LogFormat{N}.Name` where ``N`` is either the empty string or a natural number on the interval [1,9] the text in :ref:`logging.config-format-snippet` will be inserted. In that snippet, ``NAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Name`, and ``FORMAT`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Format` for the same value of ``N``\ [#logs-format]_.

.. _logging.config-format-snippet:

.. code-block:: text
	:caption: Log Format Snippet

	NAME = format {
		Format = 'FORMAT '
	}

.. tip:: The order in which these Parameters are considered is exactly the numerical ordering implied by ``N`` (starting with it being empty). However, each section is generated for all values of ``N`` before moving on to the next.

Furthermore, for a given value of ``N`` - as before restricted to either the empty string or a natural number on the interval [1,9] -, if a Parameter exists on the :term:`cache server`'s :ref:`Profile <profiles>` having this Config File value with the :ref:`parameter-name` :file:`LogFilter{N}.Name`, a line of the format :file:`{NAME} = filter.{TYPE}.('{FILTER}')` will be inserted, where ``NAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Name`, ``TYPE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Type`, and ``FILTER`` is the Value_ of the Parameter with the name :file:`LogFilter{N}.Filter`\ [#logs-filter]_.

.. note:: When, for a given value of ``N``, a Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Name` exists, but a Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Type` does *not* exist, the value of ``TYPE`` will be ``accept``.

Finally, for a given value of ``N``, if a Parameter exists on the :term:`cache server`'s :ref:`Profile <profiles>` having this Config File value with the :ref:`parameter-name` :file:`LogObject{N}.Filename`, the text in :ref:`logging.config-object-snippet` will be inserted. In that snippet, ``TYPE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type`

.. _logging.config-object-snippet:

.. code-block:: text
	:caption: Log Object Snippet

	log.TYPE {
	  Format = FORMAT,
	  Filename = 'FILENAME',

.. note:: When, for a given value of ``N`` a Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Filename` exists, but a Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type` does *not* exist, the value of ``TYPE`` in :ref:`logging.config-object-snippet` will be ``ascii``.

At this point, if the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type` is **exactly** ``pipe``, a line of the format :file:`\ \ Filters = { FILTERS }` will be inserted where ``FILTERS`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Filters`, followed by a line containing only a closing "curly brace" (:kbd:`}`) - *if and* **only** *if said Parameter is* **not** *empty*. If, however, the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type` is **not** exactly ``pipe``, then the text in :ref:`logging.config-object-not-pipe-snippet` is inserted.

.. _logging.config-object-not-pipe-snippet:

.. code-block:: text
	:caption: Log Object (not a "pipe") Snippet

	  RollingEnabled = ROLLING,
	  RollingIntervalSec = INTERVAL,
	  RollingOffsetHr = OFFSET,
	  RollingSizeMb = SIZE
	}

In this snippet, ``ROLLING`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingEnabled`, ``INTERVAL`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingIntervalSec`, ``OFFSET`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingOffsetHr`, and ``SIZE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.SizeMb` - all still having the same value of ``N``, and the Config File value ``logging.config``, of course.

.. warning:: The contents of these fields are not validated by Traffic Control - handle with care!

.. seealso:: `The Apache Traffic Server documentation for the logging.config configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/logging.config.en.html>`_

logging.yaml
''''''''''''
This is a YAML-format configuration file used by :term:`cache servers` that use Apache Traffic Server version 8 or higher - for lower versions, users/administrators/developers should instead be configuring ``logging.config``. This configuration always starts with (after the header) the single line: :literal:`format:\ `. Afterward, for every Parameter assigned to this Config File with a :ref:`parameter-name` like :file:`LogFormat{N}.Name` where ``N`` is either the empty string or a natural number on the interval [1,9], the YAML fragment shown in :ref:`logging.yaml-format-snippet` will be inserted. In this snippet, ``NAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Name`, and for the same value of ``N`` ``FORMAT`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Format`.

.. _logging.yaml-format-snippet:

.. code-block:: yaml
	:caption: Log Format Snippet

	 - name: NAME
	   format: 'FORMAT'

.. tip:: The order in which these Parameters are considered is exactly the numerical ordering implied by ``N`` (starting with it being empty). However, each section is generated for all values of ``N`` before moving on to the next.

After this, a single line containing only ``filters:`` is inserted. Then, for each Parameter on the :term:`cache server`'s :ref:`Profile <profiles>` with a :ref:`parameter-name` like :file:`LogFilter{N}.Name` where ``N`` is either the empty string or a natural number on the interval [1,9], the YAML fragment in :ref:`logging.yaml-filter-snippet` will be inserted. In that snippet, ``NAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Name`, ``TYPE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Type` for the same value of ``N``, and ``FILTER`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Filter` for the same value of ``N``.

.. _logging.yaml-filter-snippet:

.. code-block:: yaml
	:caption: Log Filter Snippet

	- name: NAME
	  action: TYPE
	  condition: FILTER

.. note:: When, for a given value of ``N``, a Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Name` exists, but a Parameter with the :ref:`parameter-name` :file:`LogFilter{N}.Type` does *not* exist, the value of ``TYPE`` in :ref:`logging.yaml-filter-snippet` will be ``accept``.

At this point, a single line containing only ``logs:`` is inserted. Finally, for each Parameter on the :term:`cache server`'s :ref:`Profile <profiles>` assigned to this Config File with a :ref:`parameter-name` like :file:`LogObject{N}.Filename` where ``N`` is once again either an empty string or a natural number on the interval [1,9] the YAML fragment in :ref:`logging.yaml-object-snippet` will be inserted. In this snippet, for a given value of ``N`` ``TYPE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type`, ``FILENAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Filename`, ``FORMAT`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Format`.

.. _logging.yaml-object-snippet:

.. code-block:: yaml
	:caption: Log Object Snippet

	- mode: TYPE
	  filename: FILENAME
	  format: FORMAT
	  ROLLING_OR_FILTERS

.. note:: When, for a given value of ``N`` a Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Filename` exists, but a Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type` does *not* exist, the value of ``TYPE`` in :ref:`logging.yaml-object-snippet` will be ``ascii``.

``ROLLING_OR_FILTERS`` will be one of two YAML fragments based on the Value_ of the Parameter with the name :file:`LogObject{N}.Type`. In particular, if it is exactly ``pipe``, then ``ROLLING_OR_FILTERS`` will be :file:`filters: [{FILTERS}]` where ``FILTERS`` is the Value_ of the Parameter assigned to this Config File with the :ref:`parameter-name` :file:`LogObject{N}.Filters` for the same value of ``N``. If, however, the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Type` is **not** exactly ``pipe``, ``ROLLING_OR_FILTERS`` will have the format given by :ref:`logging.yaml-object-not-pipe-snippet`. In that snippet, ``ROLLING`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingEnabled`, ``INTERVAL`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingIntervalSec`, ``OFFSET`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingOffsetHr`, and ``SIZE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingSizeMb` - all for the same value of ``N`` and assigned to the ``logging.yaml`` Config File, obviously.

.. _logging.yaml-object-not-pipe-snippet:

.. code-block:: yaml
	:caption: Log Object (not a "pipe") Snippet

	  rolling_enabled: ROLLING
	  rolling_interval_sec: INTERVAL
	  rolling_offset_hr: OFFSET
	  rolling_size_mb: SIZE


.. seealso:: For an explanation of YAML syntax, refer to the `official specification thereof <https://yaml.org/>`_. For an explanation of the syntax of a valid Apache Traffic Server ``logging.yaml`` configuration file, refer to `that project's dedicated documentation <https://docs.trafficserver.apache.org/en/8.0.x/admin-guide/files/logging.yaml.en.html>`_.

logs_xml.config
'''''''''''''''
This configuration file is somewhat more complex than most Config Files, in that it generates XML document tree segments\ [#xml-caveat]_ for each Parameter on the :term:`cache server`'s :ref:`Profile <profiles>` rather than simply a plain-text line. Specifically, up to ten of the document fragment shown in :ref:`logs_xml-format-snippet` will be inserted, one for each Parameter with this Config File value on the :term:`cache server`'s :ref:`Profile <profiles>` that has a :ref:`parameter-name` like :file:`LogFormat{N}.Name` where ``N`` is either the empty string or a natural number on the range [1,9]. In that snippet, the string ``NAME`` is actually the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Name"` ``FORMAT`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogFormat{N}.Format`\ [#logs-format]_, where again ``N`` is either the empty string or a natural number on the interval [1,9] - same-valued ``N`` Parameters are associated.

.. _logs_xml-format-snippet:

.. code-block:: text
	:caption: LogFormat Snippet

	<LogFormat>
		<Name = "NAME"/>
		<Format = "FORMAT"/>
	</LogFormat>

.. tip:: The order in which these Parameters are considered is exactly the numerical ordering implied by ``N`` (starting with it being empty).

Furthermore, for a given value of ``N``, if a Parameter exists on the :term:`cache server`'s :ref:`Profile <profiles>` having this Config File value with the :ref:`parameter-name` :file:`LogObject{N}.Filename`, the document fragment shown in :ref:`logs_xml-object-snippet` will be inserted. In that snippet, ``OBJ_FORMAT`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Format`, ``FILENAME`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Filename`, ``ROLLING`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingEnabled`, ``INTERVAL`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingIntervalSec`, ``OFFSET`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingOffsetHr`, ``SIZE`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.RollingSizeMb`, and ``HEADER`` is the Value_ of the Parameter with the :ref:`parameter-name` :file:`LogObject{N}.Header` - all having the same value of ``N``, and the Config File value ``logs_xml.config``, of course.

.. _logs_xml-object-snippet:

.. code-block:: text
	:caption: LogObject Snippet

	<LogObject>
		<Format = "OBJ_FORMAT"/>
		<Filename = "FILENAME"/>
		<RollingEnabled = ROLLING/>
		<RollingInterval = INTERVAL/>
		<RollingOffsetHr = OFFSET/>
		<RollingSizeMb = SIZE/>
		<Header = "HEADER"/>
	</LogObject>

.. warning:: The contents of these fields are not validated by Traffic Control - handle with care!

.. seealso:: The `Apache Traffic Control documentation on the logs_xml.config configuration file <https://docs.trafficserver.apache.org/en/6.2.x/admin-guide/files/logs_xml.config.en.html>`_

.. deprecated:: ATCv3.0

	This file is only used by Apache Traffic Server version 6.x. The use of Apache Traffic Server version < 7.1 has been deprecated, and will not be supported in the future. Developers are encouraged to instead configure the `logging.config`_ configuration file.

package
'''''''
This is a special, reserved Config File that isn't a file at all. When a Parameter's Config File is ``package``, then its name is interpreted as the name of a package. :term:`ORT` on the server using the :ref:`Profile <profiles>` that has this Parameter will attempt to install a package by that name, interpreting the Parameter's Value_ as a version string if it is not empty. The package manager used will be :manpage:`yum(8)`, regardless of system (though the Python version of :term:`ORT` will attempt to use the host system's package manager - :manpage:`yum(8)`, :manpage:`apt(8)` and ``pacman`` are supported) but that shouldn't be a problem because only CentOS 7 and CentOS 8 are supported.

The current implementation of :term:`ORT` will expect Parameters to exist on a :term:`cache server`'s :ref:`Profile <profiles>` with the :ref:`Names <parameter-name>` ``astats_over_http`` and ``trafficserver`` before being run the first time, as both of these are required for a :term:`cache server` to operate within a Traffic Control CDN. It is possible to install these outside of :term:`ORT` - and indeed even outside of :manpage:`yum(8)` - but such configuration is not officially supported.

packages
''''''''
This Config File is reserved, and is used by :term:`ORT` to pull bulk information about all of the Parameters with Config File values of package_. It doesn't actually correspond to any configuration file.

parent.config
'''''''''''''
This configuration file is generated entirely from :term:`Cache Group` relationships, as well as :term:`Delivery Service` configuration. This file *can* be affected by Parameters on the server's :ref:`Profile <Profiles>` if and only if its :ref:`parameter-name` is one of the following:

- ``algorithm``
- ``qstring``
- ``psel.qstring_handling``
- ``not_a_parent`` - unlike the other Parameters listed (which have a 1:1 correspondence with Apache Traffic Server configuration options), this Parameter affects the generation of :term:`parent` relationships between :term:`cache servers`. When a Parameter with this :ref:`parameter-name` and Config File exists on a :ref:`Profile <profiles>` used by a :term:`cache server`, it will not be added as a :term:`parent` of any other :term:`cache server`, regardless of :term:`Cache Group` hierarchy. Under ordinary circumstances, there's no real reason for this Parameter to exist.

Additionally, :term:`Delivery Service` :ref:`Profiles <ds-profile>` can have special Parameters with the :ref:`parameter-name` "mso.parent_retry" to :ref:`multi-site-origin-qht`.

.. seealso:: To see how the :ref:`Values <parameter-value>` of these Parameters are interpreted, refer to the `Apache Traffic Server documentation on the parent.config configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/parent.config.en.html>`_

plugin.config
'''''''''''''
For each Parameter with this Config File value on the same :ref:`Profile <profiles>`, a line in the resulting configuration file is produced in the format :file:`{NAME} {VALUE}` where ``NAME`` is the Parameter's :ref:`parameter-name` with trailing characters matching the regular expression :regexp:`__\\d+$` stripped out and ``VALUE`` is the Parameter's Value_.

.. caution:: In order for Parameters for Config Files relating to Apache Traffic Server plugins - e.g. `regex_revalidate.config`_ - to have any effect, a Parameter must exist with this Config File value to instruct Apache Traffic Server to load the plugin. Typically, this is more easily achieved by assigning these Parameters to `The GLOBAL Profile`_ than on a server-by-server basis.

.. seealso:: `The Apache Traffic server documentation on the plugin.config configuration file <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/plugin.config.en.html>`_ explains what Value_ and :ref:`parameter-name` a Parameter should have to be valid.

.. _tm-related-cache-server-params:

rascal.properties
'''''''''''''''''
This Config File is meant to be on Parameters assigned to either Traffic Monitor Profiles_ or :term:`cache server` Profiles_. Its allowed :ref:`Parameter Names <parameter-name>` are all configuration options for Traffic Monitor. The :ref:`Names <parameter-name>` with meaning are as follows.

.. seealso:: :ref:`health-proto`

.. _param-health-polling-format:

health.polling.format
	The Value_ of this Parameter should be the name of a parsing format supported by Traffic Monitor, used to decode statistics when polling for health and statistics. If this Parameter does not exist on a :term:`cache server`'s :ref:`Profile <Profiles>`, the default format (``astats``) will be used. The only supported values are

	- ``astats`` parses the statistics output from the `astats_over_http plugin <https://github.com/apache/trafficcontrol/tree/master/traffic_server/plugins/astats_over_http/README.md>`_.
	- ``stats_over_http`` parses the statistics output from the `stats_over_http plugin <https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/stats_over_http.en.html>`_.
	- ``noop`` no statistics are parsed; the :term:`cache servers` using this Value_ will always be considered healthy, but statistics will never be gathered for them.

	For more information on Traffic Monitor plug-ins that can expand the parsed formats, refer to :ref:`admin-tm-extensions`.

.. _param-health-polling-url:

health.polling.url
	The Value_ of this Parameter sets the URL requested when Traffic Monitor polls cache servers that have this Parameter in their Profiles_. Specifically, the Value_ is interpreted as a template - in a format reminiscent of variable interpolation in double-quoted strings in Bash -, that offers the following substitutions:

	- ``${hostname}`` Replaced by the *IP Address* of the :term:`cache server` being polled, and **not** its (short) hostname. The IP address used will be its IPv4 service address if it has one, otherwise its IPv6 service address. IPv6 addresses are properly formatted when inserted into the template, so the template need not include "square brackets" (:kbd:`[` and :kbd:`]`) around ``${hostname}``\ s even when they anticipate they will be IPv6 addresses.
	- ``${interface_name}`` Replaced by the name of the network interface that contains the :term:`cache server`'s service address(es). For most cache servers (specifically those using the ``stats_over_http`` :abbr:`ATS (Apache Traffic Server)` plugin to report their health and statistics) using this in a template won't be necessary.

	If the template doesn't include a specific port number, the :term:`cache server`'s TCP port will be inserted if the URL uses the HTTP scheme, or its HTTPS Port if the :term:`cache server` uses the the HTTPS scheme.

	Table :ref:`tbl-health-polling-url-examples` gives some examples of templates, inputs, and outputs.

	.. _tbl-health-polling-url-examples:

	.. table:: health.polling.url Value Examples

		+---------------------------------------------------------------+-------------------+----------+------------+----------------+--------------------------------------------------+
		| Template                                                      | Chosen Service IP | TCP Port | HTTPS Port | Interface Name | Output                                           |
		+===============================================================+===================+==========+============+================+==================================================+
		| ``http://${hostname}/_astats?inf.name=${interface_name}``     | 192.0.2.42        | 8080     | 8443       | eth0           | ``http://192.0.2.42:8080/_astats?inf.name=eth0`` |
		+---------------------------------------------------------------+-------------------+----------+------------+----------------+--------------------------------------------------+
		| ``https://${hostname}/_stats``                                | 2001:DB8:0:0:1::1 | 8080     | 8443       | eth0           | ``https://[2001:DB8:0:0:1::1]/_stats``           |
		+---------------------------------------------------------------+-------------------+----------+------------+----------------+--------------------------------------------------+
		| ``http://${hostname}:80/custom/stats/path/${interface_name}`` | 192.0.2.42        | 8080     | 8443       | eth0           | ``http://192.0.2.42:80/custom/stats/path/eth0``  |
		+---------------------------------------------------------------+-------------------+----------+------------+----------------+--------------------------------------------------+

health.threshold.loadavg
	The Value_ of this Parameter sets the "load average" above which the associated :ref:`Profile <profiles>`'s :term:`cache server` will be considered "unhealthy".

	.. seealso:: The definition of a "load average" can be found in the documentation for the Linux/Unix command :manpage:`uptime(1)`.

	.. caution:: If more than one Parameter with this :ref:`parameter-name` and Config File exist on the same :ref:`Profile <profiles>` with different :ref:`Values <parameter-value>`, the actual Value_ used by any given Traffic Monitor instance is undefined (though it will be the Value_ of one of those Parameters).

health.threshold.availableBandwidthInKbps
	The Value_ of this Parameter sets the amount of bandwidth (in kilobits per second) that Traffic Control will try to keep available on the :term:`cache server` - for all network interfaces. For example a Value_ of ">1500000" indicates that the :term:`cache server` will be marked "unhealthy" if its available remaining bandwidth across all of the network interfaces used by the caching proxy fall below 1.5Gbps.

	.. caution:: If more than one Parameter with this :ref:`parameter-name` and Config File exist on the same :ref:`Profile <profiles>` with different :ref:`Values <parameter-value>`, the actual Value_ used by any given Traffic Monitor instance is undefined (though it will be the Value_ of one of those Parameters).

history.count
	The Value_ of this Parameter sets the maximum number of collected statistics will retain at a time. For example, if this is "30", then Traffic Monitor will keep up to the past 30 collected statistics runs for the :term:`cache servers` using the :ref:`Profile <profiles>` that has this Parameter. The minimum history size is 1, and if this Parameter's Value_ is set below that, it will be treated as though it were 1.

	.. caution:: This **must** be an integer. What happens when the Value_ of this Parameter is *not* an integer is not known to this author; at a guess, in all likelihood it would be treated as though it were 1 and warnings/errors would be logged by Traffic Monitor and/or Traffic Ops. However, this is not known and setting it improperly is potentially dangerous, so *please ensure it is* **always** *an integer*.

records.config
''''''''''''''
For each Parameter with this Config File value on the same :ref:`Profile <profiles>`, a line in the resulting configuration file is produced in the format :file:`{NAME} {VALUE}` where ``NAME`` is the Parameter's :ref:`parameter-name` with trailing characters matching the regular expression :regexp:`__\\d+$` stripped out and ``VALUE`` is the Parameter's Value_.

.. seealso:: `The Apache Traffic Server records.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/records.config.en.html>`_

:file:`regex_remap_{anything}.config`
''''''''''''''''''''''''''''''''''''''''''''
Config Files matching this pattern - where ``anything`` is zero or more characters - are generated entirely from :term:`Delivery Service` configuration, which cannot be affected by any Parameters (except :ref:`"location" <parameter-name-location>`).

.. seealso:: For the syntax of configuration files for the "Regex Remap" plugin, see `the Regex Remap plugin's official documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/regex_remap.en.html>`_. For instructions on how to enable a plugin, consult, the `plugin.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/plugin.config.en.html>`_.

regex_revalidate.config
'''''''''''''''''''''''
This configuration file can only be affected by the special ``maxRevalDurationDays``, which is discussed in the `The GLOBAL Profile`_ section.

.. seealso:: For the syntax of configuration files for the "Regex Revalidate" plugin, see `the Regex Revalidate plugin's official documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/plugins/regex_revalidate.en.html#revalidation-rules>`_. For instructions on how to enable a plugin, consult, the `plugin.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/plugin.config.en.html>`_.

remap.config
''''''''''''
This configuration file can only be affected by Parameters on a :ref:`Profile <profiles>` assigned to a :term:`Delivery Service`. Then, for every Parameter assigned to that :ref:`Profile <profiles>` that has the Config File value "remap.config" -, a parameter will be added to the line for that :term:`Delivery Service` like so: :file:`@pparam={Value}` where ``Name`` is of the form `<plugin>.pparam` and ``Value`` are the plugin pparam arguments.

Previously this configuration file can only be affected by one Parameter on a :ref:`Profile <profiles>` assigned to a :term:`Delivery Service`. Then, for every Parameter assigned to that :ref:`Profile <profiles>` that has the Config File value "cachekey.config" - **NOT this Config File** -, a parameter will be added to the line for that :term:`Delivery Service` like so: :file:`pparam=--{Name}={Value}` where ``Name`` is the Parameter's :ref:`parameter-name`, and ``Value`` is its Value_.

The following plugins have support adding args with following parameter `Config File`
  - ``cachekey.pparam`` `<https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/cachekey.en.html>`
  - ``cache_range_requests.pparam`` `<https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/cache_range_requests.en.html>`
  - ``slice.pparam`` `<https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/slice.en.html>`  Note the --blocksize=<val> is already handled in the :ref:`tp-services-delivery-service` view in Traffic Portal.

.. note:: cachekey.config is deprecated but available for backwards compatibility.  cachekey.config Name and Value will be converted to the pparam syntax with '--' added as a prefix to the name.  Any "empty" param value (ie: separator) will add an extra '=' to the key.

.. table:: Equivalent cachekey.config/cachekey.pparam entries

  +------------------------+---------------------+--------------------------+----------------------------------+
  | :ref:`parameter-name`  | `Config File`_      | Value_                   | Result                           |
  +========================+=====================+==========================+==================================+
  | remove-all-params      | cachekey.config     | true                     | @pparam=--remove-all-params=true |
  | cachekey.pparam        | remap.config        | --remove-all-params=true | @pparam=--remove-all-params=true |
  | separator              | cachekey.config     |                          | @pparam=--separator=             |
  | cachekey.pparam        | remap.config        | --separator=             | @pparam=--separator=             |
  | cachekey.pparam        | cachekey.pparam     | -o                       | @pparam=-o (no config equiv)     |
  +------------------------+---------------------+--------------------------+----------------------------------+


.. seealso:: For an explanation of the syntax of this configuration file, refer to `the Apache Traffic Server remap.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/remap.config.en.html>`_.

:file:`set_dscp_{anything}.config`
''''''''''''''''''''''''''''''''''
Configuration files matching this pattern - where ``anything`` is a string of zero or more characters is generated entirely from a :ref:`"location" <parameter-name-location>` Parameter.

.. tip:: ``anything`` in that Config File name only has meaning if it is a natural number - specifically, one of each value of :ref:`ds-dscp` on every :term:`Delivery Service` to which the :term:`cache server` using the :ref:`Profile <profiles>` on which the Parameter(s) exist(s).

ssl_multicert.config
''''''''''''''''''''
This configuration file is generated from the SSL keys of :term:`Delivery Services`, and is unaffected by any Parameters (except :ref:`"location" <parameter-name-location>`)

.. seealso:: `The official ssl_multicert.config documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/ssl_multicert.config.en.html>`_

storage.config
''''''''''''''
This configuration file can only be affected by a handful of Parameters. If a Parameter with the :ref:`parameter-name` "Drive Prefix" exists the generated configuration file will have a line inserted in the format :file:`{PREFIX}{LETTER} volume=1` for each letter in the comma-delimited list that is the Value_ of the Parameter on the same :ref:`Profile <profiles>` with the :ref:`parameter-name` "Drive Letters", where ``PREFIX`` is the Value_ of the Parameter with the :ref:`parameter-name` "Drive Prefix", and ``LETTER`` is each of the aforementioned letters in turn. Additionally, if a Parameter on the same :ref:`Profile <profiles>` exists with the :ref:`parameter-name` "RAM Drive Prefix" then for each letter in the comma-delimited list that is the Value_ of the Parameter on the same :ref:`Profile <profiles>` with the :ref:`parameter-name` "RAM Drive Letters", a line will be generated in the format :file:`{PREFIX}{LETTER} volume={i}` where ``PREFIX`` is the Value_ of the Parameter with the :ref:`parameter-name` "RAM Drive Prefix", ``LETTER`` is each of the aforementioned letters in turn, and ``i`` is 1 *if and* **only** *if* a Parameter does **not** exist on the same :ref:`Profile <profiles>` with the :ref:`parameter-name` "Drive Prefix" and is 2 otherwise. Finally, if a Parameter exists on the same :ref:`Profile <profiles>` with the :ref:`parameter-name` "SSD Drive Prefix", then a line is inserted for each letter in the comma-delimited list that is the Value_ of the Parameter on the same :ref:`Profile <profiles>` with the :ref:`parameter-name` "SSD Drive Letters" in the format :file:`{PREFIX}{LETTER} volume={i}` where ``PREFIX`` is the Value_ of the Parameter with the :ref:`parameter-name` "SSD Drive Prefix", ``LETTER`` is each of the aforementioned letters in turn, and ``i`` is 1 *if and* **only** *if* **both** a Parameter with the :ref:`parameter-name` "Drive Prefix" and a Parameter with the :ref:`parameter-name` "RAM Drive Prefix" *don't exist on the same* :ref:`Profile <profiles>`, or 2 if only **one** of them exists, or otherwise 3.

.. seealso:: `The Apache Traffic Server storage.config file documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/storage.config.en.html>`_.

traffic_stats.config
''''''''''''''''''''
This Config File value is only handled specially when the :ref:`Profile <profiles>` to which it is assigned is of the special TRAFFIC_STATS Type_. In that case, the :ref:`parameter-name` of any Parameters with this Config File is restrained to one of "CacheStats" or "DsStats". When it is "Cache Stats", the Value_ is interpreted specially based on whether or not it starts with "ats.". If it does, then what follows must be the name of one of `the core Apache Traffic Server statistics <https://docs.trafficserver.apache.org/en/latest/admin-guide/monitoring/statistics/core-statistics.en.html>`_. This signifies to Traffic Stats that it should store that statistic for :term:`cache servers` within Traffic Control. Additionally, the special statistics "bandwidth", "maxKbps" are supported as :ref:`Names <parameter-name>` - and in fact it is suggested that they exist in every Traffic Control deployment.

When the Parameter :ref:`parameter-name` is "DSStats", the allowed :ref:`Values <parameter-value>` are:

- kbps
- status_4xx
- status_5xx
- tps_2xx
- tps_3xx
- tps_4xx
- tps_5xx
- tps_total

.. seealso:: For more information on the statistics gathered by Traffic Stats, see :ref:`ts-admin`. For information about how these statics are gathered, consult the only known documentation of the "astats_over_http" Apache Traffic Server plugin: :atc-file:`traffic_server/plugins/astats_over_http/README.md`.

sysctl.config
'''''''''''''
For each Parameter with this Config File value on the same :ref:`Profile <profiles>`, a line in the resulting configuration file is produced in the format :file:`{NAME} = {VALUE}` where ``NAME`` is the Parameter's :ref:`parameter-name` with trailing characters matching the regular expression :regexp:`__\\d+$` stripped out and ``VALUE`` is the Parameter's Value_.

:file:`uri_signing_{anything}.config`
'''''''''''''''''''''''''''''''''''''
Config Files matching this pattern - where ``anything`` is zero or more characters - are generated entirely from the URI Signing Keys configured on a :term:`Delivery Service` through either the :ref:`to-api` or the :ref:`tp-services-delivery-service` view in Traffic Portal.

.. seealso:: `The draft RFC for uri_signing <https://tools.ietf.org/html/draft-ietf-cdni-uri-signing-16>`_ - note, however that the current implementation of uri_signing uses Draft 12 of that RFC document, **NOT** the latest.

:file:`url_sig_{anything}.config`
'''''''''''''''''''''''''''''''''
Config Files that match this pattern - where ``anything`` is zero or more characters - are mostly generated using the URL Signature Keys as configured either through the :ref:`to-api` or the :ref:`tp-services-delivery-service` view in Traffic Portal. However, if no such keys have been configured, they may be provided by fall-back Parameters. In this case, for each Parameter on assigned to this Config File on the same :ref:`Profile <profiles>` a line is inserted into the resulting configuration file in the format :file:`{NAME} = {VALUE}` where ``NAME`` is the Parameter's :ref:`parameter-name` and ``VALUE`` is the Parameter's Value_.

.. seealso:: `The Apache Trafficserver documentation for the url_sig plugin <https://docs.trafficserver.apache.org/en/8.0.x/admin-guide/plugins/url_sig.en.html>`_.

volume.config
'''''''''''''
This Config File is peculiar in that it depends only on the existence of Parameters, and not each Parameter's actual Value_. The Parameters that affect the generated configuration file are the Parameters with the :ref:`Names <parameter-name>` "Drive Prefix", "RAM Drive Prefix", and "SSD Drive Prefix". Each of these Parameters must be assigned to the ``storage.config`` Config File - **NOT this Config File** - and, of course, be on the same :ref:`Profile <profiles>`. The contents of the generated Config File will be between zero and three lines (excluding headers) where the number of lines is equal to the number of the aforementioned Parameters that actually exist on the same :ref:`Profile <profiles>`. Each line has the format :file:`volume={i} scheme=http size={SIZE}%` where ``i`` is a natural number that ranges from 1 to the number of those Parameters that exist. ``SIZE`` is :math:`100 / N` - where :math:`N` is the number of those special Parameters that exist - truncated to the nearest natural number, e.g. :math:`100 / 3 = 33`.

.. seealso:: `The Apache Traffic Server volume.config file documentation <https://docs.trafficserver.apache.org/en/7.1.x/admin-guide/files/volume.config.en.html>`_.

.. _parameter-id:

ID
""
An integral, unique identifier for a Parameter. Note that Parameters must have a unique combination of `Config File`_, :ref:`parameter-name`, and Value_, and so those should be used for identifying a unique Parameter whenever possible.

.. impl-detail:: If two Profiles_ have been assigned Parameters that have the same values for `Config File`_, :ref:`parameter-name`, and Value_ then Traffic Ops actually only stores one Parameter object and merely *links* it to both Profiles_. This can be seen by inspecting the Parameters' IDs, as they will be the same. There are many cases where a user or developer must rely on this implementation detail, but both are encouraged to do so only when absolutely necessary.

.. _parameter-name:

Name
""""
The Name of a Parameter has different meanings depending on the type of any and all Profiles_ to which it is assigned, as well as the `Config File`_ to which the Parameter belongs, but most generally it is used in `Apache Traffic Server configuration files`_ as the name of a configuration option in a name/value pair. Traffic Ops interprets the Name and Value_ of a Parameter in intelligent ways depending on the type of object to which the :ref:`Profile <Profiles>` using the Parameter is assigned. For example, if `Config File`_ is ``records.config`` and the Parameter's :ref:`Profile <Profiles>` is assigned to a :term:`cache server`, then a single line is placed in the configuration file specified by `Config File`_, and that line will have the contents :file:`{Name} {Value}`. However, if the `Config File`_ of the Parameter is something without special meaning to Traffic Ops e.g. "foo", then a line containing **only** the Parameter's Value_ would be inserted into that file (presuming it also has a Parameter with a Name of :ref:`"location" <parameter-name-location>` and a `Config File`_ of "foo"). Additionally, there are a few Names that are treated specially by Traffic Control.

.. _parameter-name-location:

location
	The Value_ of this Parameter is to be interpreted as a path under which the configuration file specified by `Config File`_ shall be found (or written, if not found). Any configuration file that is to exist on a server must have an associated "location" Parameter, even if the contents of the file cannot be affected by Parameters.

	.. caution:: If a single :ref:`Profile <profiles>` has multiple "location" Parameters for the same `Config File`_ with different :ref:`Values <parameter-value>`, the actual location of the generated configuration file is undefined (but will be one of those Parameters' :ref:`Values <parameter-value>`).

header
	If the :ref:`Profile <profiles>` containing this Parameter is assigned to a server, **and** if the `Config File`_ is not one of the special values that Traffic Ops uses to determine special syntax formatting, then the Value_ of this Parameter will be used instead of the typical Traffic Ops header - *unless* it is the special string "none", in which case no header will be inserted at all.

	.. caution:: If a single :ref:`Profile <profiles>` has multiple "header" Parameters for the same `Config File`_ with different :ref:`Values <parameter-value>`, the actual header is undefined (but will be one of those Parameters' :ref:`Values <parameter-value>`).

.. _parameter-secure:

Secure
""""""
When this is 'true', a user requesting to see this Parameter will see the value ``********`` instead of its actual value if the user's permission :term:`Role` isn't 'admin'.

.. _parameter-value:

Value
"""""
In general, a Parameter's :dfn:`Value` can be anything, and in the vast majority of cases the Value is *in no way validated by Traffic Control*. Usually, though, the Value has a special meaning depending on the values of the Parameter's `Config File`_ and/or :ref:`parameter-name`.

.. [#xml-caveat] The contents of this file are not valid XML, but are rather XML-like so developers writing procedures that will consume and parse it should be aware of this, and note the actual syntax as specified in the `Apache Traffic Server documentation for logs_xml.config <https://docs.trafficserver.apache.org/en/6.2.x/admin-guide/files/logs_xml.config.en.html>`_
.. [#logs-format] This Value_ may safely contain double quotes (:kbd:`"`) as they will be backslash-escaped in the generated output.
.. [#logs-filter] This Value_ may safely contain backslashes (:kbd:`\\`) and single quotes (:kbd:`'`), as they will be backslash-escaped in the generated output.
