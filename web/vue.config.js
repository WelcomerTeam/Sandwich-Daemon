// const BundleAnalyzerPlugin = require("webpack-bundle-analyzer").BundleAnalyzerPlugin;
const webpack = require("webpack");

module.exports = {
  runtimeCompiler: true,
  integrity: true,
  pwa: {
    name: "Sandwich Daemon",
    themeColor: "#212529",
    msTileColor: "#212529",
    appleMobileWebAppCapable: "yes",
    appleMobileWebAppStatusBarStyle: "black-translucent",
    manifestOptions: {
      short_name: "Sandwich Daemon",
      name: "Sandwich Daemon",
      lang: "en",
      description:
        "Sandwich Daemon allows you to manage your discord bot from a central place",
      background_color: "#ffffff",
      theme_color: "#212529",
      dir: "ltr",
      display: "standalone",
      orientation: "any",
      prefer_related_applications: false,
    },
    workboxOptions: {
      skipWaiting: true,
      clientsClaim: true,
    },
  },
  configureWebpack: {
    plugins: [
      // new BundleAnalyzerPlugin(),
      new webpack.ContextReplacementPlugin(/moment[/\\]locale$/, /en/),
    ],
    externals: {
      moment: "moment",
    },
  },
  devServer: {
    proxy: {
      "^/api": {
        methods: ["GET", "POST", "PUT", "HEAD"],
        target: "http://127.0.0.1:5469",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/login": {
        target: "http://127.0.0.1:5469",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/logout": {
        target: "http://127.0.0.1:5469",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
      "/oauth2/callback": {
        target: "http://127.0.0.1:5469",
        ws: true,
        changeOrigin: true,
        withCredentials: true,
        secure: false,
      },
    },
  },
};
