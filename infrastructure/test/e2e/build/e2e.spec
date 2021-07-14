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

Summary: Traffic Control End-to-End test framework
Name: tc-e2e
Version: %{version}
Release: %{build_number}
License: Apache License, Version 2.0
Group: Base System/System Tools
Prefix: /usr/sbin/%{name}
Source: %{_sourcedir}/%{name}-%{version}.tgz
URL: https://github.com/apache/trafficcontrol
Distribution: CentOS Linux
Vendor: Apache Software Foundation
BuildRoot: %{buildroot}

# %define PACKAGEDIR %{prefix}

%description
An End-to-End test framework for the Apache Traffic Control CDN.

%prep

%build
tar -xvzf %{_sourcedir}/%{name}-%{version}.tgz --directory %{_builddir}

%install
rm -rf %{buildroot}/usr/sbin/%{name}
mkdir -p %{buildroot}/usr/sbin/
cp -p %{name} %{buildroot}/usr/sbin/

rm -rf %{buildroot}/etc/%{name}
mkdir -p -m 777 %{buildroot}/etc/%{name}
cp -p conf/%{name}.cfg %{buildroot}/etc/%{name}

%clean
echo "cleaning"
rm -r -f %{buildroot}

%files
/usr/sbin/%{name}
%config(noreplace) /etc/%{name}
