// Package pluginsdk is the gateway plugin SDK, generation 2.
//
// v1 handed a plugin a Host facade and a unix-socket RPC wire of the gateway's
// own design. v2 replaces both: a plugin embeds plugin.Base, reaches the
// gateway through the one bus.Bus it inherits, and talks to the broker over
// standard NATS on the socket the host splices. What used to be a private wire
// is now a documented floor any language with a NATS client can meet.
//
// This module is the GATEWAY half of the contract: the handshake, the
// transport, the packaging tool and the aliases. The contract types themselves
// — bus, plugin, the typed layer, the manifest — live in
// bytedesk-sdk-dependencies/v2 and are re-exported here so a plugin author
// imports one module.
//
// A spawned plugin's whole lifecycle is ServePlugin. Everything below it —
// waiting for credentials, dialling, negotiating, binding, mounting lifecycle
// endpoints, serving HTTP — is arranged so that a plugin author writes Start
// and Stop and nothing else.
package pluginsdk
