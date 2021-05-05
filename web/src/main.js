import Vue from "vue";
import Vuex from "vuex";
import VueClipboard from "vue-clipboard2";

import axios from "axios";
import JSONBig from "json-bigint";
var jsonBig = JSONBig({ storeAsString: true });

import { TooltipPlugin } from 'bootstrap-vue'

import "bootstrap";
import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap-vue/dist/bootstrap-vue.css";

import "moment";
import "chartjs-adapter-moment";

import App from "./App.vue";
import router from "./router";

import "./registerServiceWorker";

Vue.config.productionTip = false;

Vue.use(TooltipPlugin)
Vue.use(VueClipboard);
Vue.use(Vuex);

const store = new Vuex.Store({
  state: {
    error: "",
    userLoading: true,
    userAuthenticated: false,
    user: {}
  }
});

new Vue({
  store,
  router,
  render: h => h(App),
  mounted() {
    this.fetchMe();
  },
  data() {
    return {
      version: "🥪"
    };
  },
  methods: {
    fetchMe() {
      axios
        .get("/api/me", { transformResponse: [data => jsonBig.parse(data)] })
        .then(result => {
          var data = result.data;
          if (data.success) {
            store.state.userAuthenticated = data.data.authenticated;
            store.state.user = data.data.user;
          } else {
            store.state.error = data.error;
          }
        })
        .catch(error => {
          if (error.response?.data) {
            store.state.error =
              error.response.data.error || error.response.data;
          } else {
            store.state.error = error.text || error.toString();
          }
        })
        .finally(() => {
          store.state.userLoading = false;
        });
    }
  }
}).$mount("#app");
