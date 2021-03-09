# Circonus Unified Agent Installer

A small, basic installer script for the circonus unified agent.

## Notes

* Currently works with `rpm` and `deb` packages for amd64, see [latest release](https://github.com/circonus-labs/circonus-unified-agent/releases/latest)
* A valid Circonus API token key is required
* Requires access to the internet ([github](github.com) and Circonus)
* Works with Circonus SaaS (not inside deployments)
* More options and flexibility will be added as needed

## Options

```sh
Circonus Unified Agent Install Help

Usage

  install.sh --key <apikey>

Options

  --key           Circonus API key/token **REQUIRED**
  [--app]         Circonus API app name (authorized w/key) Default: circonus-unified-agent
  [--help]        This message

Note: Provide an authorized app for the key or ensure api
      key/token has adequate privileges (default app state:allow)
```

## Examples

### With only a key (ensure, key has default app state set to allow)

```sh
curl -sSL "https://raw.githubusercontent.com/circonus-labs/circonus-unified-agent/master/install/install.sh" | bash -s -- --key <circonus api key>
```

### With key and explicit app name already allowed with key

```sh
curl -sSL "https://raw.githubusercontent.com/circonus-labs/circonus-unified-agent/master/install/install.sh" | bash -s -- --key <circonus api key> --app <app named for key>
```

## Docker

1. Create configuration file (e.g. use example from the [repository](https://github.com/circonus-labs/circonus-unified-agent/tree/master/etc))
2. In `outputs.circonus`:
    a. Set `api_token`
    b. Set `check_name_prefix` to ensure the check target can be found again when container is restarted - agent will container's "hostname" as the check target by default...

Use one of the following commands:

```sh
docker run -d --name=circonus-unified-agent \
    -v $PWD/circonus-unified-agent.conf:/etc/circonus-unified-agent/circonus-unified-agent.conf:ro \
    circonus/circonus-unified-agent
```

or

```sh
docker run -d --name=circonus-unified-agent \
    --mount type=bind,src=$PWD/circonus-unified-agent.conf,dst=/etc/circonus-unified-agent/circonus-unified-agent.conf \
    circonus/circonus-unified-agent
```

**NOTE:** To collect HOST metrics from within the container, use one of the following commands:

```sh
docker run -d --name=circonus-unified-agent \
    -v $PWD/circonus-unified-agent.conf:/etc/circonus-unified-agent/circonus-unified-agent.conf:ro \
    -v /:/hostfs:ro \
    -e HOST_ETC=/hostfs/etc \
    -e HOST_PROC=/hostfs/proc \
    -e HOST_SYS=/hostfs/sys \
    -e HOST_VAR=/hostfs/var \
    -e HOST_RUN=/hostfs/run \
    -e HOST_MOUNT_PREFIX=/hostfs \
    -e ENABLE_DEFAULT_PLUGINS=true \
    circonus/circonus-unified-agent
```

or

```sh
docker run -d --name=circonus-unified-agent \
    --mount type=bind,src=$PWD/circonus-unified-agent.conf,dst=/etc/circonus-unified-agent/circonus-unified-agent.conf \
    -v /:/hostfs:ro \
    -e HOST_ETC=/hostfs/etc \
    -e HOST_PROC=/hostfs/proc \
    -e HOST_SYS=/hostfs/sys \
    -e HOST_VAR=/hostfs/var \
    -e HOST_RUN=/hostfs/run \
    -e HOST_MOUNT_PREFIX=/hostfs \
    -e ENABLE_DEFAULT_PLUGINS=true \
    circonus/circonus-unified-agent
```
