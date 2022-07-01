# Doomsday

Doomsday is a server (and also a CLI) which can be configured to track
certificates from different storage backends (Vault, Credhub, Pivotal
Ops Manager, or actual websites) and provide a tidy view into when certificates
will expire. Doomsday provides no automation for renewal - Doomsday simply
provides the information required for maintainers to take action.

## Server configuration

Create a configuration manifest and start the server against it with
`doomsday server -m <pathtomanifest>`

The manifest should be written in YAML. An example schema with documentation
can be found at [docs/ddayconfig.yml](docs/ddayconfig.yml)

## Pushing to CloudFoundry

You'll want to make a directory that has three
files.

* A binary of doomsday for the correct operating system
* A doomsday server configuration manifest
* A cf application manifest for deploying doomsday

The binary can be found at the releases page for this Github repo.

An example manifest can be found at
[docs/ddayconfig.yml](docs/ddayconfig.yml). Omit the `server.port` property
from the manifest. This will cause the server to look for the `PORT` environment
variable for which port to have the API listen on (which is what CF wants).

The cf application manifest will probably look something like this,
assuming that your binary is called `doomsday`, and your configuration
manifest is called `doomsdayconf.yml`.

```yml
---
applications:
  - name: doomsday
    memory: 256M
    instances: 1
    command: ./doomsday server -m doomsdayconf.yml
    buildpack: binary_buildpack
```

Then, if your cf app manifest is called `manifest.yml`, run

```sh
cf push -f manifest.yml
```

## Development Notes

Make sure your GOPATH has your $HOME/go and this directory in its settings.
We are assuming the GOROOT has been set when you installed the go package.

```sh
export GOPATH="$HOME/go:$PWD"
```

