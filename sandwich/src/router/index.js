import { createWebHistory, createRouter } from "vue-router";

import Status from "../pages/Status.vue";

const routes = [
  {
    path: "/",
    name: "Service Status",
    component: Status,
  },
  {
    path: "/dashboard",
    name: "Dashboard",
    component: () =>
      import(/* webpackChunkName: "dashboard" */ "../pages/Dashboard.vue"),
    children: [
      {
        path: "settings",
        component: () =>
          import(
            /* webpackChunkName: "dashboard" */ "../pages/dashboard/Settings.vue"
          ),
      },
      {
        path: "managers",
        component: () =>
          import(
            /* webpackChunkName: "dashboard" */ "../pages/dashboard/Managers.vue"
          ),
        children: [
          {
            path: ":id",
            component: () =>
              import(
                /* webpackChunkName: "dashboard" */ "../pages/dashboard/ManagerView.vue"
              ),
          },
        ],
      },
    ],
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
