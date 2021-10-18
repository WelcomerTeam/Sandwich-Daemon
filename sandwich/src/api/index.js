import axios from "axios";

export function fetch(url, callback, errorCallback) {
  axios
    .get(url)
    .then((result) => {
      if (result?.data?.ok) {
        callback(result?.data?.data);
      } else {
        errorCallback(result?.data?.error);
      }
    })
    .catch((e) => {
      errorCallback(e);
    });
}
