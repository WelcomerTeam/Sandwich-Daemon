<template>
  <div id="app">
    <div v-cloak>
      <nav class="navbar navbar-expand-sm navbar-dark bg-dark">
        <div class="container-fluid">
          <router-link class="navbar-brand" to="/">
            Sandwich Daemon
            <span class="text-secondary h6">Version 🥪</span>
          </router-link>
          <button
            class="navbar-toggler"
            type="button"
            data-toggle="collapse"
            data-target="#navbarSupportedContent"
            aria-controls="navbarSupportedContent"
            aria-expanded="false"
            aria-label="Toggle navigation"
          >
            <span class="navbar-toggler-icon"></span>
          </button>
          <div
            class="collapse navbar-collapse justify-content-end"
            id="navbarSupportedContent"
          >
            <ul class="navbar-nav">
              <li class="nav-item">
                <a
                  class="nav-link"
                  aria-current="page"
                  href="https://github.com/TheRockettek/Sandwich-Daemon"
                  >GitHub</a
                >
              </li>
              <li class="nav-item" v-if="$store.state.user.id">
                <span
                  class="nav-link dropdown-toggle"
                  id="userDropdown"
                  role="button"
                  data-toggle="dropdown"
                  aria-expanded="false"
                >
                  <img
                    width="24"
                    height="24"
                    :src="
                      'https://cdn.discordapp.com/avatars/' +
                        $store.state.user.id +
                        '/' +
                        $store.state.user.avatar +
                        '.png?size=32'
                    "
                    alt="User profile picture"
                  />
                </span>
                <ul
                  class="dropdown-menu dropdown-menu-right"
                  aria-label="Profile"
                >
                  <li>
                    <span class="dropdown-item">
                      Signed in as <b>{{ $store.state.user.username }}</b>
                    </span>
                  </li>
                  <li>
                    <hr class="dropdown-divider" />
                  </li>
                  <li>
                    <a class="dropdown-item" href="/logout">Sign Out</a>
                  </li>
                </ul>
              </li>
              <li class="nav-item" v-else>
                <a class="nav-link" aria-current="page" href="/login">Login</a>
              </li>
            </ul>
          </div>
        </div>
      </nav>

      <div class="text-center py-5" v-if="$store.state.error">
        <error :message="$store.state.error" />
      </div>
      <div v-else>
        <router-view />
      </div>
    </div>
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

<script>
import Error from "@/components/Error.vue";

export default {
  components: {
    Error
  }
};
</script>
