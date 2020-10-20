<template>
  <div id="viewport">
    <h1>LOGIN</h1>
    <button v-on:click="$router.push({name: 'Dashboard'})">
      to dashboard
    </button>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component
export default class Login extends Vue {}
</script>
