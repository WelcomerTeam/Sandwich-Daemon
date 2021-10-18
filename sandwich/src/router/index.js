import { createWebHistory, createRouter } from "vue-router";

import Status from "../pages/Status.vue";
import Dashboard from "../pages/Dashboard.vue";

const routes = [
  {
    path: "/",
    name: "Service Status",
    component: Status,
  },
  {
    path: "/dashboard",
    name: "Dashboard",
    component: Dashboard,
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
