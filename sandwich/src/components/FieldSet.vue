<template>
  <div
    :class="[
      'relative border border-gray-300 rounded-md px-3 py-2 shadow-sm',
      extendedClass,
    ]"
  >
    <span
      for="name"
      class="
        absolute
        -top-2
        left-2
        -mt-px
        inline-block
        px-1
        bg-white
        text-xs
        font-medium
        text-gray-900
      "
      >{{ name }}</span
    >
    <slot />
  </div>
</template>

<script>
export default {
  props: ["name", "extendedClass"],
};
</script>
