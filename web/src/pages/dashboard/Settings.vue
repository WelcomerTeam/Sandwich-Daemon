<template>
  <div>
    <div v-if="settings">
      <form>
        <field-set class="mb-4" name="Identity">
          <text-input
            type="text"
            v-model="settings.identify.url"
            name="identify_url"
            label="URL"
            description="If specified, will call the URL to check if the shard can identify. Can use shard_id, shard_count, token, token_hash and max_concurrency values in URL."
          />
          <text-input
            type="text"
            v-model="identify_headers"
            name="identify_headers"
            label="Headers (JSON)"
            description="Key value of headers to pass in the request"
            :invalid="identify_headers_invalid"
          />
        </field-set>
        <field-set class="mb-4 space-y-4" name="Producer">
          <text-input
            type="text"
            v-model="settings.producer.type"
            name="producer_type"
            label="Producer Type"
            description="Type of producer. Accepts: stan, kafka, redis, websocket"
          />
          <text-input
            type="text"
            v-model="producer_configuration"
            name="producer_configuration"
            label="Configuration (JSON)"
            description="Producer configuration."
          />
        </field-set>

        <field-set class="mb-4 space-y-4" name="HTTP">
          <text-input
            type="text"
            v-model="http_oauth"
            name="http_oauth_config"
            label="OAuth config (JSON)"
            :invalid="http_oauth_invalid"
          />
          <text-input
            type="text"
            v-model="user_access"
            name="http_user_access"
            label="User Access"
            description="Comma seperated list of discord users who have access to the dashboard"
          />
        </field-set>
        <field-set class="mb-4 space-y-4" name="Webhooks">
          <text-input
            type="text"
            v-model="webhooks"
            name="webhooks"
            label="Webhooks"
            description="Comma seperated list of webhooks to send status messages to"
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
          @click.prevent="updateSandwichConfig"
        >
          Save
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import dashboardAPI from "../../api/dashboard";
import store from "../../store";

import { ref } from "vue";

import TextInput from "../../components/TextInput.vue";
import FieldSet from "../../components/FieldSet.vue";

export default {
  components: {
    FieldSet,
    TextInput,
  },
  setup() {
    return {
      identify_headers: ref(""),
      identify_headers_invalid: ref(false),

      producer_configuration: ref(""),
      producer_configuration_invalid: ref(false),

      http_oauth: ref(""),
      http_oauth_invalid: ref(false),

      user_access: ref(""),
      webhooks: ref(""),

      settings: ref(null),
    };
  },
  mounted() {
    this.refreshSettings();
  },
  beforeRouteUpdate(to, from, next) {
    this.refreshSettings();
    next();
  },
  watch: {
    identify_headers() {
      try {
        this.settings.identify.headers = JSON.parse(this.identify_headers);
        this.identify_headers_invalid = false;
      } catch {
        this.identify_headers_invalid = true;
      }
    },
    producer_configuration() {
      try {
        this.settings.producer.configuration = JSON.parse(
          this.producer_configuration
        );
        this.producer_configuration_invalid = false;
      } catch {
        this.producer_configuration_invalid = true;
      }
    },
    http_oauth() {
      try {
        this.settings.http.oauth = JSON.parse(this.http_oauth);
        this.http_oauth_invalid = false;
      } catch {
        this.http_oauth_invalid = true;
      }
    },
    user_access() {
      this.settings.http.user_access = this.user_access.split(",");
    },
    webhooks() {
      this.settings.webhooks = this.webhooks.split(",");
    },
  },
  methods: {
    refreshSettings() {
      if (typeof store.getters.getConfiguration !== "undefined") {
        this.settings = JSON.parse(
          JSON.stringify(store.getters.getConfiguration)
        );
        this.identify_headers = JSON.stringify(
          store.getters.getConfiguration.identify.headers
        );
        this.producer_configuration = JSON.stringify(
          store.getters.getConfiguration.producer.configuration
        );
        this.http_oauth = JSON.stringify(
          store.getters.getConfiguration.http.oauth
        );
        this.user_access =
          store.getters.getConfiguration.http.user_access?.join(",");
        delete this.settings.managers;
        this.webhooks = store.getters.getConfiguration.webhooks?.join(",");
      } else {
        this.settings = null;
      }
    },
    updateSandwichConfig() {
      dashboardAPI.updateSandwichConfig(
        this.settings,
        (response) => {
          alert(response);
        },
        (e) => {
          alert(e);
        }
      );
    },
  },
};
</script>
