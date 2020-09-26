<template>
	<div id="navbar" class="sticky">
    <div id="hamburger-box" class="navbar-content navbar-border navbar-button" v-if="show-menu">
      <svg id="hamburger" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="none" viewBox="0 0 201 146.17"><defs><style>.cls-1{fill:#ffffff;stroke:#fff;stroke-miterlimit:10;}</style></defs><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><rect class="cls-1" x="0.5" y="0.5" width="200" height="29.27"/><rect class="cls-1" x="0.5" y="59.04" width="200" height="29.27"/><rect class="cls-1" x="0.5" y="116.4" width="200" height="29.27"/></g></g></svg>
    </div>
    <div class="navbar-content navbar-border" id="logo-box">
      <div id="logo">doomsday</div>
    </div>
    <div class="navbar-content">
      <div id="subtitle">an x509 monitoring tool</div>
    </div>
	</div>
</template>

<script>
export default {
  props: [ 'show-menu' ]
}
</script>

<style>

</style>