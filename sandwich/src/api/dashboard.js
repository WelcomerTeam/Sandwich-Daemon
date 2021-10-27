import { fetch } from "./index";

export default {
  getDashboardConfig(callback, errorCallback) {
    fetch({ url: "/api/dashboard" }, callback, errorCallback);
  },
  getStatus(callback, errorCallback) {
    fetch({ url: "/api/status" }, callback, errorCallback);
  },
  getManagerStatus(manager, callback, errorCallback) {
    fetch({ url: `/api/status?manager=${manager}` }, callback, errorCallback);
  },
  updateManagerConfig(data, callback, errorCallback) {
    fetch(
      { url: "/api/manager", method: "post", data: data },
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
};
