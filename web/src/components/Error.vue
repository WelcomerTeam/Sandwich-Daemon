<template>
  <div class="text-center my-5">
    <svg-icon
      class="mb-2 text-muted"
      type="mdi"
      :width="64"
      :height="64"
      :path="mdiAlertCircle"
    />
    <h3>Oops, something happened.</h3>
    <p class="m-auto my-2 col-10 text-muted">
      Something unexpected happened whilst handling your request, sorry.
    </p>
    <kbd>{{ this.message }}</kbd>
  </div>
</template>

<script>
import SvgIcon from "@jamescoyle/vue-icon";
import { mdiAlertCircle } from "@mdi/js";

export default {
  components: {
    SvgIcon
  },
  props: ["message"],
  data() {
    return {
      mdiAlertCircle: mdiAlertCircle
    };
  }
};
</script>
