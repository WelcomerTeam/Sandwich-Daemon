# Sandwich Daemon

<img src="https://raw.githubusercontent.com/WelcomerTeam/Sandwich-Daemon/master/assets/icon.svg" width="500"/>

## Install Notes

- `gci` => ``go install github.com/daixiang0/gci@latest``
- `gofumpt` => ``go install mvdan.cc/gofumpt@latest``
- `goimports` => ``go install golang.org/x/tools/cmd/goimports@latest``

Sandwich Daemon is a utility that handles gateway connections, state and event processing. It handles the events from discord, handles filtering out events you do not want, stores users, members, guilds etc. in its internal store then sends the resulting data into a message queue for further handling from consumers.

Easily create new managers, scale up utilising rolling restarts on already running bots with shardgroups and configure and manage everything with a dashboard.

Includes an easy to use GRPC interface for fetching information from Sandwich including (but not limited to) searching for members by ID or name, fetching mutual guilds and returning what shard and manager a guild may be on.

Sandwich can filter out events on an absolute level so it will not even know it received the event and also the possibility to internally process (such as add to state) but then not publish to a consumer.

## Consuming

Consuming Sandwich Daemon events should be fairly similar to a regular discord payload however also include a few changes to both fit the purpose of sandwich and support forward compatibility.

A sandwich payload received by a consumer will be a JSON payload similar to a regular discord payload. Sandwich events will also include the exact same `op,d,s,t` keys from a regular discord payload and these should not be affected by sandwich between it receiving and it being published.

Any extra data that sandwich includes will be under the keys `__extra, __sandwich, __sandwich_trace`.

- `__extra`: This is the sandwich equivelant of the `d` key in the regular discord payload. This will include extra data that will be useful contextually, at the moment this will only be present on `_UPDATE` events and will include the previous state.
- `__sandwich_trace`: This will include trace times (will be introduced at a later point in time. This will be key pairs of map[string]int).
- `__sandwich`: This includes any metadata that will be useful for consumers identifying the origin of the message. Metadata includes the keys `v,i,a,s`

  - `__sandwich.v`: The current version of Sandwich.
  - `__sandwich.i`: The ProducerIdentifier of a manager.
  - `__sandwich.a`: The Application name of a manager. This will be the regular identifier.
  - `__sandwich.s`: This is a list of integers. This represents: `shardgroupID, shardID, shardCount`

  ```json
  {
    "op":0,
    "d": ...,
    "s":117,
    "t":"MESSAGE_CREATE",
    "__extra": null,
    "__sandwich":{
      "v":"1.0.1",
      "i":"welcomer",
      "a":"welcomer_dogfd",
      "s":[ 1, 0, 1 ]
    },
    "__sandwich_trace": null
  }
  ```

## Websocket support

This fork of Sandwich Daemon includes support for the `msg_websockets` messaging system.

## Virtual/Synthetic Sharding

In many cases, it is desirable to have a fixed number of consumers that does *not* vary with Discords shard count. This can lead to improved scaling etc. Also, using a fixed number of consumers solves the problem of resharding across consumers as only Sandwich needs to be resharded versus all consumers and allows better control over how many guilds are on each shard. While this is possible directly through Discord, doing so may constitute API abuse and at the very least uses up the identity limit. 

To solve this, Sandwich provides a feature called Virtual Sharding which allows setting a fixed number of virtual shards. When using Sandwich's Get Gateway Bot (or otherwise setting the virtual shard count on your bot), Sandwich remaps all events to virtual shards using the ``guild_id`` as identifier and ``(guild_id >> 22) % virtual_shard_count`` as the virtual shard ID. 

### Send Events

**Note that the below only applies when using msg_websockets as that is the only messaging system that supports Send Events in Sandwich**

Regarding send events, here are the semantics for the currently supported ones:
- ``Request Guild Members`` (chunking) will remap the ``guild_id`` to its real shard ID transparently. This means that all guild chunks will be dispatched correctly back to its virtual shard

When using virtual sharding, the following limitations apply:

- ``Update Presence`` is not supported when using virtual shards
- Shard group id will always be ``0`` when using virtual shards

Planned features that are not implemented yet
- ``Update Voice State`` (can be dispatched from the real shard through the ``guild_id`` identifier)