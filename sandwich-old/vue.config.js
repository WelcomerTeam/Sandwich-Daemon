const webpack = require("webpack");

module.export = {
  integrity: false,
  productionSourceMap: false,
  configureWebpack: {
    plugins: [new webpack.ContextReplacementPlugin(/moment[/\\]locale$/, /en/)],
  },
  devServer: {
    proxy: {
      "^/api": {
        methods: ["GET", "POST", "PUT", "HEAD"],
        target: "http://127.0.0.1:14999",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/login": {
        target: "http://127.0.0.1:14999",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/logout": {
        target: "http://127.0.0.1:14999",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/oauth2/callback": {
        target: "http://127.0.0.1:14999",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
    },
  },
};
