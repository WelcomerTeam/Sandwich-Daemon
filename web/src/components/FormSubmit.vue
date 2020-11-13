<template>
  <button type="submit" class="btn btn-dark" @click="onClick">
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
