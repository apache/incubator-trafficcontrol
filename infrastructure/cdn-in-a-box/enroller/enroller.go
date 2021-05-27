package main

// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/apache/trafficcontrol/lib/go-log"
	tc "github.com/apache/trafficcontrol/lib/go-tc"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"

	"github.com/fsnotify/fsnotify"
	"github.com/kelseyhightower/envconfig"
)

var startedFile = "enroller-started"

type session struct {
	*client.Session
}

func newSession(reqTimeout time.Duration, toURL string, toUser string, toPass string) (session, error) {
	s, _, err := client.LoginWithAgent(toURL, toUser, toPass, true, "cdn-in-a-box-enroller", true, reqTimeout)
	return session{s}, err
}

func (s session) getParameter(m tc.Parameter, header http.Header) (tc.Parameter, error) {
	// TODO: s.GetParameterByxxx() does not seem to work with values with spaces --
	// doing this the hard way for now
	opts := client.RequestOptions{Header: header}
	parameters, _, err := s.GetParameters(opts)
	if err != nil {
		return m, fmt.Errorf("getting Parameters: %v - alerts: %+v", err, parameters.Alerts)
	}
	for _, p := range parameters.Response {
		if p.Name == m.Name && p.Value == m.Value && p.ConfigFile == m.ConfigFile {
			return p, nil
		}
	}
	return m, fmt.Errorf("no parameter matching name %s, configFile %s, value %s", m.Name, m.ConfigFile, m.Value)
}

// enrollType takes a json file and creates a Type object using the TO API
func enrollType(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Type
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Type: %s", err)
		return err
	}

	alerts, _, err := toSession.CreateType(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("Type '%s' already exists", s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Type: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollCDN takes a json file and creates a CDN object using the TO API
func enrollCDN(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.CDN
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding CDN: %v", err)
		return err
	}

	alerts, _, err := toSession.CreateCDN(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if strings.Contains(alert.Text, "already exists") {
				log.Infof("CDN '%s' already exists", s.Name)
				return nil
			}
		}
		log.Infof("error creating CDN: %v - alerts: %+v", err, alerts.Alerts)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollASN(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.ASN
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding ASN: %s\n", err)
		return err
	}

	alerts, _, err := toSession.CreateASN(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if strings.Contains(alert.Text, "already exists") {
				log.Infof("asn %d already exists", s.ASN)
				return nil
			}
		}
		err = fmt.Errorf("error creating ASN: %s - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollCachegroup takes a json file and creates a Cachegroup object using the TO API
func enrollCachegroup(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.CacheGroupNullable
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Cache Group: '%s'", err)
		return err
	}

	alerts, _, err := toSession.CreateCacheGroup(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if strings.Contains(alert.Text, "already exists") {
				log.Infof("Cache Group '%s' already exists", *s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Cache Group: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollTopology(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Topology
	err := dec.Decode(&s)
	if err != nil && err != io.EOF {
		log.Infof("error decoding Topology: %s", err)
		return err
	}

	alerts, _, err := toSession.CreateTopology(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("topology %s already exists", s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Topology: %v - alerts: %+v", err, alerts.Alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollDeliveryService(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.DeliveryServiceV4
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding DeliveryService: %v", err)
		return err
	}

	alerts, _, err := toSession.CreateDeliveryService(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if strings.Contains(alert.Text, "already exists") {
				log.Infof("Delivery Service '%s' already exists", s.XMLID)
				return nil
			}
		}
		log.Infof("error creating Delivery Service: %v - alerts: %+v", err, alerts.Alerts)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollDeliveryServicesRequiredCapability takes a json file and creates a DeliveryServicesRequiredCapability object using the TO API
func enrollDeliveryServicesRequiredCapability(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var dsrc tc.DeliveryServicesRequiredCapability
	err := dec.Decode(&dsrc)
	if err != nil {
		log.Infof("error decoding Delivery Services Required Capability: %s\n", err)
		return err
	}

	if dsrc.XMLID == nil {
		return errors.New("required capability had no XMLID")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", *dsrc.XMLID)
	dses, _, err := toSession.GetDeliveryServices(opts)
	if err != nil {
		log.Infof("getting Delivery Service by XMLID %s: %s", *dsrc.XMLID, err.Error())
		return err
	}
	if len(dses.Response) < 1 {
		err = fmt.Errorf("could not find a Delivey Service with XMLID %s", *dsrc.XMLID)
		log.Infoln(err)
		return err
	}
	dsrc.DeliveryServiceID = dses.Response[0].ID

	alerts, _, err := toSession.CreateDeliveryServicesRequiredCapability(dsrc, client.RequestOptions{})
	if err != nil {
		log.Infof("error creating Delivery Services Required Capability: %v", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollDeliveryServiceServer(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)

	// DeliveryServiceServers lists ds xmlid and array of server names.  Use that to create multiple DeliveryServiceServer objects
	var dss tc.DeliveryServiceServers
	err := dec.Decode(&dss)
	if err != nil {
		log.Infof("error decoding DeliveryServiceServer: %s\n", err)
		return err
	}

	opts := client.RequestOptions{QueryParameters: url.Values{"xmlId": []string{dss.XmlId}}}
	dses, _, err := toSession.GetDeliveryServices(opts)
	if err != nil {
		return err
	}
	if len(dses.Response) == 0 {
		return errors.New("no deliveryservice with name " + dss.XmlId)
	}
	if dses.Response[0].ID == nil {
		return errors.New("Deliveryservice with name " + dss.XmlId + " has a nil ID")
	}
	dsID := *dses.Response[0].ID

	opts.QueryParameters = url.Values{}
	var serverIDs []int
	for _, sn := range dss.ServerNames {
		opts.QueryParameters.Set("hostName", sn)
		servers, _, err := toSession.GetServers(opts)
		if err != nil {
			return err
		}
		if len(servers.Response) == 0 {
			return errors.New("no server with hostName " + sn)
		}
		if servers.Response[0].ID == nil {
			return fmt.Errorf("Traffic Ops gave back a representation for server '%s' with null or undefined ID", sn)
		}
		serverIDs = append(serverIDs, *servers.Response[0].ID)
	}
	resp, _, err := toSession.CreateDeliveryServiceServers(dsID, serverIDs, true, client.RequestOptions{})
	if err != nil {
		log.Infof("error assigning servers %v to Delivery Service #%d: %v - alerts: %+v", serverIDs, dsID, err, resp.Alerts)
	}

	return err
}

func enrollDivision(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Division
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Division: %s", err)
		return err
	}

	alerts, _, err := toSession.CreateDivision(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if strings.Contains(alert.Text, "already exists") {
				log.Infof("division %s already exists", s.Name)
				return nil
			}
		}
		log.Infof("error creating Division: %v - alerts: %+v", err, alerts.Alerts)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollOrigin(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Origin
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Origin: %v", err)
		return err
	}
	if s.Name == nil {
		return errors.New("cannot create an Origin with no name")
	}

	alerts, _, err := toSession.CreateOrigin(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("Origin '%s' already exists", *s.Name)
				return nil
			}
		}
		log.Infof("error creating Origin: %v - alerts: %+v", err, alerts.Alerts)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollParameter(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var params []tc.Parameter
	err := dec.Decode(&params)
	if err != nil {
		log.Infof("error decoding Parameter: %s\n", err)
		return err
	}

	for _, p := range params {
		eparam, err := toSession.getParameter(p, nil)
		var alerts tc.Alerts
		if err == nil {
			// existing param -- update
			alerts, _, err = toSession.UpdateParameter(eparam.ID, p, client.RequestOptions{})
			if err != nil {
				log.Infof("error updating parameter %d: %v with %+v - alerts: %+v ", eparam.ID, err, p, alerts.Alerts)
				break
			}
		} else {
			alerts, _, err = toSession.CreateParameter(p, client.RequestOptions{})
			if err != nil {
				log.Infof("error creating parameter: %v from %+v - alerts: %+v", err, p, alerts.Alerts)
				return err
			}
			eparam, err = toSession.getParameter(p, nil)
			if err != nil {
				return err
			}
		}

		// link parameter with profiles
		if len(p.Profiles) > 0 {
			var profiles []string
			err = json.Unmarshal(p.Profiles, &profiles)
			if err != nil {
				log.Infof("%v", err)
				return err
			}

			opts := client.NewRequestOptions()
			for _, n := range profiles {
				opts.QueryParameters.Set("name", n)
				profiles, _, err := toSession.GetProfiles(opts)
				if err != nil {
					return err
				}
				if len(profiles.Response) == 0 {
					return errors.New("no profile with name " + n)
				}

				pp := tc.ProfileParameterCreationRequest{ParameterID: eparam.ID, ProfileID: profiles.Response[0].ID}
				resp, _, err := toSession.CreateProfileParameter(pp, client.RequestOptions{})
				if err != nil {
					found := false
					for _, alert := range resp.Alerts {
						if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
							found = true
							break
						}
					}
					if found {
						continue
					}
					// the original code didn't actually do anything if the error wasn't that the
					// Profile/Parameter association already exists.
					// TODO: handle other errors?
				}
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(&alerts)
	}
	return err
}

func enrollPhysLocation(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.PhysLocation
	err := dec.Decode(&s)
	if err != nil {
		err = fmt.Errorf("error decoding Physical Location: %v", err)
		log.Infoln(err)
		return err
	}

	alerts, _, err := toSession.CreatePhysLocation(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("Physical Location %s already exists", s.Name)
				return nil
			}

		}
		err = fmt.Errorf("error creating Physical Location '%s': %v - alerts: %+v", s.Name, err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollRegion(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Region
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Region: %s\n", err)
		return err
	}

	alerts, _, err := toSession.CreateRegion(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("a Region named '%s' already exists", s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Region '%s': %v - alerts: %+v", s.Name, err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollStatus(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.StatusNullable
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Status: %s", err)
		return err
	}

	alerts, _, err := toSession.CreateStatus(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("status %s already exists", *s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Status: %v - alerts: %+v", err, alerts.Alerts)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollTenant(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.Tenant
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Tenant: %s", err)
		return err
	}

	alerts, _, err := toSession.CreateTenant(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("tenant %s already exists", s.Name)
				return nil
			}
		}
		err = fmt.Errorf("error creating Tenant: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

func enrollUser(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.User
	err := dec.Decode(&s)
	log.Infof("User is %++v\n", s)
	if err != nil {
		log.Infof("error decoding User: %v", err)
		return err
	}

	alerts, _, err := toSession.CreateUser(s, client.RequestOptions{})
	if err != nil {
		for _, alert := range alerts.Alerts.Alerts {
			if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
				log.Infof("user %s already exists\n", *s.Username)
				return nil
			}
		}
		err = fmt.Errorf("error creating User: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollProfile takes a json file and creates a Profile object using the TO API
func enrollProfile(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var profile tc.Profile

	err := dec.Decode(&profile)
	if err != nil {
		log.Infof("error decoding Profile: %s\n", err)
		return err
	}
	// get a copy of the parameters
	parameters := profile.Parameters

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("  ", "")
	enc.Encode(profile)

	if len(profile.Name) == 0 {
		log.Infoln("missing name on profile")
		return errors.New("missing name on profile")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", profile.Name)
	profiles, _, err := toSession.GetProfiles(opts)

	createProfile := false
	if err != nil || len(profiles.Response) == 0 {
		// no profile by that name -- need to create it
		createProfile = true
	} else {
		// updating - ID needs to match
		profile = profiles.Response[0]
	}

	var alerts tc.Alerts
	var action string
	if createProfile {
		alerts, _, err = toSession.CreateProfile(profile, client.RequestOptions{})
		if err != nil {
			found := false
			for _, alert := range alerts.Alerts {
				if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
					found = true
					break
				}
			}
			if found {
				log.Infof("profile %s already exists", profile.Name)
			} else {
				log.Infof("error creating profile from %+v: %v - alerts: %+v", profile, err, alerts.Alerts)
			}
		}
		profiles, _, err = toSession.GetProfiles(opts)
		if err != nil {
			log.Infof("error getting profile ID from %+v: %v - alerts: %+v", profile, err, profiles.Alerts)
		}
		if len(profiles.Response) == 0 {
			err = fmt.Errorf("no results returned for getting profile ID from %+v", profile)
			log.Infoln(err)
			return err
		}
		profile = profiles.Response[0]
		action = "creating"
	} else {
		alerts, _, err = toSession.UpdateProfile(profile.ID, profile, client.RequestOptions{})
		action = "updating"
	}

	if err != nil {
		log.Infof("error "+action+" from %s: %s", err)
		return err
	}

	for _, p := range parameters {
		var name, configFile, value string
		var secure bool
		if p.ConfigFile != nil {
			configFile = *p.ConfigFile
		}
		if p.Name != nil {
			name = *p.Name
		}
		if p.Value != nil {
			value = *p.Value
		}
		param := tc.Parameter{ConfigFile: configFile, Name: name, Value: value, Secure: secure}
		eparam, err := toSession.getParameter(param, nil)
		if err != nil {
			// create it
			log.Infof("creating param %+v", param)
			newAlerts, _, err := toSession.CreateParameter(param, client.RequestOptions{})
			if err != nil {
				log.Infof("can't create parameter %+v: %s, %v", param, err, newAlerts.Alerts)
				continue
			}
			eparam, err = toSession.getParameter(param, nil)
			if err != nil {
				log.Infof("error getting new parameter %+v: \n", param)
				log.Infof(err.Error())
				continue
			}
		} else {
			log.Infof("found param %+v\n", eparam)
		}

		if eparam.ID < 1 {
			log.Infof("param ID not found for %v", eparam)
			continue
		}
		pp := tc.ProfileParameterCreationRequest{ProfileID: profile.ID, ParameterID: eparam.ID}
		resp, _, err := toSession.CreateProfileParameter(pp, client.RequestOptions{})
		if err != nil {
			found := false
			for _, alert := range resp.Alerts {
				if alert.Level == tc.ErrorLevel.String() && strings.Contains(alert.Text, "already exists") {
					found = true
					break
				}
			}
			if !found {
				log.Infof("error creating profileparameter %+v: %v - alerts: %+v", pp, err, resp.Alerts)
			}
		}
	}

	//enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollServer takes a json file and creates a Server object using the TO API
func enrollServer(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.ServerV40
	err := dec.Decode(&s)
	if err != nil {
		log.Infof("error decoding Server: %v", err)
		return err
	}

	alerts, _, err := toSession.CreateServer(s, client.RequestOptions{})
	if err != nil {
		err = fmt.Errorf("error creating Server: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollServerCapability takes a json file and creates a ServerCapability object using the TO API
func enrollServerCapability(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.ServerCapability
	err := dec.Decode(&s)
	if err != nil {
		err = fmt.Errorf("error decoding Server Capability: %v", err)
		log.Infoln(err)
		return err
	}

	alerts, _, err := toSession.CreateServerCapability(s, client.RequestOptions{})
	if err != nil {
		err = fmt.Errorf("error creating Server Capability: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

// enrollFederation takes a json file and creates a Federation object using the TO API.
// It also assigns a Delivery Service, the CDN in a Box admin user, IPv4 resolvers,
// and IPv6 resolvers to that Federation.
func enrollFederation(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var federation tc.AllDeliveryServiceFederationsMapping
	err := dec.Decode(&federation)
	if err != nil {
		log.Infof("error decoding Server Capability: %s\n", err)
		return err
	}
	opts := client.NewRequestOptions()
	for _, mapping := range federation.Mappings {
		var cdnFederation tc.CDNFederation
		var cdnName string
		{
			xmlID := string(federation.DeliveryService)
			opts.QueryParameters.Set("xmlId", xmlID)
			deliveryServices, _, err := toSession.GetDeliveryServices(opts)
			opts.QueryParameters.Del("xmlId")
			if err != nil {
				err = fmt.Errorf("getting Delivery Service '%s': %v - alerts: %+v", xmlID, err, deliveryServices.Alerts)
				log.Infoln(err)
				return err
			}
			if len(deliveryServices.Response) != 1 {
				err = fmt.Errorf("wanted 1 Delivery Service with XMLID %s but received %d Delivery Services", xmlID, len(deliveryServices.Response))
				log.Infoln(err)
				return err
			}
			deliveryService := deliveryServices.Response[0]
			if deliveryService.CDNName == nil || deliveryService.ID == nil {
				err = fmt.Errorf("Delivery Service '%s' as returned from Traffic Ops had null or undefined CDN name and/or ID", xmlID)
				log.Infoln(err)
				return err
			}
			cdnName = *deliveryService.CDNName
			cdnFederation = tc.CDNFederation{
				CName: mapping.CName,
				TTL:   mapping.TTL,
			}
			resp, _, err := toSession.CreateCDNFederation(cdnFederation, cdnName, client.RequestOptions{})
			if err != nil {
				err = fmt.Errorf("creating CDN Federation: %v - alerts: %+v", err, resp.Alerts)
				log.Infoln(err)
				return err
			}
			cdnFederation = resp.Response
			if cdnFederation.ID == nil {
				err = fmt.Errorf("Federation returned from creation through Traffic Ops with null or undefined ID")
				log.Infoln(err)
				return err
			}
			if alerts, _, err := toSession.CreateFederationDeliveryServices(*cdnFederation.ID, []int{*deliveryService.ID}, true, client.RequestOptions{}); err != nil {
				err = fmt.Errorf("assigning Delivery Service %s to Federation with ID %d: %v - alerts: %+v", xmlID, *cdnFederation.ID, err, alerts.Alerts)
				log.Infoln(err)
				return err
			}
		}
		{
			user, _, err := toSession.GetUserCurrent(client.RequestOptions{})
			if err != nil {
				err = fmt.Errorf("getting the Current User: %v - alerts: %+v", err, user.Alerts)
				log.Infoln(err)
				return err
			}
			if user.Response.ID == nil {
				err = errors.New("current user returned from Traffic Ops had null or undefined ID")
				log.Infoln(err)
				return err
			}
			resp, _, err := toSession.CreateFederationUsers(*cdnFederation.ID, []int{*user.Response.ID}, true, client.RequestOptions{})
			if err != nil {
				var username string
				if user.Response.UserName != nil {
					username = *user.Response.UserName
				}
				err = fmt.Errorf("assigning User '%s' to Federation with ID %d: %v - alerts: %+v", username, *cdnFederation.ID, err, resp.Alerts)
				log.Infoln(err)
				return err
			}
		}
		var allResolverIDs []int
		{
			resolverTypes := []tc.FederationResolverType{tc.FederationResolverType4, tc.FederationResolverType6}
			resolverArrays := [][]string{mapping.Resolve4, mapping.Resolve6}
			for index, resolvers := range resolverArrays {
				resolverIDs, err := createFederationResolversOfType(toSession, resolverTypes[index], resolvers)
				if err != nil {
					return err
				}
				allResolverIDs = append(allResolverIDs, resolverIDs...)
			}
		}
		if resp, _, err := toSession.AssignFederationFederationResolver(*cdnFederation.ID, allResolverIDs, true, client.RequestOptions{}); err != nil {
			err = fmt.Errorf("assigning Federation Resolvers to Federation with ID %d: %v - alerts: %+v", *cdnFederation.ID, err, resp.Alerts)
			log.Infoln(err)
			return err
		}
		opts.QueryParameters.Set("id", strconv.Itoa(*cdnFederation.ID))
		response, _, err := toSession.GetCDNFederationsByName(cdnName, opts)
		opts.QueryParameters.Del("id")
		if err != nil {
			err = fmt.Errorf("getting CDN Federation with ID %d: %v - alerts: %+v", *cdnFederation.ID, err, response.Alerts)
			return err
		}
		if len(response.Response) < 1 {
			err = fmt.Errorf("unable to GET a CDN Federation ID %d in CDN %s", *cdnFederation.ID, cdnName)
			log.Infoln(err)
			return err
		}
		cdnFederation = response.Response[0]

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(&cdnFederation)
		if err != nil {
			err = fmt.Errorf("encoding CDNFederation %s with ID %d: %v", *cdnFederation.CName, *cdnFederation.ID, err)
			log.Infoln(err)
			return err
		}
	}
	return err
}

// createFederationResolversOfType creates Federation Resolvers of either RESOLVE4 type or RESOLVE6 type.
func createFederationResolversOfType(toSession *session, resolverTypeName tc.FederationResolverType, ipAddresses []string) ([]int, error) {
	typeNameString := string(resolverTypeName)
	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("name", typeNameString)
	types, _, err := toSession.GetTypes(opts)
	if err != nil {
		err = fmt.Errorf("getting resolver type '%s': %v - alerts: %+v", typeNameString, err, types.Alerts)
		log.Infoln(err)
		return nil, err
	}
	if len(types.Response) < 1 {
		err := fmt.Errorf("unable to get a type with name %s", typeNameString)
		log.Infof(err.Error())
		return nil, err
	}
	typeID := uint(types.Response[0].ID)

	var resolverIDs []int
	for _, ipAddress := range ipAddresses {
		resolver := tc.FederationResolver{
			IPAddress: &ipAddress,
			TypeID:    &typeID,
		}
		response, _, err := toSession.CreateFederationResolver(resolver, client.RequestOptions{})
		if err != nil {
			err = fmt.Errorf("creating Federation Resolver with IP address %s: %v - alerts: %+v", ipAddress, err, response.Alerts)
			return nil, err
		}
		if response.Response.ID == nil {

		}
		resolverIDs = append(resolverIDs, int(*response.Response.ID))
	}
	return resolverIDs, nil
}

// enrollServerServerCapability takes a json file and creates a ServerServerCapability object using the TO API
func enrollServerServerCapability(toSession *session, r io.Reader) error {
	dec := json.NewDecoder(r)
	var s tc.ServerServerCapability
	err := dec.Decode(&s)
	if err != nil {
		err = fmt.Errorf("error decoding Server/Capability relationship: %s", err)
		log.Infoln(err)
		return err
	}
	if s.Server == nil {
		err = errors.New("server/Capability relationship did not specify a server")
		return err
	}

	resp, _, err := toSession.GetServers(client.RequestOptions{QueryParameters: url.Values{"hostName": []string{*s.Server}}})
	if err != nil {
		err = fmt.Errorf("getting server '%s': %v - alerts: %+v", *s.Server, err, resp.Alerts)
		log.Infoln(err)
		return err
	}
	if len(resp.Response) < 1 {
		err = fmt.Errorf("could not find Server %s", *s.Server)
		log.Infoln(err.Error())
		return err
	}
	if len(resp.Response) > 1 {
		err = fmt.Errorf("found more than 1 Server with hostname %s", *s.Server)
		log.Infoln(err.Error())
		return err
	}
	s.ServerID = resp.Response[0].ID

	alerts, _, err := toSession.CreateServerServerCapability(s, client.RequestOptions{})
	if err != nil {
		err = fmt.Errorf("error creating Server Server Capability: %v - alerts: %+v", err, alerts.Alerts)
		log.Infoln(err.Error())
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(&alerts)

	return err
}

type dirWatcher struct {
	*fsnotify.Watcher
	TOSession *session
	watched   map[string]func(toSession *session, fn string) error
}

func newDirWatcher(toSession *session) (*dirWatcher, error) {
	var err error
	var dw dirWatcher
	dw.Watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dw.watched = make(map[string]func(toSession *session, fn string) error)
	go func() {
		const (
			processed = ".processed"
			rejected  = ".rejected"
			retry     = ".retry"
		)
		originalNameRegex := regexp.MustCompile(`(\.retry)*$`)

		emptyCount := map[string]int{}
		const maxEmptyTries = 10

		for {
			select {
			case event, ok := <-dw.Events:
				if !ok {
					log.Infoln("event not ok")
					continue
				}

				// ignore all but Create events
				if event.Op&fsnotify.Create != fsnotify.Create {
					continue
				}

				// skip already processed files
				if strings.HasSuffix(event.Name, processed) || strings.HasSuffix(event.Name, rejected) {
					continue
				}

				i, err := os.Stat(event.Name)
				if err != nil || i.IsDir() {
					log.Infoln("skipping " + event.Name)
					continue
				}
				log.Infoln("new file :", event.Name)

				// what directory is the file in?  Invoke the matching func
				dir := filepath.Base(filepath.Dir(event.Name))
				suffix := rejected
				if f, ok := dw.watched[dir]; ok {
					t := filepath.Base(dir)
					log.Infoln("creating " + t + " from " + event.Name)
					// Sleep for 100 milliseconds so that the file content is probably there when the directory watcher
					// sees the file
					time.Sleep(100 * time.Millisecond)

					err := f(toSession, event.Name)
					// If a file is empty, try reading from it 10 times before giving up on that file
					if err == io.EOF {
						originalName := originalNameRegex.ReplaceAllString(event.Name, "")
						if _, exists := emptyCount[originalName]; !exists {
							emptyCount[originalName] = 0
						}
						emptyCount[originalName]++
						log.Infof("empty json object %s: %s\ntried file %d out of %d times", originalName, err.Error(), emptyCount[originalName], maxEmptyTries)
						if emptyCount[originalName] < maxEmptyTries {
							newName := event.Name + retry
							if err := os.Rename(event.Name, newName); err != nil {
								log.Infof("error renaming %s to %s: %s", event.Name, newName, err)
							}
							continue
						}
					}
					if err != nil {
						log.Infof("error creating %s from %s: %s\n", dir, event.Name, err.Error())
					} else {
						suffix = processed
					}
				} else {
					log.Infof("no method for creating %s\n", dir)
				}
				// rename the file indicating if processed or rejected
				err = os.Rename(event.Name, event.Name+suffix)
				if err != nil {
					log.Infof("error renaming %s to %s: %s\n", event.Name, event.Name+suffix, err.Error())
				}
			case err, ok := <-dw.Errors:
				log.Infof("error from fsnotify: ok? %v;  error: %v\n", ok, err)
				continue
			}
		}
	}()
	return &dw, err
}

// watch starts f when a new file is created in dir
func (dw *dirWatcher) watch(watchdir, t string, f func(*session, io.Reader) error) {
	dir := watchdir + "/" + t
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		// attempt to create dir
		if err = os.Mkdir(dir, os.ModeDir|0700); err != nil {
			log.Infoln("cannot watch " + dir + ": not a directory")
			return
		}
	}

	log.Infoln("watching " + dir)
	dw.Add(dir)
	dw.watched[t] = func(toSession *session, fn string) error {
		fh, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer log.Close(fh, "could not close file")
		return f(toSession, fh)
	}
}

func startWatching(watchDir string, toSession *session, dispatcher map[string]func(*session, io.Reader) error) (*dirWatcher, error) {
	// watch for file creation in directories
	dw, err := newDirWatcher(toSession)
	if err == nil {
		for d, f := range dispatcher {
			dw.watch(watchDir, d, f)
		}
	}
	return dw, err
}

func startServer(httpPort string, toSession *session, dispatcher map[string]func(*session, io.Reader) error) error {
	baseEP := "/api/2.0/"
	for d, f := range dispatcher {
		http.HandleFunc(baseEP+d, func(w http.ResponseWriter, r *http.Request) {
			defer log.Close(r.Body, "could not close reader")
			f(toSession, r.Body)
		})
	}

	go func() {
		server := &http.Server{
			Addr:      httpPort,
			TLSConfig: nil,
			ErrorLog:  log.Error,
		}
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("stopping server: %v\n", err)
			panic(err)
		}
	}()

	log.Infoln("http service started on " + httpPort)
	return nil
}

// Set up the log config -- all messages go to stdout
type logConfig struct{}

func (cfg logConfig) ErrorLog() log.LogLocation {
	return log.LogLocationStdout
}
func (cfg logConfig) WarningLog() log.LogLocation {
	return log.LogLocationStdout
}
func (cfg logConfig) InfoLog() log.LogLocation {
	return log.LogLocationStdout
}
func (cfg logConfig) DebugLog() log.LogLocation {
	return log.LogLocationStdout
}
func (cfg logConfig) EventLog() log.LogLocation {
	return log.LogLocationStdout
}

func main() {
	var watchDir, httpPort string

	flag.StringVar(&startedFile, "started", startedFile, "file indicating service was started")
	flag.StringVar(&watchDir, "dir", "", "base directory to watch")
	flag.StringVar(&httpPort, "http", "", "act as http server for POST on this port (e.g. :7070)")
	flag.Parse()

	err := log.InitCfg(logConfig{})
	if err != nil {
		panic(err.Error())
	}
	if watchDir == "" && httpPort == "" {
		// if neither -dir nor -http provided, default to watching the current dir
		watchDir = "."
	}

	var toCreds struct {
		URL      string `envconfig:"TO_URL"`
		User     string `envconfig:"TO_USER"`
		Password string `envconfig:"TO_PASSWORD"`
	}

	envconfig.Process("", &toCreds)

	reqTimeout := time.Second * time.Duration(60)

	log.Infoln("Starting TrafficOps session")
	toSession, err := newSession(reqTimeout, toCreds.URL, toCreds.User, toCreds.Password)
	if err != nil {
		log.Errorln("error starting TrafficOps session: " + err.Error())
		os.Exit(1)
	}
	log.Infoln("TrafficOps session established")

	// dispatcher maps an API endpoint name to a function to act on the JSON input Reader
	dispatcher := map[string]func(*session, io.Reader) error{
		"types":                                  enrollType,
		"cdns":                                   enrollCDN,
		"cachegroups":                            enrollCachegroup,
		"topologies":                             enrollTopology,
		"profiles":                               enrollProfile,
		"parameters":                             enrollParameter,
		"servers":                                enrollServer,
		"server_capabilities":                    enrollServerCapability,
		"server_server_capabilities":             enrollServerServerCapability,
		"asns":                                   enrollASN,
		"deliveryservices":                       enrollDeliveryService,
		"deliveryservices_required_capabilities": enrollDeliveryServicesRequiredCapability,
		"deliveryservice_servers":                enrollDeliveryServiceServer,
		"divisions":                              enrollDivision,
		"federations":                            enrollFederation,
		"origins":                                enrollOrigin,
		"phys_locations":                         enrollPhysLocation,
		"regions":                                enrollRegion,
		"statuses":                               enrollStatus,
		"tenants":                                enrollTenant,
		"users":                                  enrollUser,
	}

	if len(httpPort) != 0 {
		log.Infoln("Starting http server on " + httpPort)
		err := startServer(httpPort, &toSession, dispatcher)
		if err != nil {
			log.Errorln("http server on " + httpPort + " failed: " + err.Error())
		}
	}

	if len(watchDir) != 0 {
		log.Infoln("Watching directory " + watchDir)
		dw, err := startWatching(watchDir, &toSession, dispatcher)
		defer log.Close(dw, "could not close dirwatcher")
		if err != nil {
			log.Errorf("dirwatcher on %s failed: %s", watchDir, err.Error())
		}
	}

	// create this file to indicate the enroller is ready
	f, err := os.Create(startedFile)
	if err != nil {
		panic(err)
	}
	log.Infoln("Created " + startedFile)
	log.Close(f, "could not close file")

	var waitforever chan struct{}
	<-waitforever
}
