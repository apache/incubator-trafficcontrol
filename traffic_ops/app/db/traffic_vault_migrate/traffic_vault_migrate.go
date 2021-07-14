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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/pborman/getopt/v2"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

var (
	fromType    string
	toType      string
	fromCfgPath string
	toCfgPath   string
	logCfgPath  string
	keyFile     string
	dry         bool
	compare     bool
	noConfirm   bool
	dump        bool
	logLevel    string

	cfg config = config{
		LogLocationError:   log.LogLocationStderr,
		LogLocationWarning: log.LogLocationStdout,
		LogLocationInfo:    log.LogLocationStdout,
		LogLocationDebug:   log.LogLocationNull,
		LogLocationEvent:   log.LogLocationNull,
	}
	riakBE RiakBackend = RiakBackend{}
	pgBE   PGBackend   = PGBackend{}
)

func init() {
	fromType = riakBE.Name()
	getopt.FlagLong(&fromType, "fromType", 't', fmt.Sprintf("From server types (%v)", strings.Join(supportedTypes(), "|")))

	toType = pgBE.Name()
	getopt.FlagLong(&fromType, "toType", 'o', fmt.Sprintf("From server types (%v)", strings.Join(supportedTypes(), "|")))

	toCfgPath = "pg.json"
	getopt.FlagLong(&toCfgPath, "toCfgPath", 'g', "To server config file")

	fromCfgPath = "riak.json"
	getopt.FlagLong(&fromCfgPath, "fromCfgPath", 'f', "From server config file")

	getopt.FlagLong(&dry, "dry", 'r', "Do not perform writes").
		SetOptional().
		SetFlag().
		SetGroup("no_insert")

	getopt.FlagLong(&compare, "compare", 'c', "Compare to and from server records").
		SetOptional().
		SetFlag().
		SetGroup("no_insert")

	getopt.FlagLong(&noConfirm, "noConfirm", 'm', "Requires confirmation before inserting records").
		SetFlag()

	getopt.FlagLong(&dump, "dump", 'd', "Write keys (from 'from' server) to disk").
		SetOptional().
		SetGroup("disk_bck").
		SetFlag()

	getopt.FlagLong(&keyFile, "fill", 'i', "Insert data into `to` server with data this directory").
		SetOptional().
		SetGroup("disk_bck")

	getopt.FlagLong(&logCfgPath, "logCfg", 'l', "Log configuration file").
		SetOptional().
		SetGroup("log")

	getopt.FlagLong(&logLevel, "logLevel", 'e', "Print everything at above specified log level (error|warning|info|debug|event)").
		SetOptional().
		SetGroup("log")
}

// supportBackends returns the backends available in this tool.
func supportedBackends() []TVBackend {
	return []TVBackend{
		&riakBE, &pgBE,
	}
}

func main() {
	getopt.ParseV2()

	initConfig()

	var fromSrv TVBackend
	var toSrv TVBackend

	importData := keyFile != ""
	toSrvUsed := !dump && !dry || keyFile != ""

	if !importData {
		log.Infof("Initiating fromSrv %s...\n", fromType)
		if !validateType(fromType) {
			log.Errorln("Unknown fromType " + fromType)
			os.Exit(1)
		}
		fromSrv = getBackendFromType(fromType)
		if err := fromSrv.ReadConfigFile(fromCfgPath); err != nil {
			log.Errorf("Unable to read fromSrv cfg: %v", err)
			os.Exit(1)
		}

		if err := fromSrv.Start(); err != nil {
			log.Errorf("issue starting fromSrv: %v", err)
			os.Exit(1)
		}
		defer log.Close(fromSrv, "closing fromSrv")

		if err := fromSrv.Ping(); err != nil {
			log.Errorf("Unable to ping fromSrv: %v", err)
			os.Exit(1)
		}
	}

	if toSrvUsed {
		log.Infof("Initiating toSrv %s...\n", toType)
		if !validateType(toType) {
			log.Errorln("Unknown toType " + toType)
			os.Exit(1)
		}
		toSrv = getBackendFromType(toType)

		if err := toSrv.ReadConfigFile(toCfgPath); err != nil {
			log.Errorf("Unable to read toSrv cfg: %v", err)
			os.Exit(1)
		}

		if err := toSrv.Start(); err != nil {
			log.Errorf("issue starting toSrv: %v", err)
			os.Exit(1)
		}
		defer log.Close(toSrv, "closing toSrv")

		if err := toSrv.Ping(); err != nil {
			log.Errorf("Unable to ping toSrv: %v", err)
			os.Exit(1)
		}
	}

	var fromSecret Secrets
	if !importData {
		var err error
		log.Infof("Fetching data from %s...\n", fromSrv.Name())
		if err = fromSrv.Fetch(); err != nil {
			log.Errorf("Unable to fetch fromSrv data: %v", err)
			os.Exit(1)
		}

		if fromSecret, err = GetKeys(fromSrv); err != nil {
			log.Errorln(err)
			os.Exit(1)
		}

		if err := Validate(fromSrv); err != nil {
			log.Errorln(err)
			os.Exit(1)
		}

	} else {
		err := fromSecret.fill(keyFile)
		if err != nil {
			log.Errorln(err)
			os.Exit(1)
		}
	}

	if dump {
		log.Infof("Dumping data from %s...\n", fromSrv.Name())
		fromSecret.dump("dump")
		return
	}

	if compare {
		log.Infof("Fetching data from %s...\n", toSrv.Name())
		if err := toSrv.Fetch(); err != nil {
			log.Errorf("Unable to fetch toSrv data: %v\n", err)
			os.Exit(1)
		}

		toSecret, err := GetKeys(toSrv)
		if err != nil {
			log.Errorln(err)
			os.Exit(1)
		}
		log.Infoln("Validating " + toSrv.Name())
		if err := toSrv.ValidateKey(); err != nil && len(err) > 0 {
			log.Errorln(strings.Join(err, "\n"))
			os.Exit(1)
		}

		fromSecret.sort()
		toSecret.sort()

		if !importData {
			log.Infoln(fromSrv.String())
		} else {
			log.Infof("Disk backup:\n\tSSL Keys: %d\n\tDNSSec Keys: %d\n\tURI Keys: %d\n\tURL Keys: %d\n", len(fromSecret.sslkeys), len(fromSecret.dnssecKeys), len(fromSecret.uriKeys), len(fromSecret.urlKeys))
		}
		log.Infoln(toSrv.String())

		if !reflect.DeepEqual(fromSecret.sslkeys, toSecret.sslkeys) {
			log.Errorln("from sslkeys and to sslkeys don't match")
			os.Exit(1)
		}
		if !reflect.DeepEqual(fromSecret.dnssecKeys, toSecret.dnssecKeys) {
			log.Errorln("from dnssec and to dnssec don't match")
			os.Exit(1)
		}
		if !reflect.DeepEqual(fromSecret.uriKeys, toSecret.uriKeys) {
			log.Errorln("from uri and to uri don't match")
			os.Exit(1)
		}
		if !reflect.DeepEqual(fromSecret.urlKeys, toSecret.urlKeys) {
			log.Errorln("from url and to url don't match")
			os.Exit(1)
		}
		log.Infoln("Both data sources have the same keys")
		return
	}

	if toSrvUsed {
		log.Infof("Setting %s keys...\n", toSrv.Name())
		if err := SetKeys(toSrv, fromSecret); err != nil {
			log.Errorln(err)
			os.Exit(1)
		}

		if err := Validate(toSrv); err != nil {
			log.Errorln(err)
			os.Exit(1)
		}
	}

	if !importData {
		log.Infoln(fromSrv.String())
	} else {
		log.Infof("Disk backup:\n\tSSL Keys: %d\n\tDNSSec Keys: %d\n\tURI Keys: %d\n\tURL Keys: %d\n", len(fromSecret.sslkeys), len(fromSecret.dnssecKeys), len(fromSecret.uriKeys), len(fromSecret.urlKeys))
	}

	if dry {
		return
	}

	if !noConfirm {
		ans := "q"
		for {
			fmt.Print("Confirm data insertion (y/n): ")
			if _, err := fmt.Scanln(&ans); err != nil {
				log.Errorln("unable to get user input")
				os.Exit(1)
			}

			if ans == "y" {
				break
			} else if ans == "n" {
				return
			}
		}
	}
	log.Infof("Inserting data into %s...\n", toSrv.Name())
	if err := toSrv.Insert(); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

// Validate runs the ValidateKey method on the backend.
func Validate(be TVBackend) error {
	if errs := be.ValidateKey(); errs != nil && len(errs) > 0 {
		return errors.New(fmt.Sprintf("Validation Errors (%s): \n%s", be.Name(), strings.Join(errs, "\n")))
	}
	return nil
}

// SetKeys will set all of the keys for a backend.
func SetKeys(be TVBackend, s Secrets) error {
	if err := be.SetSSLKeys(s.sslkeys); err != nil {
		return fmt.Errorf("Unable to set %s ssl keys: %w", be.Name(), err)
	}
	if err := be.SetDNSSecKeys(s.dnssecKeys); err != nil {
		return fmt.Errorf("Unable to set %s dnssec keys: %w", be.Name(), err)
	}
	if err := be.SetURLSigKeys(s.urlKeys); err != nil {
		return fmt.Errorf("Unable to set %v url keys: %v", be.Name(), err)
	}
	if err := be.SetURISignKeys(s.uriKeys); err != nil {
		return fmt.Errorf("Unable to set %v uri keys: %v", be.Name(), err)
	}
	return nil
}

// GetKeys will get all of the keys for a backend.
func GetKeys(be TVBackend) (Secrets, error) {
	var secret Secrets
	var err error
	if secret.sslkeys, err = be.GetSSLKeys(); err != nil {
		return Secrets{}, fmt.Errorf("Unable to get %v sslkeys: %v", be.Name(), err)
	}
	if secret.dnssecKeys, err = be.GetDNSSecKeys(); err != nil {
		return Secrets{}, fmt.Errorf("Unable to get %v dnssec keys: %v", be.Name(), err)
	}
	if secret.uriKeys, err = be.GetURISignKeys(); err != nil {
		return Secrets{}, fmt.Errorf("Unable to get %v uri keys: %v", be.Name(), err)
	}
	if secret.urlKeys, err = be.GetURLSigKeys(); err != nil {
		return Secrets{}, fmt.Errorf("Unable to %v url keys: %v", be.Name(), err)
	}
	return secret, nil
}

// UnmarshalConfig takes in a config file and a type and will read the config file into the reflected type.
func UnmarshalConfig(configFile string, config interface{}) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}

	return nil
}

// TVBackend represents a TV backend that can be have data migrated to/from
type TVBackend interface {
	// Start initiates the connection to the backend DB.
	Start() error
	// Close terminates the connection to the backend DB.
	Close() error
	// Ping checks the connection to the backend DB.
	Ping() error
	// ValidateKey validates that the keys are valid (in most cases, certain fields are not null).
	ValidateKey() []string
	// Name returns the name for this backend.
	Name() string
	// ReadConfigFile takes in a filename and will read it into the backends config.
	ReadConfigFile(string) error
	// String returns a high level overview of the backend and its keys.
	String() string

	// Fetch gets all of the keys from the backend DB.
	Fetch() error
	// Insert takes the current keys and inserts them into the backend DB.
	Insert() error

	// GetSSLKeys converts the backends internal key representation into the common representation (SSLKey).
	GetSSLKeys() ([]SSLKey, error)
	// SetSSLKeys takes in keys and converts & encrypts the data into the backends internal format.
	SetSSLKeys([]SSLKey) error

	// GetDNSSecKeys converts the backends internal key representation into the common representation (DNSSecKey).
	GetDNSSecKeys() ([]DNSSecKey, error)
	// SetDNSSecKeys takes in keys and converts & encrypts the data into the backends internal format.
	SetDNSSecKeys([]DNSSecKey) error

	// GetURISignKeys converts the pg internal key representation into the common representation (URISignKey).
	GetURISignKeys() ([]URISignKey, error)
	// SetURISignKeys takes in keys and converts & encrypts the data into the backends internal format.
	SetURISignKeys([]URISignKey) error

	// GetURLSigKeys converts the backends internal key representation into the common representation (URLSigKey).
	GetURLSigKeys() ([]URLSigKey, error)
	// SetURLSigKeys takes in keys and converts & encrypts the data into the backends internal format.
	SetURLSigKeys([]URLSigKey) error
}

// Secrets contains every key to be migrated.
type Secrets struct {
	sslkeys    []SSLKey
	dnssecKeys []DNSSecKey
	uriKeys    []URISignKey
	urlKeys    []URLSigKey
}

func (s *Secrets) sort() {
	sort.Slice(s.sslkeys[:], func(a, b int) bool {
		return s.sslkeys[a].CDN < s.sslkeys[b].CDN ||
			s.sslkeys[a].CDN == s.sslkeys[b].CDN && s.sslkeys[a].DeliveryService < s.sslkeys[b].DeliveryService ||
			s.sslkeys[a].CDN == s.sslkeys[b].CDN && s.sslkeys[a].DeliveryService == s.sslkeys[b].DeliveryService && s.sslkeys[a].Version < s.sslkeys[b].Version
	})
	sort.Slice(s.dnssecKeys[:], func(a, b int) bool {
		return s.dnssecKeys[a].CDN < s.dnssecKeys[b].CDN
	})
	sort.Slice(s.uriKeys[:], func(a, b int) bool {
		return s.uriKeys[a].DeliveryService < s.uriKeys[b].DeliveryService
	})
	sort.Slice(s.urlKeys[:], func(a, b int) bool {
		return s.urlKeys[a].DeliveryService < s.urlKeys[b].DeliveryService
	})
}
func (s *Secrets) dump(directory string) {
	if err := os.Mkdir(directory, 0750); err != nil {
		if !os.IsExist(err) {
			log.Errorln(err)
			os.Exit(1)
		}
	}
	if err := writeKeys(directory+"/sslkeys.json", &s.sslkeys); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
	if err := writeKeys(directory+"/dnsseckeys.json", &s.dnssecKeys); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
	if err := writeKeys(directory+"/urlkeys.json", &s.urlKeys); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
	if err := writeKeys(directory+"/urikeys.json", &s.uriKeys); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}
func (s *Secrets) fill(directory string) error {
	if err := readKeys(directory+"/sslkeys.json", &s.sslkeys); err != nil {
		return err
	}
	if err := readKeys(directory+"/dnsseckeys.json", &s.dnssecKeys); err != nil {
		return err
	}
	if err := readKeys(directory+"/urlkeys.json", &s.urlKeys); err != nil {
		return err
	}
	if err := readKeys(directory+"/urikeys.json", &s.uriKeys); err != nil {
		return err
	}
	return nil
}

// SSLKey is the common representation of a SSL Key.
type SSLKey struct {
	tc.DeliveryServiceSSLKeys
	Version string
}

// DNSSecKey is the common representation of a DNSSec Key.
type DNSSecKey struct {
	CDN string
	tc.DNSSECKeysTrafficVault
}

// URISignKey is the common representation of an URI Signing Key.
type URISignKey struct {
	DeliveryService string
	Keys            map[string]tc.URISignerKeyset
}

// URLSigKey is the common representation of an URL Sig Key.
type URLSigKey struct {
	DeliveryService string
	tc.URLSigKeys
}

type config struct {
	LogLocationError   string `json:"error_log"`
	LogLocationWarning string `json:"warning_log"`
	LogLocationInfo    string `json:"info_log"`
	LogLocationDebug   string `json:"debug_log"`
	LogLocationEvent   string `json:"event_log"`
}

func (c config) ErrorLog() log.LogLocation   { return log.LogLocation(c.LogLocationError) }
func (c config) WarningLog() log.LogLocation { return log.LogLocation(c.LogLocationWarning) }
func (c config) InfoLog() log.LogLocation    { return log.LogLocation(c.LogLocationInfo) }
func (c config) DebugLog() log.LogLocation   { return log.LogLocation(c.LogLocationDebug) }
func (c config) EventLog() log.LogLocation   { return log.LogLocation(c.LogLocationEvent) }

func writeKeys(filename string, data interface{}) error {
	bytes, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(filename, bytes, 0600); err != nil {
		return err
	}

	return nil
}
func readKeys(filename string, data interface{}) error {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return errors.New("file " + filename + " does not exist")
		}
	}

	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileBytes, &data)
	if err != nil {
		return err
	}
	return nil
}
func supportedTypes() []string {
	var typs []string
	for _, be := range supportedBackends() {
		typs = append(typs, be.Name())
	}
	return typs
}
func validateType(typ string) bool {
	for _, be := range supportedBackends() {
		if typ == be.Name() {
			return true
		}
	}
	return false
}
func getBackendFromType(typ string) TVBackend {
	for _, be := range supportedBackends() {
		if be.Name() == typ {
			return be
		}
	}
	return nil
}
func initConfig() {
	if logCfgPath != "" {
		if _, err := os.Stat(logCfgPath); err != nil {
			if os.IsNotExist(err) {
				stdlog.Fatal("file '" + logCfgPath + "' does not exist")
			}
		}
		data, err := ioutil.ReadFile(logCfgPath)
		if err != nil {
			stdlog.Fatal(err)
		}

		var newCfg config
		err = json.Unmarshal(data, &newCfg)
		if err != nil {
			stdlog.Fatal(err)
		}
		cfg = newCfg
	} else if logLevel != "" {
		switch logLevel {
		case "event":
			cfg.LogLocationEvent = log.LogLocationStdout
			fallthrough
		case "debug":
			cfg.LogLocationDebug = log.LogLocationStdout
			fallthrough
		case "info":
			cfg.LogLocationInfo = log.LogLocationStdout
			fallthrough
		case "warning":
			cfg.LogLocationWarning = log.LogLocationStdout
			fallthrough
		case "error":
			cfg.LogLocationError = log.LogLocationStderr
		default:
			stdlog.Fatal("unknown logLevel " + logLevel)
		}
	}

	err := log.InitCfg(cfg)
	if err != nil {
		stdlog.Fatal(err)
	}
}
