import axios from "axios";

export function fetch(config, callback, errorCallback) {
  axios
    .request(config)
    .then((result) => {
      if (result?.data?.ok) {
        console.debug("Callback", result?.data?.data);
        callback(result?.data?.data);
      } else {
        console.debug("NOK Callback", result?.data?.error);
        errorCallback(result?.data?.error);
      }
    })
    .catch((e) => {
      console.debug("Catchback", e);
      if (e.request?.status == 401) {
        window.open("/login", "_self");
      }
      errorCallback(e);
    });
}
