<template>
  <div id="app">
    <div v-cloak>
    </div>

    <pre>loading: {{ $store.state.userLoading }}</pre>
    <pre>user: {{ JSON.stringify($store.state.user) }}</pre>
    <pre>auth: {{ $store.state.userAuthenticated }}</pre>
    <pre>error: {{ $store.state.error }}</pre>
  </div>
</template>

<style>
html {
  min-height: 100%;
}

[v-cloak] {
  visibility: hidden;
}
</style>