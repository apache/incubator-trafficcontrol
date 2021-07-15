module github.com/apache/trafficcontrol

// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

go 1.15

replace (
	github.com/fsnotify/fsnotify v1.4.9 => github.com/fsnotify/fsnotify v1.3.0
	github.com/golang/protobuf v1.4.2 => github.com/golang/protobuf v0.0.0-20171021043952-1643683e1b54
	gopkg.in/yaml.v2 v2.3.0 => gopkg.in/yaml.v2 v2.2.8
)

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20180108190415-b31f603f5e1e
	github.com/GehirnInc/crypt v0.0.0-20190301055215-6c0105aabd46
	github.com/asaskevich/govalidator v0.0.0-20180319081651-7d2e70ef918f
	github.com/basho/backoff v0.0.0-20150307023525-2ff7c4694083 // indirect
	github.com/basho/riak-go-client v1.7.1-0.20170327205844-5587c16e0b8b
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cihub/seelog v0.0.0-20170110094445-7bfb7937d106
	github.com/dchest/siphash v1.1.0
	github.com/dgrijalva/jwt-go v3.2.1-0.20190620180102-5e25c22bd5d6+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-acme/lego v2.7.2+incompatible
	github.com/go-ozzo/ozzo-validation v3.0.3-0.20180119232150-44af65fe9adf+incompatible
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gofrs/flock v0.7.2-0.20190320160742-5135e617513b
	github.com/google/uuid v1.1.2
	github.com/hydrogen18/stoppableListener v0.0.0-20151210151943-dadc9ccc400c
	github.com/influxdata/influxdb v1.1.1-0.20170104212736-6a94d200c826
	github.com/jmoiron/sqlx v0.0.0-20170430194603-d9bd385d68c0
	github.com/json-iterator/go v1.1.6-0.20181024152841-05d041de1043
	github.com/kelseyhightower/envconfig v1.3.1-0.20180308190516-b2c5c876e265
	github.com/kylelemons/godebug v1.1.1-0.20201107061927-e693023230a4
	github.com/lestrrat-go/jwx v0.9.1-0.20190702045520-e35178ac2b1f
	github.com/lestrrat/go-jwx v0.0.0-20171104074836-2857e17763b6
	github.com/lib/pq v0.0.0-20170707053602-dd1fe2071026
	github.com/mattn/go-sqlite3 v1.14.5 // indirect
	github.com/miekg/dns v1.0.6-0.20180406150955-01d59357d468
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/pborman/getopt/v2 v2.1.0
	github.com/pkg/errors v0.8.2-0.20190227000051-27936f6d90f9
	github.com/stretchr/testify v1.6.1 // indirect
	go.etcd.io/bbolt v1.3.5
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0
	gopkg.in/asn1-ber.v1 v1.0.0-20170511165959-379148ca0225 // indirect
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.3.0
)
