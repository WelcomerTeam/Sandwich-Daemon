<template>
  <Layout title="Status">
    <div v-if="$store.getters.hasStatusLoaded">
      <h2 class="uppercase font-bold text-gray-700">Discord Bots</h2>

      <dl class="space-y-6 divide-y divide-gray-200">
        <Disclosure
          as="div"
          v-for="manager in $store.getters.getStatus.managers"
          :key="manager.display_name"
          class="pt-6"
          v-slot="{ open }"
        >
          <dt class="text-lg">
            <DisclosureButton
              class="
                text-left
                w-full
                flex
                justify-between
                items-start
                text-gray-400
              "
            >
              <span class="font-medium text-gray-900">
                {{ manager.display_name }}
                <span
                  :class="[
                    'inline-flex items-center px-3 py-0.5 rounded-full text-sm font-medium',
                    getManagerBadgeColour(manager),
                  ]"
                >
                  {{ getManagerBadgeText(manager) }}
                </span>
              </span>
              <span class="ml-6 h-7 flex items-center">
                <ChevronDownIcon
                  :class="[
                    open ? '-rotate-180' : 'rotate-0',
                    'h-6 w-6 transform',
                  ]"
                  aria-hidden="true"
                />
              </span>
            </DisclosureButton>
          </dt>

          <DisclosurePanel as="dd" class="mt-2">
            <p class="text-base text-gray-500">
              <div v-bind:key="shard_group" v-for="shard_group in manager.shard_groups">
                <div class="flex flex-wrap justify-center">
                  <div v-bind:key="shard" v-for="shard in shard_group.shards" class="has-tooltip p-1">
                    <div :class="['w-7 h-7 rounded-md', getShardColour(shard)]" />
                    <p class="tooltip bg-blue-500 text-white">
                      Shard {{shard[0]}} - {{getShardStatus(shard)}}<br><br>
                      Guilds: {{shard[3]}}<br>
                      Latency: {{shard[2]}}ms<br>
                    </p>
                  </div>
                </div>
                <div class="relative">
                  <div class="absolute inset-0 flex items-center" aria-hidden="true">
                    <div class="w-full border-t border-gray-300" />
                  </div>
                  <div class="relative flex justify-center">
                    <span class="px-2 bg-white text-sm text-gray-500">
                      Shards: {{shard_group.shards.length}}
                      Guilds: {{getShardGroupGuildCount(shard_group)}}
                      Latency: {{getShardGroupAverageLatency(shard_group)}}ms
                    </span>
                  </div>
                </div>
              </div>
            </p>
          </DisclosurePanel>
        </Disclosure>
      </dl>
    </div>
    <div v-else-if="$store.getters.getStatusFetchError">
      <Error :icon="mdiConnection">
        Failed to fetch status: {{$store.getters.getStatusFetchError}}
      </Error>
    </div>
    <div v-else-if="showLoading">
      <div class="flex justify-center">
        <LoadingIcon />
      </div>
    </div>
  </Layout>
</template>

<script>
import Layout from "../components/Layout.vue";
import Error from "../components/Error.vue";
import LoadingIcon from "../components/LoadingIcon.vue";

import store from "../store/index";
import { ref } from "vue";

import { mdiChevronDown, mdiConnection } from '@mdi/js';
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import { ChevronDownIcon } from "@heroicons/vue/outline";

const managerType = [
  ["bg-gray-200 text-gray-800", "Idle", true], // No shardgroups
  ["bg-blue-200 text-blue-800", "Connecting", true], // First shardgroup any shard connecting
  ["bg-red-200 text-red-800", "Total Outage", true], // First shardgroup shards all erroring
  ["bg-yellow-200 text-yellow-800", "Partial Outage", true], // First shardgroup shards any erroring
  ["bg-green-200 text-green-800", "Operational", true], // First shardgroup shards all not erroring
];

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

const shardStatusConnecting = 1;
const shardStatusErroring = 7;

export default {
  components: {
    Layout,
    Error,
    LoadingIcon,

    Disclosure,
    DisclosureButton,
    DisclosurePanel,
    ChevronDownIcon,
  },
  setup() {
    const showLoading = ref(false);

    return {
      showLoading,

      mdiConnection,
      mdiChevronDown,
    };
  },
  mounted() {
    store.dispatch("fetchStatus");
    setInterval(() => {
      store.dispatch("fetchStatus")
    }, 30000);

    // Only show loading after ~1 second of page loading.
    setTimeout(this.setShowLoading, 1000);
  },
  methods: {
    setShowLoading() {
      this.showLoading = true;
    },
    getManagerType(manager) {
      if (manager.shard_groups.length === 0) {
        return managerType[0];
      }

      var erroringShards = 0;
      var primaryShardGroup = manager.shard_groups[0];

      primaryShardGroup.shards.forEach((shard) => {
        if (shard[1] === shardStatusErroring) {
          erroringShards++;
        }

        if (shard[1] === shardStatusConnecting) {
          return managerType[1];
        }
      });

      // All shards erroring
      if (erroringShards === primaryShardGroup.shards.length) {
        return managerType[2];
      }

      // Return partial outage if any shards erroring else return operational.
      return managerType[erroringShards > 0 ? 3 : 4];
    },
    getManagerBadgeColour(manager) {
      return this.getManagerType(manager)[0];
    },
    getManagerBadgeText(manager) {
      return this.getManagerType(manager)[1];
    },
    getManagerShow(manager) {
      return this.getManagerType(manager)[2];
    },
    getShardGroupAverageLatency(shard_group) {
      var shardLatencyTotal = 0;
      var shardLatencyCount = 0;

      shard_group.shards.forEach((shard) => {
        if (shard[2] > 0) {
          shardLatencyTotal += shard[2];
          shardLatencyCount++;
        }
      })

      var latency = Math.round(shardLatencyTotal/shardLatencyCount);
      return Number.isNaN(latency) ? '-' : latency;
    },
    getShardGroupGuildCount(shard_group) {
      var guildCount = 0;

      shard_group.shards.forEach((shard) => {
        guildCount += shard[3];
      })

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

<style></style>
