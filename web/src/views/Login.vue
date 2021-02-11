<template>
  <div id="login-box">
    <form id="login-form">

      <div class="login-row">
        <label for="login-form-username-input">username</label>
        <input type="text"
               name="username"
               id="login-form-username-input"
               placeholder="username"
               autofocus
        />
      </div>

      <div class="login-row">
        <label for="login-form-password-input">password</label>
        <input type="password"
               name="password"
               id="login-form-password-input"
               placeholder="************"
        />
      </div>

      <div class="login-button-row">
        <input type="submit" id="login-form-submit-button" value="log in"/>
      </div>

    </form>

    <div v-if="errorMessage" class="login-error">{{ errorMessage }}</div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component
export default class Login extends Vue {
  errorMessage = "";
}
</script>

<style scoped>
#login-box {
  background-color: #252525;
  border-radius: 14px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-content: center;
  width: 325px;
  padding: 1em;
}

#login-form {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-content: center;
}

.login-row {
  width: 300px;
  margin: 0.75em 1em;
  text-align: left;
}

.login-row label {
  padding-right: 0.5em;
  margin-left: 4px;
  margin-bottom: 4px;
  display: grid;
  font-size: 16px;
}

.login-row input[type=text], input[type=password] {
  height: 25px;
  font-size: 20px;
  border-radius: 6px;
  font-family: inherit;
  padding: 0.25em 10px;
  width: 275px;
}

input[type=text]:focus, input[type=password]:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgb(229, 53, 69);
}

.login-button-row {
  height: 30px;
  width: 300px;
  margin: 0.75em 0em;
}
.login-button-row input[type=submit] {
  height: 30px;
  width: 300px;
  font-size: 20px;
  font-family: inherit;
  border-style: outset;
  border-width: 2px;
  border-radius: 6px;
  border-color: white;
  background: rgba(37,37,37,1);
  color: white;
}

.login-button-row input[type=submit]:hover {
  background: rgba(255,255,255,0.1) !important;
}

.login-button-row input[type=submit]:active {
  background: rgba(255,255,255,0.2) !important;
}

.login-button-row input[type=submit]:focus {
  outline: none;
}

</style>