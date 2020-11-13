<template>
  <button type="submit" class="btn btn-dark mr-1" @click="onClick">
    {{ label }}
  </button>
</template>

<script>
export default {
  name: "FormSubmit",
  props: {
    label: {
      default: "Save Changes"
    }
  },
  methods: {
    onClick() {
      this.$emit("click");
    }
  }
};
</script>
