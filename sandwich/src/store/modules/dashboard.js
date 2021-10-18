import dashboardAPI from "../../api/dashboard";

const state = () => ({
  configurationLoaded: false,
  configurationFetchError: "",
  configuration: null,

  statusLoaded: false,
  statusFetchError: "",
  status: null,
});

const getters = {
  // Dashboard configuration
  hasConfigurationLoaded: (state) => {
    return state.configurationLoaded;
  },
  getConfigurationFetchError: (state) => {
    return state.configurationFetchError;
  },
  getManagers: (state) => {
    return state.configuration?.managers;
  },
  getConfiguration: (state) => {
    return state.configuration;
  },

  // Status
  hasStatusLoaded: (state) => {
    return state.statusLoaded;
  },
  getStatusFetchError: (state) => {
    return state.statusFetchError;
  },
  getStatus: (state) => {
    return state.status;
  },
};

const actions = {
  fetchDashboardConfig({ commit }) {
    dashboardAPI.getDashboardConfig(
      (dashboard) => {
        commit("setDashboardConfig", [dashboard, null]);
      },
      (e) => {
        commit("setDashboardConfig", [null, e]);
      }
    );
  },
  fetchStatus({ commit }) {
    dashboardAPI.getStatus(
      (status) => {
        commit("setStatus", [status, null]);
      },
      (e) => {
        commit("setStatus", [null, e]);
      }
    );
  },
};

const mutations = {
  setDashboardConfig(state, [dashboardObject, errorObject]) {
    if (errorObject == undefined) {
      state.configurationLoaded = true;
    }
    state.configurationFetchError = errorObject;
    state.configuration = dashboardObject?.configuration;
  },
  setStatus(state, [statusObject, errorObject]) {
    console.log(statusObject, errorObject);
    if (errorObject == undefined) {
      state.statusLoaded = true;
    }
    state.statusFetchError = errorObject;
    state.status = statusObject;
  },
};

export default {
  namespaced: false,
  state,
  getters,
  actions,
  mutations,
};
