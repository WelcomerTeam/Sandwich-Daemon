<template>
  <div>
    <div class="bg-blue-600 pb-32">
      <nav class="bg-blue-600">
        <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
          <div class="border-b border-blue-700">
            <div class="flex items-center justify-between h-16 px-4 sm:px-0">
              <div class="flex items-center">
                <div class="block">
                  <div class="flex items-baseline space-x-4">
                    <router-link
                      to="/"
                      class="
                        text-white
                        hover:text-gray-200
                        py-2
                        rounded-md
                        text-sm
                        font-medium
                      "
                      >Sandwich Daemon</router-link
                    >
                  </div>
                </div>
              </div>
              <div class="block">
                <div class="ml-4 flex items-center md:ml-6">
                  <div class="ml-3 relative">
                    <div>
                      <router-link
                        to="/dashboard"
                        class="text-white hover:text-gray-200 font-medium"
                      >
                        Login
                      </router-link>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </nav>
      <header class="py-10">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <h1 class="text-3xl font-bold text-white">{{ title }}</h1>
        </div>
      </header>
    </div>

    <main class="-mt-32">
      <div class="max-w-7xl mx-auto pb-12 px-4 sm:px-6 lg:px-8">
        <div class="bg-white rounded-lg shadow px-5 py-6 sm:px-6">
          <slot />
        </div>
      </div>
    </main>
  </div>
</template>

<script>
export default {
  props: ["title"],
};
</script>
