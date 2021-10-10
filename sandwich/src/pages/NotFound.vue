<template>
  <Layout title="404">
    Page not found.
    <a class="underline cursor-pointer" @click.prevent="handleBack('/')"
      >Go Back</a
    >
  </Layout>
</template>

<script>
import Layout from "../components/Layout.vue";

export default {
  components: {
    Layout,
  },

  data() {
    return {
      fromRoute: null,
    };
  },

  beforeRouteEnter(to, from, next) {
    next((vm) => {
      vm.fromRoute = from;
    });
  },

  methods: {
    handleBack(fallback) {
      if (!this.fromRoute?.name) {
        this.$router.push(fallback);
      } else {
        this.$router.back();
      }
    },
  },
};
</script>

<style></style>
