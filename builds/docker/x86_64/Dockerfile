FROM alpine:3.12
RUN echo 'hosts: files dns' >> /etc/nsswitch.conf
RUN apk add --no-cache iputils ca-certificates net-snmp-tools procps lm_sensors && \
    update-ca-certificates

COPY sbin/circonus-unified-agentd /
COPY etc/example-circonus-unified-agent.conf /etc/circonus-unified-agent/circonus-unified-agent.conf

# control default plugins in a container - disable but allow override
# set to any other value than "false" to enable default plugins.
ENV ENABLE_DEFAULT_PLUGINS=false

EXPOSE 8125/udp 8092/udp 8094

ENTRYPOINT ["/circonus-unified-agentd"]
