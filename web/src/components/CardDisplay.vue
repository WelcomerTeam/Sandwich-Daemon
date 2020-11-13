<template>
  <div class="col justify-content-center d-flex">
    <div :class="bg + ' card text-white m-1'" style="width: 18rem;">
      <div class="card-header">{{ title }}</div>
      <div class="card-body">
        <h5 class="card-title">{{ value }}</h5>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "CardDisplay",
  props: ["title", "value", "bg"]
};
</script>
