package tc

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-util"
)

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

// ServersV4Response is the format of a response to a GET request for API v4.x /servers.
type ServersV4Response struct {
	Response []ServerV40 `json:"response"`
	Summary  struct {
		Count uint64 `json:"count"`
	} `json:"summary"`
	Alerts
}

// ServersV3Response is the format of a response to a GET request for /servers.
type ServersV3Response struct {
	Response []ServerV30 `json:"response"`
	Summary  struct {
		Count uint64 `json:"count"`
	} `json:"summary"`
	Alerts
}

// ServersResponse is a list of Servers as a response to an API v2 request.
// This can't change because it will break ORT. Unfortunately.
type ServersResponse struct {
	Response []Server `json:"response"`
	Alerts
}

// ServersDetailResponse is the JSON object returned for a single server.
type ServersDetailResponse struct {
	Response Server `json:"response"`
	Alerts
}

// ServersV1Response is a list of Servers for v1 as a response.
type ServersV1Response struct {
	Response []ServerV1 `json:"response"`
	Alerts
}

type ServerDetailV11 struct {
	ServerDetail
	LegacyInterfaceDetails
}

// ServerDetailV30 is the details for a server for API v3
type ServerDetailV30 struct {
	ServerDetail
	ServerInterfaces *[]ServerInterfaceInfo `json:"interfaces"`
}

// ServerDetailV40 is the details for a server for API v4
type ServerDetailV40 struct {
	ServerDetailBaseV40
	ServerInterfaces *[]ServerInterfaceInfoV40 `json:"interfaces"`
}

// ServersV1DetailResponse is the JSON object returned for a single server for v1.
type ServersV1DetailResponse struct {
	Response []ServerDetailV11 `json:"response"`
	Alerts
}

// ServersV3DetailResponse is the JSON object returned for a single server for v3.
type ServersV3DetailResponse struct {
	Response []ServerDetailV30 `json:"response"`
	Alerts
}

// ServersV4DetailResponse is the JSON object returned for a single server for v4.
type ServersV4DetailResponse struct {
	Response []ServerDetailV40 `json:"response"`
	Alerts
}

// ServerIPAddress is the data associated with a server's interface's IP address.
type ServerIPAddress struct {
	Address        string  `json:"address" db:"address"`
	Gateway        *string `json:"gateway" db:"gateway"`
	ServiceAddress bool    `json:"serviceAddress" db:"service_address"`
}

// ServerInterfaceInfo is the data associated with a server's interface.
type ServerInterfaceInfo struct {
	IPAddresses  []ServerIPAddress `json:"ipAddresses" db:"ip_addresses"`
	MaxBandwidth *uint64           `json:"maxBandwidth" db:"max_bandwidth"`
	Monitor      bool              `json:"monitor" db:"monitor"`
	MTU          *uint64           `json:"mtu" db:"mtu"`
	Name         string            `json:"name" db:"name"`
}

// ServerInterfaceInfoV40 is the data associated with a V40 server's interface.
type ServerInterfaceInfoV40 struct {
	ServerInterfaceInfo
	RouterHostName string `json:"routerHostName" db:"router_host_name"`
	RouterPort     string `json:"routerPort" db:"router_port"`
}

func (detailV40 ServerDetailBaseV40) ToServerDetailFromV4(routerHostName, routerPort *string) ServerDetail {
	var detail ServerDetail
	detail.CacheGroup = detailV40.CacheGroup
	detail.CDNName = detailV40.CDNName
	detail.DeliveryServiceIDs = detailV40.DeliveryServiceIDs
	detail.DomainName = detailV40.DomainName
	detail.GUID = detailV40.GUID
	detail.HardwareInfo = detailV40.HardwareInfo
	detail.HostName = detailV40.HostName
	detail.HTTPSPort = detailV40.HTTPSPort
	detail.ID = detailV40.ID
	detail.ILOIPAddress = detailV40.ILOIPAddress
	detail.ILOIPGateway = detailV40.ILOIPGateway
	detail.ILOIPNetmask = detailV40.ILOIPNetmask
	detail.ILOPassword = detailV40.ILOPassword
	detail.ILOUsername = detailV40.ILOUsername
	detail.MgmtIPAddress = detailV40.MgmtIPAddress
	detail.MgmtIPGateway = detailV40.MgmtIPGateway
	detail.MgmtIPNetmask = detailV40.MgmtIPNetmask
	detail.OfflineReason = detailV40.OfflineReason
	detail.PhysLocation = detailV40.PhysLocation
	detail.Profile = detailV40.Profile
	detail.ProfileDesc = detailV40.ProfileDesc
	detail.Rack = detailV40.Rack
	detail.Status = detailV40.Status
	detail.TCPPort = detailV40.TCPPort
	detail.Type = detailV40.Type
	detail.XMPPID = detailV40.XMPPID
	detail.XMPPPasswd = detailV40.XMPPPasswd
	detail.RouterHostName = routerHostName
	detail.RouterPortName = routerPort
	return detail
}

func (detail ServerDetail) ToV4ServerDetail() ServerDetailBaseV40 {
	var detailV40 ServerDetailBaseV40
	detailV40.CacheGroup = detail.CacheGroup
	detailV40.CDNName = detail.CDNName
	detailV40.DeliveryServiceIDs = detail.DeliveryServiceIDs
	detailV40.DomainName = detail.DomainName
	detailV40.GUID = detail.GUID
	detailV40.HardwareInfo = detail.HardwareInfo
	detailV40.HostName = detail.HostName
	detailV40.HTTPSPort = detail.HTTPSPort
	detailV40.ID = detail.ID
	detailV40.ILOIPAddress = detail.ILOIPAddress
	detailV40.ILOIPGateway = detail.ILOIPGateway
	detailV40.ILOIPNetmask = detail.ILOIPNetmask
	detailV40.ILOPassword = detail.ILOPassword
	detailV40.ILOUsername = detail.ILOUsername
	detailV40.MgmtIPAddress = detail.MgmtIPAddress
	detailV40.MgmtIPGateway = detail.MgmtIPGateway
	detailV40.MgmtIPNetmask = detail.MgmtIPNetmask
	detailV40.OfflineReason = detail.OfflineReason
	detailV40.PhysLocation = detail.PhysLocation
	detailV40.Profile = detail.Profile
	detailV40.ProfileDesc = detail.ProfileDesc
	detailV40.Rack = detail.Rack
	detailV40.Status = detail.Status
	detailV40.TCPPort = detail.TCPPort
	detailV40.Type = detail.Type
	detailV40.XMPPID = detail.XMPPID
	detailV40.XMPPPasswd = detail.XMPPPasswd
	return detailV40
}

// GetDefaultAddress returns the IPv4 and IPv6 service addresses of the interface.
func (i *ServerInterfaceInfo) GetDefaultAddress() (string, string) {
	var ipv4 string
	var ipv6 string
	for _, ip := range i.IPAddresses {
		if ip.ServiceAddress {
			address, _, err := net.ParseCIDR(ip.Address)
			if err != nil || address == nil {
				continue
			}
			if address.To4() != nil {
				ipv4 = ip.Address
			} else if address.To16() != nil {
				ipv6 = ip.Address
			}

			if ipv4 != "" && ipv6 != "" {
				break
			}
		}
	}
	return ipv4, ipv6
}

// Value implements the driver.Valuer interface
// marshals struct to json to pass back as a json.RawMessage
func (sii *ServerInterfaceInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(sii)
	return b, err
}

// Scan implements the sql.Scanner interface
// expects json.RawMessage and unmarshals to a ServerInterfaceInfo struct
func (sii *ServerInterfaceInfo) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected deliveryservice in byte array form; got %T", src)
	}

	return json.Unmarshal([]byte(b), sii)
}

// LegacyInterfaceDetails is the details for interfaces on servers for API v1 and v2.
type LegacyInterfaceDetails struct {
	InterfaceMtu  *int    `json:"interfaceMtu" db:"interface_mtu"`
	InterfaceName *string `json:"interfaceName" db:"interface_name"`
	IP6Address    *string `json:"ip6Address" db:"ip6_address"`
	IP6Gateway    *string `json:"ip6Gateway" db:"ip6_gateway"`
	IPAddress     *string `json:"ipAddress" db:"ip_address"`
	IPGateway     *string `json:"ipGateway" db:"ip_gateway"`
	IPNetmask     *string `json:"ipNetmask" db:"ip_netmask"`
}

// ToInterfaces converts a LegacyInterfaceDetails to a slice of
// ServerInterfaceInfo structures. No interfaces will be marked for monitoring,
// and it will generate service addresses according to the passed indicators
// for each address family.
func (lid *LegacyInterfaceDetails) ToInterfaces(ipv4IsService, ipv6IsService bool) ([]ServerInterfaceInfo, error) {
	var iface ServerInterfaceInfo
	if lid.InterfaceMtu == nil {
		return nil, errors.New("interfaceMtu is null")
	}
	mtu := uint64(*lid.InterfaceMtu)
	iface.MTU = &mtu

	if lid.InterfaceName == nil {
		return nil, errors.New("interfaceName is null")
	}
	iface.Name = *lid.InterfaceName

	var ips []ServerIPAddress
	if lid.IPAddress != nil && *lid.IPAddress != "" {
		if lid.IPGateway != nil && *lid.IPGateway == "" {
			lid.IPGateway = nil
		}

		ipStr := *lid.IPAddress
		if lid.IPNetmask != nil && *lid.IPNetmask != "" {
			mask := net.ParseIP(*lid.IPNetmask).To4()
			if mask == nil {
				return nil, fmt.Errorf("Failed to parse netmask '%s'", *lid.IPNetmask)
			}
			cidr, _ := net.IPv4Mask(mask[0], mask[1], mask[2], mask[3]).Size()
			ipStr = fmt.Sprintf("%s/%d", ipStr, cidr)
		}

		ips = append(ips, ServerIPAddress{
			Address:        ipStr,
			Gateway:        lid.IPGateway,
			ServiceAddress: ipv4IsService,
		})
	}

	if lid.IP6Address != nil && *lid.IP6Address != "" {
		if lid.IP6Gateway != nil && *lid.IP6Gateway == "" {
			lid.IP6Gateway = nil
		}
		ips = append(ips, ServerIPAddress{
			Address:        *lid.IP6Address,
			Gateway:        lid.IP6Gateway,
			ServiceAddress: ipv6IsService,
		})
	}

	iface.IPAddresses = ips
	return []ServerInterfaceInfo{iface}, nil
}

func ToInterfacesV4(oldInterfaces []ServerInterfaceInfo, routerName, routerPort *string) ([]ServerInterfaceInfoV40, error) {
	v4Interfaces := make([]ServerInterfaceInfoV40, 0)
	var v4Int ServerInterfaceInfoV40
	for _, i := range oldInterfaces {
		v4Int.ServerInterfaceInfo = i
		if routerName != nil {
			v4Int.RouterHostName = *routerName
		}
		if routerPort != nil {
			v4Int.RouterPort = *routerPort
		}
		v4Interfaces = append(v4Interfaces, v4Int)
	}
	return v4Interfaces, nil
}

func (lid *LegacyInterfaceDetails) ToInterfacesV4(ipv4IsService, ipv6IsService bool, routerName, routerPort *string) ([]ServerInterfaceInfoV40, error) {
	var iface ServerInterfaceInfoV40
	if lid.InterfaceMtu == nil {
		return nil, errors.New("interfaceMtu is null")
	}
	mtu := uint64(*lid.InterfaceMtu)
	iface.MTU = &mtu

	if lid.InterfaceName == nil {
		return nil, errors.New("interfaceName is null")
	}
	iface.Name = *lid.InterfaceName

	var ips []ServerIPAddress
	if lid.IPAddress != nil && *lid.IPAddress != "" {
		if lid.IPGateway != nil && *lid.IPGateway == "" {
			lid.IPGateway = nil
		}

		ipStr := *lid.IPAddress
		if lid.IPNetmask != nil && *lid.IPNetmask != "" {
			mask := net.ParseIP(*lid.IPNetmask).To4()
			if mask == nil {
				return nil, fmt.Errorf("Failed to parse netmask '%s'", *lid.IPNetmask)
			}
			cidr, _ := net.IPv4Mask(mask[0], mask[1], mask[2], mask[3]).Size()
			ipStr = fmt.Sprintf("%s/%d", ipStr, cidr)
		}

		ips = append(ips, ServerIPAddress{
			Address:        ipStr,
			Gateway:        lid.IPGateway,
			ServiceAddress: ipv4IsService,
		})
	}

	if lid.IP6Address != nil && *lid.IP6Address != "" {
		if lid.IP6Gateway != nil && *lid.IP6Gateway == "" {
			lid.IP6Gateway = nil
		}
		ips = append(ips, ServerIPAddress{
			Address:        *lid.IP6Address,
			Gateway:        lid.IP6Gateway,
			ServiceAddress: ipv6IsService,
		})
	}

	iface.IPAddresses = ips
	if routerName != nil {
		iface.RouterHostName = *routerName
	}
	if routerPort != nil {
		iface.RouterPort = *routerPort
	}
	return []ServerInterfaceInfoV40{iface}, nil
}

// String implements the fmt.Stringer interface.
func (lid LegacyInterfaceDetails) String() string {
	var b strings.Builder
	b.Write([]byte("LegacyInterfaceDetails(InterfaceMtu="))

	if lid.InterfaceMtu == nil {
		b.Write([]byte("nil"))
	} else {
		b.WriteString(strconv.FormatInt(int64(*lid.InterfaceMtu), 10))
	}

	b.Write([]byte(", InterfaceName="))
	if lid.InterfaceName != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.InterfaceName)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.Write([]byte(", IP6Address="))
	if lid.IP6Address != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.IP6Address)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.Write([]byte(", IP6Gateway="))
	if lid.IP6Gateway != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.IP6Gateway)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.Write([]byte(", IPAddress="))
	if lid.IPAddress != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.IPAddress)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.Write([]byte(", IPGateway="))
	if lid.IPGateway != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.IPGateway)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.Write([]byte(", IPNetmask="))
	if lid.IPNetmask != nil {
		b.WriteRune('\'')
		b.WriteString(*lid.IPNetmask)
		b.WriteRune('\'')
	} else {
		b.Write([]byte("nil"))
	}

	b.WriteRune(')')

	return b.String()
}

func V4InterfaceInfoToV3Interfaces(serverInterfaces []ServerInterfaceInfoV40) ([]ServerInterfaceInfo, error) {
	var interfaces []ServerInterfaceInfo

	for _, intFace := range serverInterfaces {
		var interfaceV3 ServerInterfaceInfo
		interfaceV3.IPAddresses = intFace.IPAddresses
		interfaceV3.Monitor = intFace.Monitor
		interfaceV3.MaxBandwidth = intFace.MaxBandwidth
		interfaceV3.MTU = intFace.MTU
		interfaceV3.Name = intFace.Name
		interfaces = append(interfaces, interfaceV3)
	}

	return interfaces, nil
}

func V4InterfaceInfoToLegacyInterfaces(serverInterfaces []ServerInterfaceInfoV40) (LegacyInterfaceDetails, error) {
	var legacyDetails LegacyInterfaceDetails

	for _, intFace := range serverInterfaces {

		foundServiceInterface := false

		for _, addr := range intFace.IPAddresses {
			if !addr.ServiceAddress {
				continue
			}

			foundServiceInterface = true

			address := addr.Address
			gateway := addr.Gateway

			var parsedIp net.IP
			var mask *net.IPNet
			var err error
			parsedIp, mask, err = net.ParseCIDR(address)
			if err != nil {
				parsedIp = net.ParseIP(address)
				if parsedIp == nil {
					return legacyDetails, fmt.Errorf("Failed to parse '%s' as network or CIDR string: %v", address, err)
				}
			}

			if parsedIp.To4() == nil {
				legacyDetails.IP6Address = &address
				legacyDetails.IP6Gateway = gateway
			} else if mask != nil {
				legacyDetails.IPAddress = util.StrPtr(parsedIp.String())
				legacyDetails.IPGateway = gateway
				legacyDetails.IPNetmask = util.StrPtr(fmt.Sprintf("%d.%d.%d.%d", mask.Mask[0], mask.Mask[1], mask.Mask[2], mask.Mask[3]))
			} else {
				legacyDetails.IPAddress = util.StrPtr(parsedIp.String())
				legacyDetails.IPGateway = gateway
				legacyDetails.IPNetmask = new(string)
			}

			if intFace.MTU != nil {
				legacyDetails.InterfaceMtu = util.IntPtr(int(*intFace.MTU))
			}

			// This should no longer matter now that short-circuiting is better,
			// but this temporary variable is necessary because the 'intFace'
			// variable is referential, so taking '&intFace.Name' would cause
			// problems when intFace is reassigned.
			name := intFace.Name
			legacyDetails.InterfaceName = &name

			// we can jump out here since servers can only legally have one
			// IPv4 and one IPv6 service address
			if legacyDetails.IPAddress != nil && *legacyDetails.IPAddress != "" && legacyDetails.IP6Address != nil && *legacyDetails.IP6Address != "" {
				break
			}
		}

		if foundServiceInterface {
			return legacyDetails, nil
		}
	}

	return legacyDetails, errors.New("no service addresses found")
}

// InterfaceInfoToLegacyInterfaces converts a ServerInterfaceInfo to an
// equivalent LegacyInterfaceDetails structure. It does this by creating the
// IP address fields using the "service" interface's IP addresses. All others
// are discarded, as the legacy format is incapable of representing them.
func InterfaceInfoToLegacyInterfaces(serverInterfaces []ServerInterfaceInfo) (LegacyInterfaceDetails, error) {
	var legacyDetails LegacyInterfaceDetails

	for _, intFace := range serverInterfaces {

		foundServiceInterface := false

		for _, addr := range intFace.IPAddresses {
			if !addr.ServiceAddress {
				continue
			}

			foundServiceInterface = true

			address := addr.Address
			gateway := addr.Gateway

			var parsedIp net.IP
			var mask *net.IPNet
			var err error
			parsedIp, mask, err = net.ParseCIDR(address)
			if err != nil {
				parsedIp = net.ParseIP(address)
				if parsedIp == nil {
					return legacyDetails, fmt.Errorf("Failed to parse '%s' as network or CIDR string: %v", address, err)
				}
			}

			if parsedIp.To4() == nil {
				legacyDetails.IP6Address = &address
				legacyDetails.IP6Gateway = gateway
			} else if mask != nil {
				legacyDetails.IPAddress = util.StrPtr(parsedIp.String())
				legacyDetails.IPGateway = gateway
				legacyDetails.IPNetmask = util.StrPtr(fmt.Sprintf("%d.%d.%d.%d", mask.Mask[0], mask.Mask[1], mask.Mask[2], mask.Mask[3]))
			} else {
				legacyDetails.IPAddress = util.StrPtr(parsedIp.String())
				legacyDetails.IPGateway = gateway
				legacyDetails.IPNetmask = new(string)
			}

			if intFace.MTU != nil {
				legacyDetails.InterfaceMtu = util.IntPtr(int(*intFace.MTU))
			}

			// This should no longer matter now that short-circuiting is better,
			// but this temporary variable is necessary because the 'intFace'
			// variable is referential, so taking '&intFace.Name' would cause
			// problems when intFace is reassigned.
			name := intFace.Name
			legacyDetails.InterfaceName = &name

			// we can jump out here since servers can only legally have one
			// IPv4 and one IPv6 service address
			if legacyDetails.IPAddress != nil && *legacyDetails.IPAddress != "" && legacyDetails.IP6Address != nil && *legacyDetails.IP6Address != "" {
				break
			}
		}

		if foundServiceInterface {
			return legacyDetails, nil
		}
	}

	return legacyDetails, errors.New("no service addresses found")
}

// Server is a non-"nullable" representation of a Server as it appeared in API
// version 2.0
// Deprecated: Please use versioned and nullable structures from now on.
type Server struct {
	Cachegroup       string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID     int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID            int                 `json:"cdnId" db:"cdn_id"`
	CDNName          string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices map[string][]string `json:"deliveryServices,omitempty"`
	DomainName       string              `json:"domainName" db:"domain_name"`
	FQDN             *string             `json:"fqdn,omitempty"`
	FqdnTime         time.Time           `json:"-"`
	GUID             string              `json:"guid" db:"guid"`
	HostName         string              `json:"hostName" db:"host_name"`
	HTTPSPort        int                 `json:"httpsPort" db:"https_port"`
	ID               int                 `json:"id" db:"id"`
	ILOIPAddress     string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway     string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask     string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword      string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername      string              `json:"iloUsername" db:"ilo_username"`
	InterfaceMtu     int                 `json:"interfaceMtu" db:"interface_mtu"`
	InterfaceName    string              `json:"interfaceName" db:"interface_name"`
	IP6Address       string              `json:"ip6Address" db:"ip6_address"`
	IP6IsService     bool                `json:"ip6IsService" db:"ip6_address_is_service"`
	IP6Gateway       string              `json:"ip6Gateway" db:"ip6_gateway"`
	IPAddress        string              `json:"ipAddress" db:"ip_address"`
	IPIsService      bool                `json:"ipIsService" db:"ip_address_is_service"`
	IPGateway        string              `json:"ipGateway" db:"ip_gateway"`
	IPNetmask        string              `json:"ipNetmask" db:"ip_netmask"`
	LastUpdated      TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress    string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway    string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask    string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason    string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation     string              `json:"physLocation" db:"phys_location"`
	PhysLocationID   int                 `json:"physLocationId" db:"phys_location_id"`
	Profile          string              `json:"profile" db:"profile"`
	ProfileDesc      string              `json:"profileDesc" db:"profile_desc"`
	ProfileID        int                 `json:"profileId" db:"profile_id"`
	Rack             string              `json:"rack" db:"rack"`
	RevalPending     bool                `json:"revalPending" db:"reval_pending"`
	RouterHostName   string              `json:"routerHostName" db:"router_host_name"`
	RouterPortName   string              `json:"routerPortName" db:"router_port_name"`
	Status           string              `json:"status" db:"status"`
	StatusID         int                 `json:"statusId" db:"status_id"`
	TCPPort          int                 `json:"tcpPort" db:"tcp_port"`
	Type             string              `json:"type" db:"server_type"`
	TypeID           int                 `json:"typeId" db:"server_type_id"`
	UpdPending       bool                `json:"updPending" db:"upd_pending"`
	XMPPID           string              `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd       string              `json:"xmppPasswd" db:"xmpp_passwd"`
}

type ServerV1 struct {
	Cachegroup       string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID     int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID            int                 `json:"cdnId" db:"cdn_id"`
	CDNName          string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices map[string][]string `json:"deliveryServices,omitempty"`
	DomainName       string              `json:"domainName" db:"domain_name"`
	FQDN             *string             `json:"fqdn,omitempty"`
	FqdnTime         time.Time           `json:"-"`
	GUID             string              `json:"guid" db:"guid"`
	HostName         string              `json:"hostName" db:"host_name"`
	HTTPSPort        int                 `json:"httpsPort" db:"https_port"`
	ID               int                 `json:"id" db:"id"`
	ILOIPAddress     string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway     string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask     string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword      string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername      string              `json:"iloUsername" db:"ilo_username"`
	InterfaceMtu     int                 `json:"interfaceMtu" db:"interface_mtu"`
	InterfaceName    string              `json:"interfaceName" db:"interface_name"`
	IP6Address       string              `json:"ip6Address" db:"ip6_address"`
	IP6Gateway       string              `json:"ip6Gateway" db:"ip6_gateway"`
	IPAddress        string              `json:"ipAddress" db:"ip_address"`
	IPGateway        string              `json:"ipGateway" db:"ip_gateway"`
	IPNetmask        string              `json:"ipNetmask" db:"ip_netmask"`
	LastUpdated      TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress    string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway    string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask    string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason    string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation     string              `json:"physLocation" db:"phys_location"`
	PhysLocationID   int                 `json:"physLocationId" db:"phys_location_id"`
	Profile          string              `json:"profile" db:"profile"`
	ProfileDesc      string              `json:"profileDesc" db:"profile_desc"`
	ProfileID        int                 `json:"profileId" db:"profile_id"`
	Rack             string              `json:"rack" db:"rack"`
	RevalPending     bool                `json:"revalPending" db:"reval_pending"`
	RouterHostName   string              `json:"routerHostName" db:"router_host_name"`
	RouterPortName   string              `json:"routerPortName" db:"router_port_name"`
	Status           string              `json:"status" db:"status"`
	StatusID         int                 `json:"statusId" db:"status_id"`
	TCPPort          int                 `json:"tcpPort" db:"tcp_port"`
	Type             string              `json:"type" db:"server_type"`
	TypeID           int                 `json:"typeId" db:"server_type_id"`
	UpdPending       bool                `json:"updPending" db:"upd_pending"`
	XMPPID           string              `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd       string              `json:"xmppPasswd" db:"xmpp_passwd"`
}

// CommonServerProperties is just the collection of properties which are
// shared by all servers across API versions.
type CommonServerProperties struct {
	Cachegroup       *string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID     *int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID            *int                 `json:"cdnId" db:"cdn_id"`
	CDNName          *string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices *map[string][]string `json:"deliveryServices,omitempty"`
	DomainName       *string              `json:"domainName" db:"domain_name"`
	FQDN             *string              `json:"fqdn,omitempty"`
	FqdnTime         time.Time            `json:"-"`
	GUID             *string              `json:"guid" db:"guid"`
	HostName         *string              `json:"hostName" db:"host_name"`
	HTTPSPort        *int                 `json:"httpsPort" db:"https_port"`
	ID               *int                 `json:"id" db:"id"`
	ILOIPAddress     *string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway     *string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask     *string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword      *string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername      *string              `json:"iloUsername" db:"ilo_username"`
	LastUpdated      *TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress    *string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway    *string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask    *string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason    *string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation     *string              `json:"physLocation" db:"phys_location"`
	PhysLocationID   *int                 `json:"physLocationId" db:"phys_location_id"`
	Profile          *string              `json:"profile" db:"profile"`
	ProfileDesc      *string              `json:"profileDesc" db:"profile_desc"`
	ProfileID        *int                 `json:"profileId" db:"profile_id"`
	Rack             *string              `json:"rack" db:"rack"`
	RevalPending     *bool                `json:"revalPending" db:"reval_pending"`
	RouterHostName   *string              `json:"routerHostName" db:"router_host_name"`
	RouterPortName   *string              `json:"routerPortName" db:"router_port_name"`
	Status           *string              `json:"status" db:"status"`
	StatusID         *int                 `json:"statusId" db:"status_id"`
	TCPPort          *int                 `json:"tcpPort" db:"tcp_port"`
	Type             string               `json:"type" db:"server_type"`
	TypeID           *int                 `json:"typeId" db:"server_type_id"`
	UpdPending       *bool                `json:"updPending" db:"upd_pending"`
	XMPPID           *string              `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd       *string              `json:"xmppPasswd" db:"xmpp_passwd"`
}

// CommonServerPropertiesV40 is just the collection of properties which are
// shared by all servers across API versions >= 4.0.
type CommonServerPropertiesV40 struct {
	Cachegroup       *string              `json:"cachegroup" db:"cachegroup"`
	CachegroupID     *int                 `json:"cachegroupId" db:"cachegroup_id"`
	CDNID            *int                 `json:"cdnId" db:"cdn_id"`
	CDNName          *string              `json:"cdnName" db:"cdn_name"`
	DeliveryServices *map[string][]string `json:"deliveryServices,omitempty"`
	DomainName       *string              `json:"domainName" db:"domain_name"`
	FQDN             *string              `json:"fqdn,omitempty"`
	FqdnTime         time.Time            `json:"-"`
	GUID             *string              `json:"guid" db:"guid"`
	HostName         *string              `json:"hostName" db:"host_name"`
	HTTPSPort        *int                 `json:"httpsPort" db:"https_port"`
	ID               *int                 `json:"id" db:"id"`
	ILOIPAddress     *string              `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway     *string              `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask     *string              `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword      *string              `json:"iloPassword" db:"ilo_password"`
	ILOUsername      *string              `json:"iloUsername" db:"ilo_username"`
	LastUpdated      *TimeNoMod           `json:"lastUpdated" db:"last_updated"`
	MgmtIPAddress    *string              `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway    *string              `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask    *string              `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason    *string              `json:"offlineReason" db:"offline_reason"`
	PhysLocation     *string              `json:"physLocation" db:"phys_location"`
	PhysLocationID   *int                 `json:"physLocationId" db:"phys_location_id"`
	Profile          *string              `json:"profile" db:"profile"`
	ProfileDesc      *string              `json:"profileDesc" db:"profile_desc"`
	ProfileID        *int                 `json:"profileId" db:"profile_id"`
	Rack             *string              `json:"rack" db:"rack"`
	RevalPending     *bool                `json:"revalPending" db:"reval_pending"`
	Status           *string              `json:"status" db:"status"`
	StatusID         *int                 `json:"statusId" db:"status_id"`
	TCPPort          *int                 `json:"tcpPort" db:"tcp_port"`
	Type             string               `json:"type" db:"server_type"`
	TypeID           *int                 `json:"typeId" db:"server_type_id"`
	UpdPending       *bool                `json:"updPending" db:"upd_pending"`
	XMPPID           *string              `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd       *string              `json:"xmppPasswd" db:"xmpp_passwd"`
}

// ServerNullableV11 is a server as it appeared in API version 1.1.
type ServerNullableV11 struct {
	LegacyInterfaceDetails
	CommonServerProperties
}

// ServerNullableV2 is a server as it appeared in API v2.
type ServerNullableV2 struct {
	ServerNullableV11
	IPIsService  *bool `json:"ipIsService" db:"ip_address_is_service"`
	IP6IsService *bool `json:"ip6IsService" db:"ip6_address_is_service"`
}

// ToNullable converts the Server to an equivalent, "nullable" structure.
//
// Note that "zero" values (e.g. the empty string "") are NOT coerced to actual
// null values. In particular, the only fields that will possibly be nil are
// FQDN - if the original server had a nil FQDN - and DeliveryServices - which
// will actually be a pointer to a nil map if the original server had a nil
// DeliveryServices map.
// Further note that this makes "shallow" copies of member properties; if
// reference types (map, slice, pointer etc.) are altered on the original after
// conversion, the changes WILL affect the nullable copy.
func (s Server) ToNullable() ServerNullableV2 {
	return ServerNullableV2{
		ServerNullableV11: ServerNullableV11{
			CommonServerProperties: CommonServerProperties{
				Cachegroup:       &s.Cachegroup,
				CachegroupID:     &s.CachegroupID,
				CDNID:            &s.CDNID,
				CDNName:          &s.CDNName,
				DeliveryServices: &s.DeliveryServices,
				DomainName:       &s.DomainName,
				FQDN:             s.FQDN,
				FqdnTime:         s.FqdnTime,
				GUID:             &s.GUID,
				HostName:         &s.HostName,
				HTTPSPort:        &s.HTTPSPort,
				ID:               &s.ID,
				ILOIPAddress:     &s.ILOIPAddress,
				ILOIPGateway:     &s.ILOIPGateway,
				ILOIPNetmask:     &s.ILOIPNetmask,
				ILOPassword:      &s.ILOPassword,
				ILOUsername:      &s.ILOUsername,
				LastUpdated:      &s.LastUpdated,
				MgmtIPAddress:    &s.MgmtIPAddress,
				MgmtIPGateway:    &s.MgmtIPGateway,
				MgmtIPNetmask:    &s.MgmtIPNetmask,
				OfflineReason:    &s.OfflineReason,
				PhysLocation:     &s.PhysLocation,
				PhysLocationID:   &s.PhysLocationID,
				Profile:          &s.Profile,
				ProfileDesc:      &s.ProfileDesc,
				ProfileID:        &s.ProfileID,
				Rack:             &s.Rack,
				RevalPending:     &s.RevalPending,
				RouterHostName:   &s.RouterHostName,
				RouterPortName:   &s.RouterPortName,
				Status:           &s.Status,
				StatusID:         &s.StatusID,
				TCPPort:          &s.TCPPort,
				Type:             s.Type,
				TypeID:           &s.TypeID,
				UpdPending:       &s.UpdPending,
				XMPPID:           &s.XMPPID,
				XMPPPasswd:       &s.XMPPPasswd,
			},
			LegacyInterfaceDetails: LegacyInterfaceDetails{
				InterfaceMtu:  &s.InterfaceMtu,
				InterfaceName: &s.InterfaceName,
				IPAddress:     &s.IPAddress,
				IPGateway:     &s.IPGateway,
				IPNetmask:     &s.IPNetmask,
				IP6Address:    &s.IP6Address,
				IP6Gateway:    &s.IP6Gateway,
			},
		},
		IPIsService:  &s.IPIsService,
		IP6IsService: &s.IP6IsService,
	}
}

func (s ServerV1) ToNullable() ServerNullableV11 {
	return ServerNullableV11{
		CommonServerProperties: CommonServerProperties{
			Cachegroup:       &s.Cachegroup,
			CachegroupID:     &s.CachegroupID,
			CDNID:            &s.CDNID,
			CDNName:          &s.CDNName,
			DeliveryServices: &s.DeliveryServices,
			DomainName:       &s.DomainName,
			FQDN:             s.FQDN,
			FqdnTime:         s.FqdnTime,
			GUID:             &s.GUID,
			HostName:         &s.HostName,
			HTTPSPort:        &s.HTTPSPort,
			ID:               &s.ID,
			ILOIPAddress:     &s.ILOIPAddress,
			ILOIPGateway:     &s.ILOIPGateway,
			ILOIPNetmask:     &s.ILOIPNetmask,
			ILOPassword:      &s.ILOPassword,
			ILOUsername:      &s.ILOUsername,
			LastUpdated:      &s.LastUpdated,
			MgmtIPAddress:    &s.MgmtIPAddress,
			MgmtIPGateway:    &s.MgmtIPGateway,
			MgmtIPNetmask:    &s.MgmtIPNetmask,
			OfflineReason:    &s.OfflineReason,
			PhysLocation:     &s.PhysLocation,
			PhysLocationID:   &s.PhysLocationID,
			Profile:          &s.Profile,
			ProfileDesc:      &s.ProfileDesc,
			ProfileID:        &s.ProfileID,
			Rack:             &s.Rack,
			RevalPending:     &s.RevalPending,
			RouterHostName:   &s.RouterHostName,
			RouterPortName:   &s.RouterPortName,
			Status:           &s.Status,
			StatusID:         &s.StatusID,
			TCPPort:          &s.TCPPort,
			Type:             s.Type,
			TypeID:           &s.TypeID,
			UpdPending:       &s.UpdPending,
			XMPPID:           &s.XMPPID,
			XMPPPasswd:       &s.XMPPPasswd,
		},
		LegacyInterfaceDetails: LegacyInterfaceDetails{
			InterfaceMtu:  &s.InterfaceMtu,
			InterfaceName: &s.InterfaceName,
			IPAddress:     &s.IPAddress,
			IPGateway:     &s.IPGateway,
			IPNetmask:     &s.IPNetmask,
			IP6Address:    &s.IP6Address,
			IP6Gateway:    &s.IP6Gateway,
		},
	}
}

func coerceBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
func coerceInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func coerceString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ToNonNullable converts the ServerNullableV2 safely to a Server structure.
func (s ServerNullableV2) ToNonNullable() Server {
	ret := Server{
		Cachegroup:     coerceString(s.Cachegroup),
		CachegroupID:   coerceInt(s.CachegroupID),
		CDNID:          coerceInt((s.CDNID)),
		CDNName:        coerceString(s.CDNName),
		DomainName:     coerceString(s.DomainName),
		FQDN:           s.FQDN,
		FqdnTime:       s.FqdnTime,
		GUID:           coerceString(s.GUID),
		HostName:       coerceString(s.HostName),
		HTTPSPort:      coerceInt(s.HTTPSPort),
		ID:             coerceInt(s.ID),
		ILOIPAddress:   coerceString(s.ILOIPAddress),
		ILOIPGateway:   coerceString(s.ILOIPGateway),
		ILOIPNetmask:   coerceString(s.ILOIPNetmask),
		ILOPassword:    coerceString(s.ILOPassword),
		ILOUsername:    coerceString(s.ILOUsername),
		InterfaceMtu:   coerceInt(s.InterfaceMtu),
		InterfaceName:  coerceString(s.InterfaceName),
		IP6Address:     coerceString(s.IP6Address),
		IP6IsService:   coerceBool(s.IP6IsService),
		IP6Gateway:     coerceString(s.IP6Gateway),
		IPAddress:      coerceString(s.IPAddress),
		IPIsService:    coerceBool(s.IPIsService),
		IPGateway:      coerceString(s.IPGateway),
		IPNetmask:      coerceString(s.IPNetmask),
		MgmtIPAddress:  coerceString(s.MgmtIPAddress),
		MgmtIPGateway:  coerceString(s.MgmtIPGateway),
		MgmtIPNetmask:  coerceString(s.MgmtIPNetmask),
		OfflineReason:  coerceString(s.OfflineReason),
		PhysLocation:   coerceString(s.PhysLocation),
		PhysLocationID: coerceInt(s.PhysLocationID),
		Profile:        coerceString(s.Profile),
		ProfileDesc:    coerceString(s.ProfileDesc),
		ProfileID:      coerceInt(s.ProfileID),
		Rack:           coerceString(s.Rack),
		RevalPending:   coerceBool(s.RevalPending),
		RouterHostName: coerceString(s.RouterHostName),
		RouterPortName: coerceString(s.RouterPortName),
		Status:         coerceString(s.Status),
		StatusID:       coerceInt(s.StatusID),
		TCPPort:        coerceInt(s.TCPPort),
		Type:           s.Type,
		TypeID:         coerceInt(s.TypeID),
		UpdPending:     coerceBool(s.UpdPending),
		XMPPID:         coerceString(s.XMPPID),
		XMPPPasswd:     coerceString(s.XMPPPasswd),
	}

	if s.DeliveryServices == nil {
		ret.DeliveryServices = nil
	} else {
		ret.DeliveryServices = *s.DeliveryServices
	}

	if s.LastUpdated == nil {
		ret.LastUpdated = TimeNoMod{}
	} else {
		ret.LastUpdated = *s.LastUpdated
	}

	return ret
}

// Upgrade upgrades the ServerNullableV2 to the new ServerNullable structure.
//
// Note that this makes "shallow" copies of all underlying data, so changes to
// the original will affect the upgraded copy.
func (s ServerNullableV2) Upgrade() (ServerV30, error) {
	ipv4IsService := false
	if s.IPIsService != nil {
		ipv4IsService = *s.IPIsService
	}
	ipv6IsService := false
	if s.IP6IsService != nil {
		ipv6IsService = *s.IP6IsService
	}

	upgraded := ServerV30{
		CommonServerProperties: s.CommonServerProperties,
	}

	infs, err := s.LegacyInterfaceDetails.ToInterfaces(ipv4IsService, ipv6IsService)
	if err != nil {
		return upgraded, err
	}
	upgraded.Interfaces = infs
	return upgraded, nil
}

func (s ServerV30) UpgradeToV40() (ServerV40, error) {
	common := CommonServerPropertiesV40{
		Cachegroup:       s.CommonServerProperties.Cachegroup,
		CachegroupID:     s.CommonServerProperties.CachegroupID,
		CDNID:            s.CommonServerProperties.CDNID,
		CDNName:          s.CommonServerProperties.CDNName,
		DeliveryServices: s.CommonServerProperties.DeliveryServices,
		DomainName:       s.CommonServerProperties.DomainName,
		FQDN:             s.CommonServerProperties.FQDN,
		FqdnTime:         s.CommonServerProperties.FqdnTime,
		GUID:             s.CommonServerProperties.GUID,
		HostName:         s.CommonServerProperties.HostName,
		HTTPSPort:        s.CommonServerProperties.HTTPSPort,
		ID:               s.CommonServerProperties.ID,
		ILOIPAddress:     s.CommonServerProperties.ILOIPAddress,
		ILOIPGateway:     s.CommonServerProperties.ILOIPGateway,
		ILOIPNetmask:     s.CommonServerProperties.ILOIPNetmask,
		ILOPassword:      s.CommonServerProperties.ILOPassword,
		ILOUsername:      s.CommonServerProperties.ILOUsername,
		LastUpdated:      s.CommonServerProperties.LastUpdated,
		MgmtIPAddress:    s.CommonServerProperties.MgmtIPAddress,
		MgmtIPGateway:    s.CommonServerProperties.MgmtIPGateway,
		MgmtIPNetmask:    s.CommonServerProperties.MgmtIPNetmask,
		OfflineReason:    s.CommonServerProperties.OfflineReason,
		PhysLocation:     s.CommonServerProperties.PhysLocation,
		PhysLocationID:   s.CommonServerProperties.PhysLocationID,
		Profile:          s.CommonServerProperties.Profile,
		ProfileDesc:      s.CommonServerProperties.ProfileDesc,
		ProfileID:        s.CommonServerProperties.ProfileID,
		Rack:             s.CommonServerProperties.Rack,
		RevalPending:     s.CommonServerProperties.RevalPending,
		Status:           s.CommonServerProperties.Status,
		StatusID:         s.CommonServerProperties.StatusID,
		TCPPort:          s.CommonServerProperties.TCPPort,
		Type:             s.CommonServerProperties.Type,
		TypeID:           s.CommonServerProperties.TypeID,
		UpdPending:       s.CommonServerProperties.UpdPending,
		XMPPID:           s.CommonServerProperties.XMPPID,
		XMPPPasswd:       s.CommonServerProperties.XMPPPasswd,
	}
	upgraded := ServerV40{
		CommonServerPropertiesV40: common,
		StatusLastUpdated:         s.StatusLastUpdated,
	}
	infs, err := ToInterfacesV4(s.Interfaces, s.RouterHostName, s.RouterPortName)
	if err != nil {
		return upgraded, err
	}
	upgraded.Interfaces = infs
	return upgraded, nil
}

func (s ServerNullableV2) UpgradeToV40() (ServerV40, error) {
	ipv4IsService := false
	if s.IPIsService != nil {
		ipv4IsService = *s.IPIsService
	}
	ipv6IsService := false
	if s.IP6IsService != nil {
		ipv6IsService = *s.IP6IsService
	}
	common := CommonServerPropertiesV40{
		Cachegroup:       s.CommonServerProperties.Cachegroup,
		CachegroupID:     s.CommonServerProperties.CachegroupID,
		CDNID:            s.CommonServerProperties.CDNID,
		CDNName:          s.CommonServerProperties.CDNName,
		DeliveryServices: s.CommonServerProperties.DeliveryServices,
		DomainName:       s.CommonServerProperties.DomainName,
		FQDN:             s.CommonServerProperties.FQDN,
		FqdnTime:         s.CommonServerProperties.FqdnTime,
		GUID:             s.CommonServerProperties.GUID,
		HostName:         s.CommonServerProperties.HostName,
		HTTPSPort:        s.CommonServerProperties.HTTPSPort,
		ID:               s.CommonServerProperties.ID,
		ILOIPAddress:     s.CommonServerProperties.ILOIPAddress,
		ILOIPGateway:     s.CommonServerProperties.ILOIPGateway,
		ILOIPNetmask:     s.CommonServerProperties.ILOIPNetmask,
		ILOPassword:      s.CommonServerProperties.ILOPassword,
		ILOUsername:      s.CommonServerProperties.ILOUsername,
		LastUpdated:      s.CommonServerProperties.LastUpdated,
		MgmtIPAddress:    s.CommonServerProperties.MgmtIPAddress,
		MgmtIPGateway:    s.CommonServerProperties.MgmtIPGateway,
		MgmtIPNetmask:    s.CommonServerProperties.MgmtIPNetmask,
		OfflineReason:    s.CommonServerProperties.OfflineReason,
		PhysLocation:     s.CommonServerProperties.PhysLocation,
		PhysLocationID:   s.CommonServerProperties.PhysLocationID,
		Profile:          s.CommonServerProperties.Profile,
		ProfileDesc:      s.CommonServerProperties.ProfileDesc,
		ProfileID:        s.CommonServerProperties.ProfileID,
		Rack:             s.CommonServerProperties.Rack,
		RevalPending:     s.CommonServerProperties.RevalPending,
		Status:           s.CommonServerProperties.Status,
		StatusID:         s.CommonServerProperties.StatusID,
		TCPPort:          s.CommonServerProperties.TCPPort,
		Type:             s.CommonServerProperties.Type,
		TypeID:           s.CommonServerProperties.TypeID,
		UpdPending:       s.CommonServerProperties.UpdPending,
		XMPPID:           s.CommonServerProperties.XMPPID,
		XMPPPasswd:       s.CommonServerProperties.XMPPPasswd,
	}
	upgraded := ServerV40{
		CommonServerPropertiesV40: common,
	}

	infs, err := s.LegacyInterfaceDetails.ToInterfacesV4(ipv4IsService, ipv6IsService, s.RouterHostName, s.RouterPortName)
	if err != nil {
		return upgraded, err
	}
	upgraded.Interfaces = infs
	return upgraded, nil
}

// ServerV40 is the representation of a Server in version 3.1 of the Traffic Ops API
type ServerV40 struct {
	CommonServerPropertiesV40
	Interfaces        []ServerInterfaceInfoV40 `json:"interfaces" db:"interfaces"`
	StatusLastUpdated *time.Time               `json:"statusLastUpdated" db:"status_last_updated"`
}

// ServerV30 is the representation of a Server in version 3 of the Traffic Ops API.
type ServerV30 struct {
	CommonServerProperties
	Interfaces        []ServerInterfaceInfo `json:"interfaces" db:"interfaces"`
	StatusLastUpdated *time.Time            `json:"statusLastUpdated" db:"status_last_updated"`
}

// ServerNullable represents an ATC server, as returned by the TO API.
// Deprecated: please use versioned structures instead of this alias from now on.
type ServerNullable ServerV30

// ToServerV2 converts the server to an equivalent ServerNullableV2 structure,
// if possible. If the conversion could not be performed, an error is returned.
func (s *ServerNullable) ToServerV2() (ServerNullableV2, error) {
	nullable := ServerV30(*s)
	return nullable.ToServerV2()
}

// ToServerV2 converts the server to an equivalent ServerNullableV2 structure,
// if possible. If the conversion could not be performed, an error is returned.
func (s *ServerV30) ToServerV2() (ServerNullableV2, error) {
	legacyServer := ServerNullableV2{
		ServerNullableV11: ServerNullableV11{
			CommonServerProperties: s.CommonServerProperties,
		},
		IPIsService:  new(bool),
		IP6IsService: new(bool),
	}

	var err error
	legacyServer.LegacyInterfaceDetails, err = InterfaceInfoToLegacyInterfaces(s.Interfaces)
	if err != nil {
		return legacyServer, err
	}

	*legacyServer.IPIsService = legacyServer.LegacyInterfaceDetails.IPAddress != nil && *legacyServer.LegacyInterfaceDetails.IPAddress != ""
	*legacyServer.IP6IsService = legacyServer.LegacyInterfaceDetails.IP6Address != nil && *legacyServer.LegacyInterfaceDetails.IP6Address != ""

	return legacyServer, nil
}

func (s *ServerV40) ToServerV3FromV4() (ServerV30, error) {
	common := CommonServerProperties{
		Cachegroup:       s.CommonServerPropertiesV40.Cachegroup,
		CachegroupID:     s.CommonServerPropertiesV40.CachegroupID,
		CDNID:            s.CommonServerPropertiesV40.CDNID,
		CDNName:          s.CommonServerPropertiesV40.CDNName,
		DeliveryServices: s.CommonServerPropertiesV40.DeliveryServices,
		DomainName:       s.CommonServerPropertiesV40.DomainName,
		FQDN:             s.CommonServerPropertiesV40.FQDN,
		FqdnTime:         s.CommonServerPropertiesV40.FqdnTime,
		GUID:             s.CommonServerPropertiesV40.GUID,
		HostName:         s.CommonServerPropertiesV40.HostName,
		HTTPSPort:        s.CommonServerPropertiesV40.HTTPSPort,
		ID:               s.CommonServerPropertiesV40.ID,
		ILOIPAddress:     s.CommonServerPropertiesV40.ILOIPAddress,
		ILOIPGateway:     s.CommonServerPropertiesV40.ILOIPGateway,
		ILOIPNetmask:     s.CommonServerPropertiesV40.ILOIPNetmask,
		ILOPassword:      s.CommonServerPropertiesV40.ILOPassword,
		ILOUsername:      s.CommonServerPropertiesV40.ILOUsername,
		LastUpdated:      s.CommonServerPropertiesV40.LastUpdated,
		MgmtIPAddress:    s.CommonServerPropertiesV40.MgmtIPAddress,
		MgmtIPGateway:    s.CommonServerPropertiesV40.MgmtIPGateway,
		MgmtIPNetmask:    s.CommonServerPropertiesV40.MgmtIPNetmask,
		OfflineReason:    s.CommonServerPropertiesV40.OfflineReason,
		PhysLocation:     s.CommonServerPropertiesV40.PhysLocation,
		PhysLocationID:   s.CommonServerPropertiesV40.PhysLocationID,
		Profile:          s.CommonServerPropertiesV40.Profile,
		ProfileDesc:      s.CommonServerPropertiesV40.ProfileDesc,
		ProfileID:        s.CommonServerPropertiesV40.ProfileID,
		Rack:             s.CommonServerPropertiesV40.Rack,
		RevalPending:     s.CommonServerPropertiesV40.RevalPending,
		Status:           s.CommonServerPropertiesV40.Status,
		StatusID:         s.CommonServerPropertiesV40.StatusID,
		TCPPort:          s.CommonServerPropertiesV40.TCPPort,
		Type:             s.CommonServerPropertiesV40.Type,
		TypeID:           s.CommonServerPropertiesV40.TypeID,
		UpdPending:       s.CommonServerPropertiesV40.UpdPending,
		XMPPID:           s.CommonServerPropertiesV40.XMPPID,
		XMPPPasswd:       s.CommonServerPropertiesV40.XMPPPasswd,
	}
	if len(s.Interfaces) != 0 {
		common.RouterHostName = &s.Interfaces[0].RouterHostName
		common.RouterPortName = &s.Interfaces[0].RouterPort
	}

	interfaces := make([]ServerInterfaceInfo, 0)
	i := ServerInterfaceInfo{}
	for _, in := range s.Interfaces {
		i.Name = in.Name
		i.MTU = in.MTU
		i.MaxBandwidth = in.MaxBandwidth
		i.Monitor = in.Monitor
		i.IPAddresses = in.IPAddresses
		interfaces = append(interfaces, i)
	}
	serverV30 := ServerV30{
		CommonServerProperties: common,
		Interfaces:             interfaces,
		StatusLastUpdated:      s.StatusLastUpdated,
	}

	return serverV30, nil
}

func (s *ServerV40) ToServerV2FromV4() (ServerNullableV2, error) {
	common := CommonServerProperties{
		Cachegroup:       s.CommonServerPropertiesV40.Cachegroup,
		CachegroupID:     s.CommonServerPropertiesV40.CachegroupID,
		CDNID:            s.CommonServerPropertiesV40.CDNID,
		CDNName:          s.CommonServerPropertiesV40.CDNName,
		DeliveryServices: s.CommonServerPropertiesV40.DeliveryServices,
		DomainName:       s.CommonServerPropertiesV40.DomainName,
		FQDN:             s.CommonServerPropertiesV40.FQDN,
		FqdnTime:         s.CommonServerPropertiesV40.FqdnTime,
		GUID:             s.CommonServerPropertiesV40.GUID,
		HostName:         s.CommonServerPropertiesV40.HostName,
		HTTPSPort:        s.CommonServerPropertiesV40.HTTPSPort,
		ID:               s.CommonServerPropertiesV40.ID,
		ILOIPAddress:     s.CommonServerPropertiesV40.ILOIPAddress,
		ILOIPGateway:     s.CommonServerPropertiesV40.ILOIPGateway,
		ILOIPNetmask:     s.CommonServerPropertiesV40.ILOIPNetmask,
		ILOPassword:      s.CommonServerPropertiesV40.ILOPassword,
		ILOUsername:      s.CommonServerPropertiesV40.ILOUsername,
		LastUpdated:      s.CommonServerPropertiesV40.LastUpdated,
		MgmtIPAddress:    s.CommonServerPropertiesV40.MgmtIPAddress,
		MgmtIPGateway:    s.CommonServerPropertiesV40.MgmtIPGateway,
		MgmtIPNetmask:    s.CommonServerPropertiesV40.MgmtIPNetmask,
		OfflineReason:    s.CommonServerPropertiesV40.OfflineReason,
		PhysLocation:     s.CommonServerPropertiesV40.PhysLocation,
		PhysLocationID:   s.CommonServerPropertiesV40.PhysLocationID,
		Profile:          s.CommonServerPropertiesV40.Profile,
		ProfileDesc:      s.CommonServerPropertiesV40.ProfileDesc,
		ProfileID:        s.CommonServerPropertiesV40.ProfileID,
		Rack:             s.CommonServerPropertiesV40.Rack,
		RevalPending:     s.CommonServerPropertiesV40.RevalPending,
		Status:           s.CommonServerPropertiesV40.Status,
		StatusID:         s.CommonServerPropertiesV40.StatusID,
		TCPPort:          s.CommonServerPropertiesV40.TCPPort,
		Type:             s.CommonServerPropertiesV40.Type,
		TypeID:           s.CommonServerPropertiesV40.TypeID,
		UpdPending:       s.CommonServerPropertiesV40.UpdPending,
		XMPPID:           s.CommonServerPropertiesV40.XMPPID,
		XMPPPasswd:       s.CommonServerPropertiesV40.XMPPPasswd,
	}
	if len(s.Interfaces) != 0 {
		common.RouterHostName = &s.Interfaces[0].RouterHostName
		common.RouterPortName = &s.Interfaces[0].RouterPort
	}
	legacyServer := ServerNullableV2{
		ServerNullableV11: ServerNullableV11{
			CommonServerProperties: common,
		},
		IPIsService:  new(bool),
		IP6IsService: new(bool),
	}

	interfaces := make([]ServerInterfaceInfo, 0)
	i := ServerInterfaceInfo{}
	for _, in := range s.Interfaces {
		i.Name = in.Name
		i.MTU = in.MTU
		i.MaxBandwidth = in.MaxBandwidth
		i.Monitor = in.Monitor
		i.IPAddresses = in.IPAddresses
		interfaces = append(interfaces, i)
	}
	var err error
	legacyServer.LegacyInterfaceDetails, err = InterfaceInfoToLegacyInterfaces(interfaces)
	if err != nil {
		return legacyServer, err
	}

	*legacyServer.IPIsService = legacyServer.LegacyInterfaceDetails.IPAddress != nil && *legacyServer.LegacyInterfaceDetails.IPAddress != ""
	*legacyServer.IP6IsService = legacyServer.LegacyInterfaceDetails.IP6Address != nil && *legacyServer.LegacyInterfaceDetails.IP6Address != ""

	return legacyServer, nil
}

type ServerUpdateStatus struct {
	HostName           string `json:"host_name"`
	UpdatePending      bool   `json:"upd_pending"`
	RevalPending       bool   `json:"reval_pending"`
	UseRevalPending    bool   `json:"use_reval_pending"`
	HostId             int    `json:"host_id"`
	Status             string `json:"status"`
	ParentPending      bool   `json:"parent_pending"`
	ParentRevalPending bool   `json:"parent_reval_pending"`
}

type ServerPutStatus struct {
	Status        util.JSONNameOrIDStr `json:"status"`
	OfflineReason *string              `json:"offlineReason"`
}

type ServerInfo struct {
	Cachegroup   string
	CachegroupID int
	CDNID        int
	DomainName   string
	HostName     string
	ID           int
	Status       string
	Type         string
}

type ServerDetail struct {
	CacheGroup         *string           `json:"cachegroup" db:"cachegroup"`
	CDNName            *string           `json:"cdnName" db:"cdn_name"`
	DeliveryServiceIDs []int64           `json:"deliveryservices,omitempty"`
	DomainName         *string           `json:"domainName" db:"domain_name"`
	GUID               *string           `json:"guid" db:"guid"`
	HardwareInfo       map[string]string `json:"hardwareInfo"`
	HostName           *string           `json:"hostName" db:"host_name"`
	HTTPSPort          *int              `json:"httpsPort" db:"https_port"`
	ID                 *int              `json:"id" db:"id"`
	ILOIPAddress       *string           `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway       *string           `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask       *string           `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword        *string           `json:"iloPassword" db:"ilo_password"`
	ILOUsername        *string           `json:"iloUsername" db:"ilo_username"`
	MgmtIPAddress      *string           `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway      *string           `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask      *string           `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason      *string           `json:"offlineReason" db:"offline_reason"`
	PhysLocation       *string           `json:"physLocation" db:"phys_location"`
	Profile            *string           `json:"profile" db:"profile"`
	ProfileDesc        *string           `json:"profileDesc" db:"profile_desc"`
	Rack               *string           `json:"rack" db:"rack"`
	RouterHostName     *string           `json:"routerHostName" db:"router_host_name"`
	RouterPortName     *string           `json:"routerPortName" db:"router_port_name"`
	Status             *string           `json:"status" db:"status"`
	TCPPort            *int              `json:"tcpPort" db:"tcp_port"`
	Type               string            `json:"type" db:"server_type"`
	XMPPID             *string           `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd         *string           `json:"xmppPasswd" db:"xmpp_passwd"`
}

type ServerDetailBaseV40 struct {
	CacheGroup         *string           `json:"cachegroup" db:"cachegroup"`
	CDNName            *string           `json:"cdnName" db:"cdn_name"`
	DeliveryServiceIDs []int64           `json:"deliveryservices,omitempty"`
	DomainName         *string           `json:"domainName" db:"domain_name"`
	GUID               *string           `json:"guid" db:"guid"`
	HardwareInfo       map[string]string `json:"hardwareInfo"`
	HostName           *string           `json:"hostName" db:"host_name"`
	HTTPSPort          *int              `json:"httpsPort" db:"https_port"`
	ID                 *int              `json:"id" db:"id"`
	ILOIPAddress       *string           `json:"iloIpAddress" db:"ilo_ip_address"`
	ILOIPGateway       *string           `json:"iloIpGateway" db:"ilo_ip_gateway"`
	ILOIPNetmask       *string           `json:"iloIpNetmask" db:"ilo_ip_netmask"`
	ILOPassword        *string           `json:"iloPassword" db:"ilo_password"`
	ILOUsername        *string           `json:"iloUsername" db:"ilo_username"`
	MgmtIPAddress      *string           `json:"mgmtIpAddress" db:"mgmt_ip_address"`
	MgmtIPGateway      *string           `json:"mgmtIpGateway" db:"mgmt_ip_gateway"`
	MgmtIPNetmask      *string           `json:"mgmtIpNetmask" db:"mgmt_ip_netmask"`
	OfflineReason      *string           `json:"offlineReason" db:"offline_reason"`
	PhysLocation       *string           `json:"physLocation" db:"phys_location"`
	Profile            *string           `json:"profile" db:"profile"`
	ProfileDesc        *string           `json:"profileDesc" db:"profile_desc"`
	Rack               *string           `json:"rack" db:"rack"`
	Status             *string           `json:"status" db:"status"`
	TCPPort            *int              `json:"tcpPort" db:"tcp_port"`
	Type               string            `json:"type" db:"server_type"`
	XMPPID             *string           `json:"xmppId" db:"xmpp_id"`
	XMPPPasswd         *string           `json:"xmppPasswd" db:"xmpp_passwd"`
}

// ServerQueueUpdateRequest encodes the request data for the POST
// servers/{{ID}}/queue_update endpoint.
type ServerQueueUpdateRequest struct {
	Action string `json:"action"`
}

// ServerQueueUpdateResponse decodes the full response with alerts from the POST
// servers/{{ID}}/queue_update endpoint.
type ServerQueueUpdateResponse struct {
	Response ServerQueueUpdate `json:"response"`
	Alerts
}

// ServerQueueUpdate decodes the update data from the POST
// servers/{{ID}}/queue_update endpoint.
type ServerQueueUpdate struct {
	ServerID util.JSONIntStr `json:"serverId"`
	Action   string          `json:"action"`
}
