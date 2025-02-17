import { createRouter, createWebHistory } from "vue-router";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: () => import("./views/Home.vue") },
    { path: "/login", component: () => import("./views/Login.vue") },
    {
      path: "/register",
      component: () => import("./views/Register.vue"),
    },
  ],
});
