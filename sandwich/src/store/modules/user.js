import userAPI from "../../api/user";

const state = () => ({
  isLoggedIn: false,
  isAuthenticated: false,
  user: null,
});

const getters = {
  isLoggedIn: (state) => {
    return state.isLoggedIn;
  },
  isAuthenticated: (state) => {
    return state.isAuthenticated;
  },
  getUser: (state) => {
    return state.user;
  },
};

const actions = {
  fetchUser({ commit }) {
    userAPI.getUser(
      (u) => {
        commit("setUser", [u]);
      },
      () => {
        commit("setUser", [null]);
      }
    );
  },
};

const mutations = {
  setUser(state, [userObject]) {
    state.user = userObject?.user;
    state.isLoggedIn = userObject?.logged_in;
    state.isAuthenticated = userObject?.authenticated;
  },
};

export default {
  namespaced: false,
  state,
  getters,
  actions,
  mutations,
};
