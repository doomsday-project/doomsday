<template>
  <div id="nav-container">
    <div id="navbar">
      <div id="hamburger-box" 
          class="hamburger-box navbar-content navbar-border navbar-button"
          v-if="hamburger"
          v-bind:class="{ hamburgerBoxOpen: hamburger && menuOpen }"
          v-on:click="menuOpen=!menuOpen">
        <svg id="hamburger" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="none" viewBox="0 0 201 146.17"><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><rect class="cls-1" x="0.5" y="0.5" width="200" height="29.27"/><rect class="cls-1" x="0.5" y="59.04" width="200" height="29.27"/><rect class="cls-1" x="0.5" y="116.4" width="200" height="29.27"/></g></g></svg>
      </div>

      <div id="logo-box" class="navbar-content navbar-border no-select" >
        <div id="logo">doomsday</div>
      </div>

      <div id="sub-logo" class="navbar-content no-select">
        <div id="subtitle">an x509 monitoring tool</div>
      </div>
    </div>

		<div id="hamburger-menu" 
         class="sticky navbar-border"
         v-bind:class="{ hamburgerMenuOpen: hamburger && menuOpen }">
			<div id="logout-button" 
           class="hamburger-menu-button navbar-button no-select">
        logout
      </div>
		</div>
  </div>
</template>

<script lang="ts">

import { Component, Prop, Vue } from 'vue-property-decorator';

@Component
export default class Navigation extends Vue{
  @Prop() private hamburger!: boolean;
  menuOpen = false;
}
</script>

<style scoped>
#navbar {
	background-color: #252525;
  display: flex;
  position: fixed;
	height: 60px;
  z-index: 100;
	top: 0;
	width: 100%;
}

#logo {
  color: white;
	font-size: 40px;
}

#sub-logo {
  color: white;
  font-size: 20px;
}

#hamburger {
	height: 25px;
	width: 25px;
	flex: 1 1 auto;
}

.no-select {
	-webkit-touch-callout: none;
  -webkit-user-select: none;
  -khtml-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
}

.navbar-button:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.navbar-button:active {
  background-color: rgba(255, 255, 255, 0.2);
}

.navbar-content {
	display: flex;
	align-items: center;
	justify-content: center;
	padding-left: 15px;
	padding-right: 15px;
}

.navbar-border {
	border-right-color: rgba(255, 255, 255, 0.3);
	border-right-style: solid;
	border-right-width: thin;
}

.cls-1{fill:#ffffff;stroke:#fff;stroke-miterlimit:10;}

#hamburger-box {
  width: 25px;
  transition: width .19s;
  transition-timing-function: linear;
}

.hamburgerBoxOpen {
  width: 220px !important;
  transition-delay: .05s !important;
}

#hamburger-menu {
	background-color: rgba(37, 37, 37, 0.9);
	height: calc(100% - 60px);
	width: 250px;
	z-index: 100;
	top: 60px;
	left: -251px;
  display: inline-block;
  position: fixed;
  transition: left .25s;
  transition-timing-function: linear;
}

.hamburgerMenuOpen {
  left: -1px !important;
}

.hamburger-menu-button {
	display: flex;
	align-items: center;
	align-content: center;
	justify-content: center;
	height: 30px;
	padding: 5px 20px 5px 20px;
	font-size: 24px;
	color: white;
	border-style: solid none solid none;
	border-width: thin;
	border-color: rgba(255, 255, 255, 0.3);
	-webkit-touch-callout: none;
  -webkit-user-select: none;
  -khtml-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
	background-color: #252525;
}

.hamburger-menu-button-inactive {
	color: grey !important;
}
</style>