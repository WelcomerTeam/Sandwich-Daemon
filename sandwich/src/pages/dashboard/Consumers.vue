<template>
  <div>
    <div>
      <form>
        <div class="mb-4">
          <prism-editor
            class="bg-gray-100 p-4 rounded-sm shadow-sm"
            v-model="config"
            :highlight="highlighter"
            line-numbers
          />
        </div>

        <button
          class="
            inline-flex
            items-center
            px-4
            py-2
            border border-transparent
            text-sm
            font-medium
            rounded-md
            shadow-sm
            text-white
            bg-blue-600
            hover:bg-blue-700
            focus:outline-none
            focus:ring-2
            focus:ring-offset-2
            focus:ring-blue-500
          "
          @click.prevent="updateConsumerConfiguration"
        >
          Save
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import { PrismEditor } from "vue-prism-editor";
import "vue-prism-editor/dist/prismeditor.min.css";

import { highlight, languages } from "prismjs/components/prism-core";
import "prismjs/components/prism-json";
import "prismjs/themes/prism-tomorrow.css";

import { ref } from "@vue/reactivity";

import FieldSet from "../../components/FieldSet.vue";

export default {
  components: {
    FieldSet,
    PrismEditor,
  },
  setup() {
    return {
      config: ref(""),
      invalid: ref(false),
    };
  },
  methods: {
    highlighter(code) {
      return highlight(code, languages.json);
    },
  },
};
</script>
