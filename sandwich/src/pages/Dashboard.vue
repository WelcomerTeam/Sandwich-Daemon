<template>
  <Layout title="Dashboard">
    <div v-if="$store.getters.hasConfigurationLoaded">
      <div class="block">
        <nav class="flex flex-wrap mb-4" aria-label="Tabs">
          <router-link
            v-for="tab in tabs"
            :key="tab.name"
            :to="tab.href"
            :class="[
              $route.path === tab.href
                ? 'bg-gray-100 text-gray-700'
                : 'text-gray-500 hover:text-gray-700',
              'p-2 font-medium text-sm rounded-md',
            ]"
            :aria-current="tab.current ? 'page' : undefined"
          >
            {{ tab.name }}
          </router-link>
        </nav>
        <router-view />
      </div>
    </div>
    <div v-else-if="$store.getters.getConfigurationFetchError">
      <Error :icon="mdiConnection">
        Failed to fetch dashboard:
        {{ $store.getters.getConfigurationFetchError }}
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

import { mdiConnection } from "@mdi/js";

var tabs = [
  {
    name: "Managers",
    href: "/dashboard/managers",
  },
  {
    name: "Global Settings",
    href: "/dashboard/settings",
  },
  {
    name: "Consumers",
    href: "/dashboard/consumers",
  },
];

export default {
  components: {
    Layout,
    Error,
    LoadingIcon,
  },
  setup() {
    const showLoading = ref(false);

    return {
      showLoading,
      tabs,

      mdiConnection,
    };
  },
  mounted() {
    store.dispatch("fetchDashboardConfig");

    // Only show loading after ~1 second of page loading.
    setTimeout(this.setShowLoading, 1000);
  },
  methods: {
    setShowLoading() {
      this.showLoading = true;
    },
  },
};
</script>
