# Doomsday

## Server configuration

Create a configuration manifest and start the server against it with
`doomsday server -m <pathtomanifest>`

The manifest should be written in YAML, and the schema looks like this:

`backend.type`: _(string, enum)_ Which kind of storage backend containing your
certs that the server should target. Currently supports `vault`, `opsman`, or `credhub`.

`backend.address`: _(string)_ Where the storage backend's API is located

`backend.insecure_skip_verify`: _(bool)_ Set if the server should not verify
storage backend's CA cert.

`backend.auth`: _(hash)_ A backend specific authentication configuration.

`backend.config`: _(hash)_ Additional configuration
specific to a backend

### For `vault` storage backends

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.token`: _(string)_ The authentication token to be used

`backend.config.base_path`: _(string)_ The root of the
secret tree to begin searching from. Defaults to `secret`.

### For `opsman` or `credhub` storage backends

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.grant_type`: _(string,enum)_ The type of oauth grant to
 perform to the backend. One of `client_credentials` or `password`

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.client_id`: _(string)_ The client id for the oauth
client to authenticate as.

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.client_secret`: _(string)_ The client secret for the
oauth client to authenticate as.

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.username`: _(string)_ The username to authenticate as.
Only required if using the `password` grant type.

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.password`: _(string)_ The password for the user to
 authenticate with. Only required if using the `password` grant type.

`server.port`: _(int)_ The port that the doomsday server API listens on. If this isn't set, then it defaults to the PORT
environment variable.

`server.auth.type`: _(string, enum)_ The type of authentication to use for the server
API. The only valid value is currently `userpass`. Omit this key to have no
server auth.

`server.auth.config`: _(hash)_ Configuration for the selected auth type

&nbsp;&nbsp;&nbsp;&nbsp;
**For `userpass` auth:**

&nbsp;&nbsp;&nbsp;&nbsp;
`server.auth.config.username`: The username for the allowed user

`server.auth.config.password`: The password for the allowed user

## Pushing to CloudFoundry

You'll want to make a directory that has three
files.

* A binary of doomsday for the correct operating system
* A doomsday server configuration manifest
* A cf application manifest for deploying doomsday

The binary can be found at the releases page for this Github repo.

The server configuration manifest might look something like this,
although you'll need to use the above documentation to tweak it for
your needs.

```yml
---
backend:
  type: vault
  address: https://127.0.0.1:8200
  insecure_skip_verify: true
  auth:
    token: a-vault-root-token

server:
  auth:
    type: userpass
    config:
      username: doomsday
      password: password
```

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

## Development

This project uses https://github.com/kardianos/govendor for vendoring.