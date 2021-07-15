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
	"time"
)

// DeliveryServiceServerResponse is the type of a response from Traffic Ops
// to a GET request to the /deliveryserviceserver endpoint.
type DeliveryServiceServerResponse struct {
	Orderby  string                  `json:"orderby"`
	Response []DeliveryServiceServer `json:"response"`
	Size     int                     `json:"size"`
	Limit    int                     `json:"limit"`
	Alerts
}

type DSSMapResponse struct {
	DsId    int   `json:"dsId"`
	Replace bool  `json:"replace"`
	Servers []int `json:"servers"`
}

type DSSReplaceResponse struct {
	Alerts
	Response DSSMapResponse `json:"response"`
}

type DSServersResponse struct {
	Response DeliveryServiceServers `json:"response"`
	Alerts
}

type DeliveryServiceServers struct {
	ServerNames []string `json:"serverNames"`
	XmlId       string   `json:"xmlId"`
}

// DeliveryServiceServer ...
type DeliveryServiceServer struct {
	Server          *int       `json:"server" db:"server"`
	DeliveryService *int       `json:"deliveryService" db:"deliveryservice"`
	LastUpdated     *TimeNoMod `json:"lastUpdated" db:"last_updated"`
}

type Filter int

const (
	Assigned Filter = iota
	Unassigned
	Eligible
)

// DSServerBase contains the base information for a Delivery Service Server.
type DSServerBase struct {
	Cachegroup                  *string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID                *int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID                       *int                 `json:"cdnId" db:"cdn_id"`
	CDNName                     *string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices            *map[string][]string `json:"deliveryServices,omitempty"`
	DomainName                  *string              `json:"domainName" db:"domain_name"`
	FQDN                        *string              `json:"fqdn,omitempty"`
	FqdnTime                    time.Time            `json:"-"`
	GUID                        *string              `json:"guid" db:"guid"`
	HostName                    *string              `json:"hostName" db:"host_name"`
	HTTPSPort                   *int                 `json:"httpsPort" db:"https_port"`
	ID                          *int                 `json:"id" db:"id"`
	ILOIPAddress                *string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway                *string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask                *string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword                 *string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername                 *string              `json:"iloUsername" db:"ilo_username"`
	LastUpdated                 *TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress               *string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway               *string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask               *string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason               *string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation                *string              `json:"physLocation" db:"phys_location"`
	PhysLocationID              *int                 `json:"physLocationId" db:"phys_location_id"`
	Profile                     *string              `json:"profile" db:"profile"`
	ProfileDesc                 *string              `json:"profileDesc" db:"profile_desc"`
	ProfileID                   *int                 `json:"profileId" db:"profile_id"`
	Rack                        *string              `json:"rack" db:"rack"`
	RouterHostName              *string              `json:"routerHostName" db:"router_host_name"`
	RouterPortName              *string              `json:"routerPortName" db:"router_port_name"`
	Status                      *string              `json:"status" db:"status"`
	StatusID                    *int                 `json:"statusId" db:"status_id"`
	TCPPort                     *int                 `json:"tcpPort" db:"tcp_port"`
	Type                        string               `json:"type" db:"server_type"`
	TypeID                      *int                 `json:"typeId" db:"server_type_id"`
	UpdPending                  *bool                `json:"updPending" db:"upd_pending"`
	ServerCapabilities          []string             `json:"-" db:"server_capabilities"`
	DeliveryServiceCapabilities []string             `json:"-" db:"deliveryservice_capabilities"`
}

// DSServerBaseV4 contains the base information for a Delivery Service Server.
type DSServerBaseV4 struct {
	Cachegroup                  *string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID                *int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID                       *int                 `json:"cdnId" db:"cdn_id"`
	CDNName                     *string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices            *map[string][]string `json:"deliveryServices,omitempty"`
	DomainName                  *string              `json:"domainName" db:"domain_name"`
	FQDN                        *string              `json:"fqdn,omitempty"`
	FqdnTime                    time.Time            `json:"-"`
	GUID                        *string              `json:"guid" db:"guid"`
	HostName                    *string              `json:"hostName" db:"host_name"`
	HTTPSPort                   *int                 `json:"httpsPort" db:"https_port"`
	ID                          *int                 `json:"id" db:"id"`
	ILOIPAddress                *string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway                *string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask                *string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword                 *string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername                 *string              `json:"iloUsername" db:"ilo_username"`
	LastUpdated                 *TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress               *string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway               *string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask               *string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason               *string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation                *string              `json:"physLocation" db:"phys_location"`
	PhysLocationID              *int                 `json:"physLocationId" db:"phys_location_id"`
	Profile                     *string              `json:"profile" db:"profile"`
	ProfileDesc                 *string              `json:"profileDesc" db:"profile_desc"`
	ProfileID                   *int                 `json:"profileId" db:"profile_id"`
	Rack                        *string              `json:"rack" db:"rack"`
	Status                      *string              `json:"status" db:"status"`
	StatusID                    *int                 `json:"statusId" db:"status_id"`
	TCPPort                     *int                 `json:"tcpPort" db:"tcp_port"`
	Type                        string               `json:"type" db:"server_type"`
	TypeID                      *int                 `json:"typeId" db:"server_type_id"`
	UpdPending                  *bool                `json:"updPending" db:"upd_pending"`
	ServerCapabilities          []string             `json:"-" db:"server_capabilities"`
	DeliveryServiceCapabilities []string             `json:"-" db:"deliveryservice_capabilities"`
}

// DSServerV11 contains the legacy format for a Delivery Service Server.
type DSServerV11 struct {
	DSServerBase
	LegacyInterfaceDetails
}

// DSServer contains information for a Delivery Service Server.
type DSServer struct {
	DSServerBase
	ServerInterfaces *[]ServerInterfaceInfo `json:"interfaces" db:"interfaces"`
}

type DSServerResponseV30 struct {
	Response []DSServer `json:"response"`
	Alerts
}

// DSServerV4 contains information for a V4.x Delivery Service Server.
type DSServerV4 struct {
	DSServerBaseV4
	ServerInterfaces *[]ServerInterfaceInfoV40 `json:"interfaces" db:"interfaces"`
}

// DSServerResponseV40 is the type of a response from Traffic Ops to a request
// for servers assigned to a Delivery Service - in API version 4.0.
type DSServerResponseV40 struct {
	Response []DSServerV4 `json:"response"`
	Alerts
}

// DSServerResponseV4 is the type of a response from Traffic Ops to a request
// for servers assigned to a Delivery Service - in the latest minor version of
// API version 4.
type DSServerResponseV4 = DSServerResponseV40

func (oldBase DSServerBase) ToDSServerBaseV4() DSServerBaseV4 {
	var dsServerBaseV4 DSServerBaseV4
	dsServerBaseV4.Cachegroup = oldBase.Cachegroup
	dsServerBaseV4.CachegroupID = oldBase.CachegroupID
	dsServerBaseV4.CDNID = oldBase.CDNID
	dsServerBaseV4.CDNName = oldBase.CDNName
	dsServerBaseV4.DeliveryServices = oldBase.DeliveryServices
	dsServerBaseV4.DomainName = oldBase.DomainName
	dsServerBaseV4.FQDN = oldBase.FQDN
	dsServerBaseV4.FqdnTime = oldBase.FqdnTime
	dsServerBaseV4.GUID = oldBase.GUID
	dsServerBaseV4.HostName = oldBase.HostName
	dsServerBaseV4.HTTPSPort = oldBase.HTTPSPort
	dsServerBaseV4.ID = oldBase.ID
	dsServerBaseV4.ILOIPAddress = oldBase.ILOIPAddress
	dsServerBaseV4.ILOIPGateway = oldBase.ILOIPGateway
	dsServerBaseV4.ILOIPNetmask = oldBase.ILOIPNetmask
	dsServerBaseV4.ILOPassword = oldBase.ILOPassword
	dsServerBaseV4.ILOUsername = oldBase.ILOUsername
	dsServerBaseV4.LastUpdated = oldBase.LastUpdated
	dsServerBaseV4.MgmtIPAddress = oldBase.MgmtIPAddress
	dsServerBaseV4.MgmtIPGateway = oldBase.MgmtIPGateway
	dsServerBaseV4.MgmtIPNetmask = oldBase.MgmtIPNetmask
	dsServerBaseV4.OfflineReason = oldBase.OfflineReason
	dsServerBaseV4.PhysLocation = oldBase.PhysLocation
	dsServerBaseV4.PhysLocationID = oldBase.PhysLocationID
	dsServerBaseV4.Profile = oldBase.Profile
	dsServerBaseV4.ProfileDesc = oldBase.ProfileDesc
	dsServerBaseV4.ProfileID = oldBase.ProfileID
	dsServerBaseV4.Rack = oldBase.Rack
	dsServerBaseV4.Status = oldBase.Status
	dsServerBaseV4.StatusID = oldBase.StatusID
	dsServerBaseV4.TCPPort = oldBase.TCPPort
	dsServerBaseV4.Type = oldBase.Type
	dsServerBaseV4.TypeID = oldBase.TypeID
	dsServerBaseV4.UpdPending = oldBase.UpdPending
	dsServerBaseV4.ServerCapabilities = oldBase.ServerCapabilities
	dsServerBaseV4.DeliveryServiceCapabilities = oldBase.DeliveryServiceCapabilities
	return dsServerBaseV4
}

func (baseV4 DSServerBaseV4) ToDSServerBase(routerHostName, routerPort *string) DSServerBase {
	var dsServerBase DSServerBase
	dsServerBase.Cachegroup = baseV4.Cachegroup
	dsServerBase.CachegroupID = baseV4.CachegroupID
	dsServerBase.CDNID = baseV4.CDNID
	dsServerBase.CDNName = baseV4.CDNName
	dsServerBase.DeliveryServices = baseV4.DeliveryServices
	dsServerBase.DomainName = baseV4.DomainName
	dsServerBase.FQDN = baseV4.FQDN
	dsServerBase.FqdnTime = baseV4.FqdnTime
	dsServerBase.GUID = baseV4.GUID
	dsServerBase.HostName = baseV4.HostName
	dsServerBase.HTTPSPort = baseV4.HTTPSPort
	dsServerBase.ID = baseV4.ID
	dsServerBase.ILOIPAddress = baseV4.ILOIPAddress
	dsServerBase.ILOIPGateway = baseV4.ILOIPGateway
	dsServerBase.ILOIPNetmask = baseV4.ILOIPNetmask
	dsServerBase.ILOPassword = baseV4.ILOPassword
	dsServerBase.ILOUsername = baseV4.ILOUsername
	dsServerBase.LastUpdated = baseV4.LastUpdated
	dsServerBase.MgmtIPAddress = baseV4.MgmtIPAddress
	dsServerBase.MgmtIPGateway = baseV4.MgmtIPGateway
	dsServerBase.MgmtIPNetmask = baseV4.MgmtIPNetmask
	dsServerBase.OfflineReason = baseV4.OfflineReason
	dsServerBase.PhysLocation = baseV4.PhysLocation
	dsServerBase.PhysLocationID = baseV4.PhysLocationID
	dsServerBase.Profile = baseV4.Profile
	dsServerBase.ProfileDesc = baseV4.ProfileDesc
	dsServerBase.ProfileID = baseV4.ProfileID
	dsServerBase.Rack = baseV4.Rack
	dsServerBase.Status = baseV4.Status
	dsServerBase.StatusID = baseV4.StatusID
	dsServerBase.TCPPort = baseV4.TCPPort
	dsServerBase.Type = baseV4.Type
	dsServerBase.TypeID = baseV4.TypeID
	dsServerBase.UpdPending = baseV4.UpdPending
	dsServerBase.ServerCapabilities = baseV4.ServerCapabilities
	dsServerBase.DeliveryServiceCapabilities = baseV4.DeliveryServiceCapabilities
	dsServerBase.RouterHostName = routerHostName
	dsServerBase.RouterPortName = routerPort
	return dsServerBase
}
