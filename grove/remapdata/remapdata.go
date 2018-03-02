package remapdata

// remapdata exists as a package to avoid import cycles, for packages that need remap objects and are also included by remap.

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/apache/incubator-trafficcontrol/grove/chash"
	"github.com/apache/incubator-trafficcontrol/grove/icache"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"
)

type Reuse int

const (
	ReuseCan Reuse = iota
	ReuseCannot
	ReuseMustRevalidate
	// ReuseMustRevalidateCanStale indicates the response must be revalidated, but if the parent cannot be reached, may be served stale, per RFC7234§4.2.4
	ReuseMustRevalidateCanStale
)

// ParentSelectionType is the algorithm to use for selecting parents.
type ParentSelectionType string

const (
	ParentSelectionTypeConsistentHash = ParentSelectionType("consistent-hash")
	ParentSelectionTypeRoundRobin     = ParentSelectionType("round-robin")
	ParentSelectionTypeInvalid        = ParentSelectionType("")
)

func (t ParentSelectionType) String() string {
	switch t {
	case ParentSelectionTypeConsistentHash:
		return "consistent-hash"
	case ParentSelectionTypeRoundRobin:
		return "round-robin"
	default:
		return "invalid"
	}
}

func ParentSelectionTypeFromString(s string) ParentSelectionType {
	s = strings.ToLower(s)
	if s == "consistent-hash" {
		return ParentSelectionTypeConsistentHash
	}
	if s == "round-robin" {
		return ParentSelectionTypeRoundRobin
	}
	return ParentSelectionTypeInvalid
}

type RemapRulesStats struct {
	Allow []*net.IPNet
	Deny  []*net.IPNet
}

func (statRules RemapRulesStats) Allowed(ip net.IP) bool {
	for _, network := range statRules.Deny {
		if network.Contains(ip) {
			log.Debugf("deny contains ip\n")
			return false
		}
	}
	if len(statRules.Allow) == 0 {
		log.Debugf("Allowed len 0\n")
		return true
	}
	for _, network := range statRules.Allow {
		if network.Contains(ip) {
			log.Debugf("allow contains ip\n")
			return true
		}
	}
	return false
}

type RemapRuleBase struct {
	Name               string          `json:"name"`
	From               string          `json:"from"`
	CertificateFile    string          `json:"certificate-file"`
	CertificateKeyFile string          `json:"certificate-key-file"`
	ConnectionClose    bool            `json:"connection-close"`
	QueryString        QueryStringRule `json:"query-string"`
	// ConcurrentRuleRequests is the number of concurrent requests permitted to a remap rule, that is, to an origin. If this is 0, the global config is used.
	ConcurrentRuleRequests int  `json:"concurrent_rule_requests"`
	RetryNum               *int `json:"retry_num"`
	DSCP                   int  `json:"dscp"`
}

type RemapRule struct {
	RemapRuleBase
	Timeout         *time.Duration
	ParentSelection *ParentSelectionType
	To              []RemapRuleTo
	Allow           []*net.IPNet
	Deny            []*net.IPNet
	RetryCodes      map[int]struct{}
	ConsistentHash  chash.ATSConsistentHash
	Cache           icache.Cache
	Plugins         map[string]interface{}
}

func (r *RemapRule) Allowed(ip net.IP) bool {
	for _, network := range r.Deny {
		if network.Contains(ip) {
			log.Debugf("deny contains ip\n")
			return false
		}
	}
	if len(r.Allow) == 0 {
		log.Debugf("Allowed len 0\n")
		return true
	}
	for _, network := range r.Allow {
		if network.Contains(ip) {
			log.Debugf("allow contains ip\n")
			return true
		}
	}
	return false
}

// URI takes a request URI and maps it to the real URI to proxy-and-cache. The `failures` parameter indicates how many parents have tried and failed, indicating to skip to the nth hashed parent. Returns the URI to request, and the proxy URL (if any)
func (r RemapRule) URI(fromURI string, path string, query string, failures int) (string, *url.URL) {
	fromHash := path
	if r.QueryString.Remap && query != "" {
		fromHash += "?" + query
	}

	// fmt.Println("RemapRule.URI fromURI " + fromHash)
	to, proxyURI := r.uriGetTo(fromHash, failures)
	uri := to + fromURI[len(r.From):]
	if !r.QueryString.Remap {
		if i := strings.Index(uri, "?"); i != -1 {
			uri = uri[:i]
		}
	}
	return uri, proxyURI
}

// uriGetTo is a helper func for URI. It returns the To URL, based on the Parent Selection type. In the event of failure, it logs the error and returns the first parent. Also returns the URL's Proxy URI (if any).
func (r RemapRule) uriGetTo(fromURI string, failures int) (string, *url.URL) {
	switch *r.ParentSelection {
	case ParentSelectionTypeConsistentHash:
		return r.uriGetToConsistentHash(fromURI, failures)
	default:
		log.Errorf("RemapRule.URI: Rule '%v': Unknown Parent Selection type %v - using first URI in rule\n", r.Name, r.ParentSelection)
		return r.To[0].URL, r.To[0].ProxyURL
	}
}

// uriGetToConsistentHash is a helper func for URI, uriGetTo. It returns the To URL using Consistent Hashing. In the event of failure, it logs the error and returns the first parent. Also returns the Proxy URI (if any).
func (r RemapRule) uriGetToConsistentHash(fromURI string, failures int) (string, *url.URL) {
	// fmt.Printf("DEBUGL uriGetToConsistentHash RemapRule %+v\n", r)
	if r.ConsistentHash == nil {
		log.Errorf("RemapRule.URI: Rule '%v': Parent Selection Type ConsistentHash, but rule.ConsistentHash is nil! Using first parent\n", r.Name)
		return r.To[0].URL, r.To[0].ProxyURL
	}

	// fmt.Printf("DEBUGL uriGetToConsistentHash\n")
	iter, _, err := r.ConsistentHash.Lookup(fromURI)
	if err != nil {
		// if r.ConsistentHash.First() == nil {
		// 	fmt.Printf("DEBUGL uriGetToConsistentHash NodeMap empty!\n")
		// }
		// fmt.Printf("DEBUGL uriGetToConsistentHash fromURI '%v' err %v returning '%v'\n", fromURI, err, r.To[0].URL)
		log.Errorf("RemapRule.URI: Rule '%v': Error looking up Consistent Hash! Using first parent\n", r.Name)
		return r.To[0].URL, r.To[0].ProxyURL
	}

	for i := 0; i < failures; i++ {
		iter = iter.NextWrap()
	}

	return iter.Val().Name, iter.Val().ProxyURL
}

func (r RemapRule) CacheKey(method string, fromURI string) string {
	// TODO don't cache on `to`, since it's affected by Parent Selection
	// TODO add parent selection
	to := r.To[0].URL
	uri := to + fromURI[len(r.From):]
	if !r.QueryString.Cache {
		if i := strings.Index(uri, "?"); i != -1 {
			uri = uri[:i]
		}
	}
	key := method + ":" + uri
	return key
}

type RemapRuleToBase struct {
	URL      string   `json:"url"`
	Weight   *float64 `json:"weight"`
	RetryNum *int     `json:"retry_num"`
}

type RemapRuleTo struct {
	RemapRuleToBase
	ProxyURL   *url.URL
	Timeout    *time.Duration
	RetryCodes map[int]struct{}
}

type QueryStringRule struct {
	Remap bool `json:"remap"`
	Cache bool `json:"cache"`
}
