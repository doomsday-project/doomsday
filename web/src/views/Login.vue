<template>
  <div id="login-box">
    <form id="login-form" v-on:submit.prevent="onSubmit">

      <div class="login-row">
        <label for="login-form-username-input">username</label>
        <input type="text"
               name="username"
               v-model.lazy="formUsername"
               id="login-form-username-input"
               placeholder="username"
               ref="usernameField"
               autofocus
        />
      </div>

      <div class="login-row">
        <label for="login-form-password-input">password</label>
        <input type="password"
               name="password"
               v-model.lazy="formPassword"
               id="login-form-password-input"
               placeholder="************"
               ref="passwordField"
        />
      </div>

      <div class="login-button-row">
        <input-button type="submit" v-bind:text="buttonText" v-bind:pending="buttonPending">
        </input-button>
      </div>

    </form>

    <transition name="error">
      <div class="login-row"
           v-if="errorMessage"
      > 
        <div id="login-error"
              v-if="errorMessage"
        >
        {{ errorMessage }}
        </div>
      </div>
    </transition>
  </div>
</template>

<script lang="ts">
import { Component, Watch, Vue } from 'vue-property-decorator';
import InputButton from '@/components/inputs/InputButton.vue'

class BadPasswordError extends Error {}

@Component({
  components: {
    InputButton
  }
})
export default class Login extends Vue {
  errorMessage = "";
  formUsername = "";
  formPassword = "";
  errorCount   = 0;
  buttonPending = false;
  buttonText = "log in"

  mounted(): void {
    this.activateButton();
    this.moveToUsername();
  }

  onSubmit(): void {
    //TODO

    if (this.formUsername.length == 0) {
      this.setError("Please provide a username.");
      this.moveToUsername();
      return
    }

    if (this.formPassword.length == 0) {
      this.setError("Please provide a password.");
      this.moveToPassword();
      return;
    }

    this.suspendButton();
    this.errorMessage = "";

    this.doLogin(this.formUsername, this.formPassword)
      .then(() => {
        this.errorMessage = "";
        this.$router.push({name: "Dashboard"});
      })
      .catch((e: Error) => {
        this.setError(e.message);
        if (e instanceof BadPasswordError) {
          this.moveToPassword();
        }
      })
      .finally(() => {
        this.activateButton();
      });
  }

  @Watch('errorCount')
  onPropertyChanged() {
    console.log("Error message updated");
    const element = document.getElementById("login-error"); 
    if (element == null) {
      return
    }
    element.classList.remove("error-flash");
    //triggers reflow so that when animation is re-added, it plays.
    void element.offsetWidth;
    element.classList.add("error-flash");
  }

  setError(msg: string): void {
    this.errorMessage = msg;
    this.errorCount += 1;
  }

  doLogin(username: string, password: string): Promise<void | null> {
    return new Promise(resolve => { setTimeout(resolve, 2000)})
      .then(() => {
        const correctUsername = "foo";
        const correctPassword = "bar";

        if (username != correctUsername ||
          password != correctPassword) {
          throw new BadPasswordError("The username or password was incorrect.");
        }
      });
  }

  moveToUsername(): void {
    (this.$refs.usernameField as HTMLElement).focus();
  }

  moveToPassword(): void {
    const passField = this.$refs.passwordField as HTMLInputElement;
    passField.focus();
    passField.select();
  }

  suspendButton(): void {
    this.buttonPending = true;
    this.buttonText    = "logging in...";
  }

  activateButton(): void {
    this.buttonPending = false;
    this.buttonText    = "log in"
  }
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
  padding: 10px;
}

#login-form {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-content: center;
}

.login-row {
  box-sizing: border-box;
  width: 300px;
  margin: 10px 1em;
  text-align: left;
}

.login-row label {
  padding-right: 0.5em;
  margin-left: 4px;
  margin-bottom: 6px;
  display: grid;
  font-size: 16px;
  font-weight: bold;
}

.login-row input[type=text], input[type=password] {
  height: 25px;
  font-size: 20px;
  border-radius: 6px;
  font-family: inherit;
  padding: 0.25em 10px;
  width: 275px;
}

input {
  outline: none;
}

input[type=text]:focus, input[type=password]:focus {
  box-shadow: 0 0 0 2px rgb(229, 53, 69);
}

.login-button-row {
  box-sizing: border-box;
  height: 30px;
  width: 300px;
  margin: 10px 0 10px 0;
}

#login-error {
  display: flex;
  border-radius: 6px;
  background-color: rgb(229, 53, 69);
  width: 300px;
  color: black;
  padding: 0.3em;
  text-align: center;
  box-sizing: border-box;
  align-items: center;
  justify-content: center;
}

.error-enter {
  opacity: 0 !important;
  margin: 0 !important;
  padding: 0 !important;
  max-height: 0 !important;
}

.error-enter-to {
  max-height: 1000px;
}

.error-enter-active {
  transition: opacity 1s, max-height 1s, margin 1s, padding 1s;
}

.error-enter-to {
  transform: scaleY(1);
}

@keyframes error-flash-frames {
  0% { 
    background-color: rgb(229,53,69); 
  } 
  100% { 
    background-color: rgb(235,98,111);
    color: white;
  }
}

.error-flash {
  animation-name: error-flash-frames;
  animation-duration: 0.3s;
  animation-direction: alternate;
  animation-iteration-count: 2;
  animation-timing-function: linear;
}
</style>