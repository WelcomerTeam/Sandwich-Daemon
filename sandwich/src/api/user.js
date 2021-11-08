import { fetch } from "./index";

export default {
  getUser(callback, errorCallback) {
    fetch({ url: "/api/user" }, callback, errorCallback);
  },
};
