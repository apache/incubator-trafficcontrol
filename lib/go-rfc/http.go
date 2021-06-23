package rfc

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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-util"
)

// These are the names of HTTP Headers, for convenience and so that typos are
// caught at compile-time.
const (
	AcceptEncoding     = "Accept-Encoding"     // RFC7231§5.3.4
	CacheControl       = "Cache-Control"       // RFC7234§5.2
	ContentDisposition = "Content-Disposition" // RFC6266
	ContentEncoding    = "Content-Encoding"    // RFC7231§3.1.2.2
	ContentType        = "Content-Type"        // RFC7231§3.1.1.5
	PermissionsPolicy  = "Permissions-Policy"  // W3C "Permissions Policy"
	Server             = "Server"              // RFC7231§7.4.2
	UserAgent          = "User-Agent"          // RFC7231§5.5.3
	Location           = "Location"            // RFC7231§6.3.2
	Vary               = "Vary"                // RFC7231§7.1.4
)

// These are (some) valid values for content encoding and MIME types, for
// convenience and so that typos are caught at compile-time.
const (
	ApplicationJSON           = "application/json"         // RFC4627§6
	ApplicationOctetStream    = "application/octet-stream" // RFC2046§4.5.2
	ContentTypeMultiPartMixed = "multipart/mixed"          // RFC1341§7.2
	ContentTypeTextPlain      = "text/plain"               // RFC2046§4.1
	ContentTypeURIList        = "text/uri-list"            // RFC2483§5
	Gzip                      = "gzip"                     // RFC7230§4.2.3
)

// LastModifiedFormat is the format used by dates in the HTTP Last-Modified
// header.
const LastModifiedFormat = "Mon, 02 Jan 2006 15:04:05 MST" // RFC1123

// ValidHTTPCodes provides fast lookup of whether a HTTP response code is valid.
var ValidHTTPCodes = map[int]struct{}{
	http.StatusContinue:           {}, // RFC 7231, 6.2.1
	http.StatusSwitchingProtocols: {}, // RFC 7231, 6.2.2
	http.StatusProcessing:         {}, // RFC 2518, 10.1
	http.StatusEarlyHints:         {}, // RFC 8297

	http.StatusOK:                   {}, // RFC 7231, 6.3.1
	http.StatusCreated:              {}, // RFC 7231, 6.3.2
	http.StatusAccepted:             {}, // RFC 7231, 6.3.3
	http.StatusNonAuthoritativeInfo: {}, // RFC 7231, 6.3.4
	http.StatusNoContent:            {}, // RFC 7231, 6.3.5
	http.StatusResetContent:         {}, // RFC 7231, 6.3.6
	http.StatusPartialContent:       {}, // RFC 7233, 4.1
	http.StatusMultiStatus:          {}, // RFC 4918, 11.1
	http.StatusAlreadyReported:      {}, // RFC 5842, 7.1
	http.StatusIMUsed:               {}, // RFC 3229, 10.4.1

	http.StatusMultipleChoices:   {}, // RFC 7231, 6.4.1
	http.StatusMovedPermanently:  {}, // RFC 7231, 6.4.2
	http.StatusFound:             {}, // RFC 7231, 6.4.3
	http.StatusSeeOther:          {}, // RFC 7231, 6.4.4
	http.StatusNotModified:       {}, // RFC 7232, 4.1
	http.StatusUseProxy:          {}, // RFC 7231, 6.4.5
	306:                          {}, // 306 is reserved for future use - valid, but unused
	http.StatusTemporaryRedirect: {}, // RFC 7231, 6.4.7
	http.StatusPermanentRedirect: {}, // RFC 7538, 3

	http.StatusBadRequest:                   {}, // RFC 7231, 6.5.1
	http.StatusUnauthorized:                 {}, // RFC 7235, 3.1
	http.StatusPaymentRequired:              {}, // RFC 7231, 6.5.2
	http.StatusForbidden:                    {}, // RFC 7231, 6.5.3
	http.StatusNotFound:                     {}, // RFC 7231, 6.5.4
	http.StatusMethodNotAllowed:             {}, // RFC 7231, 6.5.5
	http.StatusNotAcceptable:                {}, // RFC 7231, 6.5.6
	http.StatusProxyAuthRequired:            {}, // RFC 7235, 3.2
	http.StatusRequestTimeout:               {}, // RFC 7231, 6.5.7
	http.StatusConflict:                     {}, // RFC 7231, 6.5.8
	http.StatusGone:                         {}, // RFC 7231, 6.5.9
	http.StatusLengthRequired:               {}, // RFC 7231, 6.5.10
	http.StatusPreconditionFailed:           {}, // RFC 7232, 4.2
	http.StatusRequestEntityTooLarge:        {}, // RFC 7231, 6.5.11
	http.StatusRequestURITooLong:            {}, // RFC 7231, 6.5.12
	http.StatusUnsupportedMediaType:         {}, // RFC 7231, 6.5.13
	http.StatusRequestedRangeNotSatisfiable: {}, // RFC 7233, 4.4
	http.StatusExpectationFailed:            {}, // RFC 7231, 6.5.14
	http.StatusTeapot:                       {}, // RFC 7168, 2.3.3
	http.StatusMisdirectedRequest:           {}, // RFC 7540, 9.1.2
	http.StatusUnprocessableEntity:          {}, // RFC 4918, 11.2
	http.StatusLocked:                       {}, // RFC 4918, 11.3
	http.StatusFailedDependency:             {}, // RFC 4918, 11.4
	http.StatusTooEarly:                     {}, // RFC 8470, 5.2.
	http.StatusUpgradeRequired:              {}, // RFC 7231, 6.5.15
	http.StatusPreconditionRequired:         {}, // RFC 6585, 3
	http.StatusTooManyRequests:              {}, // RFC 6585, 4
	http.StatusRequestHeaderFieldsTooLarge:  {}, // RFC 6585, 5
	http.StatusUnavailableForLegalReasons:   {}, // RFC 7725, 3

	http.StatusInternalServerError:           {}, // RFC 7231, 6.6.1
	http.StatusNotImplemented:                {}, // RFC 7231, 6.6.2
	http.StatusBadGateway:                    {}, // RFC 7231, 6.6.3
	http.StatusServiceUnavailable:            {}, // RFC 7231, 6.6.4
	http.StatusGatewayTimeout:                {}, // RFC 7231, 6.6.5
	http.StatusHTTPVersionNotSupported:       {}, // RFC 7231, 6.6.6
	http.StatusVariantAlsoNegotiates:         {}, // RFC 2295, 8.1
	http.StatusInsufficientStorage:           {}, // RFC 4918, 11.5
	http.StatusLoopDetected:                  {}, // RFC 5842, 7.2
	http.StatusNotExtended:                   {}, // RFC 2774, 7
	http.StatusNetworkAuthenticationRequired: {}, // RFC 6585, 6
}

// AcceptsGzip returns whether r accepts gzip encoding, per RFC7231§5.3.4.
func AcceptsGzip(r *http.Request) bool {
	encodingHeaders := r.Header[AcceptEncoding] // headers are case-insensitive, but Go promises to Canonical-Case requests
	for _, encodingHeader := range encodingHeaders {
		encodingHeader = util.StripAllWhitespace(encodingHeader)
		encodings := strings.Split(encodingHeader, ",")
		for _, encoding := range encodings {
			if strings.ToLower(encoding) == Gzip { // encoding is case-insensitive, per the RFC7231§3.1.2.1.
				return true
			}
		}
	}
	return false
}

// GetHTTPDate is a helper function which gets an HTTP date from the given map
// (which is typically a `http.Header` or `CacheControl`.
//
// Returns false if the given key doesn't exist in the map, or if the value isn't
// a valid HTTP Date per RFC2616§3.3.
func GetHTTPDate(headers http.Header, key string) (time.Time, bool) {
	maybeDate := headers.Get(key)
	if maybeDate == "" {
		return time.Time{}, false
	}
	return ParseHTTPDate(maybeDate)
}

// ParseHTTPDate parses the given RFC7231§7.1.1 HTTP-date.
func ParseHTTPDate(d string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC1123, d); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC850, d); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.ANSIC, d); err == nil {
		return t, true
	}
	return time.Time{}, false

}

// FormatHTTPDate formats t as an RFC7231§7.1.1 HTTP-date.
func FormatHTTPDate(t time.Time) string {
	return t.Format(time.RFC1123)
}

// GetHTTPDeltaSeconds is a helper function which gets an HTTP Delta Seconds
// from the given map (which is typically a `http.Header` or `CacheControl`).
// Returns false if the given key doesn't exist in the map, or if the value
// isn't a valid Delta Seconds per RFC2616§3.3.2.
func GetHTTPDeltaSeconds(m map[string][]string, key string) (time.Duration, bool) {
	maybeSeconds, ok := m[key]
	if !ok {
		return 0, false
	}
	if len(maybeSeconds) == 0 {
		return 0, false
	}
	maybeSec := maybeSeconds[0]

	seconds, err := strconv.ParseUint(maybeSec, 10, 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}
