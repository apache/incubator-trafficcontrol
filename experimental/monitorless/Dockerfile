# docker run must:
#
# - pass in the CRConfig as a volume to /external/CRConfig.json
#   e.g. '--volume $(pwd)/CRConfig.json:/CRConfig.json'
#   - Note the container copies the file, and promises not to change the host file.
#     Unfortunately, it's not possible in Docker to copy in a file or unmount a volume.
#
# - pass the environment variable HEALTH_PORT
#   e.g. '--env HEALTH_PORT=8089'
#
# - pass the environment variable CRSTATES_PORT
#   e.g. '--env CRSTATES_PORT=8088'
#
FROM centos/systemd

COPY remapgen/remapgen /
COPY astatstwo/astatstwo /
COPY healthcombiner/healthcombiner /
# TODO add build arg for rpm file
COPY trafficserver.rpm /
COPY logging.yaml /
COPY Dockerfile_run.sh /

RUN yum update -y
RUN yum install -y epel-release
RUN yum install -y jq
RUN yum install -y initscripts net-tools # debug
RUN yum install -y trafficserver.rpm
RUN mkdir -p /opt/trafficserver/etc/trafficserver/
RUN mkdir -p /etc/trafficserver/
RUN cp logging.yaml /opt/trafficserver/etc/trafficserver/ # handle both /opt/ and / installs
RUN cp logging.yaml /etc/trafficserver/

RUN printf $'CONFIG proxy.config.http.parent_proxy_routing_enable INT 1\n\
CONFIG proxy.config.http.insert_response_via_str INT 3\n\
CONFIG proxy.config.http.insert_request_via_str INT 1\n\
' > /opt/trafficserver/etc/trafficserver/records.config

RUN cp  /opt/trafficserver/etc/trafficserver/records.config /etc/trafficserver/records.config


RUN mkdir -p /external
RUN touch /external/CRConfig.json # file needs to exist, so when it's mounted as a volume, it's mounted as a file not a dir

CMD /Dockerfile_run.sh
