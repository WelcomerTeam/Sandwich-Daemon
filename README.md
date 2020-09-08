
<img src="https://raw.githubusercontent.com/TheRockettek/Sandwich-Daemon/master/web/static/daemon-icon.svg" width="500"/>
Sandwich Daemon allows you to manage multiple bots within a single application. Easily create new bots and scale up already running bots with rolling restarts, multithreaded and microserviced with a hint of web dashboard for easy customization.

# How To Get Started
I use docker-compose when testing so i have attached an example-docker-compose.yml that i use to deploy STAN and Redis. Ensure you specify NATs-Streaming (STAN) instead of just NATs.

1) Launch STAN (NATs-Streaming) and Redis  
2) Rename sandwich-default.yml to sandwich.yml  
3) Edit the default token (go to 7) or remove the Manager (go to 4 for dashboard management)
4) To create a new manager go to localhost:5469 then head to clusters and click `create new manager`, fill out settings then click create.  
5) Click on the Manager you made then click `Scale Cluster`. Specify a custom shardcount (auto determine will use whatever the gateway provides). Provide a shard id range (auto determine will use all whilst adhering to clusters if specified)  
6) Click create shardgroup and wait until the shardgroup is ready. If an error occurs it will show the error.  
7) Go to the dashboard, go to clusters and it will show the status of the managers you have created. If there are any errors it will display the problem it had whilst connecting.  

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
- [x] Custom Tailored Events to allow for easier programming (including before and after in UPDATE events, invited_by in MEMBER_JOIN may be coming soon)
- [ ] Auto Shard Scaling (coming soon)
- [ ] Ability to use your own messaging service such as Kafka (Utilises NATS/STAN for Only-Once messaging) (Kafka Exactly-Once support may be coming soon)
- [ ] Selfbot support (no. Why do you need this for selfbots anyway?)

### Contact Me
Contact me on Github or ImRock#0001 on discord :)
