<template>
  <div class="container-sm d-flex">
    <div class="align-items-center text-center m-auto d-flex flex-wrap mt-5">
      <img
        @click="item++"
        width="128"
        height="128"
        class="justify-content-start m-auto flex-grow-1 p-2"
        :src="items[item % items.length]"
      />
      <div class="justify-content-end text-center flex-grow-2 mx-auto p-2">
        <h1>Sandwich Daemon</h1>
        <a class="mr-1" href="https://github.com/TheRockettek/Sandwich-Daemon"
          ><button type="button" class="btn btn-dark">Github</button></a
        >
        <router-link class="ml-1" to="/dashboard"
          ><button type="button" class="btn btn-dark">
            Dashboard
          </button></router-link
        >
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "Index",
  data: function() {
    return {
      item: 0,
      items: ["/img/brand.svg", "/img/comfy.svg"]
    };
  }
};
</script>
