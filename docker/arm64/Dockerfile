FROM arm64v8/alpine:latest
RUN echo 'hosts: files dns' >> /etc/nsswitch.conf
RUN apk add --no-cache iputils ca-certificates net-snmp-tools procps lm_sensors && \
    update-ca-certificates

COPY circonus-unified-agentd /
COPY etc/example-circonus-unified-agent.conf /etc/circonus-unified-agent/circonus-unified-agent.conf

EXPOSE 8125/udp 8092/udp 8094

ENTRYPOINT ["/circonus-unified-agentd"]