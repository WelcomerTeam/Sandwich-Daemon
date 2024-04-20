import discord
from discord.ext.commands import Bot
import os
import re
from collections import namedtuple
from urllib.parse import quote_plus

import discord
import discord.gateway
import discord.http
import yarl
import logging


def patch_with_gateway(env_gateway):
    class ProductionHTTPClient(discord.http.HTTPClient):
        async def get_gateway(self, **_):
            return f"{env_gateway}?encoding=json&v=9"

        async def get_bot_gateway(self, **_):
            try:
                data = await self.request(discord.http.Route("GET", "/gateway/bot"))
            except discord.HTTPException as exc:
                raise discord.GatewayNotFound() from exc
            return data["shards"], f"{env_gateway}?encoding=json&v=9"

    class ProductionDiscordWebSocket(discord.gateway.DiscordWebSocket):
        DEFAULT_GATEWAY = yarl.URL(env_gateway)

        def is_ratelimited(self):
            return False

    class ProductionBot(Bot):
        async def before_identify_hook(self, shard_id, *, initial):
            pass

        def is_ws_ratelimited(self):
            return False

    class ProductionReconnectWebSocket(Exception):
        def __init__(self, shard_id, *, resume=False):
            self.shard_id = shard_id
            self.resume = False
            self.op = "IDENTIFY"

    discord.http.HTTPClient.get_gateway = ProductionHTTPClient.get_gateway
    discord.http.HTTPClient.get_bot_gateway = ProductionHTTPClient.get_bot_gateway
    discord.gateway.DiscordWebSocket.DEFAULT_GATEWAY = ProductionDiscordWebSocket.DEFAULT_GATEWAY
    discord.gateway.DiscordWebSocket.is_ratelimited = ProductionDiscordWebSocket.is_ratelimited
    discord.gateway.ReconnectWebSocket.__init__ = ProductionReconnectWebSocket.__init__
    return ProductionBot

bot = patch_with_gateway("ws://localhost:3600")

client = bot(command_prefix="!", intents=discord.Intents.all())

@client.event
async def on_ready():
    print(f"We have logged in as {client.user}")
    for g in client.guilds:
        print(f"Connected to {g.name}")
        await g.chunk()

@client.command()
async def ping(ctx):
    await ctx.send(f"Pong! {ctx.bot.latency}")

client.run(os.environ.get("DISCORD_BOT_TOKEN"), log_level=logging.DEBUG)