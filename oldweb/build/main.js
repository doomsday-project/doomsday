import Vue from "vue";
import VueRouter from "vue-router";
import LoginView from "./views/LoginView.vue";
const router = new VueRouter({
    routes: [
        { path: '/', component: LoginView }
    ]
});
const app = new Vue({
    router,
}).$mount("#app");
