<template>
  <div id="viewport">
    <h1>DASHBOARD</h1>
    <button v-on:click="$router.push({name: 'Login'})">
      to login
    </button>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import Navigation from '../components/navigation/Navigation.vue';

@Component({
  components: {
    Navigation,
  },
})
export default class Dashboard extends Vue {}
</script>