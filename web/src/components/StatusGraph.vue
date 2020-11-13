<template>
  <div class="progress">
    <div
      v-for="(value, index) in this.keys"
      v-bind:key="index"
      :class="'progress-bar bg-' + colours[index]"
      role="progressbar"
      :style="'width: ' + (value / total) * 100 + '%'"
      :aria-valuenow="(value / total) * 100"
      aria-valuemin="0"
      aria-valuemax="100"
    ></div>
  </div>
</template>

<script>
export default {
  name: "StatusGraph",
  props: ["value", "colours"],
  data() {
    return {
      keys: {},
      total: 0
    };
  },
  watch: {
    value: function() {
      this.loadValues();
    }
  },
  mounted() {
    this.loadValues();
  },
  methods: {
    loadValues() {
      this.keys = {};
      this.total = 0;
      var shards = Object.values(this.value.shards);
      for (var shindex in shards) {
        var shard = shards[shindex];
        if (shard.status in this.keys) {
          this.keys[shard.status]++;
        } else {
          this.keys[shard.status] = 1;
        }
        this.total++;
      }
      this.$forceUpdate();
    }
  }
};
</script>
