<template>
  <div class="text-center my-5">
    <svg-icon
      class="mb-2 text-muted"
      type="mdi"
      :width="64"
      :height="64"
      :path="mdiHelpRhombus"
    />
    <h3>404</h3>
    <p class="m-auto my-2 col-10 text-muted">
      It seems you are not where you want to be...
    </p>
    <kbd>{{ this.$route.path }}</kbd>
  </div>
</template>

<script>
import { mdiHelpRhombus } from "@mdi/js";
import SvgIcon from "@jamescoyle/vue-icon";
export default {
  components: {
    SvgIcon
  },
  name: "NotFoundPage",
  data() {
    return {
      mdiHelpRhombus: mdiHelpRhombus
    };
  }
};
</script>
