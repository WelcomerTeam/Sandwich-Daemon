<template>
  <div class="text-center">
    <svg-icon
      class="mb-2 mx-auto text-gray-300"
      type="mdi"
      :width="48"
      :height="48"
      :path="icon"
    />
    <p class="text-gray-900 font-medium">An error occurred</p>
    <p class="text-gray-500"><slot /></p>
  </div>
</template>

<script>
import SvgIcon from "@jamescoyle/vue-icon";

export default {
  components: {
    SvgIcon,
  },
  props: ["errorText", "icon"],
};
</script>
