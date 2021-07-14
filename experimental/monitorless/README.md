# Monitorless Monitoring

This proof-of-concept aims to replace the Traffic Monitor in Traffic Control with ATS remap rules.

## Benefits

By using ATS to do the expensive, difficult to write, high-performance things the Monitor currently does, developers on Traffic Control don't have to solve the difficult high-performance problems that ATS already solves.

In addition, this also eliminates a highly CPU- and network-intensive application from production deployments.

# Apps

Monitorless Monitoring consists of 3 applications:

1. `astatstwo` replaces the astats plugin, returning health data.
2. `remapgen` generates the remap.config rules needed for routing health requests through ATS
3. `healthcombiner` aggregates health from all caches into the legacy /CrStates.json consumed by Traffic Router.
    - this is not strictly required, TR could make requests for each cache directly to ATS. But this allows Monitorless Monitoring to work without modifying Traffic Router.

# Instructions

The Proof-of-Concept creates Dockerfiles performing the ATS remapping and "monitorless monitoring."


## Prerequisites

You'll need Docker, Go, and `jq` installed locally.
You'll also need a Apache Traffic Server 8.0 RPM in this directory named `trafficserver.rpm`.
    - Should work with an RPM that installs at /opt/trafficserver
    - I tried to make it  work with https://ci.trafficserver.apache.org/RPMS/CentOS7/trafficserver-8.0.3-1.el7.x86_64.rpm which installs to / and uses dynamic linking, but I haven't gotten the dynamic linking working in Docker yet.
        - Note that is an unstable development build! The ATS project does not provide Release RPMs. If you need Traffic Server for any kind of real or production use, it is highly recommended to build it from a release source.

## Installation

```sh
(cd remapgen && env GOOS=linux GOARCH=amd64 go build)
(cd astatstwo && env GOOS=linux GOARCH=amd64 go build)
(cd healthcombiner && env GOOS=linux GOARCH=amd64 go build)

docker build --no-cache --rm --tag mm:0.1 .
docker network create mm
./create-containers.sh
```

## Cleanup

```sh
./create-containers.sh clean
docker rmi mm:0.1
docker network remove mm
rm remapgen/remapgen
rm astatstwo/astatstwo
rm healthcombiner/healthcombiner
```

## Example Commands

Example commands to play with the health protocol.

Entering a container:
`docker exec -it houston-ec0 /bin/bash`

Inside one container, requesting the health of another:
`curl -Lvsk -H 'Host: near.health.mm-seattle-ec2.cdn.test:8081' http://localhost:8081`
  - observe the Via header, to see which caches were requested.

Requesting the `astatstwo` service:
`curl -Lvsk -H http://localhost:8089`

Requesting the `healthcombiner` service:
`curl -Lvsk -H http://localhost:8088/CRStates.json`

Set a container's `astatstwo` service to serve Unavailable (503):n
`curl -Lvsk -X POST http://localhost:8083/debug?available=false`

See what the remap rules look like (they include detailed comments as to what was chosen and why):
`cat /opt/trafficserver/etc/trafficserver/remap.config`

See what the parent rules look like (also include detailed comments):
`cat /opt/trafficserver/etc/trafficserver/parent.config`

Containers are built from the CRConfig.json in this directory. You should be able to add and rename servers and cachegroups, to change how containers are created.
    - Note remap rules don't work for servers with different ports yet. All servers need the same port, for now.
    - Note create-containers.sh doesn't use the IPs, but rather copies the file and overwrites them with the dynamically-created IPs of the Docker containers.

You can also change `health_port` and `crstates_port` in create-containers.sh, and it should work.
