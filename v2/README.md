# bytedesk-remote-gateway-plugin-sdk/v2

The gateway plugin SDK, generation 2. A plugin embeds `plugin.Base`, reaches
the gateway through the one `bus.Bus` it inherits, and talks to the broker over
standard NATS on the socket the host splices.

The contract types — `bus`, `plugin`, the typed layer, the manifest — live in
`bytedesk-sdk-dependencies/v2` and are re-exported here, so a plugin author
imports one module.

## What is here

| Package | What it is |
|---|---|
| `.` | aliases, the v2 handshake (`ServePlugin`), HTTP serving, packaging |
| `transport/natsconn` | the NATS transport, and the **only** importer of `github.com/nats-io/*` |
| `v1compat` | the v1 `Host` facade over a bound v2 `Base`, for migrating plugins |
| `cmd/plugin-sdk` | `validate`, `pack` (prints the grants digest), `digest`, `mcp` |
| `internal/fakebroker` | a hand-written NATS peer for the SDK's own tests |

## Writing a plugin

```go
type Files struct {
	pluginsdk.Base // Bus(), Logger(), Profiling(), StateDir(), Identity()
}

func (p *Files) ID() string                     { return "files" }
func (p *Files) Manifest() pluginsdk.Manifest   { return manifest }
func (p *Files) Start(ctx context.Context) error {
	// Bus() is already live: Bind ran before Start, which is why Start takes
	// no host argument.
	_, err := p.Bus().Services().Serve(ctx, spec)
	return err
}
func (p *Files) Stop(ctx context.Context) error { return nil }

func main() {
	if err := pluginsdk.ServePlugin(context.Background(), &Files{}, pluginsdk.PluginConfig{}); err != nil {
		log.Fatal(err)
	}
}
```

## The handshake

`ServePlugin` runs one ordered sequence, and the order is load bearing:

```
identity → manifest → wait ≤5s for credentials → dial → negotiate →
CheckProtocol → Bind → Validate → Start → lifecycle endpoints → HTTP
```

Nothing that can refuse happens after `Start`, and nothing that needs the bus
happens before `Bind`. A failure at any step stops the steps that ran before it.

The host sets four environment variables. `GATEWAY_PLUGIN_SOCKET` and
`GATEWAY_PLUGIN_ID` are unchanged from v1; `GATEWAY_BUS_SOCKET` and
`GATEWAY_BUS_CREDS` are new and both name paths inside the plugin's own run
directory, so the containment bind list does not change.

The credential wait exists because the host writes the `.creds` file **after**
the containment receipt commits. A credential that never arrives is a
`FaultDenied`, not a hang.

## Lifecycle hooks are endpoints now

v1 negotiated a hook list and the host called each hook over a bespoke HTTP
command. v2 mounts them as ordinary service endpoints in the plugin's own
namespace:

```
svc.<id>.lifecycle.v1.activation.check
svc.<id>.lifecycle.v1.ready
svc.<id>.lifecycle.v1.health
```

Implement `ActivationChecker`, `Readier` or `HealthContributor` and the endpoint
appears; implement none and no lifecycle service is mounted at all.

## What rc.1 does not implement yet

`transport/natsconn` implements core pub/sub, request/reply, services and
correlation. It does **not** implement streams, KV, objects or schedules.

They are reported `false` by `Capabilities()` rather than stubbed true, so a
manifest whose `needs` names one fails **closed at the handshake**, naming the
capability — not at first use, with a nil dereference. The JetStream surfaces
land with the durable re-platforms; implementing them is a change to
`TransportCapabilities()` and the four surface types, and nothing else in the
SDK moves.

Logging goes to stderr, which the host captures. A host-driven profiling switch
arrives when the gateway adopts v2 and defines the subject that carries it;
until then `Profiling()` is off unless a caller passes `PluginConfig.Profiler`.

## The import boundary

No `nats` type appears in any exported signature anywhere in this module, and
`github.com/nats-io/*` is imported by exactly one package. Both halves are
tested, and both tests are accompanied by one that breaks them on purpose:
`TestImportScanDetectsAForbiddenImport` and `TestReflectWalkDetectsAForeignType`.
A fence that cannot detect its own vacuity is not a fence.

Unexported fields are deliberately outside the rule. `natsconn.Conn` holds a
`*nats.Conn` privately; that is how the abstraction works, not a hole in it.
