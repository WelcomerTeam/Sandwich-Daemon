<template>
  <div>
    <div class="form-check" v-if="type == 'checkbox'">
      <input
        class="form-check-input"
        type="checkbox"
        :id="id"
        :checked="value"
        v-on:change="updateValue($event.target.checked)"
        :disabled="disabled"
      />
      <label class="form-check-label" :for="id">{{ label }}</label>
    </div>
    <div class="mb-3" v-else-if="type == 'text'">
      <label :for="id" v-if="label" class="col-sm-12 form-label">{{ label }}</label>
      <input
        type="text"
        class="form-control"
        :id="id"
        :value="value"
        v-on:change="updateValue($event.target.value)"
        :disabled="disabled"
        :placeholder="placeholder"
      />
    </div>
    <div class="mb-3" v-else-if="type == 'list'">
      <label :for="id" v-if="label" class="col-sm-12 form-label">{{ label }}</label>
      <input
        type="text"
        class="form-control"
        :id="id"
        :value="value"
        placeholder="A,B,C"
        v-on:change="
          updateValue(Array.from(new Set($event.target.value.split(','))))
        "
        :disabled="disabled"
      />
    </div>
    <div class="mb-3" v-else-if="type == 'number'">
      <label :for="id" v-if="label" class="col-sm-12 form-label">{{ label }}</label>
      <input
        type="number"
        class="form-control"
        :id="id"
        :value="value"
        v-on:change="updateValue(Number($event.target.value))"
        :disabled="disabled"
        :placeholder="placeholder"
      />
    </div>
    <div class="mb-3" v-else-if="type == 'password'">
      <label :for="id" v-if="label" class="col-sm-12 form-label">{{ label }}</label>
      <div class="input-group">
        <input
          type="password"
          class="form-control"
          :id="id"
          autocomplete
          :value="value"
          v-on:change="updateValue($event.target.value)"
          :disabled="disabled"
          :placeholder="placeholder"
        />
        <button
          class="btn btn-outline-dark"
          type="button"
          v-clipboard:copy="value"
        >
          Copy
        </button>
      </div>
    </div>
    <div class="mb-3" v-else-if="type == 'select'">
      <label :for="id" v-if="label" class="col-sm-12 form-label">{{ label }}</label>
      <select
        class="form-select"
        :id="id"
        v-on:change="updateValue($event.target.value)"
        :disabled="disabled"
      >
        <option
          v-for="(item, index) in values"
          v-bind:key="index"
          selected="item == value"
          >{{ item }}</option
        >
      </select>
    </div>
    <div class="mb-3 row pb-4" v-else-if="type == 'intent'">
      <label for="managerBotIntents" class="col-sm-3 form-label">{{
        label
      }}</label>
      <div class="col-sm-9">
        <input
          type="number"
          class="form-control"
          min="0"
          max="32767"
          :value="value"
          @change="
            v => {
              v.target.value = v.target.value & 32767;
              updateValue(Number(v.target.value));
              fromIntents(v.target.value);
            }
          "
          @input="
            v => {
              updateValue(Number(v.target.value));
              fromIntents(v.target.value);
            }
          "
          :disabled="disabled"
        />
        <div class="form-row py-2">
          <div
            class="form-check form-check-inline col-sm-8 col-md-5"
            v-for="(intent, index) in this.intents"
            v-bind:key="index"
          >
            <input
              class="form-check-input"
              type="checkbox"
              v-bind:value="index"
              v-bind:id="id + index"
              v-model="selectedIntent"
              @change="calculateIntent()"
            />
            <label
              class="form-check-label"
              v-bind:for="'managerBotIntentBox' + index"
              >{{ intent }}</label
            >
          </div>
        </div>
      </div>
    </div>
    <div class="mb-3 row pb-4" v-else-if="type == 'presence'">
      <label class="col-sm-3 col-form-label">{{ label }}</label>
      <div class="col-sm-9">
        <div class="mb-3">
          <label :for="id + 'status'" class="col-sm-12 form-label"
            >Status</label
          >
          <select
            class="form-select"
            :id="id + 'status'"
            :value="value.status"
            @input="
              v => {
                value.status = v.target.value;
              }
            "
          >
            <option
              v-for="item in [
                '',
                'online',
                'dnd',
                'idle',
                'invisible',
                'offline'
              ]"
              :key="item"
              :disabled="!item"
              :selected="item == value"
              >{{ item }}</option
            >
          </select>
        </div>
        <div class="mb-3">
          <label :for="id + 'name'" class="col-sm-12 form-label">Name</label>
          <input
            type="text"
            class="form-control"
            :id="id + 'name'"
            :value="value.name"
            @input="
              v => {
                value.name = v.target.value;
              }
            "
          />
        </div>
        <div class="form-check">
          <input
            class="form-check-input"
            type="checkbox"
            :id="id + 'afk'"
            :checked="value.afk"
            @input="
              v => {
                value.afk = v.target.checked;
              }
            "
          />
          <label class="form-check-label" :for="id + 'afk'">AFK</label>
        </div>
      </div>
    </div>
    <span class="badge bg-warning text-dark" v-else
      >Invalid type "{{ type }}" for "{{ id }}"</span
    >
  </div>
</template>

<script>
export default {
  props: ["type", "id", "label", "values", "value", "disabled", "placeholder"],
  data: function() {
    return {
      intents: [
        "GUILDS",
        "GUILD_MEMBERS",
        "GUILD_BANS",
        "GUILD_INTEGRATIONS",
        "GUILD_EMOJIS",
        "GUILD_WEBHOOKS",
        "GUILD_INVITES",
        "GUILD_VOICE_STATES",
        "GUILD_PRESENCES",
        "GUILD_MESSAGES",
        "GUILD_MESSAGE_REACTIONS",
        "GUILD_MESSAGE_TYPING",
        "DIRECT_MESSAGES",
        "DIRECT_MESSAGE_REACTIONS",
        "DIRECT_MESSAGE_TYPING"
      ],
      selectedIntent: []
    };
  },
  mounted: function() {
    if (this.type == "intent") {
      this.fromIntents(this.value);
    }
  },
  methods: {
    calculateIntent() {
      this.intentValue = 0;
      this.selectedIntent.forEach(a => {
        this.intentValue += 1 << a;
      });
      this.updateValue(Number(this.intentValue));
    },
    fromIntents(val) {
      var _binary = Number(val)
        .toString(2)
        .split("")
        .reverse();
      var _newIntent = [];
      _binary.forEach((value, index) => {
        if (value === "1" && this.selectedIntent.indexOf(value) === -1) {
          _newIntent.push(index);
        }
      });
      this.selectedIntent = _newIntent;
    },
    updateValue: function(value) {
      this.$emit("input", value);
    }
  }
};
</script>
