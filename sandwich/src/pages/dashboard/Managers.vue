<template>
  <div>
    <div
      class="
        pb-5
        border-b border-gray-200
        sm:flex sm:items-center sm:justify-between
        mb-4
      "
    >
      <Listbox as="div">
        <div class="mt-1 relative max-w-full w-96">
          <ListboxButton
            class="
              bg-white
              relative
              w-full
              border border-gray-300
              rounded-md
              shadow-sm
              pl-3
              pr-10
              py-2
              text-left
              cursor-default
              focus:outline-none
              focus:ring-1
              focus:ring-blue-500
              focus:border-blue-500
              sm:text-sm
            "
          >
            <span class="block truncate">{{
              $store.getters.getSelectedManager?.identifier ||
              "Select a manager"
            }}</span>
            <span
              class="
                absolute
                inset-y-0
                right-0
                flex
                items-center
                pr-2
                pointer-events-none
              "
            >
              <SelectorIcon class="h-5 w-5 text-gray-400" aria-hidden="true" />
            </span>
          </ListboxButton>

          <transition
            leave-active-class="transition ease-in duration-100"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
          >
            <ListboxOptions
              class="
                absolute
                z-10
                mt-1
                w-full
                bg-white
                shadow-lg
                max-h-60
                rounded-md
                py-1
                text-base
                ring-1 ring-black ring-opacity-5
                overflow-auto
                focus:outline-none
                sm:text-sm
              "
            >
              <ListboxOption
                as="template"
                @click="$store.commit('setSelectedManager', null)"
              >
                <router-link :to="'/dashboard/managers'">
                  <li
                    class="
                      cursor-default
                      select-none
                      relative
                      py-2
                      pl-3
                      pr-9
                      hover:bg-blue-500 hover:text-white
                    "
                  >
                    Select a manager
                  </li>
                </router-link>
              </ListboxOption>
              <ListboxOption
                as="template"
                v-for="manager in $store.getters.getManagers"
                :key="manager"
                :value="manager.identifier"
                @click="$store.commit('setSelectedManager', manager.identifier)"
              >
                <router-link :to="'/dashboard/managers/' + manager.identifier">
                  <li
                    class="
                      cursor-default
                      select-none
                      relative
                      py-2
                      pl-3
                      pr-9
                      hover:bg-blue-500 hover:text-white
                    "
                  >
                    {{ manager.identifier }}
                  </li>
                </router-link>
              </ListboxOption>
            </ListboxOptions>
          </transition>
        </div>
      </Listbox>
      <div class="mt-3 sm:mt-0 sm:ml-4">
        <button
          type="button"
          class="
            inline-flex
            items-center
            px-4
            py-2
            border border-transparent
            rounded-md
            shadow-sm
            text-sm
            font-medium
            text-white
            bg-blue-600
            hover:bg-blue-700
            focus:outline-none
            focus:ring-2
            focus:ring-offset-2
            focus:ring-blue-500
          "
          @click="createNewManager"
        >
          Create new Manager
        </button>
      </div>
    </div>

    <router-view />
  </div>
</template>

<script>
import dashboardAPI from "../../api/dashboard";
import store from "../../store";

import TextInput from "../../components/TextInput.vue";
import {
  Listbox,
  ListboxButton,
  ListboxLabel,
  ListboxOption,
  ListboxOptions,
} from "@headlessui/vue";
import { CheckIcon, SelectorIcon } from "@heroicons/vue/solid";

export default {
  components: {
    Listbox,
    ListboxButton,
    ListboxLabel,
    ListboxOption,
    ListboxOptions,
    CheckIcon,
    SelectorIcon,
    TextInput,
  },
  methods: {
    onChange(event) {
      store.commit("setSelectedManager", event.target.value);
    },
  },
  methods: {
    createNewManager() {
      let newManagerConfiguration = {
        identifier: "",
        producer_identifier: "",
        friendly_name: "",
        token: "",
        client_name: "",
        channel_name: "",
      };

      newManagerConfiguration.identifier = prompt(
        "Enter new manager identifier",
        ""
      );
      if (newManagerConfiguration.identifier == "") {
        return;
      }

      newManagerConfiguration.producer_identifier = prompt(
        "Enter new manager producer identifier",
        newManagerConfiguration.producer_identifier
      );
      if (newManagerConfiguration.producer_identifier == "") {
        return;
      }

      newManagerConfiguration.friendly_name = prompt(
        "Enter friendly name",
        newManagerConfiguration.identifier
      );
      if (newManagerConfiguration.friendly_name == "") {
        return;
      }

      newManagerConfiguration.token = prompt("Enter manager token", "");
      newManagerConfiguration.client_name = prompt("Enter client name", "");
      newManagerConfiguration.channel_name = prompt(
        "Enter new manager channel name",
        "sandwich"
      );

      dashboardAPI.createNewManager(
        newManagerConfiguration,
        (response) => {
          alert(response);
          store.dispatch("fetchDashboardConfig");
          location.reload();
        },
        (e) => {
          alert(e);
        }
      );
    },
  },
};
</script>
