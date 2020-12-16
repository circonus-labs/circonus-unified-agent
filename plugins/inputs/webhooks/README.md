# Webhooks Input Plugin

This is a service plugin that start an http server and register multiple webhook listeners.

```sh
$ circonus-unified-agent config -input-filter webhooks -output-filter circonus > config.conf.new
```

Change the config file to point to add Circonus API token key you are using and adjust the settings to match your environment. Once that is complete:

```sh
$ cp config.conf.new /etc/circonus-unified-agent/circonus-unified-agent.conf
$ sudo service circonus-unified-agent start
```


### Configuration:

```toml
[[inputs.webhooks]]
  ## Address and port to host Webhook listener on
  service_address = ":1619"

  [inputs.webhooks.filestack]
    path = "/filestack"

  [inputs.webhooks.github]
    path = "/github"
    # secret = ""

  [inputs.webhooks.mandrill]
    path = "/mandrill"

  [inputs.webhooks.rollbar]
    path = "/rollbar"

  [inputs.webhooks.papertrail]
    path = "/papertrail"

  [inputs.webhooks.particle]
    path = "/particle"
```


### Available webhooks

- [Filestack](filestack/)
- [Github](github/)
- [Mandrill](mandrill/)
- [Rollbar](rollbar/)
- [Papertrail](papertrail/)
- [Particle](particle/)


### Adding new webhooks plugin

1. Add your webhook plugin inside the `webhooks` folder
1. Your plugin must implement the `Webhook` interface
1. Import your plugin in the `webhooks.go` file and add it to the `Webhooks` struct

Both [Github](github/) and [Rollbar](rollbar/) are good example to follow.
