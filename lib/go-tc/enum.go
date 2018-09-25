// Package enum contains enumerations and strongly typed names.
//
// These enums should be treated as enumerables, and MUST NOT be cast as anything else (integer, strings, etc). Enums MUST NOT be compared to strings or integers via casting. Enumerable data SHOULD be stored as the enumeration, not as a string or number. The *only* reason they are internally represented as strings, is to make them implicitly serialize to human-readable JSON. They should not be treated as strings. Casting or storing strings or numbers defeats the purpose of enum safety and conveniences.
//
// When storing enumumerable data in memory, it SHOULD be converted to and stored as an enum via the corresponding `FromString` function, checked whether the conversion failed and Invalid values handled, and valid data stored as the enum. This guarantees stored data is valid, and catches invalid input as soon as possible.
//
// When adding new enum types, enums should be internally stored as strings, so they implicitly serialize as human-readable JSON, unless the performance or memory of integers is necessary (it almost certainly isn't). Enums should always have the "invalid" value as the empty string (or 0), so default-initialized enums are invalid.
// Enums should always have a FromString() conversion function, to convert input data to enums. Conversion functions should usually be case-insensitive, and may ignore underscores or hyphens, depending on the use case.
//
package tc

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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CDNName is the name of a CDN in Traffic Control.
type CDNName string

// TrafficMonitorName is the hostname of a Traffic Monitor peer.
type TrafficMonitorName string

// CacheName is the hostname of a CDN cache.
type CacheName string

// CacheGroupName is the name of a CDN cachegroup.
type CacheGroupName string

// DeliveryServiceName is the name of a CDN delivery service.
type DeliveryServiceName string

// CacheType is the type (or tier) of a CDN cache.
type CacheType string

const (
	// CacheTypeEdge represents an edge cache.
	CacheTypeEdge = CacheType("EDGE")
	// CacheTypeMid represents a mid cache.
	CacheTypeMid = CacheType("MID")
	// CacheTypeInvalid represents an cache type enumeration. Note this is the default construction for a CacheType.
	CacheTypeInvalid = CacheType("")
)

func (c CacheName) String() string {
	return string(c)
}

func (t TrafficMonitorName) String() string {
	return string(t)
}

func (d DeliveryServiceName) String() string {
	return string(d)
}

// String returns a string representation of this cache type.
func (t CacheType) String() string {
	switch t {
	case CacheTypeEdge:
		return "EDGE"
	case CacheTypeMid:
		return "MID"
	default:
		return "INVALIDCACHETYPE"
	}
}

// CacheTypeFromString returns a cache type object from its string representation, or CacheTypeInvalid if the string is not a valid type.
func CacheTypeFromString(s string) CacheType {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "edge") {
		return CacheTypeEdge
	}
	if strings.HasPrefix(s, "mid") {
		return CacheTypeMid
	}
	return CacheTypeInvalid
}

const ParameterSecureCapability = `parameters-read-secure`
const ServerSecureCapability = `servers-read-secure`

// DSTypeCategory is the Delivery Service type category: HTTP or DNS
type DSTypeCategory string

const (
	// DSTypeCategoryHTTP represents an HTTP delivery service
	DSTypeCategoryHTTP = DSTypeCategory("http")
	// DSTypeCategoryDNS represents a DNS delivery service
	DSTypeCategoryDNS = DSTypeCategory("dns")
	// DSTypeCategoryInvalid represents an invalid delivery service type enumeration. Note this is the default construction for a DSTypeCategory.
	DSTypeCategoryInvalid = DSTypeCategory("")
)

// String returns a string representation of this delivery service type.
func (t DSTypeCategory) String() string {
	switch t {
	case DSTypeCategoryHTTP:
		return "HTTP"
	case DSTypeCategoryDNS:
		return "DNS"
	default:
		return "INVALIDDSTYPE"
	}
}

// DSTypeCategoryFromString returns a delivery service type object from its string representation, or DSTypeCategoryInvalid if the string is not a valid type.
func DSTypeCategoryFromString(s string) DSTypeCategory {
	s = strings.ToLower(s)
	switch s {
	case "http":
		return DSTypeCategoryHTTP
	case "dns":
		return DSTypeCategoryDNS
	default:
		return DSTypeCategoryInvalid
	}
}

// CacheStatus represents the Traffic Server status set in Traffic Ops (online, offline, admin_down, reported). The string values of this type should match the Traffic Ops values.
type CacheStatus string

const (
	// CacheStatusAdminDown represents a cache which has been administratively marked as down, but which should still appear in the CDN (Traffic Server, Traffic Monitor, Traffic Router).
	CacheStatusAdminDown = CacheStatus("ADMIN_DOWN")
	// CacheStatusOnline represents a cache which has been marked as Online in Traffic Ops, irrespective of monitoring. Traffic Monitor will always flag these caches as available.
	CacheStatusOnline = CacheStatus("ONLINE")
	// CacheStatusOffline represents a cache which has been marked as Offline in Traffic Ops. These caches will not be returned in any endpoint, and Traffic Monitor acts like they don't exist.
	CacheStatusOffline = CacheStatus("OFFLINE")
	// CacheStatusReported represents a cache which has been marked as Reported in Traffic Ops. These caches are polled for health and returned in endpoints as available or unavailable based on bandwidth, response time, and other factors. The vast majority of caches should be Reported.
	CacheStatusReported = CacheStatus("REPORTED")
	// CacheStatusInvalid represents an invalid status enumeration.
	CacheStatusInvalid = CacheStatus("")
)

// String returns a string representation of this cache status
func (t CacheStatus) String() string {
	switch t {
	case CacheStatusAdminDown:
		fallthrough
	case CacheStatusOnline:
		fallthrough
	case CacheStatusOffline:
		fallthrough
	case CacheStatusReported:
		return string(t)
	default:
		return "INVALIDCACHESTATUS"
	}
}

// CacheStatusFromString returns a CacheStatus from its string representation, or CacheStatusInvalid if the string is not a valid type.
func CacheStatusFromString(s string) CacheStatus {
	s = strings.ToLower(s)
	switch s {
	case "admin_down":
		fallthrough
	case "admindown":
		return CacheStatusAdminDown
	case "offline":
		return CacheStatusOffline
	case "online":
		return CacheStatusOnline
	case "reported":
		return CacheStatusReported
	default:
		return CacheStatusInvalid
	}
}

// LocalizationMethod represents an enabled localization method for a cachegroup. The string values of this type should match the Traffic Ops values.
type LocalizationMethod string

const (
	LocalizationMethodCZ      = LocalizationMethod("CZ")
	LocalizationMethodDeepCZ  = LocalizationMethod("DEEP_CZ")
	LocalizationMethodGeo     = LocalizationMethod("GEO")
	LocalizationMethodInvalid = LocalizationMethod("INVALID")
)

// String returns a string representation of this localization method
func (m LocalizationMethod) String() string {
	switch m {
	case LocalizationMethodCZ:
		return string(m)
	case LocalizationMethodDeepCZ:
		return string(m)
	case LocalizationMethodGeo:
		return string(m)
	default:
		return "INVALID"
	}
}

func LocalizationMethodFromString(s string) LocalizationMethod {
	switch strings.ToLower(s) {
	case "cz":
		return LocalizationMethodCZ
	case "deep_cz":
		return LocalizationMethodDeepCZ
	case "geo":
		return LocalizationMethodGeo
	default:
		return LocalizationMethodInvalid
	}
}

func (m *LocalizationMethod) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return errors.New("LocalizationMethod cannot be null")
	}
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return errors.New(string(data) + " JSON not quoted")
	}
	*m = LocalizationMethodFromString(s)
	if *m == LocalizationMethodInvalid {
		return errors.New(s + " is not a LocalizationMethod")
	}
	return nil
}

func (m LocalizationMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *LocalizationMethod) Scan(value interface{}) error {
	if value == nil {
		return errors.New("LocalizationMethod cannot be null")
	}
	sv, err := driver.String.ConvertValue(value)
	if err != nil {
		return errors.New("failed to scan LocalizationMethod: " + err.Error())
	}

	switch v := sv.(type) {
	case []byte:
		*m = LocalizationMethodFromString(string(v))
		if *m == LocalizationMethodInvalid {
			return errors.New(string(v) + " is not a valid LocalizationMethod")
		}
		return nil
	default:
		return fmt.Errorf("failed to scan LocalizationMethod, unsupported input type: %T", value)
	}
}

// DeepCachingType represents a Delivery Service's deep caching type. The string values of this type should match the Traffic Ops values.
type DeepCachingType string

const (
	DeepCachingTypeNever   = DeepCachingType("") // default value
	DeepCachingTypeAlways  = DeepCachingType("ALWAYS")
	DeepCachingTypeInvalid = DeepCachingType("INVALID")
)

// String returns a string representation of this deep caching type
func (t DeepCachingType) String() string {
	switch t {
	case DeepCachingTypeAlways:
		return string(t)
	case DeepCachingTypeNever:
		return "NEVER"
	default:
		return "INVALID"
	}
}

// DeepCachingTypeFromString returns a DeepCachingType from its string representation, or DeepCachingTypeInvalid if the string is not a valid type.
func DeepCachingTypeFromString(s string) DeepCachingType {
	switch strings.ToLower(s) {
	case "always":
		return DeepCachingTypeAlways
	case "never":
		return DeepCachingTypeNever
	case "":
		// default when omitted
		return DeepCachingTypeNever
	default:
		return DeepCachingTypeInvalid
	}
}

// UnmarshalJSON unmarshals a JSON representation of a DeepCachingType (i.e. a string) or returns an error if the DeepCachingType is invalid
func (t *DeepCachingType) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = DeepCachingTypeNever
		return nil
	}
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return errors.New(string(data) + " JSON not quoted")
	}
	*t = DeepCachingTypeFromString(s)
	if *t == DeepCachingTypeInvalid {
		return errors.New(string(data) + " is not a DeepCachingType")
	}
	return nil
}

// MarshalJSON marshals into a JSON representation
func (t DeepCachingType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// DSType is the Delivery Service type.
type DSType string

const (
	DSTypeClientSteering   DSType = "CLIENT_STEERING"
	DSTypeDNS              DSType = "DNS"
	DSTypeDNSLive          DSType = "DNS_LIVE"
	DSTypeDNSLiveNational  DSType = "DNS_LIVE_NATNL"
	DSTypeHTTP             DSType = "HTTP"
	DSTypeHTTPLive         DSType = "HTTP_LIVE"
	DSTypeHTTPLiveNational DSType = "HTTP_LIVE_NATNL"
	DSTypeHTTPNoCache      DSType = "HTTP_NO_CACHE"
	DSTypeSteering         DSType = "STEERING"
	DSTypeAnyMap           DSType = "ANY_MAP"
	DSTypeInvalid          DSType = ""
)

// String returns a string representation of this delivery service type.
func (t DSType) String() string {
	switch t {
	case DSTypeHTTPNoCache:
		fallthrough
	case DSTypeDNS:
		fallthrough
	case DSTypeDNSLive:
		fallthrough
	case DSTypeHTTP:
		fallthrough
	case DSTypeDNSLiveNational:
		fallthrough
	case DSTypeAnyMap:
		fallthrough
	case DSTypeHTTPLive:
		fallthrough
	case DSTypeSteering:
		fallthrough
	case DSTypeHTTPLiveNational:
		fallthrough
	case DSTypeClientSteering:
		return string(t)
	default:
		return "INVALID"
	}
}

// DSTypeFromString returns a delivery service type object from its string representation, or DSTypeInvalid if the string is not a valid type.
func DSTypeFromString(s string) DSType {
	s = strings.ToLower(strings.Replace(s, "_", "", -1))
	switch s {
	case "httpnocache":
		return DSTypeHTTPNoCache
	case "dns":
		return DSTypeDNS
	case "dnslive":
		return DSTypeDNSLive
	case "http":
		return DSTypeHTTP
	case "dnslivenatnl":
		return DSTypeDNSLiveNational
	case "anymap":
		return DSTypeAnyMap
	case "httplive":
		return DSTypeHTTPLive
	case "steering":
		return DSTypeSteering
	case "httplivenatnl":
		return DSTypeHTTPLiveNational
	case "clientsteering":
		return DSTypeClientSteering
	default:
		return DSTypeInvalid
	}
}

// IsHTTP returns whether the DSType is an HTTP category.
func (t DSType) IsHTTP() bool {
	switch t {
	case DSTypeHTTP:
		fallthrough
	case DSTypeHTTPLive:
		fallthrough
	case DSTypeHTTPLiveNational:
		fallthrough
	case DSTypeHTTPNoCache:
		return true
	}
	return false
}

// IsDNS returns whether the DSType is a DNS category.
func (t DSType) IsDNS() bool {
	switch t {
	case DSTypeHTTPNoCache:
		fallthrough
	case DSTypeDNS:
		fallthrough
	case DSTypeDNSLive:
		fallthrough
	case DSTypeDNSLiveNational:
		return true
	}
	return false
}

// IsSteering returns whether the DSType is a Steering category
func (t DSType) IsSteering() bool {
	switch t {
	case DSTypeSteering:
		fallthrough
	case DSTypeClientSteering:
		fallthrough
	case DSTypeDNSLive:
		return true
	}
	return false
}

// HasSSLKeys returns whether delivery services of this type have SSL keys.
func (t DSType) HasSSLKeys() bool {
	return t.IsHTTP() || t.IsDNS() || t.IsSteering()
}

// IsLive returns whether delivery services of this type are "live".
func (t DSType) IsLive() bool {
	switch t {
	case DSTypeDNSLive:
		fallthrough
	case DSTypeDNSLiveNational:
		fallthrough
	case DSTypeHTTPLive:
		fallthrough
	case DSTypeHTTPLiveNational:
		return true
	}
	return false
}

// IsLive returns whether delivery services of this type are "national".
func (t DSType) IsNational() bool {
	switch t {
	case DSTypeDNSLiveNational:
		fallthrough
	case DSTypeHTTPLiveNational:
		return true
	}
	return false
}

// UsesMidCache returns whether delivery services of this type use mid-tier caches
func (t DSType) UsesMidCache() bool {
	switch t {
	case DSTypeDNSLive:
		fallthrough
	case DSTypeHTTPLive:
		fallthrough
	case DSTypeHTTPNoCache:
		return false
	}
	return true
}

type DSMatchType string

const (
	DSMatchTypeHostRegex     DSMatchType = "HOST_REGEXP"
	DSMatchTypePathRegex     DSMatchType = "PATH_REGEXP"
	DSMatchTypeSteeringRegex DSMatchType = "STEERING_REGEXP"
	DSMatchTypeHeaderRegex   DSMatchType = "HEADER_REGEXP"
	DSMatchTypeInvalid       DSMatchType = ""
)

// String returns a string representation of this delivery service match type.
func (t DSMatchType) String() string {
	switch t {
	case DSMatchTypeHostRegex:
		fallthrough
	case DSMatchTypePathRegex:
		fallthrough
	case DSMatchTypeSteeringRegex:
		fallthrough
	case DSMatchTypeHeaderRegex:
		return string(t)
	default:
		return "INVALID_MATCH_TYPE"
	}
}

// DSMatchTypeFromString returns a delivery service match type object from its string representation, or DSMatchTypeInvalid if the string is not a valid type.
func DSMatchTypeFromString(s string) DSMatchType {
	s = strings.ToLower(strings.Replace(s, "_", "", -1))
	switch s {
	case "hostregexp":
		return DSMatchTypeHostRegex
	case "pathregexp":
		return DSMatchTypePathRegex
	case "steeringregexp":
		return DSMatchTypeSteeringRegex
	case "headerregexp":
		return DSMatchTypeHeaderRegex
	default:
		return DSMatchTypeInvalid
	}
}
