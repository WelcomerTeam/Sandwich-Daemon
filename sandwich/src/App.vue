<template>
  <router-view />
</template>

<script>
import store from "./store/index";

export default {
  created() {
    store.dispatch("fetchUser");
  },
};
</script>
