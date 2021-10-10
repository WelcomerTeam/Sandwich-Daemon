import userAPI from "../../api/user";

// initial state
const state = () => ({
  isLoggedIn: false,
  user: null,
  guilds: null,
});

// getters
const getters = {
  // get user
  // get is logged in
  // get guilds
};

// actions
const actions = {
  // fetch user
  // fetch guilds
  // add membership to guild
  // remove membership from guild
};

// mutations
const mutations = {
  // set user + isloggedin
  // set guilds
  // set guild
};

export default {
  namespaced: false,
  state,
  getters,
  actions,
  mutations,
};
