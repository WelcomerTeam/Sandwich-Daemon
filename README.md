# Sandwich-Daemon
Sandwich Daemon allows you to manage multiple bots within a single application. Easily create new bots and scale up already running bots with rolling restarts, multithreaded and microserviced with a hint of web dashboard for easy customization.

## Why should i use this?
Sandwich Daemon is useful if you have multiple bots running or you just generally want more control over how your bot interacts with the gateway or want to get into microservicing your bot. It is scalable and is fairly lightweight.

Sandwich Daemon was made to allow for a single process to interact with the gateway instead of having 18 discord.py instances each doing it themselves, it also allows for all the caches to be stored in a single place which is useful when having a bot dashboard.

## What do you offer?
- [x] Ability to run multiple bots in a single program
- [x] Seperate the bot logic from the gateway logic
- [x] Ability to make your own consumer in any language without modifying the gateway code
- [x] Ability to filter events out from consumers and make some only cached
- [x] Bots use their own cache or share a cache
- [x] Redis to allow the gateway, consumer and any external service to use (such as a website)
- [x] Support for the new gateway features (intents)
- [x] Allow to easily retrieve mutual servers with a user with a single request
- [x] Abilty to make the gateway automatically check for prefixes and ignore bots
- [x] Auto Sharding
- [x] Clustering Daemon among multiple machines
- [x] Custom Tailored Events to allow for easier programming (including before and after in UPDATE events, invited_by in MEMBER_JOIN may be comming soon)
- [ ] Auto Shard Scaling (comming soon)
- [ ] Ability to use your own messaging service such as Kafka (Utilises NATS/STAN for Only-Once messaging) (Kafka Exactly-Once support may be comming soon)
- [ ] Selfbot support (no. Why do you need this for selfbots anyway?)

### Contact Me
Contact me on Github or ImRock#0001 on discord :)