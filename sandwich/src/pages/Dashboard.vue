<template>
  <Layout title="Dashboard">
    <div v-if="$store.getters.hasConfigurationLoaded">
      <!-- <pre>{{ $store.getters.getConfiguration }}</pre> -->
      <div class="sm:hidden">
        <label for="tabs" class="sr-only">Select a tab</label>
        <!-- Use an "onChange" listener to redirect the user to the selected tab URL. -->
        <select
          id="tabs"
          name="tabs"
          class="
            block
            w-full
            focus:ring-indigo-500 focus:border-indigo-500
            border-gray-300
            rounded-md
          "
        >
          <option v-for="tab in tabs" :key="tab.name" :selected="tab.current">
            {{ tab.name }}
          </option>
        </select>
      </div>
      <div class="hidden sm:block">
        <nav class="flex space-x-4" aria-label="Tabs">
          <a
            v-for="tab in tabs"
            :key="tab.name"
            :href="tab.href"
            :class="[
              tab.current
                ? 'bg-gray-100 text-gray-700'
                : 'text-gray-500 hover:text-gray-700',
              'px-3 py-2 font-medium text-sm rounded-md',
            ]"
            :aria-current="tab.current ? 'page' : undefined"
          >
            {{ tab.name }}
          </a>
        </nav>
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
