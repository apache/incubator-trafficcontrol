package main

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */
import "fmt"
import "net/http"
import "strings"

var EDGE_PROFILE_HEADER = "# DO NOT EDIT - Generated for " + EDGE_PROFILE_NAME + " by Traffic Ops (https://localhost:443/) on " + CURRENT_TIME.UTC().Format("Mon Jan 2 15:04:05 MST 2006")
var EDGE_SERVER_HEADER = "# DO NOT EDIT - Generated for " + EDGE_SERVER_HOSTNAME + " by Traffic Ops (https://localhost:443/) on " + CURRENT_TIME.Format("Mon Jan 2 15:04:05 MST 2006")
var MID_PROFILE_HEADER = "# DO NOT EDIT - Generated for " + MID_PROFILE_NAME + " by Traffic Ops (https://localhost:443/) on " + CURRENT_TIME.Format("Mon Jan 2 15:04:05 MST 2006")
var MID_SERVER_HEADER = "# DO NOT EDIT - Generated for " + MID_SERVER_HOSTNAME + " by Traffic Ops (https://localhost:443/) on " + CURRENT_TIME.Format("Mon Jan 2 15:04:05 MST 2006")
var CDN_HEADER = "# DO NOT EDIT - Generated for CDN " + CDN + " by Traffic Ops (https://localhost:443/) on " + CURRENT_TIME.Format("Mon Jan 2 15:04:05 MST 2006")

var ASTATS_CONFIG = `
allow_ip=0.0.0.0/0
allow_ip6=::/0
path=_astats
record_types=122
`

func edgeAstatsConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER + ASTATS_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midAstatsConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER + ASTATS_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func edgeCacheConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midCacheConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func chkConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(`[{"value":"0:off\t1:off\t2:on\t3:on\t4:on\t5:on\t6:off","name":"trafficserver"}]`))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var HOSTING_CONFIG = `
hostname=*   volume=1
`

func edgeHostingConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_SERVER_HEADER + HOSTING_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midHostingConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_SERVER_HEADER + HOSTING_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var IP_ALLOW_CONFIG_TOP = `
src_ip=127.0.0.1                                                              action=ip_allow   method=ALL                 
src_ip=::1                                                                    action=ip_allow   method=ALL                 `
var IP_ALLOW_CONFIG_BOTTOM = `
src_ip=0.0.0.0-255.255.255.255                                                action=ip_deny    method=PUSH|PURGE|DELETE   
src_ip=::-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff                             action=ip_deny    method=PUSH|PURGE|DELETE   
`
var IP_ALLOW_CONFIG_EDGE_IP_PREFIX = strings.Join(strings.Split(EDGE_SERVER_IP, ".")[:2], ".")
var IP_ALLOW_CONFIG_EDGE_IP_RANGE = fmt.Sprintf("%s.0.0-%s.255.255", IP_ALLOW_CONFIG_EDGE_IP_PREFIX, IP_ALLOW_CONFIG_EDGE_IP_PREFIX)
var MID_IP_ALLOW_CONFIG = "\n" + fmt.Sprintf("src_ip=%-70s method=ip_allow   method=ALL                    ", IP_ALLOW_CONFIG_EDGE_IP_RANGE) + `
src_ip=10.0.0.0-10.255.255.255                                                action=ip_allow   method=ALL                    
src_ip=172.16.0.0-172.31.255.255                                              action=ip_allow   method=ALL                 
src_ip=192.168.0.0-192.168.255.255                                            action=ip_allow   method=ALL                 `

func edgeIp_allowConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_SERVER_HEADER + IP_ALLOW_CONFIG_TOP + IP_ALLOW_CONFIG_BOTTOM))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midIp_allowConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_SERVER_HEADER + IP_ALLOW_CONFIG_TOP + MID_IP_ALLOW_CONFIG + IP_ALLOW_CONFIG_BOTTOM))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var EDGE_PARENT_CONFIG = `
dest_domain=. parent="` + MID_SERVER_HOSTNAME + "." + MID_SERVER_DOMAIN_NAME + fmt.Sprintf(":%d", MID_SERVER_TCP_PORT) + `|0.999;" round_robin=consistent_hash go_direct=false
`

func edgeParentConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_SERVER_HEADER + EDGE_PARENT_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midParentConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_SERVER_HEADER))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var PLUGIN_CONFIG = `
astats_over_http.so 
regex_revalidate.so --config regex_revalidate.config
`

func edgePluginConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER + PLUGIN_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midPluginConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER + PLUGIN_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var EDGE_RECORDS_CONFIG = `
CONFIG proxy.config.admin.user_id STRING ats
CONFIG proxy.config.body_factory.template_sets_dir STRING /etc/trafficserver/body_factory
CONFIG proxy.config.config_dir STRING /etc/trafficserver
CONFIG proxy.config.diags.debug.enabled INT 1
CONFIG proxy.config.dns.round_robin_nameservers INT 0
CONFIG proxy.config.exec_thread.autoconfig INT 0
CONFIG proxy.config.http.cache.required_headers INT 0
CONFIG proxy.config.http.connect_attempts_timeout INT 10
CONFIG proxy.config.http.enable_http_stats INT 1
CONFIG proxy.config.http.insert_response_via_str INT 3
CONFIG proxy.config.http.parent_proxy.retry_time INT 60
CONFIG proxy.config.http.parent_proxy_routing_enable INT 1
CONFIG proxy.config.http.server_ports STRING 80 80:ipv6 443:proto=http:ssl 443:ipv6:proto=http:ssl
CONFIG proxy.config.http.slow.log.threshold INT 10000
CONFIG proxy.config.http.transaction_active_timeout_in INT 0
CONFIG proxy.config.log.logfile_dir STRING /var/log/trafficserver
CONFIG proxy.config.proxy_name STRING __FULL_HOSTNAME__
CONFIG proxy.config.reverse_proxy.enabled INT 1
CONFIG proxy.config.ssl.CA.cert.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.client.CA.cert.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.client.cert.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.client.private_key.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.ocsp.enabled INT 1
CONFIG proxy.config.ssl.server.cert.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.server.private_key.path STRING /etc/trafficserver/ssl
CONFIG proxy.config.ssl.server.ticket_key.filename STRING NULL
CONFIG proxy.config.url_remap.remap_required INT 0
`

var MID_RECORDS_CONFIG = `
CONFIG proxy.config.admin.user_id STRING ats
CONFIG proxy.config.body_factory.template_sets_dir STRING /etc/trafficserver/body_factory
CONFIG proxy.config.config_dir STRING /etc/trafficserver
CONFIG proxy.config.diags.debug.enabled INT 1
CONFIG proxy.config.dns.round_robin_nameservers INT 0
CONFIG proxy.config.exec_thread.autoconfig INT 0
CONFIG proxy.config.http.cache.required_headers INT 0
CONFIG proxy.config.http.connect_attempts_timeout INT 10
CONFIG proxy.config.http.enable_http_stats INT 1
CONFIG proxy.config.http.insert_response_via_str INT 3
CONFIG proxy.config.http.parent_proxy.retry_time INT 60
CONFIG proxy.config.http.parent_proxy_routing_enable INT 1
CONFIG proxy.config.http.server_ports STRING 80 80:ipv6
CONFIG proxy.config.http.slow.log.threshold INT 10000
CONFIG proxy.config.http.transaction_active_timeout_in INT 0
CONFIG proxy.config.log.logfile_dir STRING /var/log/trafficserver
CONFIG proxy.config.proxy_name STRING __FULL_HOSTNAME__
CONFIG proxy.config.reverse_proxy.enabled INT 0
CONFIG proxy.config.url_remap.remap_required INT 0
`

func edgeRecordsConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER + EDGE_RECORDS_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midRecordsConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER + MID_RECORDS_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func regex_revalidateConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(CDN_HEADER + "\n"))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func edgeRemapConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_SERVER_HEADER + "\n"))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midRemapConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_SERVER_HEADER + "\n"))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var SET_DSCP_N_CONFIG = `
cond %{REMAP_PSEUDO_HOOK}
set-conn-dscp %d [L]
`

func setDSCP(n int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(CDN_HEADER + fmt.Sprintf(SET_DSCP_N_CONFIG, n)))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func setDSCPn(n int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		setDSCP(n, w, r)
	}
}

var STORAGE_CONFIG = `
/var/trafficserver/cache volume=1
`

func edgeStorageConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER + STORAGE_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midStorageConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER + STORAGE_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

var VOLUME_CONFIG = `
# TRAFFIC OPS NOTE: This is running with forced volumes - the size is irrelevant
volume=1 scheme=http size=100%
`

func edgeVolumeConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(EDGE_PROFILE_HEADER + VOLUME_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func midVolumeConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", SERVER_STRING)
	if r.Method == http.MethodGet {
		w.Write([]byte(MID_PROFILE_HEADER + VOLUME_CONFIG))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
	}
}
