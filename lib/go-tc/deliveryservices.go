package tc

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/apache/trafficcontrol/lib/go-util"
)

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

const DefaultRoutingName = "cdn"
const DefaultMaxRequestHeaderBytes = 0
const MinRangeSliceBlockSize = 262144   // 265Kib
const MaxRangeSliceBlockSize = 33554432 // 32Mib

// GetDeliveryServiceResponse is deprecated use DeliveryServicesResponse...
type GetDeliveryServiceResponse struct {
	Response []DeliveryService `json:"response"`
}

// DeliveryServicesResponse ...
// Deprecated: use DeliveryServicesNullableResponse instead
type DeliveryServicesResponse struct {
	Response []DeliveryService `json:"response"`
	Alerts
}

// DeliveryServicesResponseV30 is the type of a response from the
// /api/3.0/deliveryservices Traffic Ops endpoint.
// TODO: Move these into the respective clients?
type DeliveryServicesResponseV30 struct {
	Response []DeliveryServiceNullableV30 `json:"response"`
	Alerts
}

// DeliveryServicesResponseV40 is the type of a response from the
// /api/4.0/deliveryservices Traffic Ops endpoint.
type DeliveryServicesResponseV40 struct {
	Response []DeliveryServiceV40 `json:"response"`
	Alerts
}

// DeliveryServicesResponseV4 is the type of a response from the
// /api/4.x/deliveryservices Traffic Ops endpoint.
// It always points to the type for the latest minor version of APIv4.
type DeliveryServicesResponseV4 = DeliveryServicesResponseV40

// DeliveryServicesNullableResponse ...
// Deprecated: Please only use the versioned structures.
type DeliveryServicesNullableResponse struct {
	Response []DeliveryServiceNullable `json:"response"`
	Alerts
}

// CreateDeliveryServiceResponse ...
// Deprecated: use CreateDeliveryServiceNullableResponse instead
type CreateDeliveryServiceResponse struct {
	Response []DeliveryService `json:"response"`
	Alerts
}

// CreateDeliveryServiceNullableResponse ...
// Deprecated: Please only use the versioned structures.
type CreateDeliveryServiceNullableResponse struct {
	Response []DeliveryServiceNullable `json:"response"`
	Alerts
}

// UpdateDeliveryServiceResponse ...
// Deprecated: use UpdateDeliveryServiceNullableResponse instead
type UpdateDeliveryServiceResponse struct {
	Response []DeliveryService `json:"response"`
	Alerts
}

// UpdateDeliveryServiceNullableResponse ...
// Deprecated: Please only use the versioned structures.
type UpdateDeliveryServiceNullableResponse struct {
	Response []DeliveryServiceNullable `json:"response"`
	Alerts
}

// DeleteDeliveryServiceResponse ...
type DeleteDeliveryServiceResponse struct {
	Alerts
}

// Deprecated: use DeliveryServiceNullable instead
type DeliveryService struct {
	DeliveryServiceV13
	MaxOriginConnections      int      `json:"maxOriginConnections" db:"max_origin_connections"`
	ConsistentHashRegex       string   `json:"consistentHashRegex"`
	ConsistentHashQueryParams []string `json:"consistentHashQueryParams"`
}

type DeliveryServiceV13 struct {
	DeliveryServiceV11
	DeepCachingType   DeepCachingType `json:"deepCachingType"`
	FQPacingRate      int             `json:"fqPacingRate,omitempty"`
	SigningAlgorithm  string          `json:"signingAlgorithm" db:"signing_algorithm"`
	Tenant            string          `json:"tenant"`
	TRRequestHeaders  string          `json:"trRequestHeaders,omitempty"`
	TRResponseHeaders string          `json:"trResponseHeaders,omitempty"`
}

// DeliveryServiceV11 contains the information relating to a delivery service
// that was around in version 1.1 of the API.
// TODO move contents to DeliveryServiceV12, fix references, and remove
type DeliveryServiceV11 struct {
	Active                   bool                   `json:"active"`
	AnonymousBlockingEnabled bool                   `json:"anonymousBlockingEnabled"`
	CacheURL                 string                 `json:"cacheurl"`
	CCRDNSTTL                int                    `json:"ccrDnsTtl"`
	CDNID                    int                    `json:"cdnId"`
	CDNName                  string                 `json:"cdnName"`
	CheckPath                string                 `json:"checkPath"`
	DeepCachingType          DeepCachingType        `json:"deepCachingType"`
	DisplayName              string                 `json:"displayName"`
	DNSBypassCname           string                 `json:"dnsBypassCname"`
	DNSBypassIP              string                 `json:"dnsBypassIp"`
	DNSBypassIP6             string                 `json:"dnsBypassIp6"`
	DNSBypassTTL             int                    `json:"dnsBypassTtl"`
	DSCP                     int                    `json:"dscp"`
	EdgeHeaderRewrite        string                 `json:"edgeHeaderRewrite"`
	ExampleURLs              []string               `json:"exampleURLs"`
	GeoLimit                 int                    `json:"geoLimit"`
	GeoProvider              int                    `json:"geoProvider"`
	GlobalMaxMBPS            int                    `json:"globalMaxMbps"`
	GlobalMaxTPS             int                    `json:"globalMaxTps"`
	HTTPBypassFQDN           string                 `json:"httpBypassFqdn"`
	ID                       int                    `json:"id"`
	InfoURL                  string                 `json:"infoUrl"`
	InitialDispersion        float32                `json:"initialDispersion"`
	IPV6RoutingEnabled       bool                   `json:"ipv6RoutingEnabled"`
	LastUpdated              *TimeNoMod             `json:"lastUpdated" db:"last_updated"`
	LogsEnabled              bool                   `json:"logsEnabled"`
	LongDesc                 string                 `json:"longDesc"`
	LongDesc1                string                 `json:"longDesc1"`
	LongDesc2                string                 `json:"longDesc2"`
	MatchList                []DeliveryServiceMatch `json:"matchList,omitempty"`
	MaxDNSAnswers            int                    `json:"maxDnsAnswers"`
	MidHeaderRewrite         string                 `json:"midHeaderRewrite"`
	MissLat                  float64                `json:"missLat"`
	MissLong                 float64                `json:"missLong"`
	MultiSiteOrigin          bool                   `json:"multiSiteOrigin"`
	OrgServerFQDN            string                 `json:"orgServerFqdn"`
	ProfileDesc              string                 `json:"profileDescription"`
	ProfileID                int                    `json:"profileId,omitempty"`
	ProfileName              string                 `json:"profileName"`
	Protocol                 int                    `json:"protocol"`
	QStringIgnore            int                    `json:"qstringIgnore"`
	RangeRequestHandling     int                    `json:"rangeRequestHandling"`
	RegexRemap               string                 `json:"regexRemap"`
	RegionalGeoBlocking      bool                   `json:"regionalGeoBlocking"`
	RemapText                string                 `json:"remapText"`
	RoutingName              string                 `json:"routingName"`
	Signed                   bool                   `json:"signed"`
	TypeID                   int                    `json:"typeId"`
	Type                     DSType                 `json:"type"`
	TRResponseHeaders        string                 `json:"trResponseHeaders"`
	TenantID                 int                    `json:"tenantId"`
	XMLID                    string                 `json:"xmlId"`
}

type DeliveryServiceV31 struct {
	DeliveryServiceV30
	DeliveryServiceFieldsV31
}

// DeliveryServiceFieldsV31 contains additions to delivery services in api v3.1
type DeliveryServiceFieldsV31 struct {
	MaxRequestHeaderBytes *int `json:"maxRequestHeaderBytes" db:"max_request_header_bytes"`
}

// DeliveryServiceActiveState is an "enumerated" type which encodes the valid
// values of a Delivery Service's 'Active' property (v3.0+).
type DeliveryServiceActiveState string

// A DeliveryServiceActiveState describes the availability of Delivery Service
// content from the perspective of caching servers and Traffic Routers.
const (
	// Traffic Router routes clients for this Delivery Service and cache
	// servers are configured to serve its content.
	DS_ACTIVE = DeliveryServiceActiveState("ACTIVE")
	// Traffic Router does not route for this Delivery Service and cache
	// servers cannot serve its content.
	DS_INACTIVE = DeliveryServiceActiveState("INACTIVE")
	// Traffic Router does not route for this Delivery Service, but cache
	// servers are configured to serve its content.
	DS_PRIMED = DeliveryServiceActiveState("PRIMED")
)

// DeliveryServiceV40 is a Delivery Service as it appears in version 4.0 of the
// Traffic Ops API.
type DeliveryServiceV40 struct {
	// Active dictates whether the Delivery Service is routed by Traffic Router,
	// as well as whether or not cache servers have the correct configuration to
	// serve its content.
	Active DeliveryServiceActiveState `json:"active" db:"active"`
	// AnonymousBlockingEnabled sets whether or not anonymized IP addresses
	// (e.g. Tor exit nodes) should be restricted from accessing the Delivery
	// Service's content.
	AnonymousBlockingEnabled bool `json:"anonymousBlockingEnabled" db:"anonymous_blocking_enabled"`
	// CCRDNSTTL sets the Time-to-Live - in seconds - for DNS responses for this
	// Delivery Service from Traffic Router.
	CCRDNSTTL *int `json:"ccrDnsTtl" db:"ccr_dns_ttl"`
	// CDNID is the integral, unique identifier for the CDN to which the
	// Delivery Service belongs.
	CDNID int `json:"cdnId" db:"cdn_id"`
	// CDNName is the name of the CDN to which the Delivery Service belongs.
	CDNName *string `json:"cdnName"`
	// CheckPath is a path which may be requested of the Delivery Service's
	// origin to ensure it's working properly.
	CheckPath *string `json:"checkPath" db:"check_path"`
	// ConsistentHashQueryParams is a list of al of the query string parameters
	// which ought to be considered by Traffic Router in client request URIs for
	// HTTP-routed Delivery Services in the hashing process.
	ConsistentHashQueryParams []string `json:"consistentHashQueryParams"`
	// ConsistentHashRegex is used by Traffic Router to extract meaningful parts
	// of a client's request URI for HTTP-routed Delivery Services before
	// hashing the request to find a cache server to which to direct the client.
	ConsistentHashRegex *string `json:"consistentHashRegex"`
	// DeepCachingType may only legally point to 'ALWAYS' or 'NEVER', which
	// define whether "deep caching" may or may not be used for this Delivery
	// Service, respectively.
	DeepCachingType DeepCachingType `json:"deepCachingType" db:"deep_caching_type"`
	// DisplayName is a human-friendly name that might be used in some UIs
	// somewhere.
	DisplayName string `json:"displayName" db:"display_name"`
	// DNSBypassCNAME is a fully qualified domain name to be used in a CNAME
	// record presented to clients in bypass scenarios.
	DNSBypassCNAME *string `json:"dnsBypassCname" db:"dns_bypass_cname"`
	// DNSBypassIP is an IPv4 address to be used in an A record presented to
	// clients in bypass scenarios.
	DNSBypassIP *string `json:"dnsBypassIp" db:"dns_bypass_ip"`
	// DNSBypassIP6 is an IPv6 address to be used in an AAAA record presented to
	// clients in bypass scenarios.
	DNSBypassIP6 *string `json:"dnsBypassIp6" db:"dns_bypass_ip6"`
	// DNSBypassTTL sets the Time-to-Live - in seconds - of DNS responses from
	// the Traffic Router that contain records for bypass destinations.
	DNSBypassTTL *int `json:"dnsBypassTtl" db:"dns_bypass_ttl"`
	// DSCP sets the Differentiated Services Code Point for IP packets
	// transferred between clients, origins, and cache servers when obtaining
	// and serving content for this Delivery Service.
	// See Also: https://en.wikipedia.org/wiki/Differentiated_services
	DSCP int `json:"dscp" db:"dscp"`
	// EcsEnabled describes whether or not the Traffic Router's EDNS0 Client
	// Subnet extensions should be enabled when serving DNS responses for this
	// Delivery Service. Even if this is true, the Traffic Router may still
	// have the extensions unilaterally disabled in its own configuration.
	EcsEnabled bool `json:"ecsEnabled" db:"ecs_enabled"`
	// EdgeHeaderRewrite is a "header rewrite rule" used by ATS at the Edge-tier
	// of caching. This has no effect on Delivery Services that don't use a
	// Topology.
	EdgeHeaderRewrite *string `json:"edgeHeaderRewrite" db:"edge_header_rewrite"`
	// ExampleURLs is a list of all of the URLs from which content may be
	// requested from the Delivery Service.
	ExampleURLs []string `json:"exampleURLs"`
	// FirstHeaderRewrite is a "header rewrite rule" used by ATS at the first
	// caching layer encountered in the Delivery Service's Topology, or nil if
	// there is no such rule. This has no effect on Delivery Services that don't
	// employ Topologies.
	FirstHeaderRewrite *string `json:"firstHeaderRewrite" db:"first_header_rewrite"`
	// FQPacingRate sets the maximum bytes per second a cache server will deliver
	// on any single TCP connection for this Delivery Service. This may never
	// legally point to a value less than zero.
	FQPacingRate *int `json:"fqPacingRate" db:"fq_pacing_rate"`
	// GeoLimit defines whether or not access to a Delivery Service's content
	// should be limited based on the requesting client's geographic location.
	// Despite that this is a pointer to an arbitrary integer, the only valid
	// values are 0 (which indicates that content should not be limited
	// geographically), 1 (which indicates that content should only be served to
	// clients whose IP addresses can be found within a Coverage Zone File), and
	// 2 (which indicates that content should be served to clients whose IP
	// addresses can be found within a Coverage Zone File OR are allowed access
	// according to the "array" in GeoLimitCountries).
	GeoLimit int `json:"geoLimit" db:"geo_limit"`
	// GeoLimitCountries is an "array" of "country codes" that itemizes the
	// countries within which the Delivery Service's content ought to be made
	// available. This has no effect if GeoLimit is not a pointer to exactly the
	// value 2.
	GeoLimitCountries *string `json:"geoLimitCountries" db:"geo_limit_countries"`
	// GeoLimitRedirectURL is a URL to which clients will be redirected if their
	// access to the Delivery Service's content is blocked by GeoLimit rules.
	GeoLimitRedirectURL *string `json:"geoLimitRedirectURL" db:"geolimit_redirect_url"`
	// GeoProvider names the type of database to be used for providing IP
	// address-to-geographic-location mapping for this Delivery Service. The
	// only valid values to which it may point are 0 (which indicates the use of
	// a MaxMind GeoIP2 database) and 1 (which indicates the use of a Neustar
	// GeoPoint IP address database).
	GeoProvider int `json:"geoProvider" db:"geo_provider"`
	// GlobalMaxMBPS defines a maximum number of MegaBytes Per Second which may
	// be served for the Delivery Service before redirecting clients to bypass
	// locations.
	GlobalMaxMBPS *int `json:"globalMaxMbps" db:"global_max_mbps"`
	// GlobalMaxTPS defines a maximum number of Transactions Per Second which
	// may be served for the Delivery Service before redirecting clients to
	// bypass locations.
	GlobalMaxTPS *int `json:"globalMaxTps" db:"global_max_tps"`
	// HTTPBypassFQDN is a network location to which clients will be redirected
	// in bypass scenarios using HTTP "Location" headers and appropriate
	// redirection response codes.
	HTTPBypassFQDN *string `json:"httpBypassFqdn" db:"http_bypass_fqdn"`
	// ID is an integral, unique identifier for the Delivery Service.
	ID *int `json:"id" db:"id"`
	// InfoURL is a URL to which operators or clients may be directed to obtain
	// further information about a Delivery Service.
	InfoURL *string `json:"infoUrl" db:"info_url"`
	// InitialDispersion sets the number of cache servers within the first
	// caching layer ("Edge-tier" in a non-Topology context) across which
	// content will be dispersed per Cache Group.
	InitialDispersion *int `json:"initialDispersion" db:"initial_dispersion"`
	// InnerHeaderRewrite is a "header rewrite rule" used by ATS at all caching
	// layers encountered in the Delivery Service's Topology except the first
	// and last, or nil if there is no such rule. This has no effect on Delivery
	// Services that don't employ Topologies.
	InnerHeaderRewrite *string `json:"innerHeaderRewrite" db:"inner_header_rewrite"`
	// IPV6RoutingEnabled controls whether or not routing over IPv6 should be
	// done for this Delivery Service.
	IPV6RoutingEnabled bool `json:"ipv6RoutingEnabled" db:"ipv6_routing_enabled"`
	// LastHeaderRewrite is a "header rewrite rule" used by ATS at the first
	// caching layer encountered in the Delivery Service's Topology, or nil if
	// there is no such rule. This has no effect on Delivery Services that don't
	// employ Topologies.
	LastHeaderRewrite *string `json:"lastHeaderRewrite" db:"last_header_rewrite"`
	// LastUpdated is the time and date at which the Delivery Service was last
	// updated.
	LastUpdated time.Time `json:"lastUpdated" db:"last_updated"`
	// LogsEnabled controls nothing. It is kept only for legacy compatibility.
	LogsEnabled bool `json:"logsEnabled" db:"logs_enabled"`
	// LongDesc is a description of the Delivery Service, having arbitrary
	// length.
	LongDesc *string `json:"longDesc" db:"long_desc"`
	// LongDesc1 is a description of the Delivery Service, having arbitrary
	// length.
	LongDesc1 *string `json:"longDesc1" db:"long_desc_1"`
	// LongDesc2 is a description of the Delivery Service, having arbitrary
	// length.
	LongDesc2 *string `json:"longDesc2" db:"long_desc_2"`
	// MatchList is a list of Regular Expressions used for routing the Delivery
	// Service. Order matters, and the array is not allowed to be sparse.
	MatchList []DeliveryServiceMatch `json:"matchList"`
	// MaxDNSAnswers sets the maximum number of records which should be returned
	// by Traffic Router in DNS responses to requests for resolving names for
	// this Delivery Service.
	MaxDNSAnswers *int `json:"maxDnsAnswers" db:"max_dns_answers"`
	// MaxOriginConnections defines the total maximum  number of connections
	// that the highest caching layer ("Mid-tier" in a non-Topology context) is
	// allowed to have concurrently open to the Delivery Service's Origin. This
	// may never legally point to a value less than 0.
	MaxOriginConnections *int `json:"maxOriginConnections" db:"max_origin_connections"`
	// MaxRequestHeaderBytes is the maximum size (in bytes) of the request
	// header that is allowed for this Delivery Service.
	MaxRequestHeaderBytes *int `json:"maxRequestHeaderBytes" db:"max_request_header_bytes"`
	// MidHeaderRewrite is a "header rewrite rule" used by ATS at the Mid-tier
	// of caching. This has no effect on Delivery Services that don't use a
	// Topology.
	MidHeaderRewrite *string `json:"midHeaderRewrite" db:"mid_header_rewrite"`
	// MissLat is a latitude to default to for clients of this Delivery Service
	// when geolocation attempts fail.
	MissLat *float64 `json:"missLat" db:"miss_lat"`
	// MissLong is a longitude to default to for clients of this Delivery
	// Service when geolocation attempts fail.
	MissLong *float64 `json:"missLong" db:"miss_long"`
	// MultiSiteOrigin determines whether or not the Delivery Service makes use
	// of "Multi-Site Origin".
	MultiSiteOrigin *bool `json:"multiSiteOrigin" db:"multi_site_origin"`
	// OriginShield is a field that does nothing. It is kept only for legacy
	// compatibility reasons.
	OriginShield *string `json:"originShield" db:"origin_shield"`
	// OrgServerFQDN is the URL - NOT Fully Qualified Domain Name - of the
	// origin of the Delivery Service's content.
	OrgServerFQDN *string `json:"orgServerFqdn" db:"org_server_fqdn"`
	// ProfileDesc is the Description of the Profile used by the Delivery
	// Service, if any.
	ProfileDesc *string `json:"profileDescription"`
	// ProfileID is the integral, unique identifier of the Profile used by the
	// Delivery Service, if any.
	ProfileID *int `json:"profileId" db:"profile"`
	// ProfileName is the Name of the Profile used by the Delivery Service, if
	// any.
	ProfileName *string `json:"profileName"`
	// Protocol defines the protocols by which caching servers may communicate
	// with clients. The valid values to which it may point are 0 (which implies
	// that only HTTP may be used), 1 (which implies that only HTTPS may be
	// used), 2 (which implies that either HTTP or HTTPS may be used), and 3
	// (which implies that clients using HTTP must be redirected to use HTTPS,
	// while communications over HTTPS may proceed as normal).
	Protocol *int `json:"protocol" db:"protocol"`
	// QStringIgnore sets how query strings in HTTP requests to cache servers
	// from clients are treated. The only valid values to which it may point are
	// 0 (which implies that all caching layers will pass query strings in
	// upstream requests and use them in the cache key), 1 (which implies that
	// all caching layers will pass query strings in upstream requests, but not
	// use them in cache keys), and 2 (which implies that the first encountered
	// caching layer - "Edge-tier" in a non-Topology context - will strip query
	// strings, effectively preventing them from being passed in upstream
	// requests, and not use them in the cache key).
	QStringIgnore *int `json:"qstringIgnore" db:"qstring_ignore"`
	// RangeRequestHandling defines how HTTP GET requests with a Range header
	// will be handled by cache servers serving the Delivery Service's content.
	// The only valid values to which it may point are 0 (which implies that
	// Range requests will not be cached at all), 1 (which implies that the
	// background_fetch plugin will be used to service the range request while
	// caching the whole object), 2 (which implies that the cache_range_requests
	// plugin will be used to cache ranges as unique objects), and 3 (which
	// implies that the slice plugin will be used to slice range based requests
	// into deterministic chunks.)
	RangeRequestHandling *int `json:"rangeRequestHandling" db:"range_request_handling"`
	// RangeSliceBlockSize defines the size of range request blocks - or
	// "slices" - used by the "slice" plugin. This has no effect if
	// RangeRequestHandling does not point to exactly 3. This may never legally
	// point to a value less than zero.
	RangeSliceBlockSize *int `json:"rangeSliceBlockSize" db:"range_slice_block_size"`
	// Regex Remap is a raw line to be inserted into "regex_remap.config" on the
	// cache server. Care is necessitated in its use, because the input is in no
	// way restricted, validated, or limited in scope to the Delivery Service.
	RegexRemap *string `json:"regexRemap" db:"regex_remap"`
	// RegionalGeoBlocking defines whether or not whatever Regional Geo Blocking
	// rules are configured on the Traffic Router serving content for this
	// Delivery Service will have an effect on the traffic of this Delivery
	// Service.
	RegionalGeoBlocking bool `json:"regionalGeoBlocking" db:"regional_geo_blocking"`
	// RemapText is raw text to insert in "remap.config" on the cache servers
	// serving content for this Delivery Service. Care is necessitated in its
	// use, because the input is in no way restricted, validated, or limited in
	// scope to the Delivery Service.
	RemapText *string `json:"remapText" db:"remap_text"`
	// RoutingName defines the lowest-level DNS label used by the Delivery
	// Service, e.g. if trafficcontrol.apache.org were a Delivery Service, it
	// would have a RoutingName of "trafficcontrol".
	RoutingName string `json:"routingName" db:"routing_name"`
	// ServiceCategory defines a category to which a Delivery Service may
	// belong, which will cause HTTP Responses containing content for the
	// Delivery Service to have the "X-CDN-SVC" header with a value that is the
	// XMLID of the Delivery Service.
	ServiceCategory *string `json:"serviceCategory" db:"service_category"`
	// Signed is a legacy field. It is allowed to be `true` if and only if
	// SigningAlgorithm is not nil.
	Signed bool `json:"signed"`
	// SigningAlgorithm is the name of the algorithm used to sign CDN URIs for
	// this Delivery Service's content, or nil if no URI signing is done for the
	// Delivery Service. This may only point to the values "url_sig" or
	// "uri_signing".
	SigningAlgorithm *string `json:"signingAlgorithm" db:"signing_algorithm"`
	// SSLKeyVersion incremented whenever Traffic Portal generates new SSL keys
	// for the Delivery Service, effectively making it a "generational" marker.
	SSLKeyVersion *int `json:"sslKeyVersion" db:"ssl_key_version"`
	// Tenant is the Tenant to which the Delivery Service belongs.
	Tenant *string `json:"tenant"`
	// TenantID is the integral, unique identifier for the Tenant to which the
	// Delivery Service belongs.
	TenantID int `json:"tenantId" db:"tenant_id"`
	// Topology is the name of the Topology used by the Delivery Service, or nil
	// if no Topology is used.
	Topology *string `json:"topology" db:"topology"`
	// TRResponseHeaders is a set of headers (separated by CRLF pairs as per the
	// HTTP spec) and their values (separated by a colon as per the HTTP spec)
	// which will be sent by Traffic Router in HTTP responses to client requests
	// for this Delivery Service's content. This has no effect on DNS-routed or
	// un-routed Delivery Service Types.
	TRResponseHeaders *string `json:"trResponseHeaders"`
	// TRRequestHeaders is an "array" of HTTP headers which should be logged
	// from client HTTP requests for this Delivery Service's content by Traffic
	// Router, separated by newlines. This has no effect on DNS-routed or
	// un-routed Delivery Service Types.
	TRRequestHeaders *string `json:"trRequestHeaders"`
	// Type describes how content is routed and cached for this Delivery Service
	// as well as what other properties have any meaning.
	Type *DSType `json:"type"`
	// TypeID is an integral, unique identifier for the Tenant to which the
	// Delivery Service belongs.
	TypeID int `json:"typeId" db:"type"`
	// XMLID is a unique identifier that is also the second lowest-level DNS
	// label used by the Delivery Service. For example, if a Delivery Service's
	// content may be requested from video.demo1.mycdn.ciab.test, it may be
	// inferred that the Delivery Service's XMLID is demo1.
	XMLID string `json:"xmlId" db:"xml_id"`
}

// DeliveryServiceV4 is a Delivery Service as it appears in version 4 of the
// Traffic Ops API - it always points to the highest minor version in APIv4.
type DeliveryServiceV4 = DeliveryServiceV40

type DeliveryServiceV30 struct {
	DeliveryServiceNullableV15
	DeliveryServiceFieldsV30
}

// DeliveryServiceFieldsV30 contains additions to delivery services in api v3.0
type DeliveryServiceFieldsV30 struct {
	Topology           *string `json:"topology" db:"topology"`
	FirstHeaderRewrite *string `json:"firstHeaderRewrite" db:"first_header_rewrite"`
	InnerHeaderRewrite *string `json:"innerHeaderRewrite" db:"inner_header_rewrite"`
	LastHeaderRewrite  *string `json:"lastHeaderRewrite" db:"last_header_rewrite"`
	ServiceCategory    *string `json:"serviceCategory" db:"service_category"`
}

// DeliveryServiceNullableV30 is the aliased structure that we should be using for all api 3.x delivery structure operations
// This type should always alias the latest 3.x minor version struct. For ex, if you wanted to create a DeliveryServiceV32 struct, you would do the following:
// type DeliveryServiceNullableV30 DeliveryServiceV32
// DeliveryServiceV32 = DeliveryServiceV31 + the new fields
type DeliveryServiceNullableV30 DeliveryServiceV31

// Deprecated: Use versioned structures only from now on.
type DeliveryServiceNullable DeliveryServiceNullableV15
type DeliveryServiceNullableV15 struct {
	DeliveryServiceNullableV14
	DeliveryServiceFieldsV15
}

// DeliveryServiceFieldsV15 contains additions to delivery services in api v1.5
type DeliveryServiceFieldsV15 struct {
	EcsEnabled          bool `json:"ecsEnabled" db:"ecs_enabled"`
	RangeSliceBlockSize *int `json:"rangeSliceBlockSize" db:"range_slice_block_size"`
}

type DeliveryServiceNullableV14 struct {
	DeliveryServiceNullableV13
	DeliveryServiceFieldsV14
}

// DeliveryServiceFieldsV14 contains additions to delivery services in api v1.4
type DeliveryServiceFieldsV14 struct {
	ConsistentHashRegex       *string  `json:"consistentHashRegex"`
	ConsistentHashQueryParams []string `json:"consistentHashQueryParams"`
	MaxOriginConnections      *int     `json:"maxOriginConnections" db:"max_origin_connections"`
}

type DeliveryServiceNullableV13 struct {
	DeliveryServiceNullableV12
	DeliveryServiceFieldsV13
}

// DeliveryServiceFieldsV13 contains additions to delivery services in api v1.3
type DeliveryServiceFieldsV13 struct {
	DeepCachingType   *DeepCachingType `json:"deepCachingType" db:"deep_caching_type"`
	FQPacingRate      *int             `json:"fqPacingRate" db:"fq_pacing_rate"`
	SigningAlgorithm  *string          `json:"signingAlgorithm" db:"signing_algorithm"`
	Tenant            *string          `json:"tenant"`
	TRResponseHeaders *string          `json:"trResponseHeaders"`
	TRRequestHeaders  *string          `json:"trRequestHeaders"`
}

type DeliveryServiceNullableV12 struct {
	DeliveryServiceNullableV11
}

// DeliveryServiceNullableV11 is a version of the deliveryservice that allows
// for all fields to be null.
// TODO move contents to DeliveryServiceNullableV12, fix references, and remove
type DeliveryServiceNullableV11 struct {
	DeliveryServiceNullableFieldsV11
	DeliveryServiceRemovedFieldsV11
}

type DeliveryServiceNullableFieldsV11 struct {
	Active                   *bool                   `json:"active" db:"active"`
	AnonymousBlockingEnabled *bool                   `json:"anonymousBlockingEnabled" db:"anonymous_blocking_enabled"`
	CCRDNSTTL                *int                    `json:"ccrDnsTtl" db:"ccr_dns_ttl"`
	CDNID                    *int                    `json:"cdnId" db:"cdn_id"`
	CDNName                  *string                 `json:"cdnName"`
	CheckPath                *string                 `json:"checkPath" db:"check_path"`
	DisplayName              *string                 `json:"displayName" db:"display_name"`
	DNSBypassCNAME           *string                 `json:"dnsBypassCname" db:"dns_bypass_cname"`
	DNSBypassIP              *string                 `json:"dnsBypassIp" db:"dns_bypass_ip"`
	DNSBypassIP6             *string                 `json:"dnsBypassIp6" db:"dns_bypass_ip6"`
	DNSBypassTTL             *int                    `json:"dnsBypassTtl" db:"dns_bypass_ttl"`
	DSCP                     *int                    `json:"dscp" db:"dscp"`
	EdgeHeaderRewrite        *string                 `json:"edgeHeaderRewrite" db:"edge_header_rewrite"`
	GeoLimit                 *int                    `json:"geoLimit" db:"geo_limit"`
	GeoLimitCountries        *string                 `json:"geoLimitCountries" db:"geo_limit_countries"`
	GeoLimitRedirectURL      *string                 `json:"geoLimitRedirectURL" db:"geolimit_redirect_url"`
	GeoProvider              *int                    `json:"geoProvider" db:"geo_provider"`
	GlobalMaxMBPS            *int                    `json:"globalMaxMbps" db:"global_max_mbps"`
	GlobalMaxTPS             *int                    `json:"globalMaxTps" db:"global_max_tps"`
	HTTPBypassFQDN           *string                 `json:"httpBypassFqdn" db:"http_bypass_fqdn"`
	ID                       *int                    `json:"id" db:"id"`
	InfoURL                  *string                 `json:"infoUrl" db:"info_url"`
	InitialDispersion        *int                    `json:"initialDispersion" db:"initial_dispersion"`
	IPV6RoutingEnabled       *bool                   `json:"ipv6RoutingEnabled" db:"ipv6_routing_enabled"`
	LastUpdated              *TimeNoMod              `json:"lastUpdated" db:"last_updated"`
	LogsEnabled              *bool                   `json:"logsEnabled" db:"logs_enabled"`
	LongDesc                 *string                 `json:"longDesc" db:"long_desc"`
	LongDesc1                *string                 `json:"longDesc1" db:"long_desc_1"`
	LongDesc2                *string                 `json:"longDesc2" db:"long_desc_2"`
	MatchList                *[]DeliveryServiceMatch `json:"matchList"`
	MaxDNSAnswers            *int                    `json:"maxDnsAnswers" db:"max_dns_answers"`
	MidHeaderRewrite         *string                 `json:"midHeaderRewrite" db:"mid_header_rewrite"`
	MissLat                  *float64                `json:"missLat" db:"miss_lat"`
	MissLong                 *float64                `json:"missLong" db:"miss_long"`
	MultiSiteOrigin          *bool                   `json:"multiSiteOrigin" db:"multi_site_origin"`
	OriginShield             *string                 `json:"originShield" db:"origin_shield"`
	OrgServerFQDN            *string                 `json:"orgServerFqdn" db:"org_server_fqdn"`
	ProfileDesc              *string                 `json:"profileDescription"`
	ProfileID                *int                    `json:"profileId" db:"profile"`
	ProfileName              *string                 `json:"profileName"`
	Protocol                 *int                    `json:"protocol" db:"protocol"`
	QStringIgnore            *int                    `json:"qstringIgnore" db:"qstring_ignore"`
	RangeRequestHandling     *int                    `json:"rangeRequestHandling" db:"range_request_handling"`
	RegexRemap               *string                 `json:"regexRemap" db:"regex_remap"`
	RegionalGeoBlocking      *bool                   `json:"regionalGeoBlocking" db:"regional_geo_blocking"`
	RemapText                *string                 `json:"remapText" db:"remap_text"`
	RoutingName              *string                 `json:"routingName" db:"routing_name"`
	Signed                   bool                    `json:"signed"`
	SSLKeyVersion            *int                    `json:"sslKeyVersion" db:"ssl_key_version"`
	TenantID                 *int                    `json:"tenantId" db:"tenant_id"`
	Type                     *DSType                 `json:"type"`
	TypeID                   *int                    `json:"typeId" db:"type"`
	XMLID                    *string                 `json:"xmlId" db:"xml_id"`
	ExampleURLs              []string                `json:"exampleURLs"`
}

// DeliveryServiceRemovedFieldsV11 contains additions to delivery services in api v1.1 that were later removed
// Deprecated: used for backwards compatibility  with ATC <v5.1
type DeliveryServiceRemovedFieldsV11 struct {
	CacheURL *string `json:"cacheurl" db:"cacheurl"`
}

// DowngradeToV3 converts the 4.x DS to a 3.x DS.
func (ds *DeliveryServiceV4) DowngradeToV3() DeliveryServiceNullableV30 {
	var ret DeliveryServiceNullableV30
	ret.Active = new(bool)
	if ds.Active == DS_ACTIVE {
		*ret.Active = true
	} else {
		*ret.Active = false
	}
	ret.AnonymousBlockingEnabled = new(bool)
	*ret.AnonymousBlockingEnabled = ds.AnonymousBlockingEnabled
	ret.CCRDNSTTL = ds.CCRDNSTTL
	ret.CDNID = new(int)
	*ret.CDNID = ds.CDNID
	ret.CDNName = ds.CDNName
	ret.CheckPath = ds.CheckPath
	ret.ConsistentHashRegex = ds.ConsistentHashRegex
	ret.ConsistentHashQueryParams = make([]string, len(ds.ConsistentHashQueryParams))
	copy(ret.ConsistentHashQueryParams, ds.ConsistentHashQueryParams)
	ret.DeepCachingType = new(DeepCachingType)
	*ret.DeepCachingType = ds.DeepCachingType
	ret.DisplayName = new(string)
	*ret.DisplayName = ds.DisplayName
	ret.DNSBypassCNAME = ds.DNSBypassCNAME
	ret.DNSBypassIP = ds.DNSBypassIP
	ret.DNSBypassIP6 = ds.DNSBypassIP6
	ret.DNSBypassTTL = ds.DNSBypassTTL
	ret.DSCP = new(int)
	*ret.DSCP = ds.DSCP
	ret.EcsEnabled = ds.EcsEnabled
	ret.EdgeHeaderRewrite = ds.EdgeHeaderRewrite
	ret.ExampleURLs = make([]string, len(ds.ExampleURLs))
	copy(ret.ExampleURLs, ds.ExampleURLs)
	ret.FirstHeaderRewrite = ds.FirstHeaderRewrite
	ret.FQPacingRate = ds.FQPacingRate
	ret.GeoLimit = new(int)
	*ret.GeoLimit = ds.GeoLimit
	ret.GeoLimitCountries = ds.GeoLimitCountries
	ret.GeoLimitRedirectURL = ds.GeoLimitRedirectURL
	ret.GeoProvider = new(int)
	*ret.GeoProvider = ds.GeoProvider
	ret.GlobalMaxMBPS = ds.GlobalMaxMBPS
	ret.GlobalMaxTPS = ds.GlobalMaxTPS
	ret.HTTPBypassFQDN = ds.HTTPBypassFQDN
	ret.ID = ds.ID
	ret.InfoURL = ds.InfoURL
	ret.InitialDispersion = ds.InitialDispersion
	ret.InnerHeaderRewrite = ds.InnerHeaderRewrite
	ret.IPV6RoutingEnabled = new(bool)
	*ret.IPV6RoutingEnabled = ds.IPV6RoutingEnabled
	ret.LastHeaderRewrite = ds.LastHeaderRewrite
	ret.LastUpdated = TimeNoModFromTime(ds.LastUpdated)
	ret.LogsEnabled = new(bool)
	*ret.LogsEnabled = ds.LogsEnabled
	ret.LongDesc = ds.LongDesc
	ret.LongDesc1 = ds.LongDesc1
	ret.LongDesc2 = ds.LongDesc2
	if ds.MatchList != nil {
		ret.MatchList = new([]DeliveryServiceMatch)
		*ret.MatchList = make([]DeliveryServiceMatch, len(ds.MatchList))
		copy(*ret.MatchList, ds.MatchList)
	} else {
		ret.MatchList = nil
	}
	ret.MaxDNSAnswers = ds.MaxDNSAnswers
	ret.MaxOriginConnections = ds.MaxOriginConnections
	ret.MaxRequestHeaderBytes = ds.MaxRequestHeaderBytes
	ret.MidHeaderRewrite = ds.MidHeaderRewrite
	ret.MissLat = ds.MissLat
	ret.MissLong = ds.MissLong
	ret.MultiSiteOrigin = ds.MultiSiteOrigin
	ret.OriginShield = ds.OriginShield
	ret.OrgServerFQDN = ds.OrgServerFQDN
	ret.ProfileDesc = ds.ProfileDesc
	ret.ProfileID = ds.ProfileID
	ret.ProfileName = ds.ProfileName
	ret.Protocol = ds.Protocol
	ret.QStringIgnore = ds.QStringIgnore
	ret.RangeRequestHandling = ds.RangeRequestHandling
	ret.RangeSliceBlockSize = ds.RangeSliceBlockSize
	ret.RegexRemap = ds.RegexRemap
	ret.RegionalGeoBlocking = new(bool)
	*ret.RegionalGeoBlocking = ds.RegionalGeoBlocking
	ret.RemapText = ds.RemapText
	ret.RoutingName = new(string)
	*ret.RoutingName = ds.RoutingName
	ret.ServiceCategory = ds.ServiceCategory
	ret.Signed = ds.Signed
	ret.SigningAlgorithm = ds.SigningAlgorithm
	ret.SSLKeyVersion = ds.SSLKeyVersion
	ret.Tenant = ds.Tenant
	ret.TenantID = new(int)
	*ret.TenantID = ds.TenantID
	ret.Topology = ds.Topology
	ret.TRResponseHeaders = ds.TRResponseHeaders
	ret.TRRequestHeaders = ds.TRRequestHeaders
	ret.Type = ds.Type
	ret.TypeID = new(int)
	*ret.TypeID = ds.TypeID
	ret.XMLID = new(string)
	*ret.XMLID = ds.XMLID

	return ret
}

// UpgradeToV4 converts the 3.x DS to a 4.x DS.
func (ds *DeliveryServiceNullableV30) UpgradeToV4() DeliveryServiceV4 {
	var ret DeliveryServiceV4
	if ds.Active != nil && *ds.Active {
		ret.Active = DS_ACTIVE
	} else {
		ret.Active = DS_PRIMED
	}
	if ds.AnonymousBlockingEnabled != nil && *ds.AnonymousBlockingEnabled {
		ret.AnonymousBlockingEnabled = true
	} else {
		ret.AnonymousBlockingEnabled = false
	}
	ret.CCRDNSTTL = ds.CCRDNSTTL
	if ds.CDNID != nil {
		ret.CDNID = *ds.CDNID
	}
	ret.CDNName = ds.CDNName
	ret.CheckPath = ds.CheckPath
	ret.ConsistentHashRegex = ds.ConsistentHashRegex
	ret.ConsistentHashQueryParams = make([]string, len(ds.ConsistentHashQueryParams))
	copy(ret.ConsistentHashQueryParams, ds.ConsistentHashQueryParams)
	if ds.DeepCachingType != nil && *ds.DeepCachingType == DeepCachingTypeAlways {
		ret.DeepCachingType = DeepCachingTypeAlways
	} else {
		ret.DeepCachingType = DeepCachingTypeNever
	}
	if ds.DisplayName != nil {
		ret.DisplayName = *ds.DisplayName
	}
	ret.DNSBypassCNAME = ds.DNSBypassCNAME
	ret.DNSBypassIP = ds.DNSBypassIP
	ret.DNSBypassIP6 = ds.DNSBypassIP6
	ret.DNSBypassTTL = ds.DNSBypassTTL
	if ds.DSCP != nil {
		ret.DSCP = *ds.DSCP
	}
	ret.EcsEnabled = ds.EcsEnabled
	ret.EdgeHeaderRewrite = ds.EdgeHeaderRewrite
	ret.ExampleURLs = make([]string, len(ds.ExampleURLs))
	copy(ret.ExampleURLs, ds.ExampleURLs)
	ret.FirstHeaderRewrite = ds.FirstHeaderRewrite
	ret.FQPacingRate = ds.FQPacingRate
	if ds.GeoLimit != nil && (*ds.GeoLimit == 1 || *ds.GeoLimit == 2) {
		ret.GeoLimit = *ds.GeoLimit
	} else {
		ret.GeoLimit = 0
	}
	ret.GeoLimitCountries = ds.GeoLimitCountries
	ret.GeoLimitRedirectURL = ds.GeoLimitRedirectURL
	if ds.GeoProvider != nil && *ds.GeoProvider == 1 {
		ret.GeoProvider = 1
	} else {
		ret.GeoProvider = 0
	}
	ret.GlobalMaxMBPS = ds.GlobalMaxMBPS
	ret.GlobalMaxTPS = ds.GlobalMaxTPS
	ret.HTTPBypassFQDN = ds.HTTPBypassFQDN
	ret.ID = ds.ID
	ret.InfoURL = ds.InfoURL
	ret.InitialDispersion = ds.InitialDispersion
	ret.InnerHeaderRewrite = ds.InnerHeaderRewrite
	if ds.IPV6RoutingEnabled != nil && *ds.IPV6RoutingEnabled {
		ret.IPV6RoutingEnabled = true
	} else {
		ret.IPV6RoutingEnabled = false
	}
	ret.LastHeaderRewrite = ds.LastHeaderRewrite
	if ds.LastUpdated != nil {
		ret.LastUpdated = ds.LastUpdated.Time
	}
	if ds.LogsEnabled != nil && *ds.LogsEnabled {
		ret.LogsEnabled = true
	} else {
		ret.LogsEnabled = false
	}
	ret.LongDesc = ds.LongDesc
	ret.LongDesc1 = ds.LongDesc1
	ret.LongDesc2 = ds.LongDesc2
	if ds.MatchList != nil {
		ret.MatchList = make([]DeliveryServiceMatch, len(*ds.MatchList))
		copy(ret.MatchList, *ds.MatchList)
	} else {
		ret.MatchList = nil
	}
	ret.MaxDNSAnswers = ds.MaxDNSAnswers
	ret.MaxOriginConnections = ds.MaxOriginConnections
	ret.MaxRequestHeaderBytes = ds.MaxRequestHeaderBytes
	ret.MidHeaderRewrite = ds.MidHeaderRewrite
	ret.MissLat = ds.MissLat
	ret.MissLong = ds.MissLong
	ret.MultiSiteOrigin = ds.MultiSiteOrigin
	ret.OriginShield = ds.OriginShield
	ret.OrgServerFQDN = ds.OrgServerFQDN
	ret.ProfileDesc = ds.ProfileDesc
	ret.ProfileID = ds.ProfileID
	ret.ProfileName = ds.ProfileName
	ret.Protocol = ds.Protocol
	ret.QStringIgnore = ds.QStringIgnore
	ret.RangeRequestHandling = ds.RangeRequestHandling
	ret.RangeSliceBlockSize = ds.RangeSliceBlockSize
	ret.RegexRemap = ds.RegexRemap
	if ds.RegionalGeoBlocking != nil && *ds.RegionalGeoBlocking {
		ret.RegionalGeoBlocking = true
	} else {
		ret.RegionalGeoBlocking = false
	}
	ret.RemapText = ds.RemapText
	if ds.RoutingName != nil {
		ret.RoutingName = *ds.RoutingName
	}
	ret.ServiceCategory = ds.ServiceCategory
	ret.Signed = ds.Signed
	ret.SigningAlgorithm = ds.SigningAlgorithm
	ret.SSLKeyVersion = ds.SSLKeyVersion
	ret.Tenant = ds.Tenant
	if ds.TenantID != nil {
		ret.TenantID = *ds.TenantID
	}
	ret.Topology = ds.Topology
	ret.TRResponseHeaders = ds.TRResponseHeaders
	ret.TRRequestHeaders = ds.TRRequestHeaders
	ret.Type = ds.Type
	if ds.TypeID != nil {
		ret.TypeID = *ds.TypeID
	}
	if ds.XMLID != nil {
		ret.XMLID = *ds.XMLID
	}
	return ret
}

func jsonValue(v interface{}) (driver.Value, error) {
	b, err := json.Marshal(v)
	return b, err
}

func jsonScan(src interface{}, dest interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected deliveryservice in byte array form; got %T", src)
	}
	return json.Unmarshal(b, dest)
}

// NOTE: the driver.Valuer and sql.Scanner interface implementations are
// necessary for Delivery Service Requests which store and read raw JSON
// from the database.

// Value implements the driver.Valuer interface --
// marshals struct to json to pass back as a json.RawMessage.
func (ds *DeliveryServiceNullable) Value() (driver.Value, error) {
	return jsonValue(ds)
}

// Scan implements the sql.Scanner interface --
// expects json.RawMessage and unmarshals to a DeliveryServiceNullable struct.
func (ds *DeliveryServiceNullable) Scan(src interface{}) error {
	return jsonScan(src, ds)
}

// Value implements the driver.Valuer interface --
// marshals struct to json to pass back as a json.RawMessage.
func (ds *DeliveryServiceV4) Value() (driver.Value, error) {
	return jsonValue(ds)
}

// Scan implements the sql.Scanner interface --
// expects json.RawMessage and unmarshals to a DeliveryServiceV4 struct.
func (ds *DeliveryServiceV4) Scan(src interface{}) error {
	return jsonScan(src, ds)
}

// DeliveryServiceMatch ...
type DeliveryServiceMatch struct {
	Type      DSMatchType `json:"type"`
	SetNumber int         `json:"setNumber"`
	Pattern   string      `json:"pattern"`
}

// DeliveryServiceStateResponse ...
type DeliveryServiceStateResponse struct {
	Response DeliveryServiceState `json:"response"`
}

// DeliveryServiceState ...
type DeliveryServiceState struct {
	Enabled  bool                    `json:"enabled"`
	Failover DeliveryServiceFailover `json:"failover"`
}

// DeliveryServiceFailover ...
type DeliveryServiceFailover struct {
	Locations   []string                   `json:"locations"`
	Destination DeliveryServiceDestination `json:"destination"`
	Configured  bool                       `json:"configured"`
	Enabled     bool                       `json:"enabled"`
}

// DeliveryServiceDestination ...
type DeliveryServiceDestination struct {
	Location string `json:"location"`
	Type     string `json:"type"`
}

// DeliveryServiceHealthResponse is the type of a response from Traffic Ops to
// a request for a Delivery Service's "health".
type DeliveryServiceHealthResponse struct {
	Response DeliveryServiceHealth `json:"response"`
	Alerts
}

// DeliveryServiceHealth ...
type DeliveryServiceHealth struct {
	TotalOnline  int                         `json:"totalOnline"`
	TotalOffline int                         `json:"totalOffline"`
	CacheGroups  []DeliveryServiceCacheGroup `json:"cacheGroups"`
}

// DeliveryServiceCacheGroup ...
type DeliveryServiceCacheGroup struct {
	Online  int    `json:"online"`
	Offline int    `json:"offline"`
	Name    string `json:"name"`
}

// DeliveryServiceCapacityResponse is the type of a response from Traffic Ops to
// a request for a Delivery Service's "capacity".
type DeliveryServiceCapacityResponse struct {
	Response DeliveryServiceCapacity `json:"response"`
	Alerts
}

// DeliveryServiceCapacity ...
type DeliveryServiceCapacity struct {
	AvailablePercent   float64 `json:"availablePercent"`
	UnavailablePercent float64 `json:"unavailablePercent"`
	UtilizedPercent    float64 `json:"utilizedPercent"`
	MaintenancePercent float64 `json:"maintenancePercent"`
}

type DeliveryServiceMatchesResp []DeliveryServicePatterns

type DeliveryServicePatterns struct {
	Patterns []string            `json:"patterns"`
	DSName   DeliveryServiceName `json:"dsName"`
}

type DeliveryServiceMatchesResponse struct {
	Response []DeliveryServicePatterns `json:"response"`
}

// DeliveryServiceRoutingResponse ...
type DeliveryServiceRoutingResponse struct {
	Response DeliveryServiceRouting `json:"response"`
}

// DeliveryServiceRouting ...
type DeliveryServiceRouting struct {
	StaticRoute       int     `json:"staticRoute"`
	Miss              int     `json:"miss"`
	Geo               float64 `json:"geo"`
	Err               int     `json:"err"`
	CZ                float64 `json:"cz"`
	DSR               float64 `json:"dsr"`
	Fed               int     `json:"fed"`
	RegionalAlternate int     `json:"regionalAlternate"`
	RegionalDenied    int     `json:"regionalDenied"`
}

type UserAvailableDS struct {
	ID          *int    `json:"id" db:"id"`
	DisplayName *string `json:"displayName" db:"display_name"`
	XMLID       *string `json:"xmlId" db:"xml_id"`
	TenantID    *int    `json:"-"` // tenant is necessary to check authorization, but not serialized
}

type FederationDeliveryServiceNullable struct {
	ID    *int    `json:"id" db:"id"`
	CDN   *string `json:"cdn" db:"cdn"`
	Type  *string `json:"type" db:"type"`
	XMLID *string `json:"xmlId" db:"xml_id"`
}

// FederationDeliveryServicesResponse is the type of a response from Traffic
// Ops to a request made to its /federations/{{ID}}/deliveryservices endpoint.
type FederationDeliveryServicesResponse struct {
	Response []FederationDeliveryServiceNullable `json:"response"`
	Alerts
}

type DeliveryServiceUserPost struct {
	UserID           *int   `json:"userId"`
	DeliveryServices *[]int `json:"deliveryServices"`
	Replace          *bool  `json:"replace"`
}

type UserDeliveryServicePostResponse struct {
	Alerts   []Alert                 `json:"alerts"`
	Response DeliveryServiceUserPost `json:"response"`
}

type UserDeliveryServicesNullableResponse struct {
	Response []DeliveryServiceNullable `json:"response"`
}

type DSServerIDs struct {
	DeliveryServiceID *int  `json:"dsId" db:"deliveryservice"`
	ServerIDs         []int `json:"servers"`
	Replace           *bool `json:"replace"`
}

// DeliveryserviceserverResponse - not to be confused with DSServerResponseV40
// or DSServerResponse- is the type of a response from Traffic Ops to a request
// to the /deliveryserviceserver endpoint to associate servers with a Delivery
// Service.
type DeliveryserviceserverResponse struct {
	Response DSServerIDs `json:"response"`
	Alerts
}

type CachegroupPostDSReq struct {
	DeliveryServices []int `json:"deliveryServices"`
}

type CacheGroupPostDSResp struct {
	ID               util.JSONIntStr `json:"id"`
	ServerNames      []CacheName     `json:"serverNames"`
	DeliveryServices []int           `json:"deliveryServices"`
}

type CacheGroupPostDSRespResponse struct {
	Alerts
	Response CacheGroupPostDSResp `json:"response"`
}

type AssignedDsResponse struct {
	ServerID int   `json:"serverId"`
	DSIds    []int `json:"dsIds"`
	Replace  bool  `json:"replace"`
}

// DeliveryServiceSafeUpdateRequest represents a request to update the "safe" fields of a
// Delivery Service.
type DeliveryServiceSafeUpdateRequest struct {
	DisplayName *string `json:"displayName"`
	InfoURL     *string `json:"infoUrl"`
	LongDesc    *string `json:"longDesc"`
	LongDesc1   *string `json:"longDesc1"`
}

// Validate implements the github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api.ParseValidator
// interface.
func (r *DeliveryServiceSafeUpdateRequest) Validate(*sql.Tx) error {
	if r.DisplayName == nil {
		return errors.New("displayName: cannot be null/missing")
	}
	return nil
}

// DeliveryServiceSafeUpdateResponse represents Traffic Ops's response to a PUT
// request to its /deliveryservices/{{ID}}/safe endpoint.
// Deprecated: Please only use versioned structures.
type DeliveryServiceSafeUpdateResponse struct {
	Alerts
	// Response contains the representation of the Delivery Service after it has been updated.
	Response []DeliveryServiceNullable `json:"response"`
}

// DeliveryServiceSafeUpdateResponseV30 represents Traffic Ops's response to a PUT
// request to its /api/3.0/deliveryservices/{{ID}}/safe endpoint.
type DeliveryServiceSafeUpdateResponseV30 struct {
	Alerts
	// Response contains the representation of the Delivery Service after it has
	// been updated.
	Response []DeliveryServiceNullableV30 `json:"response"`
}

// DeliveryServiceSafeUpdateResponseV40 represents Traffic Ops's response to a PUT
// request to its /api/4.0/deliveryservices/{{ID}}/safe endpoint.
type DeliveryServiceSafeUpdateResponseV40 struct {
	Alerts
	// Response contains the representation of the Delivery Service after it has
	// been updated.
	Response []DeliveryServiceV40 `json:"response"`
}

// DeliveryServiceSafeUpdateResponseV4 represents TrafficOps's response to a
// PUT request to its /api/4.x/deliveryservices/{{ID}}/safe endpoint.
// This is always a type alias for the structure of a response in the latest
// minor APIv4 version.
type DeliveryServiceSafeUpdateResponseV4 = DeliveryServiceSafeUpdateResponseV40
