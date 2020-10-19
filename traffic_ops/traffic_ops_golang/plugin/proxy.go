package plugin

/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// The proxy plugin reverse-proxies to other HTTP services, as configured.
//
// Configuration is in `cdn.conf` (like all plugins) and of the form `{"plugin_config": {"proxy":[{"path": "/my-extension-route", "uri": "https://example.net"}]}}`
//
// Users are required to be authenticated. For modifications such as removing authentication or amending the proxied request, forking this plugin is encouraged.

func init() {
	AddPlugin(10000, Funcs{load: proxyLoad, onRequest: proxyOnReq}, "proxy plugin to reverse-proxy to other HTTP services", "1.0.0")
}

type ProxyConfig []ProxyRemap

type ProxyRemap struct {
	Path string   `json:"path"`
	URI  *url.URL `json:"uri"`
}

func (r *ProxyRemap) UnmarshalJSON(b []byte) error {
	type ProxyRemapJSON struct {
		Path string `json:"path"`
		URI  string `json:"uri"`
	}
	rj := ProxyRemapJSON{}
	if err := json.Unmarshal(b, &rj); err != nil {
		return err
	}
	uri, err := url.Parse(rj.URI)
	if err != nil {
		return err
	}
	r.Path = rj.Path
	r.URI = uri
	return nil
}

func proxyLoad(b json.RawMessage) interface{} {
	cfg := ProxyConfig{}
	err := json.Unmarshal(b, &cfg)
	if err != nil {
		log.Debugln(`plugin proxy: malformed config. Config should look like: {"plugin_config": {"proxy":[{"path": "/my-extension-route", "uri": "https://example.net"}]}}`)
		return nil
	}
	log.Debugf("plugin proxy: loaded config %+v\n", cfg)
	return &cfg
}

func proxyOnReq(d OnRequestData) IsRequestHandled {
	if d.Cfg == nil {
		return RequestUnhandled
	}
	cfg, ok := d.Cfg.(*ProxyConfig)
	if !ok {
		// should never happen
		log.Errorf("plugin proxy config '%v' type '%T' expected *ProxyConfig\n", d.Cfg, d.Cfg)
		return RequestUnhandled
	}

	for _, remap := range *cfg {
		if !strings.HasPrefix(d.R.URL.Path, remap.Path) {
			continue
		}
		return proxyHandle(d.W, d.R, d, remap.URI)
	}
	return RequestUnhandled
}

func proxyHandle(w http.ResponseWriter, r *http.Request, d OnRequestData, proxyURI *url.URL) IsRequestHandled {
	_, errs := api.GetUserFromReq(w, r, d.AppCfg.Secrets[0]) // require login
	if errs.Occurred() {
		api.HandleErr(w, r, nil, errs.Code, errs.UserError, errs.SystemError)
		return RequestHandled
	}
	rp := httputil.NewSingleHostReverseProxy(proxyURI)
	rp.ServeHTTP(w, r)
	return RequestHandled
}
