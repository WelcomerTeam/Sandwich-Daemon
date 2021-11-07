<template>
  <div>
    <div v-if="$store.getters.getSelectedManagerStatus">
      <manager-status
        :manager="$store.getters.getSelectedManagerStatus"
        :showAllShardGroups="true"
        class="mb-4"
      />
      <form class="mb-4">
        <field-set class="mb-4 space-y-4" name="New ShardGroup">
          <text-input
            type="text"
            v-model="newShardGroupShardIDs"
            label="Shard IDs"
          />
          <text-input
            type="number"
            v-model="newShardGroupShardCount"
            label="Shard Count"
          />
          <text-input
            type="checkbox"
            v-model="newShardGroupAutoSharded"
            label="Autosharded"
          />
        </field-set>
        <div class="flex space-x-4">
          <button
            class="
              inline-flex
              items-center
              px-4
              py-2
              border border-transparent
              text-sm
              font-medium
              rounded-md
              shadow-sm
              text-white
              bg-blue-600
              hover:bg-blue-700
              focus:outline-none
              focus:ring-2
              focus:ring-offset-2
              focus:ring-blue-500
            "
            @click.prevent="createShardGroup"
          >
            New ShardGroup
          </button>
          <button
            class="
              ml-3
              inline-flex
              items-center
              px-4
              py-2
              border border-transparent
              text-sm
              font-medium
              rounded-md
              text-red-600
              bg-white
              focus:outline-none
              focus:ring-2
              focus:ring-offset-2
              focus:ring-red-500
            "
            @click.prevent="stopShardGroups"
          >
            Stop ShardGroups
          </button>
          <div v-if="shardGroupLoading" class="flex">
            <loading-icon />
            Connecting Shards...
          </div>
        </div>
      </form>
    </div>

    <form v-if="manager" class="space-y-4">
      <field-set name="General" class="space-y-4">
        <text-input
          type="text"
          v-model="manager.identifier"
          name="identifier"
          label="Identifier"
          description="Internal identifier of the manager. This cannot be changed and is unique."
          :disabled="allowIdentifierEdits"
        />
        <text-input
          type="text"
          v-model="manager.producer_identifier"
          name="producer_identifier"
          label="Producer Identifier"
          description="Identifier that is sent to consumers. This does not have to be unique."
        />
        <text-input
          type="text"
          v-model="manager.friendly_name"
          name="friendly_name"
          label="Friendly Name"
          description="Friendly name to display on dashboard, logs and on status page."
        />
        <text-input
          type="password"
          v-model="manager.token"
          name="token"
          label="Token"
        />
        <text-input
          type="checkbox"
          v-model="manager.auto_start"
          name="auto_start"
          label="Auto Start"
          description="When enabled, will start up the manager when Sandwich is started up."
        />
      </field-set>
      <field-set name="Bot" class="space-y-4">
        <text-input
          type="text"
          v-model="default_presence"
          name="default_presence"
          label="Default Presence (JSON)"
          description="See https://discord.com/developers/docs/topics/gateway#update-presence"
          :invalid="default_presence_invalid"
        />
        <text-input
          type="number"
          v-model="manager.bot.intents"
          name="intents"
          label="Intents Value"
          description="Numerical value representing the intents to use."
        />
        <text-input
          type="checkbox"
          v-model="manager.bot.chunk_guilds_on_startup"
          name="chunk_guilds_on_startup"
          label="Chunk Guilds on Startup"
          :disabled="true"
          description="When enabled, will request guild members on startup."
        />
      </field-set>
      <field-set name="Caching" class="space-y-4">
        <text-input
          type="checkbox"
          v-model="manager.caching.cache_users"
          name="cache_users"
          label="Cache Users"
          :disabled="true"
          description="When enabled, will keep users in cache."
        />
        <text-input
          type="checkbox"
          v-model="manager.caching.cache_members"
          name="cache_members"
          label="Cache Members"
          :disabled="true"
          description="When enabled, will keep members in cache. Noop if cache users is disabled."
        />
        <text-input
          type="checkbox"
          v-model="manager.caching.store_mutuals"
          name="store_mutuals"
          label="Store Mutuals"
          :disabled="true"
          description="When enabled, will keep track of mutual guilds for a specific user. Rely on oauth2 instead of this. Noop if cache members or cache users is disabled."
        />
      </field-set>
      <field-set name="Events" class="space-y-4">
        <text-input
          type="text"
          v-model="event_blacklist"
          name="event_blacklist"
          label="Event Blacklist"
          description="Comma seperated list of dispatch events that will be completely ignored."
        />
        <text-input
          type="text"
          v-model="produce_blacklist"
          name="produce_blacklist"
          label="Produce Blacklist"
          description="Comma seperated list of dispatch events that will not be sent to consumers."
        />
      </field-set>
      <field-set name="Messaging" class="space-y-4">
        <text-input
          type="text"
          v-model="manager.messaging.client_name"
          name="client_name"
          label="Client Name"
          description="Client name to use in messaging."
        />
        <text-input
          type="text"
          v-model="manager.messaging.channel_name"
          name="channel_name"
          label="Channel Name"
          description="Channel name to use in messaging."
        />
        <text-input
          type="checkbox"
          v-model="manager.messaging.use_random_suffix"
          name="use_random_suffix"
          label="Use Random Suffix on Client name"
          description="When enabled, client names will include a random suffix. Enabling this is recommended."
        />
      </field-set>
      <field-set name="Sharding" class="space-y-4">
        <text-input
          type="checkbox"
          v-model="manager.sharding.auto_sharded"
          name="auto_sharded"
          label="Auto Sharded"
          description="When enabled, will ignore shard_count and shard_ids and will launch how many shards discord recommends."
        />
        <text-input
          type="number"
          v-model="manager.sharding.shard_count"
          name="shard_count"
          label="Shard Count"
          description="Total number of shards irrespective of Shard IDs."
        />
        <text-input
          type="text"
          v-model="manager.sharding.shard_ids"
          name="shard_ids"
          label="Shard IDs"
          description="Shard IDs to connect with by default. Can be in format 0,1,2 or include ranges 0,1-5,6."
        />
      </field-set>
      <button
        class="
          inline-flex
          items-center
          px-4
          py-2
          border border-transparent
          text-sm
          font-medium
          rounded-md
          shadow-sm
          text-white
          bg-blue-600
          hover:bg-blue-700
          focus:outline-none
          focus:ring-2
          focus:ring-offset-2
          focus:ring-blue-500
        "
        @click.prevent="updateManagerConfig"
      >
        Save
      </button>
      <button
        class="
          ml-3
          inline-fl1ex
          items-center
          px-4
          py-2
          border border-transparent
          text-sm
          font-medium
          rounded-md
          shadow-sm
          bg-white
          border-blue-600
          text-blue-600
          hover:bg-gray-50
          focus:outline-none
          focus:ring-2
          focus:ring-offset-2
          focus:ring-blue-500
        "
        @click.prevent="initializeManager"
      >
        Initialize
      </button>
      <button
        class="
          ml-3
          inline-flex
          items-center
          px-4
          py-2
          border border-transparent
          text-sm
          font-medium
          rounded-md
          text-red-600
          hover:text-red-700
          bg-white
          focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500
        "
        @click.prevent="deleteManager"
      >
        Delete Manager
      </button>
    </form>
  </div>
</template>

<script>
import dashboardAPI from "../../api/dashboard";
import store from "../../store";

import { ref } from "vue";

import TextInput from "../../components/TextInput.vue";
import FieldSet from "../../components/FieldSet.vue";
import ManagerStatus from "../../components/ManagerStatus.vue";
import LoadingIcon from "../../components/LoadingIcon.vue";

export default {
  components: {
    TextInput,
    FieldSet,
    ManagerStatus,
    LoadingIcon,
  },
  props: {
    allowIdentifierEdits: {
      type: Boolean,
      default: false,
    },
  },
  setup() {
    return {
      default_presence: ref(""),
      default_presence_invalid: ref(false),

      newShardGroupShardIDs: ref(""),
      newShardGroupShardCount: ref(0),
      newShardGroupAutoSharded: ref(false),

      shardGroupLoading: ref(false),

      event_blacklist: ref(""),
      produce_blacklist: ref(""),
      manager: ref(null),
    };
  },
  mounted() {
    this.refreshManager();

    store.dispatch("fetchManagerStatus");
    setInterval(() => {
      store.dispatch("fetchManagerStatus");
    }, 15000);
  },
  beforeRouteUpdate(to, from, next) {
    this.refreshManager();
    next();
  },
  watch: {
    "$store.getters.getSelectedManager"() {
      this.default_presence = JSON.stringify(
        store.getters.getSelectedManager.bot.default_presence
      );
      this.event_blacklist =
        store.getters.getSelectedManager.events.event_blacklist?.join(",");
      this.produce_blacklist =
        store.getters.getSelectedManager.events.produce_blacklist?.join(",");
    },
    default_presence() {
      if (this.manager && this.manager.bot) {
        try {
          this.manager.bot.default_presence = JSON.parse(this.default_presence);
          this.default_presence_invalid = false;
        } catch {
          this.default_presence_invalid = true;
        }
      }
    },
    event_blacklist() {
      if (this.manager && this.manager.events && this.event_blacklist) {
        this.manager.events.event_blacklist = this.event_blacklist.split(",");
      }
    },
    produce_blacklist() {
      if (this.manager && this.manager.events && this.produce_blacklist) {
        this.manager.events.produce_blacklist =
          this.produce_blacklist.split(",");
      }
    },
  },
  methods: {
    stopShardGroups() {
      if (confirm("Are you sure you want to stop all shard groups?")) {
        dashboardAPI.stopShardGroups(
          this.manager.identifier,
          (response) => {
            alert(response);
          },
          (e) => {
            alert(e);
          }
        );
      }
    },
    deleteManager() {
      if (confirm("Are you sure you want to delete this manager?")) {
        dashboardAPI.deleteManager(
          this.manager.identifier,
          (response) => {
            alert(response);
            location.reload();
          },
          (e) => {
            alert(e);
          }
        );
      }
    },
    refreshManager() {
      if (typeof store.getters.getSelectedManager !== "undefined") {
        store.dispatch("fetchManagerStatus");
        this.manager = JSON.parse(
          JSON.stringify(store.getters.getSelectedManager)
        );
      } else {
        this.manager = null;
      }
    },
    initializeManager() {
      dashboardAPI.initializeManager(
        this.manager.identifier,
        (response) => {
          alert(response);
        },
        (e) => {
          alert(e);
        }
      );
    },
    updateManagerConfig() {
      dashboardAPI.updateManagerConfig(
        this.manager,
        (response) => {
          alert(response);
        },
        (e) => {
          alert(e);
        }
      );
    },
    createShardGroup() {
      if (confirm("Are you sure you want to make a new shard group?")) {
        this.shardGroupLoading = true;
        dashboardAPI.createManagerShardGroup(
          {
            shard_ids: this.newShardGroupShardIDs,
            shard_count: Number(this.newShardGroupShardCount),
            auto_sharded: this.newShardGroupAutoSharded,
            identifier: this.manager.identifier,
          },
          (response) => {
            this.shardGroupLoading = false;
            alert(response);
            store.dispatch("fetchManagerStatus");
          },
          (e) => {
            this.shardGroupLoading = false;
            alert(e);
          }
        );
      }
    },
  },
};
</script>
