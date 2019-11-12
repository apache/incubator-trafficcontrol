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

import "crypto/tls"
import "flag"
import "fmt"
import "log"
import "net/http"
import "os"
import "strconv"

import "github.com/apache/trafficcontrol/lib/go-tc"

const API_MIN_MINOR_VERSION = 1
const API_MAX_MINOR_VERSION = 5

// TODO: read this in properly from the VERSION file
const VERSION = "3.0.0"

var SERVER_STRING = "Traffic Ops/" + VERSION + " (Mock)"

// This needs to exist for... reasons
const TM_LOGO = `<?xml version="1.0" encoding="UTF-8"?>
<svg width="1391px" height="888px" viewBox="0 0 789 789" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <title>Apache Traffic Control Logo</title>
    <desc>Logo with no text</desc>
    <defs></defs>
    <g stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <path d="M933.483367,152.57612 L876.912723,209.146764 C820.077911,152.80267 741.853255,118 655.5,118 C481.806446,118 341,258.806446 341,432.5 C341,606.193554 481.806446,747 655.5,747 C742.585973,747 821.404817,711.604215 878.354845,654.414332 L933.923488,709.982975 C862.496219,781.649964 763.677363,826 654.5,826 C436.623666,826 260,649.376334 260,431.5 C260,213.623666 436.623666,37 654.5,37 C763.453296,37 862.09056,81.1681819 933.483367,152.57612 Z" id="Combined-Shape" fill="#F49224"></path>
        <path d="M416.601816,396 L504.202305,396 C501.454569,407.552665 500,419.606466 500,432 C500,517.604136 569.395864,587 655,587 C740.604136,587 810,517.604136 810,432 C810,346.395864 740.604136,277 655,277 C642.963616,277 631.247667,278.371943 620,280.967982 L620,193.456017 C631.266638,191.837544 642.78551,191 654.5,191 C787.324482,191 895,298.675518 895,431.5 C895,564.324482 787.324482,672 654.5,672 C521.675518,672 414,564.324482 414,431.5 C414,419.438696 414.88787,407.584765 416.601816,396 Z" id="Combined-Shape" fill="#F49224"></path>
        <rect id="Rectangle" fill="#F49224" x="620" y="613" width="80" height="161"></rect>
        <rect id="Rectangle" fill="#F49224" x="620" y="77" width="80" height="161"></rect>
        <rect id="Rectangle" fill="#F49224" transform="translate(394.000000, 436.000000) rotate(90.000000) translate(-394.000000, -436.000000) " x="354" y="360" width="80" height="152"></rect>
    </g>
</svg>
`

// Will be used for various `lastUpdated` fields
var CURRENT_TIME *tc.TimeNoMod = tc.NewTimeNoMod()

// Just sets some common headers
func common(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", SERVER_STRING)
}

func ping(w http.ResponseWriter, r *http.Request) {
	common(w)
	w.Write([]byte("{\"ping\":\"pong\"}\n"))
}

func mockDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Disposition", `attachment; filename="database.dat"`)
		w.Write([]byte("Mock Traffic Ops servers don't have real databases\n"))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func logo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "Traffic Ops/"+VERSION+" (Mock)")
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(TM_LOGO))
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	certPathLong := flag.String("cert-path", "", "Specify a path to an SSL certificate for the mock server to use (default ./localhost.crt)")
	certPathShort := flag.String("c", "", "Specify a path to an SSL certificate for the mock server to use (default ./localhost.crt)")
	helpLong := flag.Bool("help", false, "Print usage information and exit")
	helpShort := flag.Bool("h", false, "Print usage information and exit")
	keyPathLong := flag.String("key-path", "", "Specify a path to an SSL private key for the mock server to use (default ./localhost.key)")
	keyPathShort := flag.String("k", "", "Specify a path to an SSL private key for the mock server to use (default ./localhost.key)")
	listenAddrLong := flag.String("listen", "", "Choose the address or hostname on which the mock server will listen (default '')")
	listenAddrShort := flag.String("l", "", "Choose the address or hostname on which the mock server will listen (default '')")
	portLong := flag.Uint("port", 443, "Choose the port on which the mock server will listen for incoming connections")
	portShort := flag.Uint("p", 443, "Choose the port on which the mock server will listen for incoming connections")
	versionLong := flag.Bool("version", false, "Print version information and exit")
	versionShort := flag.Bool("v", false, "Print version information and exit")

	flag.Parse()

	version := *versionLong || *versionShort
	if version {
		fmt.Printf("Traffic Ops Mock server, built for version %s\n", VERSION)
		os.Exit(0)
	}

	help := *helpLong || *helpShort
	if help {
		flag.Usage()
		os.Exit(0)
	}

	var port uint
	if *portLong != 443 {
		if *portLong == 0 || *portLong >= 65536 {
			log.Fatalf("Invalid port number: %d\n", *portLong)
		}
		if *portShort != 443 && *portShort != *portLong {
			log.Fatalf("Must specify exactly one port number!\n")
		}
		port = *portLong
	} else if *portShort != 443 {
		if *portShort == 0 || *portShort >= 65536 {
			log.Fatalf("Invalid port number: %d\n", *portLong)
		}
		port = *portShort
	} else {
		port = 443
	}

	listenAddr := ""
	if *listenAddrLong != "" {
		if *listenAddrShort != "" && *listenAddrShort != *listenAddrLong {
			log.Fatalf("Must specify exactly one listen address!\n")
		}
		listenAddr = *listenAddrLong
	} else if *listenAddrShort != "" {
		listenAddr = *listenAddrShort
	}

	certpath := "./localhost.crt"
	if *certPathLong != "" {
		if *certPathShort != "" && *certPathShort != *certPathLong {
			log.Fatalf("Must specify exactly one path to an SSL certificate!\n")
		}
		certpath = *certPathLong
	} else if *certPathShort != "" {
		certpath = *certPathShort
	}

	keypath := "./localhost.key"
	if *keyPathLong != "" {
		if *keyPathShort != "" && *keyPathShort != *keyPathLong {
			log.Fatalf("Must specify exactly one path to an SSL private key!\n")
		}
		keypath = *keyPathLong
	} else if *keyPathShort != "" {
		keypath = *keyPathShort
	}

	server := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", listenAddr, port),
		TLSConfig: &tls.Config{InsecureSkipVerify: false},
	}

	for i := API_MIN_MINOR_VERSION; i <= API_MAX_MINOR_VERSION; i += 1 {
		v := "1." + strconv.Itoa(i)
		log.Printf("Loading API v%s\n", v)
		http.HandleFunc("/api/"+v+"/ping", ping)
		http.HandleFunc("/api/"+v+"/user/login", login)
		http.HandleFunc("/api/"+v+"/user/current", cuser)
		http.HandleFunc("/api/"+v+"/cdns", CDNs)
		http.HandleFunc("/api/"+v+"/cachegroups", cacheGroups)
		http.HandleFunc("/api/"+v+"/profiles", profiles)
		http.HandleFunc("/api/"+v+"/parameters", parameters)
		http.HandleFunc("/api/"+v+"/divisions", divisions)
		http.HandleFunc("/api/"+v+"/regions", regions)
		http.HandleFunc("/api/"+v+"/phys_locations", locations)
		http.HandleFunc("/api/"+v+"/statuses", statuses)
		http.HandleFunc("/api/"+v+"/types", types)
		http.HandleFunc("/api/"+v+"/servers", servers)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats", edgeConfigFiles)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats", midConfigFiles)
		http.HandleFunc("/api/"+v+"/servers/"+TO_SERVER_HOSTNAME+"/configfiles/ats", toConfigFiles)
		http.HandleFunc("/api/"+v+"/servers/"+TODB_SERVER_HOSTNAME+"/configfiles/ats", toDbConfigFiles)
		http.HandleFunc("/api/"+v+"/profiles/"+EDGE_PROFILE_NAME+"/configfiles/ats/astats.config", edgeAstatsConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+MID_PROFILE_NAME+"/configfiles/ats/astats.config", midAstatsConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+EDGE_PROFILE_NAME+"/configfiles/ats/cache.config", edgeCacheConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+MID_PROFILE_NAME+"/configfiles/ats/cache.config", midCacheConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats/chkconfig", chkConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats/chkconfig", chkConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats/hosting.config", edgeHostingConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats/hosting.config", midHostingConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats/ip_allow.config", edgeIp_allowConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats/ip_allow.config", midIp_allowConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats/parent.config", edgeParentConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats/parent.config", midParentConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_PROFILE_NAME+"/configfiles/ats/plugin.config", edgePluginConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_PROFILE_NAME+"/configfiles/ats/plugin.config", midPluginConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+EDGE_PROFILE_NAME+"/configfiles/ats/records.config", edgeRecordsConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+MID_PROFILE_NAME+"/configfiles/ats/records.config", midRecordsConfig)
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/regex_revalidate.config", regex_revalidateConfig)
		http.HandleFunc("/api/"+v+"/servers/"+EDGE_SERVER_HOSTNAME+"/configfiles/ats/remap.config", edgeRemapConfig)
		http.HandleFunc("/api/"+v+"/servers/"+MID_SERVER_HOSTNAME+"/configfiles/ats/remap.config", midRemapConfig)
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_0.config", setDSCPn(0))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_8.config", setDSCPn(8))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_10.config", setDSCPn(10))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_12.config", setDSCPn(12))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_14.config", setDSCPn(14))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_16.config", setDSCPn(16))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_18.config", setDSCPn(18))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_22.config", setDSCPn(22))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_24.config", setDSCPn(24))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_26.config", setDSCPn(26))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_28.config", setDSCPn(28))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_30.config", setDSCPn(30))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_32.config", setDSCPn(32))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_34.config", setDSCPn(34))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_36.config", setDSCPn(36))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_37.config", setDSCPn(37))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_38.config", setDSCPn(38))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_40.config", setDSCPn(40))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_48.config", setDSCPn(48))
		http.HandleFunc("/api/"+v+"/cdns/"+CDN+"/configfiles/ats/set_dscp_56.config", setDSCPn(56))
		http.HandleFunc("/api/"+v+"/profiles/"+EDGE_PROFILE_NAME+"/configfiles/ats/storage.config", edgeStorageConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+MID_PROFILE_NAME+"/configfiles/ats/storage.config", midStorageConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+EDGE_PROFILE_NAME+"/configfiles/ats/volume.config", edgeVolumeConfig)
		http.HandleFunc("/api/"+v+"/profiles/"+MID_PROFILE_NAME+"/configfiles/ats/volume.config", midVolumeConfig)
	}

	http.HandleFunc("/mock/geo/database.dat", mockDatabase)
	http.HandleFunc("/logo.svg", logo)

	log.Printf("Finished loading API routes at %s, server listening on %s on port %d", CURRENT_TIME.String(), listenAddr, port)

	if err := server.ListenAndServeTLS(certpath, keypath); err != nil {
		log.Fatalf("Server crashed: %v\n", err)
	}
	os.Exit(0)
}
