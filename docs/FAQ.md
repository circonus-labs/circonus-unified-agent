# Frequently Asked Questions

### Q: How can I monitor the Docker Engine Host from within a container?

You will need to setup several volume mounts as well as some environment
variables:

```shell
docker run --name cua \
    -v /:/hostfs:ro \
    -e HOST_ETC=/hostfs/etc \
    -e HOST_PROC=/hostfs/proc \
    -e HOST_SYS=/hostfs/sys \
    -e HOST_VAR=/hostfs/var \
    -e HOST_RUN=/hostfs/run \
    -e HOST_MOUNT_PREFIX=/hostfs \
    circonus-unified-agent
```

### Q: Why do I get a "no such host" error resolving hostnames that other programs can resolve?

Go uses a pure Go resolver by default for [name resolution](https://golang.org/pkg/net/#hdr-Name_Resolution).
This resolver behaves differently than the C library functions but is more
efficient when used with the Go runtime.

If you encounter problems or want to use more advanced name resolution methods
that are unsupported by the pure Go resolver, you can switch to the cgo
resolver.

If running manually set:

```shell
export GODEBUG=netdns=cgo
```

If running as a service add the environment variable to `/opt/circonus/unified-agent/etc/circonus-unified-agent.env`:

```shell
GODEBUG=netdns=cgo
```
