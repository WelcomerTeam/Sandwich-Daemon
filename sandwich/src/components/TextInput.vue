<template>
  <div>
    <label :for="name" class="block text-sm font-medium text-gray-700">{{
      label
    }}</label>
    <div class="mt-1">
      <input
        :type="type"
        :name="name"
        :id="name"
        :class="[
          'shadow-sm focus:ring-blue-500 focus:border-blue-500 block sm:text-sm border-gray-300 rounded-md',
          disabled ? 'bg-gray-200 text-gray-500 cursor-default' : '',
          invalid ? 'border-red-500 text-red-500' : '',
          type == 'checkbox' ? '' : 'max-w-1/2 w-full',
          extendedClass,
        ]"
        :placeholder="placeholder"
        :aria-describedby="name + '-description'"
        :value="modelValue"
        :disabled="disabled"
        @input="
          updateValue(
            type == 'checkbox' ? $event.target.checked : $event.target.value
          )
        "
      />
    </div>
    <p class="mt-2 text-sm text-gray-500" :id="name + '-description'">
      {{ description }}
    </p>
  </div>
</template>

<script>
export default {
  props: [
    "name",
    "type",
    "label",
    "description",
    "placeholder",
    "modelValue",
    "extendedClass",
    "disabled",
    "invalid",
  ],
  methods: {
    updateValue(value) {
      this.$emit("update:modelValue", value);
    },
  },
};
</script>
