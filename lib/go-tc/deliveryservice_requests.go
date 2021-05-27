package tc

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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-util"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

// EmailTemplate is an html/template.Template for formatting DeliveryServiceRequestRequests into
// text/html email bodies. Its direct use is discouraged, instead use
// DeliveryServiceRequestRequest.Format.
var EmailTemplate = template.Must(template.New("Email Template").Parse(`<!DOCTYPE html>
<html lang="en-US">
<head>
<meta charset="utf-8"/>
<title>Delivery Service Request for {{.Customer}}</title>
<style>
aside {
	padding: 0 1em;
	color: #6A737D;
	border-left: .25em solid #DFE2E5;
}
body {
	font-family: sans;
	background-color: white;
}
pre {
	padding: 5px;
	background-color: lightgray;
}
</style>
</head>
<body>
<h1>Delivery Service Request for {{.Customer}}</h1>
<p>{{.ServiceDesc}}</p>
<section>
	<details>
		<summary><h2>Service Description</h2></summary>
		<h3>Content Type</h3>
		<p>{{.ContentType}}</p>
		<h3>Delivery Protocol</h3>
		<p>{{.DeliveryProtocol.String}}</p>
		<h3>Routing Type</h3>
		<p>{{.RoutingType.String}}</p>
	</details>
</section>
<section>
	<details>
		<summary><h2>Traffic &amp; Library Estimates</h2></summary>
		<h3>Peak Bandwidth Estimate</h3>
		<p>{{.PeakBPSEstimate}}Bps</p>
		<h3>Peak Transactions per Second Estimate</h3>
		<p>{{.PeakTPSEstimate}}Tps</p>
		<h3>Max Library Size Estimate</h3>
		<p>{{.MaxLibrarySizeEstimate}}GB</p>
	</details>
</section>
<section>
	<details>
		<summary><h2>Origin Security</h2></summary>
		<h3>Origin Server URL</h3>
		<p><a href="{{.OriginURL}}">{{.OriginURL}}</a></p>
		<h3>Origin Dynamic Remap</h3>
		<p>{{.HasOriginDynamicRemap}}</p>
		<h3>Origin Test File</h3>
		<p>{{.OriginTestFile}}</p>
		<h3>ACL/Whitelist to Access Origin</h3>
		<p>{{.HasOriginACLWhitelist}}</p>
		{{if .OriginHeaders}}<h3>Header(s) to Access Origin</h3>
		<ul>{{range .OriginHeaders}}
			<li>{{.}}</li>{{end}}
		</ul>{{end}}
		<h3>Other Origin Security</h3>
		<p>{{if .OtherOriginSecurity}}{{.OtherOriginSecurity}}{{else}}None{{end}}</p>
	</details>
</section>
<section>
	<details>
		<summary><h2>Core Features</h2></summary>
		<h3>Query String Handling</h3>
		<p>{{.QueryStringHandling}}</p>
		<h3>Range Request Handling</h3>
		<p>{{.RangeRequestHandling}}</p>
		<h3>Signed URLs / URL Tokenization</h3>
		<p>{{.HasSignedURLs}}</p>
		<h3>Negative Caching Customization</h3>
		<p>{{.HasNegativeCachingCustomization}}</p>
		{{if or .HasNegativeCachingCustomization .NegativeCachingCustomizationNote }}<aside>
			<p>{{.NegativeCachingCustomizationNote}}</p>
		</aside>{{else if .HasNegativeCachingCustomization}}<aside>
			<p><b>No instructions given!</b></p>
		</aside>{{end}}
		{{if .ServiceAliases}}<h3>Service Aliases</h3>
		<ul>{{range .ServiceAliases}}
			<li>{{.}}</li>{{end}}
		</ul>{{end}}
	</details>
</section>
{{if or .RateLimitingGBPS .RateLimitingTPS .OverflowService}}<section>
	<details>
		<summary><h2>Service Limits</h2></summary>
		{{if .RateLimitingGBPS}}<h3>Bandwidth Limit</h3>
		<p>{{.RateLimitingGBPS}}GBps</p>{{end}}
		{{if .RateLimitingTPS}}<h3>Transactions per Second Limit</h3>
		<p>{{.RateLimitingTPS}}Tps</p>{{end}}
		{{if .OverflowService}}<h3>Overflow Service</h3>
		<p>{{.OverflowService}}</p>{{end}}
	</details>
</section>{{end}}
{{if or .HeaderRewriteEdge .HeaderRewriteMid .HeaderRewriteRedirectRouter}}<section>
	<details>
		<summary><h2>Header Customization</h2></summary>
		{{if .HeaderRewriteEdge}}<h3>Header Rewrite - Edge Tier</h3>
		<pre>{{.HeaderRewriteEdge}}</pre>{{end}}
		{{if .HeaderRewriteMid}}<h3>Header Rewrite - Mid Tier</h3>
		<pre>{{.HeaderRewriteMid}}</pre>{{end}}
		{{if .HeaderRewriteRedirectRouter}}<h3>Header Rewrite - Router</h3>
		<pre>{{.HeaderRewriteRedirectRouter}}</pre>{{end}}
	</details>
</section>{{end}}
{{if .Notes}}<section>
	<details>
		<summary><h2>Additional Notes</h2></summary>
		<p>{{.Notes}}</p>
	</details>
</section>{{end}}
</body>
</html>
`))

// IDNoMod type is used to suppress JSON unmarshalling
type IDNoMod int

// DeliveryServiceRequestRequest is a literal request to make a Delivery Service.
type DeliveryServiceRequestRequest struct {
	// EmailTo is the email address that is ultimately the destination of a formatted DeliveryServiceRequestRequest.
	EmailTo string `json:"emailTo"`
	// Details holds the actual request in a data structure.
	Details DeliveryServiceRequestDetails `json:"details"`
}

// DeliveryServiceRequestDetails holds information about what a user is trying
// to change, with respect to a delivery service.
type DeliveryServiceRequestDetails struct {
	// ContentType is the type of content to be delivered, e.g. "static", "VOD" etc.
	ContentType string `json:"contentType"`
	// Customer is the requesting customer - typically this is a Tenant.
	Customer string `json:"customer"`
	// DeepCachingType represents whether or not the Delivery Service should use Deep Caching.
	DeepCachingType *DeepCachingType `json:"deepCachingType"`
	// Delivery Protocol is the protocol clients should use to connect to the Delivery Service.
	DeliveryProtocol *Protocol `json:"deliveryProtocol"`
	// HasNegativeCachingCustomization indicates whether or not the resulting Delivery Service should
	// customize the use of negative caching. When this is `true`, NegativeCachingCustomizationNote
	// should be consulted for instructions on the customization.
	HasNegativeCachingCustomization *bool `json:"hasNegativeCachingCustomization"`
	// HasOriginACLWhitelist indicates whether or not the Origin has an ACL whitelist. When this is
	// `true`, Notes should ideally contain the actual whitelist (or viewing instructions).
	HasOriginACLWhitelist *bool `json:"hasOriginACLWhitelist"`
	// Has OriginDynamicRemap indicates whether or not the OriginURL can dynamically map to multiple
	// different actual origin servers.
	HasOriginDynamicRemap *bool `json:"hasOriginDynamicRemap"`
	// HasSignedURLs indicates whether or not the resulting Delivery Service should sign its URLs.
	HasSignedURLs *bool `json:"hasSignedURLs"`
	// HeaderRewriteEdge is an optional HeaderRewrite rule to apply at the Edge tier.
	HeaderRewriteEdge *string `json:"headerRewriteEdge"`
	// HeaderRewriteMid is an optional HeaderRewrite rule to apply at the Mid tier.
	HeaderRewriteMid *string `json:"headerRewriteMid"`
	// HeaderRewriteRedirectRouter is an optional HeaderRewrite rule to apply at routing time by
	// the Traffic Router.
	HeaderRewriteRedirectRouter *string `json:"headerRewriteRedirectRouter"`
	// MaxLibrarySizeEstimate is an estimation of the total size of content that will be delivered
	// through the resulting Delivery Service.
	MaxLibrarySizeEstimate string `json:"maxLibrarySizeEstimate"`
	// NegativeCachingCustomizationNote is an optional note describing the customization to be
	// applied to Negative Caching. This should never be `nil` (or empty) if
	// HasNegativeCachingCustomization is `true`, but in that case the recipient ought to contact
	// Customer for instructions.
	NegativeCachingCustomizationNote *string `json:"negativeCachingCustomizationNote"`
	// Notes is an optional set of extra information supplied to describe the requested Delivery
	// Service.
	Notes *string `json:"notes"`
	// OriginHeaders is an optional list of HTTP headers that must be sent in requests to the Origin. When
	// parsing from JSON, this field can be either an actual array of headers, or a string containing
	// a comma-delimited list of said headers.
	OriginHeaders *OriginHeaders `json:"originHeaders"`
	// OriginTestFile is the path to a file on the origin that can be requested to test the server's
	// operational readiness, e.g. '/test.xml'.
	OriginTestFile string `json:"originTestFile"`
	// OriginURL is the URL of the origin server that has the content to be served by the requested
	// Delivery Service.
	OriginURL string `json:"originURL"`
	// OtherOriginSecurity is an optional note about any and all other Security employed by the origin
	// server (beyond an ACL whitelist, which has its own field: HasOriginACLWhitelist).
	OtherOriginSecurity *string `json:"otherOriginSecurity"`
	// OverflowService is an optional IP Address or URL to which clients should be redirected when
	// the requested Delivery Service exceeds its operational capacity.
	OverflowService *string `json:"overflowService"`
	// PeakBPSEstimate is an estimate of the bytes per second expected at peak operation.
	PeakBPSEstimate string `json:"peakBPSEstimate"`
	// PeakTPSEstimate is an estimate of the transactions per second expected at peak operation.
	PeakTPSEstimate string `json:"peakTPSEstimate"`
	// QueryStringHandling describes the manner in which the CDN should handle query strings in client
	// requests. Generally one of "use", "drop", or "ignore-in-cache-key-and-pass-up".
	QueryStringHandling string `json:"queryStringHandling"`
	// RangeRequestHandling describes the manner in which HTTP requests are handled.
	RangeRequestHandling string `json:"rangeRequestHandling"`
	// RateLimitingGBPS is an optional rate limit for the requested Delivery Service in gigabytes per
	// second.
	RateLimitingGBPS *uint `json:"rateLimitingGBPS"`
	// RateLimitingTPS is an optional rate limit for the requested Delivery Service in transactions
	// per second.
	RateLimitingTPS *uint `json:"rateLimitingTPS"`
	// RoutingName is the top-level DNS label under which the Delivery Service should be requested.
	RoutingName string `json:"routingName"`
	// RoutingType is the type of routing Traffic Router should perform for the requested Delivery
	// Service.
	RoutingType *DSType `json:"routingType"`
	// ServiceAliases is an optional list of alternative names for the requested Delivery Service.
	ServiceAliases []string `json:"serviceAliases"`
	// ServiceDesc is a basic description of the requested Delivery Service.
	ServiceDesc string `json:"serviceDesc"`
}

// Format formats the DeliveryServiceRequestDetails into the text/html body of an email. The template
// used is EmailTemplate.
func (d DeliveryServiceRequestDetails) Format() (string, error) {
	b := &strings.Builder{}

	if err := EmailTemplate.Execute(b, d); err != nil {
		return "", fmt.Errorf("Failed to apply template: %v", err)
	}
	return b.String(), nil
}

// Validate validates that the delivery service request has all of the required fields. In some cases,
// e.g. the top-level EmailTo field, the format is also checked for correctness.
func (d *DeliveryServiceRequestRequest) Validate() error {
	errs := make([]error, 0, 2)

	err := validation.ValidateStruct(d,
		validation.Field(&d.EmailTo, validation.Required, is.Email),
	)
	if err != nil {
		errs = append(errs, err)
	}

	details := d.Details
	err = validation.ValidateStruct(&details,
		validation.Field(&details.ContentType, validation.Required),
		validation.Field(&details.Customer, validation.Required),
		validation.Field(&details.DeepCachingType, validation.By(
			func(t interface{}) error {
				if t != (*DeepCachingType)(nil) && *t.(*DeepCachingType) == DeepCachingTypeInvalid {
					return errors.New("deepCachingType: invalid Deep Caching Type")
				}
				return nil
			})),
		validation.Field(&details.DeliveryProtocol, validation.By(
			func(p interface{}) error {
				if p == (*Protocol)(nil) {
					return errors.New("deliveryProtocol: required")
				}
				if *p.(*Protocol) == ProtocolInvalid {
					return errors.New("deliveryProtocol: invalid Protocol")
				}
				return nil
			})),
		validation.Field(&details.HasNegativeCachingCustomization, validation.By(
			func(h interface{}) error {
				if h == (*bool)(nil) {
					return errors.New("hasNegativeCachingCustomization: required")
				}
				return nil
			})),
		validation.Field(&details.HasOriginACLWhitelist, validation.By(
			func(h interface{}) error {
				if h == (*bool)(nil) {
					return errors.New("hasNegativeCachingCustomization: required")
				}
				return nil
			})),
		validation.Field(&details.HasOriginDynamicRemap, validation.By(
			func(h interface{}) error {
				if h == (*bool)(nil) {
					return errors.New("hasNegativeCachingCustomization: required")
				}
				return nil
			})),
		validation.Field(&details.HasSignedURLs, validation.By(
			func(h interface{}) error {
				if h == (*bool)(nil) {
					return errors.New("hasNegativeCachingCustomization: required")
				}
				return nil
			})),
		validation.Field(&details.MaxLibrarySizeEstimate, validation.Required),
		validation.Field(&details.OriginHeaders, validation.By(
			func(h interface{}) error {
				if h == (*OriginHeaders)(nil) {
					return nil
				}
				if len(*h.(*OriginHeaders)) < 1 {
					return errors.New("originHeaders: cannot be an empty list (use 'null' if none)")
				}
				return nil
			})),
		validation.Field(&details.OriginTestFile, validation.Required),
		validation.Field(&details.OriginURL, validation.Required, is.URL),
		validation.Field(&details.PeakBPSEstimate, validation.Required),
		validation.Field(&details.PeakTPSEstimate, validation.Required),
		validation.Field(&details.QueryStringHandling, validation.Required),
		validation.Field(&details.RangeRequestHandling, validation.Required),
		validation.Field(&details.RoutingType, validation.By(
			func(t interface{}) error {
				if t == (*DSType)(nil) || *(t.(*DSType)) == "" {
					return errors.New("routingType: required")
				}
				*t.(*DSType) = DSTypeFromString(string(*t.(*DSType)))
				if *t.(*DSType) == DSTypeInvalid {
					return errors.New("routingType: invalid Routing Type")
				}
				return nil
			})),
		validation.Field(&details.ServiceDesc, validation.Required),
	)

	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return util.JoinErrs(errs)
	}
	return nil
}

// OriginHeaders represents a list of the headers that must be sent to the Origin.
type OriginHeaders []string

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *OriginHeaders) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = OriginHeaders([]string{})
		return nil
	}

	headers := []string{}
	if err := json.Unmarshal(data, &headers); err == nil {
		*o = OriginHeaders(headers)
		return nil
	}

	s, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("%s does not represent Origin Headers: %v", string(data), err)
	}

	*o = OriginHeaders(strings.Split(s, ","))
	return nil
}

// DeliveryServiceRequest is used as part of the workflow to create,
// modify, or delete a delivery service.
type DeliveryServiceRequest struct {
	AssigneeID      int             `json:"assigneeId,omitempty"`
	Assignee        string          `json:"assignee,omitempty"`
	AuthorID        IDNoMod         `json:"authorId"`
	Author          string          `json:"author"`
	ChangeType      string          `json:"changeType"`
	CreatedAt       *TimeNoMod      `json:"createdAt"`
	ID              int             `json:"id"`
	LastEditedBy    string          `json:"lastEditedBy,omitempty"`
	LastEditedByID  IDNoMod         `json:"lastEditedById,omitempty"`
	LastUpdated     *TimeNoMod      `json:"lastUpdated"`
	DeliveryService DeliveryService `json:"deliveryService"` // TODO version DeliveryServiceRequest
	Status          RequestStatus   `json:"status"`
	XMLID           string          `json:"-" db:"xml_id"`
}

// DeliveryServiceRequestNullable is used as part of the workflow to create,
// modify, or delete a delivery service.
type DeliveryServiceRequestNullable struct {
	AssigneeID      *int                        `json:"assigneeId,omitempty" db:"assignee_id"`
	Assignee        *string                     `json:"assignee,omitempty"`
	AuthorID        *IDNoMod                    `json:"authorId" db:"author_id"`
	Author          *string                     `json:"author"`
	ChangeType      *string                     `json:"changeType" db:"change_type"`
	CreatedAt       *TimeNoMod                  `json:"createdAt" db:"created_at"`
	ID              *int                        `json:"id" db:"id"`
	LastEditedBy    *string                     `json:"lastEditedBy"`
	LastEditedByID  *IDNoMod                    `json:"lastEditedById" db:"last_edited_by_id"`
	LastUpdated     *TimeNoMod                  `json:"lastUpdated" db:"last_updated"`
	DeliveryService *DeliveryServiceNullableV30 `json:"deliveryService" db:"deliveryservice"`
	Status          *RequestStatus              `json:"status" db:"status"`
	XMLID           *string                     `json:"-" db:"xml_id"`
}

// Upgrade coerces the DeliveryServiceRequestNullable to the newer
// DeliveryServiceRequestV40 structure.
//
// All reference properties are "deep"-copied so they may be modified without
// affecting the original. However, DeliveryService is constructed as a "deep"
// copy, but the properties of the underlying DeliveryServiceNullableV30 are
// "shallow" copied, and so modifying them *can* affect the original and
// vice-versa.
func (dsr DeliveryServiceRequestNullable) Upgrade() DeliveryServiceRequestV40 {
	var upgraded DeliveryServiceRequestV40
	if dsr.Assignee != nil {
		upgraded.Assignee = new(string)
		*upgraded.Assignee = *dsr.Assignee
	}
	if dsr.AssigneeID != nil {
		upgraded.AssigneeID = new(int)
		*upgraded.AssigneeID = *dsr.AssigneeID
	}
	if dsr.Author != nil {
		upgraded.Author = *dsr.Author
	}
	if dsr.AuthorID != nil {
		upgraded.AuthorID = new(int)
		*upgraded.AuthorID = int(*dsr.AuthorID)
	}
	if dsr.ChangeType != nil {
		upgraded.ChangeType = DSRChangeType(*dsr.ChangeType)
	}
	if dsr.CreatedAt != nil {
		upgraded.CreatedAt = dsr.CreatedAt.Time
	}
	if dsr.DeliveryService != nil {
		if upgraded.ChangeType == DSRChangeTypeDelete {
			upgraded.Original = new(DeliveryServiceV4)
			*upgraded.Original = dsr.DeliveryService.UpgradeToV4()
		} else {
			upgraded.Requested = new(DeliveryServiceV4)
			*upgraded.Requested = dsr.DeliveryService.UpgradeToV4()
		}
	}
	if dsr.ID != nil {
		upgraded.ID = new(int)
		*upgraded.ID = *dsr.ID
	}
	if dsr.LastEditedBy != nil {
		upgraded.LastEditedBy = *dsr.LastEditedBy
	}
	if dsr.LastEditedByID != nil {
		upgraded.LastEditedByID = new(int)
		*upgraded.LastEditedByID = int(*dsr.LastEditedByID)
	}
	if dsr.Status != nil {
		upgraded.Status = *dsr.Status
	}
	if dsr.XMLID != nil {
		upgraded.XMLID = *dsr.XMLID
	} else if dsr.DeliveryService != nil && dsr.DeliveryService.XMLID != nil {
		upgraded.XMLID = *dsr.DeliveryService.XMLID
	}
	return upgraded
}

// UnmarshalJSON implements the json.Unmarshaller interface to suppress
// unmarshalling for IDNoMod.
func (a *IDNoMod) UnmarshalJSON([]byte) error {
	return nil
}

// RequestStatus captures where in the workflow this request is
type RequestStatus string

const (
	// RequestStatusInvalid -- invalid state
	RequestStatusInvalid = RequestStatus("invalid")
	// RequestStatusDraft -- newly created; not ready to be reviewed
	RequestStatusDraft = RequestStatus("draft")
	// RequestStatusSubmitted -- newly created; ready to be reviewed
	RequestStatusSubmitted = RequestStatus("submitted")
	// RequestStatusRejected -- reviewed, but problems found
	RequestStatusRejected = RequestStatus("rejected")
	// RequestStatusPending -- reviewed and locked; ready to be implemented
	RequestStatusPending = RequestStatus("pending")
	// RequestStatusComplete -- implemented and locked
	RequestStatusComplete = RequestStatus("complete")
)

// String returns the string value of the Request Status.
func (r RequestStatus) String() string {
	return string(r)
}

// RequestStatuses -- user-visible string associated with each of the above.
var RequestStatuses = []RequestStatus{
	// "invalid" -- don't list here..
	"draft",
	"submitted",
	"rejected",
	"pending",
	"complete",
}

// UnmarshalJSON implements json.Unmarshaller.
func (r *RequestStatus) UnmarshalJSON(b []byte) error {
	u, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	// just check to see if the string represents a valid requeststatus
	_, err = RequestStatusFromString(u)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, (*string)(r))
}

// MarshalJSON implements json.Marshaller.
func (r RequestStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// Value implements driver.Valuer.
func (r *RequestStatus) Value() (driver.Value, error) {
	v, err := json.Marshal(r)
	log.Debugf("value is %v; err is %v", v, err)
	v = []byte(strings.Trim(string(v), `"`))
	return v, err
}

// Scan implements sql.Scanner.
func (r *RequestStatus) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected requeststatus in byte array form; got %T", src)
	}
	b = []byte(`"` + string(b) + `"`)
	return json.Unmarshal(b, r)
}

// RequestStatusFromString gets the status enumeration from a string.
func RequestStatusFromString(rs string) (RequestStatus, error) {
	if rs == "" {
		return RequestStatusDraft, nil
	}
	for _, s := range RequestStatuses {
		if string(s) == rs {
			return s, nil
		}
	}
	return RequestStatusInvalid, errors.New(rs + " is not a valid RequestStatus name")
}

// ValidTransition returns nil if the transition is allowed for the workflow,
// an error if not.
func (r RequestStatus) ValidTransition(to RequestStatus) error {
	if r == RequestStatusRejected || r == RequestStatusComplete {
		// once rejected or completed,  no changes allowed
		return errors.New(string(r) + " request cannot be changed")
	}

	if r == to {
		// no change -- always allowed
		return nil
	}

	// indicate if valid transitioning to this RequestStatus
	switch to {
	case RequestStatusDraft:
		// can go back to draft if submitted or rejected
		if r == RequestStatusSubmitted {
			return nil
		}
	case RequestStatusSubmitted:
		// can go be submitted if draft or rejected
		if r == RequestStatusDraft {
			return nil
		}
	case RequestStatusRejected:
		// only submitted can be rejected
		if r == RequestStatusSubmitted {
			return nil
		}
	case RequestStatusPending:
		// only submitted can move to pending
		if r == RequestStatusSubmitted {
			return nil
		}
	case RequestStatusComplete:
		// only submitted or pending requests can be completed
		if r == RequestStatusSubmitted || r == RequestStatusPending {
			return nil
		}
	}
	return errors.New("invalid transition from " + string(r) + " to " + string(to))
}

// DSRChangeType is an "enumerated" string type that encodes the legal values of
// a Delivery Service Request's Change Type.
type DSRChangeType string

// These are the valid values for Delivery Service Request Change Types.
const (
	// The original Delivery Service is being modified to match the requested
	// one.
	DSRChangeTypeUpdate = DSRChangeType("update")
	// The requested Delivery Service is being created.
	DSRChangeTypeCreate = DSRChangeType("create")
	// The requested Delivery Service is being deleted.
	DSRChangeTypeDelete = DSRChangeType("delete")
)

// DSRChangeTypeFromString converts the passed string to a DSRChangeType
// (case-insensitive), returning an error if the string is not a valid
// Delivery Service Request Change Type.
func DSRChangeTypeFromString(s string) (DSRChangeType, error) {
	switch strings.ToLower(s) {
	case "update":
		return DSRChangeTypeUpdate, nil
	case "create":
		return DSRChangeTypeCreate, nil
	case "delete":
		return DSRChangeTypeDelete, nil
	}
	return "INVALID", fmt.Errorf("invalid Delivery Service Request changeType: '%s'", s)
}

// String implements the fmt.Stringer interface, returning a textual
// representation of the DSRChangeType.
func (dsrct DSRChangeType) String() string {
	return string(dsrct)
}

// MarshalJSON implements the encoding/json.Marshaller interface.
func (dsrct DSRChangeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(dsrct))
}

// UnmarshalJSON implements the encoding/json.Unmarshaller interface.
func (dsrct *DSRChangeType) UnmarshalJSON(b []byte) error {
	// This should only happen if this method is called directly; encoding/json
	// itself guards against this.
	if dsrct == nil {
		return errors.New("UnmarshalJSON(nil *tc.DSRChangeType)")
	}

	ctStr, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	ct, err := DSRChangeTypeFromString(ctStr)
	if err != nil {
		return err
	}
	*dsrct = ct
	return nil
}

// DeliveryServiceRequestV40 is the type of a Delivery Service Request in
// Traffic Ops API version 4.0.
type DeliveryServiceRequestV40 struct {
	// Assignee is the username of the user assigned to the Delivery Service
	// Request, if any.
	Assignee *string `json:"assignee"`
	// AssigneeID is the integral, unique identifier of the user assigned to the
	// Delivery Service Request, if any.
	AssigneeID *int `json:"-" db:"assignee_id"`
	// Author is the username of the user who created the Delivery Service
	// Request.
	Author string `json:"author"`
	// AuthorID is the integral, unique identifier of the user who created the
	// Delivery Service Request, if/when it is known.
	AuthorID *int `json:"-" db:"author_id"`
	// ChangeType represents the type of change being made, must be one of
	// "create", "change" or "delete".
	ChangeType DSRChangeType `json:"changeType" db:"change_type"`
	// CreatedAt is the date/time at which the Delivery Service Request was
	// created.
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// ID is the integral, unique identifier for the Delivery Service Request
	// if/when it is known.
	ID *int `json:"id" db:"id"`
	// LastEditedBy is the username of the user by whom the Delivery Service
	// Request was last edited.
	LastEditedBy string `json:"lastEditedBy"`
	// LastEditedByID is the integral, unique identifier of the user by whom the
	// Delivery Service Request was last edited, if/when it is known.
	LastEditedByID *int `json:"-" db:"last_edited_by_id"`
	// LastUpdated is the date/time at which the Delivery Service was last
	// modified.
	LastUpdated time.Time `json:"lastUpdated" db:"last_updated"`
	// Original is the original Delivery Service for which changes are
	// requested. This is present in responses only for ChangeTypes 'change' and
	// 'delete', and is only required in requests where ChangeType is 'delete'.
	Original *DeliveryServiceV4 `json:"original,omitempty" db:"original"`
	// Requested is the set of requested changes. This is present in responses
	// only for ChangeTypes 'change' and 'create', and is only required in
	// requests in those cases.
	Requested *DeliveryServiceV4 `json:"requested,omitempty" db:"deliveryservice"`
	// Status is the status of the Delivery Service Request.
	Status RequestStatus `json:"status" db:"status"`
	// Used internally to define the affected Delivery Service.
	XMLID string `json:"-"`
}

// DeliveryServiceRequestV4 is the type of a Delivery Service Request as it
// appears in API version 4.
type DeliveryServiceRequestV4 = DeliveryServiceRequestV40

// IsOpen returns whether or not the Delivery Service Request is still "open" -
// i.e. has not been rejected or completed.
func (dsr DeliveryServiceRequestV40) IsOpen() bool {
	return !dsr.IsClosed()
}

// IsClosed returns whether or not the Delivery Service Request has been
// "closed", by being either rejected or completed.
func (dsr DeliveryServiceRequestV40) IsClosed() bool {
	return dsr.Status == RequestStatusComplete || dsr.Status == RequestStatusRejected || dsr.Status == RequestStatusPending
}

// Downgrade coerces the DeliveryServiceRequestV40 to the older
// DeliveryServiceRequestNullable structure.
//
// "XMLID" will be copied directly if it is non-empty, otherwise determined
// from the DeliveryService (if it's not 'nil').
//
// All reference properties are "deep"-copied so they may be modified without
// affecting the original. However, DeliveryService is constructed as a "deep"
// copy of "Requested", but the properties of the underlying
// DeliveryServiceNullableV30 are "shallow" copied, and so modifying them *can*
// affect the original and vice-versa.
func (dsr DeliveryServiceRequestV40) Downgrade() DeliveryServiceRequestNullable {
	downgraded := DeliveryServiceRequestNullable{
		Author:       new(string),
		ChangeType:   new(string),
		LastEditedBy: new(string),
		Status:       new(RequestStatus),
	}
	if dsr.Assignee != nil {
		downgraded.Assignee = new(string)
		*downgraded.Assignee = *dsr.Assignee
	}
	if dsr.AssigneeID != nil {
		downgraded.AssigneeID = new(int)
		*downgraded.AssigneeID = *dsr.AssigneeID
	}
	*downgraded.Author = dsr.Author
	if dsr.AuthorID != nil {
		downgraded.AuthorID = new(IDNoMod)
		*downgraded.AuthorID = IDNoMod(*dsr.AuthorID)
	}
	*downgraded.ChangeType = dsr.ChangeType.String()
	downgraded.CreatedAt = TimeNoModFromTime(dsr.CreatedAt)
	if dsr.Requested != nil {
		downgraded.DeliveryService = new(DeliveryServiceNullableV30)
		*downgraded.DeliveryService = dsr.Requested.DowngradeToV3()
	} else if dsr.Original != nil {
		downgraded.DeliveryService = new(DeliveryServiceNullableV30)
		*downgraded.DeliveryService = dsr.Original.DowngradeToV3()
	}
	if dsr.ID != nil {
		downgraded.ID = new(int)
		*downgraded.ID = *dsr.ID
	}
	*downgraded.LastEditedBy = dsr.LastEditedBy
	if dsr.LastEditedByID != nil {
		downgraded.LastEditedByID = new(IDNoMod)
		*downgraded.LastEditedByID = IDNoMod(*dsr.LastEditedByID)
	}
	downgraded.LastUpdated = TimeNoModFromTime(dsr.LastUpdated)
	*downgraded.Status = dsr.Status
	if dsr.XMLID != "" {
		downgraded.XMLID = new(string)
		*downgraded.XMLID = dsr.XMLID
	} else if dsr.Original != nil {
		downgraded.XMLID = new(string)
		*downgraded.XMLID = dsr.Original.XMLID
	} else {
		downgraded.XMLID = new(string)
		*downgraded.XMLID = dsr.Requested.XMLID
	}
	return downgraded
}

// String encodes the DeliveryServiceRequestV40 as a string, in the format
// "DeliveryServiceRequestV40({{Property}}={{Value}}[, {{Property}}={{Value}}]+)".
//
// If a property is a pointer value, then its dereferenced value is used -
// unless it's nil, in which case "<nil>" is used as the value. DeliveryService
// is omitted, because of how large it is. Times are formatted in RFC3339 format.
func (dsr DeliveryServiceRequestV40) String() string {
	var builder strings.Builder
	builder.Write([]byte("DeliveryServiceRequestV40(Assignee="))
	if dsr.Assignee != nil {
		builder.WriteRune('"')
		builder.WriteString(*dsr.Assignee)
		builder.WriteRune('"')
	} else {
		builder.Write([]byte("<nil>"))
	}
	builder.Write([]byte(", AssigneeID="))
	if dsr.AssigneeID != nil {
		builder.WriteString(strconv.Itoa(*dsr.AssigneeID))
	} else {
		builder.Write([]byte("<nil>"))
	}
	builder.Write([]byte(`, Author="`))
	builder.WriteString(dsr.Author)
	builder.Write([]byte(`", AuthorID=`))
	if dsr.AuthorID != nil {
		builder.WriteString(strconv.Itoa(*dsr.AuthorID))
	} else {
		builder.Write([]byte("<nil>"))
	}
	builder.Write([]byte(`, ChangeType="`))
	builder.WriteString(dsr.ChangeType.String())
	builder.Write([]byte(`", CreatedAt=`))
	builder.WriteString(dsr.CreatedAt.Format(time.RFC3339))
	builder.Write([]byte(", ID="))
	if dsr.ID != nil {
		builder.WriteString(strconv.Itoa(*dsr.ID))
	} else {
		builder.Write([]byte("<nil>"))
	}
	builder.Write([]byte(`, LastEditedBy="`))
	builder.WriteString(dsr.LastEditedBy)
	builder.Write([]byte(`", LastEditedByID=`))
	if dsr.LastEditedByID != nil {
		builder.WriteString(strconv.Itoa(*dsr.LastEditedByID))
	} else {
		builder.Write([]byte("<nil>"))
	}
	builder.Write([]byte(`, LastUpdated=`))
	builder.WriteString(dsr.LastUpdated.Format(time.RFC3339))
	builder.Write([]byte(`, Status="`))
	builder.WriteString(dsr.Status.String())
	builder.Write([]byte(`")`))
	return builder.String()
}

// SetXMLID sets the DeliveryServiceRequestV40's XMLID based on its DeliveryService.
func (dsr *DeliveryServiceRequestV40) SetXMLID() {
	if dsr == nil {
		return
	}

	if dsr.ChangeType == DSRChangeTypeDelete && dsr.Original != nil {
		dsr.XMLID = dsr.Original.XMLID
		return
	}

	if dsr.Requested != nil {
		dsr.XMLID = dsr.Requested.XMLID
	}
}

// StatusChangeRequest is the form of a PUT request body to
// /deliveryservice_requests/{{ID}}/status.
type StatusChangeRequest struct {
	// Status is the desired new status of the DSR.
	Status RequestStatus `json:"status"`
}

// Validate satisfies the
// github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api.ParseValidator
// interface.
func (*StatusChangeRequest) Validate(*sql.Tx) error {
	return nil
}

// DeliveryServiceRequestResponseV40 is the type of a response from
// Traffic Ops when creating, updating, or deleting a Delivery Service Request
// using API version 4.0.
type DeliveryServiceRequestResponseV40 struct {
	Response DeliveryServiceRequestV40 `json:"response"`
	Alerts
}

// DeliveryServiceRequestCUDResponseV4 is the type of a response from
// Traffic Ops when creating, updating, or deleting a Delivery Service Request
// using the latest minor version of API version 4.
type DeliveryServiceRequestResponseV4 = DeliveryServiceRequestResponseV40

// DeliveryServiceRequestResponseV40 is the type of a response from Traffic Ops
// for Delivery Service Requests using API version 4.0.
type DeliveryServiceRequestsResponseV40 struct {
	Response []DeliveryServiceRequestV40 `json:"response"`
	Alerts
}

// DeliveryServiceRequestResponseV4 is the type of a response from Traffic Ops
// for Delivery Service Requests using the latest minor version of API version
// 4.
type DeliveryServiceRequestsResponseV4 = DeliveryServiceRequestsResponseV40
