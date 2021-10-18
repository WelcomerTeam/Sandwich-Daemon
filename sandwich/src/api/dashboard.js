import { fetch } from "./index";

export default {
  getDashboardConfig(callback, errorCallback) {
    fetch("/api/dashboard", callback, errorCallback);
  },
  getStatus(callback, errorCallback) {
    fetch("/api/status", callback, errorCallback);
  },
};
