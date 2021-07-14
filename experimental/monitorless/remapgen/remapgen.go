package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"flag"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

const HealthSubdomain = `health`
const NearSubdomain = `near`
const FarSubdomain = `far`

func main() {
	host := flag.String("host", "", "hostname to generate remap and parent lines for")
	crcPath := flag.String("crconfig-path", "", "CRConfig file path")
	comments := flag.Bool("comments", true, "Whether to add comments to the generated text")
	configToGen := flag.String("config", "", "The ATS config file to generate. Options: remap, parent")
	healthPort := flag.Int("health-port", 0, "The health port of the health service (astatstwo)")
	flag.Parse()

	usageStr := `Usage: ./remapgen -host my-host-name -crconfig-path ./path/to/crconfig.json -config remap -health-port 8083` + "\n"
	if *healthPort == 0 {
		fmt.Fprintf(os.Stderr, usageStr)
		os.Exit(1)
	}
	if *host == "" || *crcPath == "" {
		fmt.Fprintf(os.Stderr, usageStr)
		os.Exit(1)
	}
	if *configToGen != "remap" && *configToGen != "parent" {
		fmt.Fprintf(os.Stderr, usageStr)
		os.Exit(1)
	}

	crcFi, err := os.Open(*crcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening CRConfig path '"+*crcPath+"': "+err.Error()+"\n")
		os.Exit(1)
	}
	defer crcFi.Close()

	crc := &tc.CRConfig{}
	if err := json.NewDecoder(crcFi).Decode(crc); err != nil {
		fmt.Fprintf(os.Stderr, "decoding CRConfig path '"+*crcPath+"': "+err.Error()+"\n")
		os.Exit(1)
	}

	remapLines, parentLines, err := GenerateRemapParentLines(crc, tc.CacheName(*host), *healthPort, *comments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating lines: "+err.Error()+"\n")
		os.Exit(1)
	}

	switch *configToGen {
	case "remap":
		fmt.Println(remapLines)
	case "parent":
		fmt.Println(parentLines)
	}
}

/*
1. remaps to every other cache which is NOT in this cache's CG
    - two remaps: to the same CG, and to the designated "far CG" which will poll that cache "from a distance."
    - remaps in order to "cache monitors," where every cache shares the same order, so one cache is always the "cache monitor" (unless it's down, in which case the next one on the list "becomes" the "cache monitor."
2. remaps to caches in this cache's CG
    - two remaps: directly to that cache (near), and to the designated "far CG" which will poll that cache "from a distance."
    - remaps directly to that cache
3. remap to self, to astatstwo
    - remaps to localhost:astatstwoport
*/

// GenerateRemapLines generates the health remap.config and parent.config lines.
// The crc.ContentServers[name].DeliveryServices is not used and may be nil (huge size/performance gain).
// Returns the remap lines, the parent lines, and any error
func GenerateRemapParentLines(crc *tc.CRConfig, hostName tc.CacheName, healthPort int, comments bool) (string, string, error) {
	cdnDomainI, ok := crc.Config[`domain_name`]
	if !ok {
		return "", "", errors.New("CRConfig Config missing 'domain_name'")
	}
	cdnDomain, ok := cdnDomainI.(string)
	if !ok {
		return "", "", fmt.Errorf("CRConfig Config 'domain_name' unexpected type %T", cdnDomainI)
	}

	remapSvName := hostName
	remapSv, ok := crc.ContentServers[string(hostName)]
	if !ok {
		return "", "", errors.New("host '" + string(hostName) + "' not in CRConfig")
	}
	if remapSv.CacheGroup == nil {
		return "", "", errors.New("host '" + string(hostName) + "' has no CacheGroup")
	}
	// remapSvCG, ok := crc.EdgeLocations[*remapSv.CacheGroup]
	// if !ok {
	// 	return "", "", errors.New("host '" + string(hostName) + "' CacheGroup '" + *remapSv.CacheGroup + "' not in CRConfig.EdgeLocations!")
	// }

	// cgServers is a cache.
	// The data exists in crc.ContentServers, this just saves iterating over every server every time.
	cgServers := map[tc.CacheGroupName][]string{}
	for svName, sv := range crc.ContentServers {
		// TODO handle mids
		if sv.ServerType == nil || tc.CacheType(*sv.ServerType) != tc.CacheTypeEdge {
			continue
		}
		if sv.CacheGroup == nil {
			return "", "", errors.New("server '" + svName + "' has no CacheGroup")
		}
		cgServers[tc.CacheGroupName(*sv.CacheGroup)] = append(cgServers[tc.CacheGroupName(*sv.CacheGroup)], svName)
	}

	// sort deterministically.
	for _, servers := range cgServers {
		// This sorts alphabetically. We could sort by something else, e.g. consistent hash
		sort.Strings(servers)
	}

	// get the "far cachegroup" for every unique cachegroup on any server
	farCGs := map[tc.CacheGroupName]tc.CacheGroupName{}
	for svName, sv := range crc.ContentServers {
		// TODO handle mids
		if sv.ServerType == nil || tc.CacheType(*sv.ServerType) != tc.CacheTypeEdge {
			continue
		}
		if sv.CacheGroup == nil {
			return "", "", errors.New("server '" + svName + "' has no CacheGroup")
		}
		cg := *sv.CacheGroup
		if _, ok := farCGs[tc.CacheGroupName(cg)]; ok {
			continue
		}
		farCG, err := GetFarCacheGroup(crc, tc.CacheGroupName(cg))
		if err != nil {
			return "", "", errors.New("getting far cachegroup: " + err.Error())
		}
		farCGs[tc.CacheGroupName(cg)] = farCG
	}

	farCGComments := ``
	for from, to := range farCGs {
		farCGComments += `# Far cachegroup for '` + string(from) + `' is '` + string(to) + `'
`
	}

	// TODO when making the remap rule for each server:
	// Determine if we _are_ the far cachegroup for that server.
	// If so, make the remap rule to that server direct

	remapLines := ""
	parentLines := ""
	if comments {
		parentLines += farCGComments
	}

	// Sort the server names to iterate, so the generated config is deterministic.
	serverNames := []string{}
	for name, _ := range crc.ContentServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, svName := range serverNames {
		sv := crc.ContentServers[svName]

		// TODO handle mids
		if sv.ServerType == nil || tc.CacheType(*sv.ServerType) != tc.CacheTypeEdge {
			continue
		}
		if sv.CacheGroup == nil {
			return "", "", errors.New("generating remap for '" + svName + "': null CacheGroup")
		}
		farCG, ok := farCGs[tc.CacheGroupName(*sv.CacheGroup)]
		if !ok {
			return "", "", errors.New("generating remap for '" + svName + "': CacheGroup '" + *sv.CacheGroup + "' not in far cachegroup list")
		}

		svRemaps, svParents, err := MakeRemapLinesServer(crc, cdnDomain, cgServers, tc.CacheName(svName), &sv, remapSvName, &remapSv, farCG, healthPort, comments)
		if err != nil {
			return "", "", errors.New("generating remap for '" + svName + "': " + err.Error())
		}
		remapLines += svRemaps
		parentLines += svParents
	}

	return remapLines, parentLines, nil
}

// MakeRemapLinesServer creates the health remap lines for the given sv server. The remapSv is the server the remap lines are being generated for.
// Each line is terminated by a newline
func MakeRemapLinesServer(
	crc *tc.CRConfig,
	cdnDomain string,
	cgServers map[tc.CacheGroupName][]string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	remapSvName tc.CacheName,
	remapSv *tc.CRConfigTrafficOpsServer,
	farCG tc.CacheGroupName,
	healthPort int,
	comments bool,
) (string, string, error) {
	if svName == remapSvName {
		return MakeRemapLinesSelf(crc, cdnDomain, svName, sv, healthPort, comments)
	}
	if *sv.CacheGroup == *remapSv.CacheGroup {
		return MakeRemapLinesSameCacheGroup(crc, cdnDomain, cgServers, svName, sv, remapSvName, remapSv, farCG, healthPort, comments)
	}
	return MakeRemapLinesOtherCacheGroup(crc, cdnDomain, cgServers, svName, sv, remapSvName, remapSv, healthPort, comments)
}

// MakeRemapLineSelf makes a remap line to the health service on this server.
// The domainPrefix is a prefix on the FQDN, typically "near" or "far". May not be empty.
func MakeRemapLineSelf(
	crc *tc.CRConfig,
	cdnDomain string,
	domainPrefix string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	healthPort int,
	comments bool,
) (string, string, error) {
	// TODO HTTPS option?
	// TODO determine if we need to add the port here, for custom ports
	remapFQDN := domainPrefix + `.` + HealthSubdomain + `.` + string(svName) + `.` + cdnDomain
	remapURI := `http://` + remapFQDN
	if sv.Port != nil && *sv.Port != 80 {
		remapURI += `:` + strconv.Itoa(*sv.Port)
	}
	remapURI += `/`

	healthPortStr := strconv.Itoa(healthPort)

	remapLine := `map ` + remapURI + ` http://localhost:` + healthPortStr + "\n"
	remapComment := `# self remap to local astats service
`

	portStr := "80"
	if sv.Port != nil {
		portStr = strconv.Itoa(*sv.Port)
	}

	if sv.Ip == nil {
		return "", "", errors.New("null IP")
	}
	// TODO verify IP is a valid IP
	// TODO add IPv4 and IPv6 rules
	// TODO add HTTPS port health

	parentComment := `
# health: self
# health: parent IP ` + *sv.Ip + ` is self
`
	//	TODO determine if self needs a parent line (for retry?)
	parentLine := `dest_domain=` + remapFQDN + ` port=` + portStr + ` parent="` + *sv.Ip + `:` + healthPortStr + `|0.999" round_robin=false qstring=ignore go_direct=false parent_is_proxy=false parent_retry=unavailable_server_retry unavailable_server_retry_responses="500" max_unavailable_server_retries=1

	`
	parentLine = "" // DEBUG
	if comments {
		parentLine = parentComment + parentLine
		remapLine = remapComment + remapLine
	}

	return remapLine, parentLine, nil
}

// MakeRemapLineDirect makes a remap on this server directly to the given server.
// This is used for servers in the same cachegroup, as well as "far" remaps where this server is in the chosen far cachegroup of the given server's cachegroup.
// The domainPrefix is a prefix on the FQDN, typically "near" or "far". May not be empty.
func MakeRemapLineDirect(
	crc *tc.CRConfig,
	cdnDomain string,
	domainPrefix string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	remapSvName tc.CacheName,
	remapSv *tc.CRConfigTrafficOpsServer,
	healthPort int,
	comments bool,
) (string, string, error) {
	remapFromFQDN := domainPrefix + `.` + HealthSubdomain + `.` + string(svName) + `.` + cdnDomain
	remapFromURI := `http://` + remapFromFQDN
	if remapSv.Port != nil && *remapSv.Port != 80 {
		remapFromURI += `:` + strconv.Itoa(*remapSv.Port)
	}
	remapFromURI += `/`

	remapToFQDN := domainPrefix + `.` + HealthSubdomain + `.` + string(svName) + `.` + cdnDomain

	remapToURI := `http://` + remapToFQDN
	if sv.Port != nil && *sv.Port != 80 {
		remapToURI += `:` + strconv.Itoa(*sv.Port)
	}
	remapToURI += `/`

	remapComment := `# map from ` + remapFromFQDN + ` on this server's port, to the same domain on the target's port, with via servers in parent.config
`
	remapLine := `map ` + remapFromURI + ` ` + remapToURI + "\n"

	toPortStr := "80"
	if sv.Port != nil {
		toPortStr = strconv.Itoa(*sv.Port)
	}

	if sv.Ip == nil {
		return "", "", errors.New("null IP")
	}

	// healthPortStr := strconv.Itoa(healthPort)

	if remapSv.CacheGroup == nil {
		return "", "", errors.New("this server " + string(remapSvName) + " had null cachegroup in CRConfig")
	}
	if sv.CacheGroup == nil {
		return "", "", errors.New("server " + string(svName) + " had null cachegroup in CRConfig")
	}

	reasonCGStr := "the same cachegroup as this server"
	if tc.CacheGroupName(*sv.CacheGroup) != tc.CacheGroupName(*remapSv.CacheGroup) {
		reasonCGStr = "cachegroup " + *sv.CacheGroup
	}
	reasonDistStr := "'near'"
	if domainPrefix == FarSubdomain {
		reasonDistStr = "'far'"
	}
	reasonStr := "in " + reasonCGStr + " for " + reasonDistStr + " health"

	parentComment := `
# health: direct to ` + string(svName) + ` ` + reasonStr + `
# health: parent IP ` + *sv.Ip + ` is ` + string(svName) + `, the server we're mapping to
`
	parentLine := `dest_domain=` + remapToFQDN + ` port=` + toPortStr + ` parent="` + *sv.Ip + `:` + strconv.Itoa(*sv.Port) + `|0.999" round_robin=false qstring=ignore go_direct=false parent_is_proxy=false parent_retry=unavailable_server_retry unavailable_server_retry_responses="500" max_unavailable_server_retries=1` + "\n"
	if comments {
		parentLine = parentComment + parentLine
		remapLine = remapComment + remapLine
	}

	return remapLine, parentLine, nil
}

// MakeRemapLineVia creates a remap line to the given server through the servers on the given cachegroup.
// This is used for "near" rules for servers not in this server's cachegroup, as well as for "far" rules for servers which are in this server's cachegroup (because if the server is in this server's CG, we need to go far away first).
// The domainPrefix is a prefix on the FQDN, typically "near" or "far". May not be empty.
func MakeRemapLineVia(
	crc *tc.CRConfig,
	cdnDomain string,
	domainPrefix string,
	cgServers map[tc.CacheGroupName][]string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	remapSvName tc.CacheName,
	remapSv *tc.CRConfigTrafficOpsServer,
	cg tc.CacheGroupName,
	healthPort int,
	comments bool,
) (string, string, error) {
	remapFromFQDN := domainPrefix + `.` + HealthSubdomain + `.` + string(svName) + `.` + cdnDomain
	remapFromURI := `http://` + remapFromFQDN
	if remapSv.Port != nil && *remapSv.Port != 80 {
		remapFromURI += `:` + strconv.Itoa(*remapSv.Port)
	}
	remapFromURI += `/`

	remapToFQDN := domainPrefix + `.` + HealthSubdomain + `.` + string(svName) + `.` + cdnDomain
	remapToURI := `http://` + remapToFQDN
	if sv.Port != nil && *sv.Port != 80 {
		remapToURI += `:` + strconv.Itoa(*sv.Port)
	}
	remapToURI += `/`

	remapComment := `# map from ` + remapFromFQDN + ` on this server's port, to the same domain on the target's port
`
	remapLine := `map ` + remapFromURI + ` ` + remapToURI + "\n"

	toPortStr := "80"
	if sv.Port != nil {
		toPortStr = strconv.Itoa(*sv.Port)
	}

	if sv.Ip == nil {
		return "", "", errors.New("null IP")
	}

	// healthPortStr := strconv.Itoa(healthPort)

	parentServers := cgServers[cg]
	if len(parentServers) == 0 {
		return "", "", errors.New("no servers for the cachegroup '" + string(cg) + "'")
	}
	parentsStrs := []string{}
	parentComments := []string{}
	for _, parentSvName := range parentServers {
		parentSv, ok := crc.ContentServers[parentSvName]
		if !ok {
			return "", "", errors.New("parent '" + parentSvName + "' not in CRConfig")
		}
		if parentSv.Ip == nil {
			return "", "", errors.New("parent '" + parentSvName + "' has null IP")
		}
		parentsStrs = append(parentsStrs, *parentSv.Ip+`:`+strconv.Itoa(*parentSv.Port)+`|0.999`)
		parentComments = append(parentComments, `# health: parent IP `+*parentSv.Ip+` is `+string(parentSvName)+` in `+string(cg))
	}

	parentStr := strings.Join(parentsStrs, `;`)

	if remapSv.CacheGroup == nil {
		return "", "", errors.New("this server '" + string(remapSvName) + "' had null CacheGroup!")
	}
	if sv.CacheGroup == nil {
		return "", "", errors.New("server '" + string(remapSvName) + "' had null CacheGroup!")
	}

	reasonCGStr := "the same cachegroup as"
	if tc.CacheGroupName(*sv.CacheGroup) != tc.CacheGroupName(*remapSv.CacheGroup) {
		reasonCGStr = "a different cachegroup than"
	}
	reasonDistStr := "'near'"
	if domainPrefix == FarSubdomain {
		reasonDistStr = "'far'"
	}
	reasonStr := "in " + reasonCGStr + " this server, for " + reasonDistStr + " health"
	parentComment := `
# health: ` + string(svName) + ` via ` + string(cg) + ` ` + reasonStr + `
` + strings.Join(parentComments, "\n") + `
`

	parentLine := `dest_domain=` + remapToFQDN + ` port=` + toPortStr + ` parent="` + parentStr + `" round_robin=false qstring=ignore go_direct=false parent_is_proxy=false parent_retry=unavailable_server_retry unavailable_server_retry_responses="500" max_unavailable_server_retries=1` + "\n"
	if comments {
		parentLine = parentComment + parentLine
		remapLine = remapComment + remapLine
	}

	return remapLine, parentLine, nil
}

// MakeRemapLinesSelf is used to generate this server's own remap to localhost:astats2
func MakeRemapLinesSelf(crc *tc.CRConfig, cdnDomain string, svName tc.CacheName, sv *tc.CRConfigTrafficOpsServer, healthPort int, comments bool) (string, string, error) {
	remapLines := ``
	parentLines := ``
	{
		remapLine, parentLine, err := MakeRemapLineSelf(crc, cdnDomain, NearSubdomain, svName, sv, healthPort, comments)
		if err != nil {
			return "", "", errors.New("making self near remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	{
		remapLine, parentLine, err := MakeRemapLineSelf(crc, cdnDomain, FarSubdomain, svName, sv, healthPort, comments)
		if err != nil {
			return "", "", errors.New("making self far remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	return remapLines, parentLines, nil
}

func MakeRemapLinesSameCacheGroup(
	crc *tc.CRConfig,
	cdnDomain string,
	cgServers map[tc.CacheGroupName][]string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	remapSvName tc.CacheName,
	remapSv *tc.CRConfigTrafficOpsServer,
	farCG tc.CacheGroupName,
	healthPort int,
	comments bool,
) (string, string, error) {
	remapLines := ``
	parentLines := ``
	{
		remapLine, parentLine, err := MakeRemapLineDirect(crc, cdnDomain, NearSubdomain, svName, sv, remapSvName, remapSv, healthPort, comments)
		if err != nil {
			return "", "", errors.New("making same cg near remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	{
		remapLine, parentLine, err := MakeRemapLineVia(crc, cdnDomain, FarSubdomain, cgServers, svName, sv, remapSvName, remapSv, farCG, healthPort, comments)
		if err != nil {
			return "", "", errors.New("making same cg far remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	return remapLines, parentLines, nil
}

// MakeRemapLinesDifferentCacheGroup is used to generate remaps to a server in this server's own CacheGroup (i.e. directly to that server, not to a deterministic "cache monitor" server in that server's CG).
func MakeRemapLinesOtherCacheGroup(
	crc *tc.CRConfig,
	cdnDomain string,
	cgServers map[tc.CacheGroupName][]string,
	svName tc.CacheName,
	sv *tc.CRConfigTrafficOpsServer,
	remapSvName tc.CacheName,
	remapSv *tc.CRConfigTrafficOpsServer,
	healthPort int,
	comments bool,
) (string, string, error) {
	remapLines := ``
	parentLines := ``
	{
		if remapSv.CacheGroup == nil {
			return "", "", errors.New("server '" + string(svName) + "has null cachegroup in CRConfig")
		}
		remapLine, parentLine, err := MakeRemapLineVia(crc, cdnDomain, NearSubdomain, cgServers, svName, sv, remapSvName, remapSv, tc.CacheGroupName(*sv.CacheGroup), healthPort, comments)
		if err != nil {
			return "", "", errors.New("making other cg near remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	{
		remapLine, parentLine, err := MakeRemapLineDirect(crc, cdnDomain, FarSubdomain, svName, sv, remapSvName, remapSv, healthPort, comments)
		if err != nil {
			return "", "", errors.New("making other cg far remap line: " + err.Error())
		}
		remapLines += remapLine
		parentLines += parentLine
	}
	return remapLines, parentLines, nil
}

// GetCacheGroupDistances returns the distance of every CacehGroup in crc from the given cg.
// Returns the distance in meters (though callers shouldn't care the unit, as distances should only ever be used relatively).
func GetCacheGroupDistances(crc *tc.CRConfig, cg tc.CRConfigLatitudeLongitude) map[tc.CacheGroupName]float64 {
	dist := map[tc.CacheGroupName]float64{}
	for name, ll := range crc.EdgeLocations {
		dist[tc.CacheGroupName(name)] = DistanceMeters(ll.Lat, ll.Lon, cg.Lat, cg.Lon)
	}
	return dist
}

// GetFarCacheGroup gets a "far" cachegroup, which will be used to generate "far" remap lines.
// The "far" cachegroup is calculated deterministically (and must be), so it will always be the same for any given cachegroup, for the same CRConfig.
// The "far" cachegroup is determined as follows:
// 1. All cachegroups with a distance greater than the median are removed.
// 2. All cachegroups with fewer caches than the average number of caches in the remaining cachegroups are removed.
// 3. The remaining cachegroups are sorted alphabetically.
//    - The order doesn't matter, and  could be changed, as long as it's deterministic.
// 4. The first cachegroup in the remaining sorted list is chosen.
//
// An error is returned if cg is not in crc, or if no other cachegroups are in crc.
func GetFarCacheGroup(crc *tc.CRConfig, cg tc.CacheGroupName) (tc.CacheGroupName, error) {
	// TODO: add "midLocations" to CRConfig. Needed by GetServerFarCacheGroup to poll mids
	cgLatLon, ok := crc.EdgeLocations[string(cg)]
	if !ok {
		return "", errors.New("cg '" + string(cg) + " not in CRConfig!")
	}

	cgSizes := map[tc.CacheGroupName]int{}
	for svName, sv := range crc.ContentServers {
		// TODO exclude OFFLINE/ADMIN_DOWN?
		if sv.CacheGroup == nil {
			return "", errors.New(svName + "had null CacheGroup!")
		}
		cgSizes[tc.CacheGroupName(*sv.CacheGroup)] = cgSizes[tc.CacheGroupName(*sv.CacheGroup)] + 1
	}

	names := []string{}
	for name, _ := range crc.EdgeLocations {
		if name == string(cg) {
			continue
		}
		if cgSizes[tc.CacheGroupName(name)] == 0 {
			continue // skip cachegroups with no caches
		}
		names = append(names, name)
	}

	cgDists := map[string]float64{}
	// TODO change to a rolling average. Adding everything then dividing loses a lot of precision.
	avgDist := 0.0
	for _, name := range names {
		dist := DistanceMeters(crc.EdgeLocations[name].Lat, crc.EdgeLocations[name].Lon, cgLatLon.Lat, cgLatLon.Lon)
		avgDist += dist
		cgDists[name] = dist
	}
	avgDist /= float64(len(names))

	farNames := []string{}
	for _, name := range names {
		if dist := cgDists[name]; dist < avgDist {
			continue
		}
		farNames = append(farNames, name)
	}

	// Of the remaining cachegroups which are greater than the average distance:
	// Get the average size.

	avgFarSize := 0.0
	for _, name := range farNames {
		if cgSizes[tc.CacheGroupName(name)] == 0 {
			continue // skip CGs with no caches
		}
		avgFarSize += float64(cgSizes[tc.CacheGroupName(name)])
	}
	avgFarSize /= float64(len(farNames)) // TODO change to rolling average; Summing and dividing loses precision

	// Remove cachegroups with less than the average number of servers.
	// This helps eliminate CGs which are testing, beta, canary, etc,
	// As well as generally sending "far" requests to bigger CGs that can handle the requests better
	// (though the request cost is tiny)

	farBigNames := []string{}
	for _, name := range farNames {
		if cgSizes[tc.CacheGroupName(name)] < int(avgFarSize) {
			continue
		}
		farBigNames = append(farBigNames, name)
	}

	// TODO determine if the alphabetical first is usually/always the same for many cachegroups.
	// That is, are we using one CG as the "far" CG for a bunch of others?
	// If so, should we make the deterministic selection more distributed, e.g. a consistent hash?
	sort.Strings(farBigNames)

	if len(farBigNames) == 0 {
		return "", errors.New("no cachegroups besides this one in the CRConfig!")
	}

	return tc.CacheGroupName(farBigNames[0]), nil
}

// DistanceMeters calculates the distance in meters between lat1,lon1 and lat2,lon2.
// From https://gist.github.com/cdipaolo/d3f8db3848278b49db68
func DistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}

func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}
