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

## Development

This project uses https://github.com/kardianos/govendor.