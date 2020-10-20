<template>
  <div id="app">
    <Navigation/>
    <div id="nav-buffer"/>
    <router-view/>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator';
import Navigation from './components/navigation/Navigation.vue';

@Component({
  components: {
    Navigation
  }
})
export default class App extends Vue {}
</script>

<style>
body {
  margin: 0;
}

#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

#nav-buffer {
  margin-top: 60px;
}
</style>
