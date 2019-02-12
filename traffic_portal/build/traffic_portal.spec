#
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
# RPM spec file for the Traffic Portal
#
%define		debug_package %{nil}
Name:		traffic_portal
Version:	%{traffic_control_version}
Release:	%{build_number}
Summary:	Traffic Portal
Group:		Applications/Communications
License:	Apache License, Version 2.0
URL:		https://github.com/apache/trafficcontrol/
Source:		%{_sourcedir}/traffic_portal-%{traffic_control_version}.tgz
AutoReqProv: no
Requires: nodejs

%define traffic_portal_home /opt/traffic_portal
%description
Installs Traffic Portal

Built: @BUILT@

%prep
rm -rf $RPM_BUILD_DIR/traffic_portal-%{version}
tar -xzvf $RPM_SOURCE_DIR/traffic_portal-%{version}.tgz

%setup

%build
    /usr/bin/npm install
    /usr/bin/grunt dist

%install
    %__mkdir -p ${RPM_BUILD_ROOT}/etc/init.d
    %__mkdir -p ${RPM_BUILD_ROOT}/etc/logrotate.d
    %__mkdir -p ${RPM_BUILD_ROOT}/etc/traffic_portal
    %__mkdir -p ${RPM_BUILD_ROOT}%{traffic_portal_home}/public
    %__mkdir -p ${RPM_BUILD_ROOT}%{traffic_portal_home}/server
    %__mkdir -p ${RPM_BUILD_ROOT}/var/log/traffic_portal

    %__cp ${RPM_BUILD_DIR}/traffic_portal-%{version}/server.js ${RPM_BUILD_ROOT}%{traffic_portal_home}/.
    %__cp -r ${RPM_BUILD_DIR}/traffic_portal-%{version}/conf ${RPM_BUILD_ROOT}/etc/traffic_portal/.
    %__cp ${RPM_BUILD_DIR}/traffic_portal-%{version}/build/etc/init.d/traffic_portal ${RPM_BUILD_ROOT}/etc/init.d/.
    %__cp ${RPM_BUILD_DIR}/traffic_portal-%{version}/build/etc/logrotate.d/traffic_portal ${RPM_BUILD_ROOT}/etc/logrotate.d/.
    %__cp ${RPM_BUILD_DIR}/traffic_portal-%{version}/build/etc/logrotate.d/traffic_portal-access ${RPM_BUILD_ROOT}/etc/logrotate.d/.
    %__cp -r ${RPM_BUILD_DIR}/traffic_portal-%{version}/app/dist/* ${RPM_BUILD_ROOT}%{traffic_portal_home}/.

	# creates dynamic json file needed at runtime for traffic portal to display release info
	VERSION=%{version}-%{build_number}
	BUILD_DATE=$(date +'%Y-%m-%d %H:%M:%S')
	VERSION="\"Version\":\"$VERSION\""
	BUILD_DATE="\"Build Date\":\"$BUILD_DATE\""
	JSON_VERSION="{\n$VERSION,\n$BUILD_DATE\n}"
	echo -e $JSON_VERSION > ${RPM_BUILD_ROOT}%{traffic_portal_home}/public/traffic_portal_release.json

%post
    echo "Successfully installed the traffic_portal assets to " %{traffic_portal_home}
    %__chmod +x %{traffic_portal_home}/node_modules/forever/bin/forever
    %__chmod +x /etc/init.d/traffic_portal
    echo "Successfully installed the 'traffic_portal' service"
    /sbin/chkconfig traffic_portal on
    echo ""
    echo "Start with 'service traffic_portal start'"

%files
%defattr(644,root,root,755)
%attr(755,root,root) /etc/init.d/traffic_portal
%attr(755,root,root) %{traffic_portal_home}/node_modules/forever/bin/*
%config(noreplace)/etc/traffic_portal/conf/config.js
%config(noreplace)%{traffic_portal_home}/public/traffic_portal_properties.json
%dir /var/log/traffic_portal
%{traffic_portal_home}/*
/etc/logrotate.d/traffic_portal
/etc/logrotate.d/traffic_portal-access
/etc/init.d/traffic_portal
