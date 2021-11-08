import { fetch } from "./index";

export default {
  getSandwichConfig(callback, errorCallback) {
    fetch({ url: "/api/sandwich" }, callback, errorCallback);
  },
  updateSandwichConfig(data, callback, errorCallback) {
    fetch(
      { url: "/api/sandwich", method: "patch", data: data },
      callback,
      errorCallback
    );
  },

  getStatus(callback, errorCallback) {
    fetch({ url: "/api/status" }, callback, errorCallback);
  },
  getManagerStatus(manager, callback, errorCallback) {
    fetch({ url: `/api/status?manager=${manager}` }, callback, errorCallback);
  },

  createNewManager(data, callback, errorCallback) {
    fetch(
      { url: "/api/manager", method: "post", data: data },
      callback,
      errorCallback
    );
  },
  updateManagerConfig(data, callback, errorCallback) {
    fetch(
      { url: "/api/manager", method: "patch", data: data },
      callback,
      errorCallback
    );
  },
  deleteManager(manager, callback, errorCallback) {
    fetch(
      { url: `/api/manager?manager=${manager}`, method: "delete" },
      callback,
      errorCallback
    );
  },

  initializeManager(manager, callback, errorCallback) {
    fetch(
      { url: `/api/manager/initialize?manager=${manager}`, method: "post" },
      callback,
      errorCallback
    );
  },

  createManagerShardGroup(data, callback, errorCallback) {
    fetch(
      { url: "/api/manager/shardgroup", method: "post", data: data },
      callback,
      errorCallback
    );
  },
  stopShardGroups(manager, callback, errorCallback) {
    fetch(
      { url: `/api/manager/shardgroup?manager=${manager}`, method: "delete" },
      callback,
      errorCallback
    );
  },
};
