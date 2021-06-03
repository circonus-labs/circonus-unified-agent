# Running Circonus Unified Agent as a Windows Service

The agent natively supports running as a Windows Service. Outlined below is are
the general steps to set it up.

1. Obtain the agent windows distribution
2. Create the directory `C:\Program Files\Circonus Unified Agent` (if you install in a different
   location simply specify the `--config` parameter with the desired location)
3. Unzip the windows release into `C:\Program Files\Circonus Unified Agent`
4. Rename `etc\example-circonus-unified-agent_windows.conf` to `etc\circonus-unified-agent.conf`
5. Edit `etc\circonus-unified-agent.conf` - at a minimum add a valid Circonus API Token to `api_token` under the `[agent.circonus]` section
6. To install the service into the Windows Service Manager, run the following in PowerShell as an administrator (If necessary, you can wrap any spaces in the file paths in double quotes ""):

   ```
   > "C:\Program Files\Circonus Unified Agent\circonus-unified-agentd.exe" --service install
   ```

5. Edit the configuration file to meet your needs
6. To check that it works, run:

   ```
   > "C:\Program Files\Circonus Unified Agent\circonus-unified-agentd.exe" --config="C:\Program Files\Circonus Unified Agent\etc\circonus-unified-agent.conf" --test
   ```

7. To start collecting data, run:

   ```
   > net start circonus-unified-agent
   ```

## Config Directory

You can also specify a `--config-directory` for the service to use:
1. Create a directory for config snippets: `C:\Program Files\Circonus Unified Agent\etc\config.d`
2. Include the `--config-directory` option when registering the service:
   ```
   > "C:\Program Files\Circonus Unified Agent\circonus-unified-agentd.exe" --service install --config="C:\Program Files\Circonus Unified Agent\circonus-unified-agent.conf" --config-directory="C:\Program Files\Circonus Unified Agent\etc\config.d"
   ```

## Other supported operations

The agent can manage its own service through the --service flag:

| Command                                           | Effect                        |
|---------------------------------------------------|-------------------------------|
| `circonus-unified-agentd.exe --service install`   | Install the service           |
| `circonus-unified-agentd.exe --service uninstall` | Remove the service            |
| `circonus-unified-agentd.exe --service start`     | Start the service             |
| `circonus-unified-agentd.exe --service stop`      | Stop the service              |

## Install multiple services

Running multiple instances of the agent is seldom needed, as you can run
multiple instances of each plugin and route metric flow using the metric
filtering options.  However, if you do need to run multiple agent instances
on a single system, you can install the service with the `--service-name` and
`--service-display-name` flags to give the services unique names:

```
> "C:\Program Files\Circonus Unified Agent\circonus-unified-agentd.exe" --service install --service-name circonus-unified-agent-1 --service-display-name "Circonus Unified Agent 1"
> "C:\Program Files\Circonus Unified Agent\circonus-unified-agentd.exe" --service install --service-name circonus-unified-agent-2 --service-display-name "Circonus Unified Agent 2"
```

## Troubleshooting

When the agent runs as a Windows service, it logs messages to Windows events log before configuration file with logging settings is loaded.
Check event log for an error reported by `circonus-unified-agent` service in case the agent service reports failure on its start: Event Viewer->Windows Logs->Application

**Troubleshooting  common error #1067**

When installing as service in Windows, always double check to specify full path of the config file, otherwise windows service will fail to start

 `--config "C:\Program Files\Circonus Unified Agent\etc\circonus-unified-agent.conf"`
