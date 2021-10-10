import { createWebHistory, createRouter } from "vue-router";

import Status from "../pages/Status.vue";

const routes = [
  {
    path: "/",
    name: "Service Status",
    component: Status,
  },
  {
    path: "/:catchAll(.*)",
    component: () => import("../pages/NotFound.vue"),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
