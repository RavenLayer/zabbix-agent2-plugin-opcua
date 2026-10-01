# Zabbix Agent 2 plugin for OPC UA

This plugin provides a native Zabbix solution to monitor industrial automation and IoT devices via the **OPC Unified Architecture (OPC UA)** binary protocol (`opc.tcp`).

It communicates directly with OPC UA servers to test connectivity, collect server runtime metrics, monitor server certificate expiration, read arbitrary NodeIDs, and discover variables/nodes dynamically using Zabbix Low-Level Discovery (LLD).

The plugin supports multiple OPC UA servers simultaneously using either connection parameters or pre-configured named sessions.

## Requirements

* Zabbix Agent 2 version 6.0.0 or newer
* OPC UA server supporting the binary protocol (`opc.tcp`)
* If building from source: Go version 1.22 or newer

## Plugin Setup

The `Plugins.OPCUA.System.Path` configuration directive must be set in the Zabbix Agent 2 configuration file to point to the OPC UA plugin executable. By default, this is defined in `opcua.conf` and included in `zabbix_agent2.conf`.

Example configuration in `opcua.conf`:

```ini
Plugins.OPCUA.System.Path=/usr/sbin/zabbix-agent2-plugin/zabbix-agent2-plugin-opcua
```

Then, include the configuration file in the main Zabbix Agent 2 configuration file (`zabbix_agent2.conf`):

```ini
Include=/etc/zabbix/zabbix_agent2.d/plugins.d/opcua.conf
```

## Options

The OPC UA plugin binary can be run independently with the following flags:

* `-h`, `--help` displays help message
* `-V`, `--version` displays plugin version and license information

## Configuration

To configure the plugin, use `opcua.conf` (or `zabbix_agent2.conf`).

### Global Plugin Options

* `Plugins.OPCUA.Timeout` — maximum time in seconds to wait for an OPC UA server to respond to requests.  
  *Default value:* inherits global agent `Timeout` (limits: 1–30 sec).

* `Plugins.OPCUA.DiscoveryMaxDepth` — maximum recursion depth for the `opcua.discovery` key.  
  *Default value:* `3`

### Session Options

Each session is configured under `Plugins.OPCUA.Sessions.<session_name>.*` or as defaults under `Plugins.OPCUA.Default.*`:

* `Uri` — OPC UA endpoint URI to connect to.  
  *Default value:* `opc.tcp://localhost:4840`  
  *Supported scheme:* `opc.tcp`

* `User` — Username for authentication.  
  *Default value:* empty (connects anonymously if omitted).

* `Password` — Password for authentication.  
  *Default value:* empty (limits: up to 512 characters).

* `SecurityMode` — Message security mode.  
  *Default value:* `None`  
  *Accepted values:* `None`, `Sign`, `SignAndEncrypt`

* `SecurityPolicy` — Security policy URI or short name.  
  *Default value:* `None`  
  *Accepted values:* `None`, `Basic128Rsa15`, `Basic256`, `Basic256Sha256`, `Aes128_Sha256_RsaOaep`, `Aes256_Sha256_RsaPss`, or a full security policy URI.

* `CertFile` — Full pathname of client X.509 certificate file (`.crt` or `.pem`).  
  *Default value:* empty

* `KeyFile` — Full pathname of client private RSA key file (`.key` or `.pem`).  
  *Default value:* empty

### Configuring Connections

A connection can be configured using either **key parameters** or **named sessions**.

#### 1. Using Key Parameters

The first three parameters for all keys are:

```
[ConnString][,User][,Password]
```

* `ConnString` can be either an endpoint URI (e.g. `opc.tcp://192.168.1.10:4840`) or a named session (configured in `opcua.conf`).
* If `ConnString` is a named session, `User` and `Password` are ignored (the session's configuration is used instead).
* If omitted, the default URI `opc.tcp://localhost:4840` is used.

*Examples*:

```bash
# Using direct endpoint URI:
zabbix_agent2 -t opcua.ping[opc.tcp://192.168.1.10:4840]

# Using credentials:
zabbix_agent2 -t opcua.get[opc.tcp://192.168.1.10:4840,opcuser,secret,"ns=2;s=Device1.Temperature"]

# Using a named session:
zabbix_agent2 -t opcua.ping[PLC1]
```

#### 2. Using Named Sessions

Named sessions allow you to configure credentials and security parameters per server in `opcua.conf`:

```ini
# Anonymous connection
Plugins.OPCUA.Sessions.PLC1.Uri=opc.tcp://192.168.1.10:4840

# User/Password authentication with SignAndEncrypt
Plugins.OPCUA.Sessions.ScadaServer.Uri=opc.tcp://scada.corp.local:4840
Plugins.OPCUA.Sessions.ScadaServer.User=operator
Plugins.OPCUA.Sessions.ScadaServer.Password=TopSecret123
Plugins.OPCUA.Sessions.ScadaServer.SecurityMode=SignAndEncrypt
Plugins.OPCUA.Sessions.ScadaServer.SecurityPolicy=Basic256Sha256
Plugins.OPCUA.Sessions.ScadaServer.CertFile=/etc/zabbix/certs/opcua_client.crt
Plugins.OPCUA.Sessions.ScadaServer.KeyFile=/etc/zabbix/certs/opcua_client.key
```

Keys can then use the session name as the first parameter:

```bash
zabbix_agent2 -t opcua.ping[ScadaServer]
zabbix_agent2 -t opcua.info[ScadaServer]
```

## Supported Keys

### `opcua.ping`

```
opcua.ping[<ConnString>[,User][,Password]]
```

Tests if the connection to the OPC UA server is alive by attempting to read node `ns=0;i=2259` (`ServerStatus.State`).

*Returns:*
* `1` — Connection is alive and server state is Good.
* `0` — Connection failed or server is unreachable.

### `opcua.info`

```
opcua.info[<ConnString>[,User][,Password]]
```

Fetches server diagnostic metrics and certificate expiration data formatted as JSON.

*JSON response fields:*
* `state` — Server status name (`Running`, `Failed`, `NoConfiguration`, `Suspended`, `Shutdown`, `Test`, `CommunicationFault`, `Unknown`)
* `state_code` — Numeric server state code (0: Running, 1: Failed, ...)
* `start_time` — Server start time (RFC3339)
* `current_time` — Server current time (RFC3339)
* `current_session_count` — Number of currently active client sessions
* `current_subscription_count` — Number of active subscriptions
* `certificate` — Server certificate details (if presented by the endpoint):
  * `subject` — Certificate subject
  * `issuer` — Certificate issuer
  * `not_before` — Validity start date
  * `not_after` — Expiration date
  * `days_until_expiration` — Number of days remaining before expiration

*Example output:*

```json
{
  "state": "Running",
  "state_code": 0,
  "start_time": "2026-01-01T08:00:00Z",
  "current_time": "2026-09-10T10:00:00Z",
  "current_session_count": 3,
  "current_subscription_count": 2,
  "certificate": {
    "subject": "CN=OPCUA Server, O=Example Inc",
    "issuer": "CN=Example Root CA",
    "not_before": "2025-01-01T00:00:00Z",
    "not_after": "2027-01-01T00:00:00Z",
    "days_until_expiration": 113
  }
}
```

### `opcua.get`

```
opcua.get[<ConnString>[,User][,Password],NodeID[,OutputFormat]]
```

Reads the value and (optionally) metadata of a single OPC UA NodeID.

*Parameters:*
* `NodeID` (mandatory): Target NodeID string (e.g. `ns=0;i=2258`, `ns=2;s=Factory.Line1.Temperature`).
* `OutputFormat` (optional): Output format selector:
  * `value`: Returns the raw node value directly (string, integer, float, boolean, etc.).
  * `json`: Returns a JSON object with raw value and metadata info.

*JSON response fields (`OutputFormat = json`):*
* `value` — Node value (or `null` if the node status is not Good)
* `status` — Lowercase OPC UA status code name (e.g. `good`, `badsensorfailure`, `badtimeout`)
* `status_code` — Numeric unsigned 32-bit status code (0 for Good)
* `severity` — Quality severity: `good`, `uncertain`, or `bad`
* `data_type` — Lowercase OPC UA variant data type (e.g. `float`, `double`, `int32`, `string`, `boolean`)
* `source_timestamp` — Source timestamp as numeric Unix epoch seconds (or 0 if not provided)
* `server_timestamp` — Server timestamp as numeric Unix epoch seconds (or 0 if not provided)

*Returns:* Raw node value (string, integer, float, boolean, etc.) or JSON formatted data.

*Example output (`OutputFormat = json`):*

```json
{
  "value": 42.5,
  "status": "good",
  "status_code": 0,
  "severity": "good",
  "data_type": "float",
  "source_timestamp": 1727169300,
  "server_timestamp": 1727169300
}
```

### `opcua.discovery`

```
opcua.discovery[<ConnString>[,User][,Password][,RootNodeID][,NodeClassFilter]]
```

Recursively browses hierarchical references starting from `RootNodeID` to produce Zabbix Low-Level Discovery (LLD) JSON.

*Parameters:*
* `RootNodeID` (optional): Root NodeID to begin browsing (default: `ns=0;i=85` — ObjectsFolder).
* `NodeClassFilter` (optional): Node class filter. Options: `Variable` (default), `Object`, `Method`, `All`.

*Returns:* Zabbix LLD JSON containing `{#NODEID}` and `{#NODENAME}` macros.

*Example output:*

```json
[
  {"{#NODEID}": "ns=2;s=Temperature", "{#NODENAME}": "Temperature"},
  {"{#NODEID}": "ns=2;s=Pressure", "{#NODENAME}": "Pressure"}
]
```

## Templates

An example template is provided under `templates/zbx_export_templates.xml` (**Template OPC Unified Architecture by Zabbix agent 2**).

*Note that this template is just an example demonstrating how metrics, server diagnostics, and certificate monitoring can be collected and structured with the plugin.*

### Template Macros

| Macro | Default | Description |
|---|---|---|
| `{$OPCUA.URI}` | `opc.tcp://127.0.0.1:4840` | OPC UA server endpoint URI or session name |
| `{$OPCUA.USER}` | *(empty)* | Username for OPC UA authentication |
| `{$OPCUA.PASSWORD}` | *(secret text)* | Password for OPC UA authentication |

## Build from Source

```bash
# Build binary
make build

# Run unit tests
make test

# Install binary and default configuration
sudo make install
```

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Disclaimer

**RavenLayer** is an independent software provider and is not affiliated with, sponsored by, or endorsed by **Zabbix LLC** or the **OPC Foundation**. Zabbix is a trademark of Zabbix LLC. OPC UA is a trademark of the OPC Foundation. The software is provided "as is", without warranty of any kind.
