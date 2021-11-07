<template>
  <div>
    <div class="text-base text-gray-500" v-if="showAllShardGroups">
      <div v-bind:key="shard_group" v-for="shard_group in manager.shard_groups">
        <field-set :name="'ShardGroup ' + shard_group.id">
          <div class="flex flex-wrap justify-center">
            <div
              v-bind:key="shard"
              v-for="shard in shard_group.shards"
              class="has-tooltip p-1"
            >
              <div :class="['w-7 h-7 rounded-md', getShardColour(shard)]" />
              <p class="tooltip bg-blue-500 text-white">
                Shard {{ shard[0] }} - {{ getShardStatus(shard) }}<br /><br />
                Guilds: {{ shard[3] }}<br />
                Latency: {{ shard[2] }}ms<br />
              </p>
            </div>
          </div>
          <div class="relative">
            <div class="absolute inset-0 flex items-center" aria-hidden="true">
              <div class="w-full border-t border-gray-300" />
            </div>
            <div class="relative flex justify-center">
              <span class="px-2 bg-white text-sm text-gray-500">
                Shards: {{ shard_group.shards.length }} Guilds:
                {{ getShardGroupGuildCount(shard_group) }} Latency:
                {{ getShardGroupAverageLatency(shard_group) }}ms
              </span>
            </div>
          </div>
        </field-set>
      </div>
    </div>
    <div class="text-base text-gray-500" v-else>
      <div v-bind:key="shard_group" v-for="shard_group in manager.shard_groups">
        <div class="flex flex-wrap justify-center">
          <div
            v-bind:key="shard"
            v-for="shard in shard_group.shards"
            class="has-tooltip p-1"
          >
            <div :class="['w-7 h-7 rounded-md', getShardColour(shard)]" />
            <p class="tooltip bg-blue-500 text-white">
              Shard {{ shard[0] }} - {{ getShardStatus(shard) }}<br /><br />
              Guilds: {{ shard[3] }}<br />
              Latency: {{ shard[2] }}ms<br />
            </p>
          </div>
        </div>
        <div class="relative">
          <div class="absolute inset-0 flex items-center" aria-hidden="true">
            <div class="w-full border-t border-gray-300" />
          </div>
          <div class="relative flex justify-center">
            <span class="px-2 bg-white text-sm text-gray-500">
              Shards: {{ shard_group.shards.length }} Guilds:
              {{ getShardGroupGuildCount(shard_group) }} Latency:
              {{ getShardGroupAverageLatency(shard_group) }}ms
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import FieldSet from "./FieldSet.vue";

const shardType = [
  ["bg-gray-300", "Idle"],
  ["bg-blue-300", "Connecting"],
  ["bg-blue-400", "Connected"],
  ["bg-green-500", "Ready"],
  ["bg-green-600", "Reconnecting"],
  ["bg-yellow-300", "Closing"],
  ["bg-gray-400", "Closed"],
  ["bg-red-500", "Erroring"],
];

export default {
  components: { FieldSet },
  props: {
    manager: {
      type: Object,
    },
    showAllShardGroups: {
      type: Boolean,
    },
  },
  methods: {
    getShardGroupAverageLatency(shard_group) {
      var shardLatencyTotal = 0;
      var shardLatencyCount = 0;

      shard_group.shards.forEach((shard) => {
        if (shard[2] > 0) {
          shardLatencyTotal += shard[2];
          shardLatencyCount++;
        }
      });

      var latency = Math.round(shardLatencyTotal / shardLatencyCount);
      return Number.isNaN(latency) ? "-" : latency;
    },
    getShardGroupGuildCount(shard_group) {
      var guildCount = 0;

      shard_group.shards.forEach((shard) => {
        guildCount += shard[3];
      });

      return guildCount;
    },
    getShardColour(shard) {
      return shardType[shard[1]][0];
    },
    getShardStatus(shard) {
      return shardType[shard[1]][1];
    },
  },
};
</script>
