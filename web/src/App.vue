<template>
  <div id="app">
    <Navigation/>
    <div id="viewport">
      <router-view/>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import Component from 'vue-class-component';
import Navigation from './components/navigation/Navigation.vue';

@Component({
  components: {
    Navigation
  }
})
export default class App extends Vue {}
</script>

<style>
html {
  margin: 0;
}

body {
  margin: 0;
}

#app {
  height: 100vh;
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  display: flex;
  flex-direction: column;
  color: white;
}

#viewport {
  display: flex;
  align-items:     center;
  justify-content: center;
  margin-top: 60px;
  flex: 1;
  overflow-y: auto;
	background: repeating-linear-gradient(
		45deg,
		rgb(68, 68, 68),
		rgb(68, 68, 68) 60px,
		rgb(96, 96, 96) 60px,
		rgb(96, 96, 96) 120px
  );
}
</style>
