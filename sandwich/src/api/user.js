import { fetch } from "./index";

export default {
  getUser(callback, errorCallback) {
    fetch("/api/user", callback, errorCallback);
  },
};
