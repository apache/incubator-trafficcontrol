package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"

	"github.com/apache/incubator-trafficcontrol/grove/cache"
	"github.com/apache/incubator-trafficcontrol/grove/config"
	"github.com/apache/incubator-trafficcontrol/grove/diskcache"
	"github.com/apache/incubator-trafficcontrol/grove/icache"
	"github.com/apache/incubator-trafficcontrol/grove/memcache"
	"github.com/apache/incubator-trafficcontrol/grove/plugin"
	"github.com/apache/incubator-trafficcontrol/grove/remap"
	"github.com/apache/incubator-trafficcontrol/grove/remapdata"
	"github.com/apache/incubator-trafficcontrol/grove/stat"
	"github.com/apache/incubator-trafficcontrol/grove/tiercache"
	"github.com/apache/incubator-trafficcontrol/grove/web"
)

const ShutdownTimeout = 60 * time.Second

func main() {
	runtime.GOMAXPROCS(32) // DEBUG
	configFileName := flag.String("cfg", "", "The config file path")
	pprof := flag.Bool("pprof", false, "Whether to profile")
	flag.Parse()

	if *configFileName == "" {
		fmt.Printf("Error starting service: The -cfg argument is required\n")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(*configFileName)
	if err != nil {
		fmt.Printf("Error starting service: loading config: %v\n", err)
		os.Exit(1)
	}

	eventW, errW, warnW, infoW, debugW, err := log.GetLogWriters(cfg)
	if err != nil {
		fmt.Printf("Error starting service: failed to create log writers: %v\n", err)
		os.Exit(1)
	}
	log.Init(eventW, errW, warnW, infoW, debugW)

	caches, err := createCaches(cfg.CacheFiles, uint64(cfg.FileMemBytes), uint64(cfg.CacheSizeBytes))
	if err != nil {
		log.Errorln("starting service: creating caches: " + err.Error())
		os.Exit(1)
	}

	plugins := plugin.Get()
	remapper, err := remap.LoadRemapper(cfg.RemapRulesFile, plugins.LoadFuncs(), caches)
	if err != nil {
		log.Errorf("starting service: loading remap rules: %v\n", err)
		os.Exit(1)
	}

	certs, err := loadCerts(remapper.Rules())
	if err != nil {
		log.Errorf("starting service: loading certificates: %v\n", err)
		os.Exit(1)
	}
	defaultCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		log.Errorf("starting service: loading default certificate: %v\n", err)
		os.Exit(1)
	}
	certs = append(certs, defaultCert)

	httpListener, httpConns, httpConnStateCallback, err := web.InterceptListen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Errorf("creating HTTP listener %v: %v\n", cfg.Port, err)
		os.Exit(1)
	}

	httpsConns := (*web.ConnMap)(nil)
	httpsServer := (*http.Server)(nil)
	httpsListener := net.Listener(nil)
	httpsConnStateCallback := (func(net.Conn, http.ConnState))(nil)
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		if httpsListener, httpsConns, httpsConnStateCallback, err = web.InterceptListenTLS("tcp", fmt.Sprintf(":%d", cfg.HTTPSPort), certs); err != nil {
			log.Errorf("creating HTTPS listener %v: %v\n", cfg.HTTPSPort, err)
			return
		}
	}

	// TODO pass total size for all file groups?
	stats := stat.New(remapper.Rules(), caches, uint64(cfg.CacheSizeBytes), httpConns, httpsConns)

	buildHandler := func(scheme string, port string, conns *web.ConnMap, stats stat.Stats, pluginContext map[string]*interface{}) *cache.HandlerPointer {
		return cache.NewHandlerPointer(cache.NewHandler(
			remapper,
			uint64(cfg.ConcurrentRuleRequests),
			stats,
			scheme,
			port,
			conns,
			cfg.RFCCompliant,
			cfg.ConnectionClose,
			time.Duration(cfg.ReqTimeoutMS)*time.Millisecond,
			time.Duration(cfg.ReqKeepAliveMS)*time.Millisecond,
			cfg.ReqMaxIdleConns,
			time.Duration(cfg.ReqIdleConnTimeoutMS)*time.Millisecond,
			plugins,
			pluginContext,
			httpConns,
			httpsConns,
			cfg.InterfaceName,
		))
	}

	pluginContext := map[string]*interface{}{}

	httpHandler := buildHandler("http", strconv.Itoa(cfg.Port), httpConns, stats, pluginContext)
	httpsHandler := buildHandler("https", strconv.Itoa(cfg.HTTPSPort), httpsConns, stats, pluginContext)

	idleTimeout := time.Duration(cfg.ServerIdleTimeoutMS) * time.Millisecond
	readTimeout := time.Duration(cfg.ServerReadTimeoutMS) * time.Millisecond
	writeTimeout := time.Duration(cfg.ServerWriteTimeoutMS) * time.Millisecond

	plugins.OnStartup(remapper.PluginCfg(), pluginContext, plugin.StartupData{Config: cfg, Shared: remapper.PluginSharedCfg()})

	// TODO add config to not serve HTTP (only HTTPS). If port is not set?
	httpServer := startServer(httpHandler, httpListener, httpConnStateCallback, cfg.Port, idleTimeout, readTimeout, writeTimeout, "http")

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		httpsServer = startServer(httpsHandler, httpsListener, httpsConnStateCallback, cfg.HTTPSPort, idleTimeout, readTimeout, writeTimeout, "https")
	}

	reloadConfig := func() {
		log.Infof("reloading config\n")
		err := error(nil)
		oldCfg := cfg
		cfg, err = config.LoadConfig(*configFileName)
		if err != nil {
			log.Errorf("reloading config: loading config file: %v\n", err)
			return
		}
		eventW, errW, warnW, infoW, debugW, err := log.GetLogWriters(cfg)
		if err != nil {
			log.Errorf("reloading config: getting log writers '%v': %v", *configFileName, err)
		}
		log.Init(eventW, errW, warnW, infoW, debugW)

		// TODO add cache file reloading
		// The problem is, the disk db needs file locks, so there's no way to close and create new files without making all requests cache miss in the meantime.
		// Thus, the file paths must be kept, diffed, only removed paths' dbs closed, only new paths opened, and dbs for existing paths passed into the new caches object.
		if cachesChanged(oldCfg, cfg) {
			log.Warnln("reloading config: caches changed in new config! Dynamic cache reloading is not supported! Old cache files and sizes will be used, and new cache config will NOT be loaded! Restart service to apply cache changes!")
		}

		remapper, err = remap.LoadRemapper(cfg.RemapRulesFile, plugins.LoadFuncs(), caches)
		if err != nil {
			log.Errorf("starting service: loading remap rules: %v\n", err)
			os.Exit(1)
		}

		if cfg.Port != oldCfg.Port {
			if httpListener, httpConns, httpConnStateCallback, err = web.InterceptListen("tcp", fmt.Sprintf(":%d", cfg.Port)); err != nil {
				log.Errorf("reloading config: creating HTTP listener %v: %v\n", cfg.Port, err)
				return
			}
		}

		if (cfg.CertFile != oldCfg.CertFile || cfg.KeyFile != oldCfg.KeyFile) && cfg.HTTPSPort != oldCfg.HTTPSPort {
			log.Warnf("config certificate changed, but port did not. Cannot recreate listener on same port without stopping the service. Restart the service to load the new certificate.\n")
		}

		if cfg.HTTPSPort != oldCfg.HTTPSPort {
			if httpsListener, httpsConns, httpsConnStateCallback, err = web.InterceptListenTLS("tcp", fmt.Sprintf(":%d", cfg.HTTPSPort), certs); err != nil {
				log.Errorf("creating HTTPS listener %v: %v\n", cfg.HTTPSPort, err)
			}
		}

		stats = stat.New(remapper.Rules(), caches, uint64(cfg.CacheSizeBytes), httpConns, httpsConns) // TODO copy stats from old stats object?

		httpCacheHandler := cache.NewHandler(
			remapper,
			uint64(cfg.ConcurrentRuleRequests),
			stats,
			"http",
			strconv.Itoa(cfg.Port),
			httpConns,
			cfg.RFCCompliant,
			cfg.ConnectionClose,
			time.Duration(cfg.ReqTimeoutMS)*time.Millisecond,
			time.Duration(cfg.ReqKeepAliveMS)*time.Millisecond,
			cfg.ReqMaxIdleConns,
			time.Duration(cfg.ReqIdleConnTimeoutMS)*time.Millisecond,
			plugins,
			pluginContext,
			httpConns,
			httpsConns,
			cfg.InterfaceName,
		)
		httpHandler.Set(httpCacheHandler)

		httpsCacheHandler := cache.NewHandler(
			remapper,
			uint64(cfg.ConcurrentRuleRequests),
			stats,
			"https",
			strconv.Itoa(cfg.HTTPSPort),
			httpsConns,
			cfg.RFCCompliant,
			cfg.ConnectionClose,
			time.Duration(cfg.ReqTimeoutMS)*time.Millisecond,
			time.Duration(cfg.ReqKeepAliveMS)*time.Millisecond,
			cfg.ReqMaxIdleConns,
			time.Duration(cfg.ReqIdleConnTimeoutMS)*time.Millisecond,
			plugins,
			pluginContext,
			httpConns,
			httpsConns,
			cfg.InterfaceName,
		)
		httpsHandler.Set(httpsCacheHandler)

		if cfg.Port != oldCfg.Port {
			ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
			defer cancel()
			if err := httpServer.Shutdown(ctx); err != nil {
				if err == context.DeadlineExceeded {
					log.Errorf("closing http server: connections didn't close gracefully in %v, forcefully closing.\n", ShutdownTimeout)
					httpServer.Close()
				} else {
					log.Errorf("closing http server: %v\n", err)
				}

			}
			httpServer = startServer(httpHandler, httpListener, httpConnStateCallback, cfg.Port, idleTimeout, readTimeout, writeTimeout, "http")
		}

		if (httpsServer == nil || cfg.HTTPSPort != oldCfg.HTTPSPort) && cfg.CertFile != "" && cfg.KeyFile != "" {
			if httpsServer != nil {
				ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
				defer cancel()
				if err := httpsServer.Shutdown(ctx); err != nil {
					if err == context.DeadlineExceeded {
						log.Errorf("closing https server: connections didn't close gracefully in %v, forcefully closing.\n", ShutdownTimeout)
						httpServer.Close()
					} else {
						log.Errorf("closing https server: %v\n", err)
					}
				}
			}

			httpsServer = startServer(httpsHandler, httpsListener, httpsConnStateCallback, cfg.HTTPSPort, idleTimeout, readTimeout, writeTimeout, "https")
		}
	}

	if *pprof {
		profile()
	}
	signalReloader(unix.SIGHUP, reloadConfig)
}

func profile() {
	go func() {
		count := 0
		for {
			count++
			filename := fmt.Sprintf("grove%d.pprof", count)
			f, err := os.Create(filename)
			if err != nil {
				log.Errorf("creating profile: %v\n", err)
				os.Exit(1)
			}
			pprof.StartCPUProfile(f)
			time.Sleep(time.Minute)
			pprof.StopCPUProfile()
			f.Close()
		}
	}()
}

func signalReloader(sig os.Signal, f func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)
	for range c {
		f()
	}
}

// startServer starts an HTTP or HTTPS server on the given port, and returns it.
func startServer(handler http.Handler, listener net.Listener, connState func(net.Conn, http.ConnState), port int, idleTimeout time.Duration, readTimeout time.Duration, writeTimeout time.Duration, protocol string) *http.Server {
	server := &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf(":%d", port),
		ConnState:    connState,
		IdleTimeout:  idleTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	go func() {
		log.Infof("listening on %s://%d\n", protocol, port)
		if err := server.Serve(listener); err != nil {
			log.Errorf("serving %s port %v: %v\n", strings.ToUpper(protocol), port, err)
		}
	}()
	return server
}

func loadCerts(rules []remapdata.RemapRule) ([]tls.Certificate, error) {
	certs := []tls.Certificate{}
	for _, rule := range rules {
		if rule.CertificateFile == "" && rule.CertificateKeyFile == "" {
			continue
		}
		if rule.CertificateFile == "" {
			return nil, errors.New("rule " + rule.Name + " has a certificate but no key, using default certificate\n")
		}
		if rule.CertificateKeyFile == "" {
			return nil, errors.New("rule " + rule.Name + " has a key but no certificate, using default certificate\n")
		}

		cert, err := tls.LoadX509KeyPair(rule.CertificateFile, rule.CertificateKeyFile)
		if err != nil {
			return nil, errors.New("loading rule " + rule.Name + " certificate: " + err.Error() + "\n")
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// createCaches creates the caches specified in the config. The nameFiles is the map of names to groups of files, nameMemBytes is the amount of memory to use for each named group, and memCacheBytes is the amount of memory to use for the default memory cache.
func createCaches(nameFiles map[string][]config.CacheFile, nameMemBytes uint64, memCacheBytes uint64) (map[string]icache.Cache, error) {
	caches := map[string]icache.Cache{}

	memCache, err := memcache.New(memCacheBytes)
	if err != nil {
		return nil, errors.New("creating memory cache: " + err.Error())
	}
	caches[""] = memCache // default empty names to the mem cache

	for name, files := range nameFiles {
		multiDiskCache, err := diskcache.NewMulti(files)
		if err != nil {
			return nil, errors.New("creating cache '" + name + "': " + err.Error())
		}
		memCache, err := memcache.New(nameMemBytes)
		if err != nil {
			return nil, errors.New("creating memory cache for '" + name + "': " + err.Error())
		}
		caches[name] = tiercache.New(memCache, multiDiskCache)
	}

	return caches, nil
}

func cachesChanged(oldCfg, newCfg config.Config) bool {
	return oldCfg.FileMemBytes == newCfg.FileMemBytes &&
		oldCfg.CacheSizeBytes != newCfg.CacheSizeBytes &&
		!reflect.DeepEqual(oldCfg.CacheFiles, newCfg.CacheFiles)
}
