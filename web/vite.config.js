import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:14999",
        changeOrigin: true,
      },
      "/(login|logout|callback)": {
        target: "http://127.0.0.1:14999",
        changeOrigin: true,
      },
    },
  },
});
