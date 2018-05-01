# Doomsday

## Server configuration

Create a configuration manifest and start the server against it with
`doomsday server -m <pathtomanifest>`

The manifest should be written in YAML, and the schema looks like this:

`backend.type`: _(string, enum)_ Which kind of storage backend containing your
certs that the server should target. Currently only supports `vault`

`backend.address`: _(string)_ Where the storage backend's API is located

`backend.insecure_skip_verify`: _(bool)_ Set if the server should not verify
storage backend's CA cert.

`backend.auth`: _(hash)_ A backend specific authentication configuration.

&nbsp;&nbsp;&nbsp;&nbsp;
**For `vault` backends:**

&nbsp;&nbsp;&nbsp;&nbsp;
`backend.auth.token`: _(string)_ The authentication token to be used

`server.port`: _(int)_ The port that the doomsday server API listens on

`server.auth.type`: _(string, enum)_ The type of authentication to use for the server
API. The only valid value is currently `userpass`. Omit this key to have no
server auth.

`server.auth.config`: _(hash)_ Configuration for the selected auth type

&nbsp;&nbsp;&nbsp;&nbsp;
**For `userpass` auth:**

&nbsp;&nbsp;&nbsp;&nbsp;
`server.auth.config.username`: The username for the allowed user

`server.auth.config.password`: The password for the allowed user