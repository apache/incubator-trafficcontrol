package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const HealthPort = 33417
const HealthSubdomain = `health`
const NearSubdomain = `near`
const FarSubdomain = `far`

func main() {
	port := flag.Int("port", 80, "Port to serve on")
	// TODO separate server and client timeouts
	serverTimeoutMS := flag.Int("server-timeout-ms", 5000, "HTTP read and write timeout for this server")
	clientTimeoutMS := flag.Int("client-timeout-ms", 5000, "HTTPread and write timeout for client requests to ATS")
	insecure := flag.Bool("insecure", false, "Whether to ignore HTTPS certificate errors when making client requests")
	debug := flag.Bool("debug", false, "Whether to enable debug HTTP directives. Unsecured, should never be enabled in a production environment.")
	crConfigPath := flag.String("crconfig-path", "", "CRConfig path")
	crConfigIntervalMS := flag.Int("crconfig-interval-ms", 30000, "CRConfig refresh interavl")
	flag.Parse()

	// TODO add flag hostname override
	// The hostname isn't actually used to request locally, but rather to look up the ATS port in the CRConfig.
	// The request is always to localhost.
	// TODO add flag override to actually request the override host, to allow the healthcombiner to run remotely.
	hostName := os.Getenv("HOSTNAME")

	if *crConfigPath == "" {
		fmt.Fprintf(os.Stderr, "-crconfig-path is required\n")
		os.Exit(1)
	}

	serverTimeout := time.Duration(*serverTimeoutMS) * time.Millisecond
	clientTimeout := time.Duration(*clientTimeoutMS) * time.Millisecond
	crConfigInterval := time.Duration(*crConfigIntervalMS) * time.Millisecond

	handlerCfg := &HandlerCfg{Debug: *debug, ClientTimeout: clientTimeout, Insecure: *insecure, HostName: hostName}

	crc, err := GetCRConfig(*crConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR getting CRConfig '"+*crConfigPath+"': "+err.Error()+"\n")
		os.Exit(1)
	}
	crcPtr := (unsafe.Pointer)(crc)
	handlerCfg.crConfig = &crcPtr

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(*port),
		Handler:      MakeHandler(handlerCfg),
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
	}

	go RefreshCRConfig(handlerCfg, crConfigInterval, *crConfigPath)

	{
		// start workers

		const numWorkers = 100
		const workerChanBufSize = 10

		reqCh := make(chan Req, workerChanBufSize)

		httpClient := &http.Client{
			Timeout: handlerCfg.ClientTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: handlerCfg.Insecure},
			},
		}

		sv, ok := crc.ContentServers[hostName]
		if !ok {
			// TODO continue and use default 80?
			fmt.Fprintf(os.Stderr, "CRConfig missing this server '"+hostName+"'")
			os.Exit(1)
		}
		localPort := 80
		if sv.Port != nil && *sv.Port != 80 {
			localPort = *sv.Port
		}

		reqFQDN := "localhost"
		reqURI := "http://" + reqFQDN
		if localPort != 80 {
			reqURI += ":" + strconv.Itoa(localPort)
		}

		reqURL, err := url.Parse(reqURI)
		if err != nil {
			fmt.Fprintf(os.Stderr, "creating URL: "+err.Error())
			os.Exit(1)
		}

		for i := 0; i < numWorkers; i++ {
			go StartReqWorker(httpClient, reqURL, reqCh)
		}
		handlerCfg.ReqCh = reqCh
	}

	fmt.Println("Listening on " + server.Addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("ERROR: " + err.Error())
		os.Exit(1)
	}
	os.Exit(0) // should never happen; unless we add a "shutdown" directive
}

type HandlerCfg struct {
	Debug         bool
	ClientTimeout time.Duration
	Insecure      bool
	HostName      string
	ReqCh         chan Req

	// crConfig is set atomically. DO NOT access outside atomic operations, i.e. GetCRConfig,SetCRConfig
	crConfig *unsafe.Pointer
}

func (h *HandlerCfg) GetCRConfig() *tc.CRConfig {
	return (*tc.CRConfig)(atomic.LoadPointer(h.crConfig))
}

func (h *HandlerCfg) SetCRConfig(crc *tc.CRConfig) {
	atomic.StorePointer(h.crConfig, (unsafe.Pointer)(crc))
}

// MakeHandler currently takes just the debug bool, but it can be changed to take lots of things if necessary.
func MakeHandler(cfg *HandlerCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, cfg)
	}
}

func handler(w http.ResponseWriter, r *http.Request, cfg *HandlerCfg) {
	handlerCRStates(w, r, cfg)
}

func handlerCRStates(w http.ResponseWriter, r *http.Request, cfg *HandlerCfg) {
	switch r.Method {
	case http.MethodGet:
		if !strings.HasPrefix(strings.ToLower(r.URL.Path), `/crstates`) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		crStates, err := buildCRStates(cfg)
		if err != nil {
			fmt.Println("ERROR building CRStates: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(crStates); err != nil {
			// TODO don't log? We don't generally log client errors.
			//      Should we encode to bytes first, to be sure it's a client err not a json err?
			fmt.Println("ERROR writing CRStates: " + err.Error())
		}
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

type CRStates struct {
	Caches          map[tc.CacheName]IsAvailable                       `json:"caches"`
	DeliveryService map[tc.DeliveryServiceName]CRStatesDeliveryService `json:"deliveryServices"`
}
type CRStatesDeliveryService struct {
	DisabledLocations []tc.CacheGroupName `json:"disabledLocations"`
	IsAvailable       bool                `json:"isAvailable"`
}
type IsAvailable struct {
	IsAvailable bool `json:"isAvailable"`
}

func buildCRStates(cfg *HandlerCfg) (*CRStates, error) {
	crs := &CRStates{
		Caches:          map[tc.CacheName]IsAvailable{},
		DeliveryService: map[tc.DeliveryServiceName]CRStatesDeliveryService{},
	}

	crc := cfg.GetCRConfig()

	cdnDomainI, ok := crc.Config[`domain_name`]
	if !ok {
		return nil, errors.New("CRConfig Config missing 'domain_name'")
	}
	cdnDomain, ok := cdnDomainI.(string)
	if !ok {
		return nil, fmt.Errorf("CRConfig Config 'domain_name' unexpected type %T", cdnDomainI)
	}

	sv, ok := crc.ContentServers[cfg.HostName]
	if !ok {
		// TODO continue and use default 80?
		return nil, fmt.Errorf("CRConfig missing this server '" + cfg.HostName + "'")
	}
	localPort := 80
	if sv.Port != nil && *sv.Port != 80 {
		localPort = *sv.Port
	}

	type RespStruct struct {
		Server string
		Near   bool
		Ch     chan bool
	}
	responses := []RespStruct{}

	// TODO also get mids
	for svName, sv := range crc.ContentServers {
		if sv.ServerStatus == nil {
			return nil, errors.New("CRConfig server '" + svName + "' missing status")
		}
		if tc.CacheStatus(*sv.ServerStatus) == tc.CacheStatusOffline {
			continue // TODO warn? Shouldn't even be in the CRConfig
		}
		if tc.CacheStatus(*sv.ServerStatus) == tc.CacheStatusAdminDown {
			fmt.Println("DEBUG server '" + svName + "' AdminDown - unavailable")
			crs.Caches[tc.CacheName(svName)] = IsAvailable{IsAvailable: false}
			continue
		}
		if tc.CacheStatus(*sv.ServerStatus) == tc.CacheStatusOnline {
			fmt.Println("DEBUG server '" + svName + "' Online - available")
			crs.Caches[tc.CacheName(svName)] = IsAvailable{IsAvailable: true}
			continue
		}
		if tc.CacheStatus(*sv.ServerStatus) != tc.CacheStatusReported {
			// TODO error: unknown status
			fmt.Println("DEBUG server '" + svName + "' unknown - unavailable")
			crs.Caches[tc.CacheName(svName)] = IsAvailable{IsAvailable: false}
			continue
		}

		// TODO remove duplicate with near and far requests/availability

		healthFQDN := HealthSubdomain + `.` + svName + `.` + cdnDomain
		hostHdr := healthFQDN
		if localPort != 80 {
			hostHdr += ":" + strconv.Itoa(localPort)
		}

		respCh := make(chan bool, 1)
		cfg.ReqCh <- Req{Host: NearSubdomain + `.` + hostHdr, Resp: respCh}
		responses = append(responses, RespStruct{Server: svName, Near: true, Ch: respCh})

		respChFar := make(chan bool, 1)
		cfg.ReqCh <- Req{Host: FarSubdomain + `.` + hostHdr, Resp: respChFar}
		responses = append(responses, RespStruct{Server: svName, Near: false, Ch: respChFar})
	}

	nearAvail := map[tc.CacheName]bool{}
	farAvail := map[tc.CacheName]bool{}

	// TODO get responses as they come in, instead of having to wait in order
	//      (requires refactoring channel design)
	for _, resp := range responses {
		avail := <-resp.Ch
		if resp.Near {
			nearAvail[tc.CacheName(resp.Server)] = avail
		} else {
			farAvail[tc.CacheName(resp.Server)] = avail
		}
	}

	for host, avail := range nearAvail {
		if avail && farAvail[host] {
			crs.Caches[host] = IsAvailable{IsAvailable: true}
		} else {
			crs.Caches[host] = IsAvailable{IsAvailable: false}
		}
	}

	return crs, nil
}

// RefreshCRConfig refreshes the CRConfig every interval. Does not return.
func RefreshCRConfig(cfg *HandlerCfg, interval time.Duration, path string) {
	// TODO change refresh from a file path to the TO URL.
	timer := time.NewTimer(0)
	for {
		<-timer.C
		crc, err := GetCRConfig(path)
		if err != nil {
			fmt.Println("ERROR refreshing CRConfig path '" + path + "': " + err.Error())
		} else {
			fmt.Println("DEBUG refreshed CRConfig path '" + path + "'")
			cfg.SetCRConfig(crc)
		}
		timer.Reset(interval)
	}
}

func GetCRConfig(path string) (*tc.CRConfig, error) {
	crcFi, err := os.Open(path)
	if err != nil {
		return nil, errors.New("opening CRConfig path '" + path + "': " + err.Error())
	}
	defer crcFi.Close()

	crc := &tc.CRConfig{}
	if err := json.NewDecoder(crcFi).Decode(crc); err != nil {
		return nil, errors.New("decoding CRConfig path '" + path + "': " + err.Error())
	}
	return crc, nil
}

type Req struct {
	Host string
	Resp chan bool
}

func StartReqWorker(httpClient *http.Client, reqURL *url.URL, reqCh chan Req) {
	for workerReq := range reqCh {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    reqURL,
			Host:   workerReq.Host,
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			// TODO log
			workerReq.Resp <- false
			continue
		}
		resp.Body.Close() // immediately close, there should be no body, and we don't care if there is
		// TODO log unavailable codes?
		workerReq.Resp <- resp.StatusCode >= 200 && resp.StatusCode <= 299
	}
}
