import axios from "axios";

export function fetch(config, callback, errorCallback) {
  axios
    .request(config)
    .then((result) => {
      if (result?.data?.ok) {
        callback(result?.data?.data);
      } else {
        errorCallback(result?.data?.error);
      }
    })
    .catch((e) => {
      if (e.request?.status == 401) {
        window.open("/login", "_self");
      }
      errorCallback(e);
    });
}
