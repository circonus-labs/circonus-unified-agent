# CUA Env vars


## Windows

- "ProgramFiles" - if unset, logs an error
- "WINDIR" - if unset, logs an error

## Linux

- "HOST_PROC" - returns "/proc" if unset.
- "HOST_MOUNT_PREFIX" - plugins/inputs/system/ps used for host volume mounts

## Universal

- "ENABLE_DEFAULT_PLUGINS" - if set to "false", disables default plugins
- "CUA_CONFIG_PATH" - if set, overrides any other config file
- "ECS_CONTAINER_METADATA_URI" - if set, enables ecs v3 endpoint
- "DOCKER_HOST" - if unset, defaults to "localhost"
