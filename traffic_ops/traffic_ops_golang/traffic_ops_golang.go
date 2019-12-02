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

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/about"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/plugin"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/routing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/sys/unix"
)

// set the version at build time: `go build -X "main.version=..."`
var version = "development"

// Traffic Ops version, set before building
const Version = "3.0.0"

func init() {
	about.SetAbout(version)
}

func main() {
	showVersion := flag.Bool("version", false, "Show version and exit")
	showPlugins := flag.Bool("plugins", false, "Show the list of plugins and exit")
	configFileName := flag.String("cfg", "", "The config file path")
	dbConfigFileName := flag.String("dbcfg", "", "The db config file path")
	riakConfigFileName := flag.String("riakcfg", "", "The riak config file path")
	flag.Parse()

	if *showVersion {
		fmt.Println(about.About.RPMVersion)
		os.Exit(0)
	}
	if *showPlugins {
		fmt.Println(strings.Join(plugin.List(), "\n"))
		os.Exit(0)
	}
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	cfg, errsToLog, blockStart := config.LoadConfig(*configFileName, *dbConfigFileName, *riakConfigFileName, version)
	for _, err := range errsToLog {
		fmt.Fprintf(os.Stderr, "Loading Config: %v\n", err)
	}
	if blockStart {
		os.Exit(1)
	}

	if err := log.InitCfg(cfg); err != nil {
		fmt.Printf("Error initializing loggers: %v\n", err)
		for _, err := range errsToLog {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	for _, err := range errsToLog {
		log.Warnln(err)
	}

	logConfig(cfg)

	err := auth.LoadPasswordBlacklist("app/conf/invalid_passwords.txt")
	if err != nil {
		log.Errorf("loading password blacklist: %v\n", err)
		os.Exit(1)
	}

	sslStr := "require"
	if !cfg.DB.SSL {
		sslStr = "disable"
	}

	db, err := sqlx.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&fallback_application_name=trafficops", cfg.DB.User, cfg.DB.Password, cfg.DB.Hostname, cfg.DB.DBName, sslStr))
	if err != nil {
		log.Errorf("opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.MaxDBConnections)
	db.SetMaxIdleConns(cfg.DBMaxIdleConnections)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeSeconds) * time.Second)

	// TODO combine
	plugins := plugin.Get(cfg)
	profiling := cfg.ProfilingEnabled

	pprofMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux() // this is so we don't serve pprof over 443.

	pprofMux.Handle("/db-stats", routing.DBStatsHandler(db))
	pprofMux.Handle("/memory-stats", routing.MemoryStatsHandler())
	go func() {
		debugServer := http.Server{
			Addr:    "localhost:6060",
			Handler: pprofMux,
		}
		log.Errorln(debugServer.ListenAndServe())
	}()

	if err := routing.RegisterRoutes(routing.ServerData{DB: db, Config: cfg, Profiling: &profiling, Plugins: plugins}); err != nil {
		log.Errorf("registering routes: %v\n", err)
		os.Exit(1)
	}

	plugins.OnStartup(plugin.StartupData{Data: plugin.Data{SharedCfg: cfg.PluginSharedConfig, AppCfg: cfg}})

	log.Infof("Listening on " + cfg.Port)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		TLSConfig:         &tls.Config{InsecureSkipVerify: cfg.Insecure},
		ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
		ErrorLog:          log.Error,
	}

	go func() {
		if cfg.KeyPath == "" {
			log.Errorf("key cannot be blank in %s", cfg.ConfigHypnotoad.Listen)
			os.Exit(1)
		}

		if cfg.CertPath == "" {
			log.Errorf("cert cannot be blank in %s", cfg.ConfigHypnotoad.Listen)
			os.Exit(1)
		}

		if file, err := os.Open(cfg.CertPath); err != nil {
			log.Errorf("cannot open %s for read: %s", cfg.CertPath, err.Error())
			os.Exit(1)
		} else {
			file.Close()
		}

		if file, err := os.Open(cfg.KeyPath); err != nil {
			log.Errorf("cannot open %s for read: %s", cfg.KeyPath, err.Error())
			os.Exit(1)
		} else {
			file.Close()
		}

		if err := server.ListenAndServeTLS(cfg.CertPath, cfg.KeyPath); err != nil {
			log.Errorf("stopping server: %v\n", err)
			os.Exit(1)
		}
	}()

	profilingLocation, err := getProcessedProfilingLocation(cfg.ProfilingLocation, cfg.LogLocationError)
	if err != nil {
		log.Errorln("unable to determine profiling location: " + err.Error())
	}

	log.Infof("profiling location: %s\n", profilingLocation)
	log.Infof("profiling enabled set to %t\n", profiling)

	if profiling {
		continuousProfile(&profiling, &profilingLocation, cfg.Version)
	}

	reloadProfilingConfig := func() {
		setNewProfilingInfo(*configFileName, &profiling, &profilingLocation, cfg.Version)
	}
	signalReloader(unix.SIGHUP, reloadProfilingConfig)
}

func setNewProfilingInfo(configFileName string, currentProfilingEnabled *bool, currentProfilingLocation *string, version string) {
	newProfilingEnabled, newProfilingLocation, err := reloadProfilingInfo(configFileName)
	if err != nil {
		log.Errorln("reloading config: ", err.Error())
		return
	}
	if newProfilingLocation != "" && *currentProfilingLocation != newProfilingLocation {
		*currentProfilingLocation = newProfilingLocation
		log.Infof("profiling location set to: %s\n", *currentProfilingLocation)
	}
	if *currentProfilingEnabled != newProfilingEnabled {
		log.Infof("profiling enabled set to %t\n", newProfilingEnabled)
		log.Infof("profiling location set to: %s\n", *currentProfilingLocation)
		*currentProfilingEnabled = newProfilingEnabled
		if *currentProfilingEnabled {
			continuousProfile(currentProfilingEnabled, currentProfilingLocation, version)
		}
	}
}

func getProcessedProfilingLocation(rawProfilingLocation string, errorLogLocation string) (string, error) {
	profilingLocation := os.TempDir()

	if errorLogLocation != "" && errorLogLocation != log.LogLocationNull && errorLogLocation != log.LogLocationStderr && errorLogLocation != log.LogLocationStdout {
		errorDir := filepath.Dir(errorLogLocation)
		if _, err := os.Stat(errorDir); err == nil {
			profilingLocation = errorDir
		}
	}

	profilingLocation = filepath.Join(profilingLocation, "profiling")
	if rawProfilingLocation != "" {
		profilingLocation = rawProfilingLocation
	} else {
		//if it isn't a provided location create the profiling directory under the default temp location if it doesn't exist
		if _, err := os.Stat(profilingLocation); err != nil {
			err = os.Mkdir(profilingLocation, 0755)
			if err != nil {
				return "", fmt.Errorf("unable to create profiling location: %s", err.Error())
			}
		}
	}
	return profilingLocation, nil
}

func reloadProfilingInfo(configFileName string) (bool, string, error) {
	cfg, err := config.LoadCdnConfig(configFileName)
	if err != nil {
		return false, "", err
	}
	profilingLocation, err := getProcessedProfilingLocation(cfg.ProfilingLocation, cfg.LogLocationError)
	if err != nil {
		return false, "", err
	}
	return cfg.ProfilingEnabled, profilingLocation, nil
}

func continuousProfile(profiling *bool, profilingDir *string, version string) {
	if *profiling && *profilingDir != "" {
		go func() {
			for {
				now := time.Now().UTC()
				filename := filepath.Join(*profilingDir, fmt.Sprintf("tocpu-%s-%s.pprof", version, now.Format(time.RFC3339)))
				f, err := os.Create(filename)
				if err != nil {
					log.Errorf("creating profile: %v\n", err)
					log.Infof("Exiting profiling")
					break
				}

				pprof.StartCPUProfile(f)
				time.Sleep(time.Minute)
				pprof.StopCPUProfile()
				f.Close()
				if !*profiling {
					break
				}
			}
		}()
	}
}

func signalReloader(sig os.Signal, f func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)
	for range c {
		log.Debugln("received SIGHUP")
		f()
	}
}

func logConfig(cfg config.Config) {
	logRiakPort := "<nil>"
	if cfg.RiakPort != nil {
		logRiakPort = strconv.Itoa(int(*cfg.RiakPort))
	}
	log.Infof(`Using Config values:
		Port:                 %s
		Db Server:            %s
		Db User:              %s
		Db Name:              %s
		Db Ssl:               %t
		Max Db Connections:   %d
		TO URL:               %s
		Insecure:             %t
		Cert Path:            %s
		Key Path:             %s
		Proxy Timeout:        %v
		Proxy KeepAlive:      %v
		Proxy tls handshake:  %v
		Proxy header timeout: %v
		Read Timeout:         %v
		Read Header Timeout:  %v
		Write Timeout:        %v
		Idle Timeout:         %v
		Error Log:            %s
		Warn Log:             %s
		Info Log:             %s
		Debug Log:            %s
		Event Log:            %s
		Riak Port:            %v
		LDAP Enabled:         %v
		InfluxDB Enabled:     %v`, cfg.Port, cfg.DB.Hostname, cfg.DB.User, cfg.DB.DBName, cfg.DB.SSL, cfg.MaxDBConnections, cfg.Listen[0], cfg.Insecure, cfg.CertPath, cfg.KeyPath, time.Duration(cfg.ProxyTimeout)*time.Second, time.Duration(cfg.ProxyKeepAlive)*time.Second, time.Duration(cfg.ProxyTLSTimeout)*time.Second, time.Duration(cfg.ProxyReadHeaderTimeout)*time.Second, time.Duration(cfg.ReadTimeout)*time.Second, time.Duration(cfg.ReadHeaderTimeout)*time.Second, time.Duration(cfg.WriteTimeout)*time.Second, time.Duration(cfg.IdleTimeout)*time.Second, cfg.LogLocationError, cfg.LogLocationWarning, cfg.LogLocationInfo, cfg.LogLocationDebug, cfg.LogLocationEvent, logRiakPort, cfg.LDAPEnabled, cfg.InfluxEnabled)
}
