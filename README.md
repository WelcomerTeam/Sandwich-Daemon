# Sandwich Daemon

<img src="https://raw.githubusercontent.com/WelcomerTeam/Sandwich-Daemon/next/assets/icon.svg" width="500"/>

Sandwich Daemon is a utility that handles gateway connections, state and event processing. It handles the events from discord, handles filtering out events you do not want, stores users, members, guilds etc. in its internal store then sends the resulting data into a message queue for further handling from consumers.

Easily create new managers, scale up utilising rolling restarts on already running bots with shardgroups and configure and manage everything with a dashboard.

Includes an easy to use GRPC interface for fetching information from Sandwich including (but not limited to) searching for members by ID or name, fetching mutual guilds and returning what shard and manager a guild may be on.

Sandwich can filter out events on an absolute level so it will not even know it received the event and also the possibility to internally process (such as add to state) but then not publish to a consumer.

## Consuming

Consuming Sandwich Daemon events should be fairly similar to a regular discord payload however also include a few changes to both fit the purpose of sandwich and support forward compatibility.

A sandwich payload received by a consumer will be a JSON payload similar to a regular discord payload. Sandwich events will also include the exact same `op,d,s,t` keys from a regular discord payload and these should not be affected by sandwich between it receiving and it being published.

Any extra data that sandwich includes will be under the keys `__extra, __sandwich, __sandwich_trace`.

- `__extra`: This is the sandwich equivelant of the `d` key in the regular discord payload. This will include extra data that will be useful contextually, at the moment this will only be present on `_UPDATE` events and will include the previous state.
- `__sandwich_trace`: This will include trace times.
- `__sandwich`: This includes any metadata that will be useful for consumers identifying the origin of the message. Metadata includes the keys `v,i,a,s`
  - `__sandwich.v`: The current version of Sandwich.
  - `__sandwich.i`: The ProducerIdentifier of a manager.
  - `__sandwich.a`: The Application name of a manager. This will be the regular identifier.
  - `__sandwich.s`: This is a list of integers. This represents: `shardgroupID, shardID, shardCount`

  ```json
  {
    "op":0,
    "d":...,
    "s":117,
    "t":"MESSAGE_CREATE",
    "__extra":null,
    "__sandwich":{
      "v":"1.0.1",
      "i":"welcomer",
      "a":"welcomer_dogfd",
      "s":[ 1, 0, 1 ]
    },
    "__sandwich_trace":null
  }```