import { fetch } from "./index";

export default {
  getDashboardConfig(callback, errorCallback) {
    fetch({ url: "/api/dashboard" }, callback, errorCallback);
  },
  getStatus(callback, errorCallback) {
    fetch({ url: "/api/status" }, callback, errorCallback);
  },
  updateManagerConfig(data, callback, errorCallback) {
    fetch(
      { url: "/api/manager", method: "post", data: data },
      callback,
      errorCallback
    );
  },
};
