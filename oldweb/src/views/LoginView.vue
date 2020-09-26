<template>
  <div id="login-page-container">
    <Navigation />
    <div id="login-box">
      <form id="login-form">
        <div class="login-text-box">
          <label for="login-form-username">Username</label>
          <input type="text"
                  name="username"
                  id="login-form-username-input"
          />
        </div>
        <div class="login-text-box">
          <label for="login-form-password">Password</label>
          <input type="password"
                  name="password"
                  id="login-form-password-input"
          />
        </div>
        <div class="login-button-box">
          <input type="submit" id="login-form-submit-button" value="sign in"/>
        </div>
      </form>
      <div class="login-error" v-if="error_message">{{ error_message }}</div>
    </div>
  </div>
</template>

import { Vue, Component }
import Navigation from "@/components/navigation/Navigation.vue"

<script>
export default {
  components: {
    Navigation,
  },
}
</script>

<style>

</style>