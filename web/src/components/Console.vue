<template>
  <div>
    <ul class="console mt-4 p-3 rounded-lg">
      <li class="text-center" v-if="!this.connected">
        <button class="btn btn-light mt-5" @click="connect()">Connect</button>
      </li>
      <li
        class="d-flex font-monospace text-white"
        v-for="(entry, index) in entries"
        v-bind:key="index"
      >
        <span class="text-white-50">{{ entry.time }}</span>
        <span
          v-if="levels[entry.level]"
          :class="'text-' + (levels[entry.level][1] || 'white')"
          >{{ levels[entry.level][0] || "???" }}</span
        >
        <span v-else class="text-white">???</span>

        <span>{{ entry.message }}</span>
        <div v-for="(arg, index) in entry.args" v-bind:key="index">
          <span v-if="index == 'error'" class="text-danger">
            <span>{{ index }}=</span>
            <span>{{ arg }}</span>
          </span>
          <span v-else>
            <span class="text-info">{{ index }}=</span>
            <span>{{ arg }}</span>
          </span>
        </div>
      </li>
    </ul>
    <div class="d-flex">
      <button class="btn btn-dark mr-2" @click="clear()">Clear</button>
      <div class="my-auto mr-2">
        <input
          class="mr-1"
          type="checkbox"
          v-model="autoscroll"
          id="autoscroll__checkbox"
        />
        <label for="autoscroll__checkbox">Autoscroll</label>
      </div>
      <div class="my-auto mr-2">
        <input
          class="mr-1"
          type="number"
          style="width: 60px"
          v-model="line_limit"
          id="max__lines"
        />
        <label for="autoscroll__checkbox">Line Limit</label>
      </div>
    </div>
  </div>
</template>

<style scoped>
.console {
  box-sizing: content-box;
  background: #0c0c0c;
  overflow: scroll;
  white-space: pre;
  height: 50vh;
}

.console > li > span {
  margin-right: 8px;
}

.console > li > div > span {
  margin-right: 8px;
}
</style>

<script>
import moment from "moment";
export default {
  name: "Console",
  props: ["wsurl", "limit", "auto"],
  data() {
    return {
      ws: undefined,
      connected: false,
      autoscroll: true,
      line_limit: this.limit,
      entries: [],
      white: ["level", "time", "message"],
      levels: {
        trace: ["TRC", "primary"],
        debug: ["DBG", "warning"],
        info: ["INF", "success"],
        warn: ["WRN", "danger"],
        error: ["ERR", "danger"],
        fatal: ["FTL", "danger"],
        panic: ["PNC", "danger"],
        "": ["???", "white"]
      }
    };
  },
  mounted: function() {
    if (this.auto) {
      this.connect();
    }
  },
  methods: {
    clear() {
      this.entries = [];
      this.addentry({ message: "Cleared logs" });
    },
    connect() {
      // this.ws = new WebSocket((window.location.protocol == 'http:' ? 'ws://' : 'wss://') + window.location.host + this.wsurl);
      this.ws = new WebSocket("ws://127.0.0.1:5469/api/console");
      this.ws.onopen = this.onopen;
      this.ws.onmessage = this.onmessage;
      this.ws.onclose = this.onclose;
      this.ws.onerror = this.onerror;
    },
    addentry(data) {
      // preserve white and then add everything else to its own KV dict which is sorted
      var message = {
        time: moment(data.time).format("MMM D HH:mm:ss"),
        level: data.level,
        message: data.message,
        args: {}
      };

      for (var key in data) {
        if (!this.white.includes(key)) {
          message.args[key] = data[key];
        }
      }

      this.entries.push(message);
      if (this.line_limit > 0) {
        this.entries =
          this.entries.length >= this.line_limit
            ? this.entries.slice(
                Math.max(this.entries.length - this.line_limit, 1)
              )
            : this.entries;
      }

      if (this.autoscroll) {
        this.$nextTick(() => {
          var container = this.$el.querySelector(".console");
          container.scrollTop = container.scrollHeight;
        });
      }
    },
    onopen() {
      this.connected = true;
      this.addentry({ message: "Connected to console websocket" });
    },
    onmessage(event) {
      this.addentry(JSON.parse(event.data));
    },
    onclose() {
      this.addentry({ message: "Connection was closed" });
      this.connected = false;
    },
    onerror() {}
  }
};
</script>
