<template>
  <button :type="type"
          v-bind:class="{ pending: pending, active: !pending }"
          v-bind:disabled="pending"
  >
    {{ text }}
  </button>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator'

@Component
export default class InputButton extends Vue{
  @Prop({required: true})    text!:    string;
  @Prop({default: "button"}) type!:    string;
  @Prop({default: false})    pending!: boolean;
}
</script>

<style scoped>
button {
  position: relative;
  overflow: hidden;
  height: 30px;
  width: 300px;
  font-size: 20px;
  font-family: inherit;
  border-style: outset;
  border-width: 2px;
  border-radius: 6px;
  border-color: white;
  background: rgb(37,37,37);
  color: white;
}

.active:hover {
  background: rgba(255,255,255,0.1) !important;
}

.active:active {
  background: rgba(255,255,255,0.2) !important;
}

@keyframes pending-scroll {
  from { transform: translateX(0px); }
  to   { transform: translateX(170px ); }
}

.pending {
  color: grey;
}

.pending::before {
  content: "";
  position: absolute;
  width: 300%;
  height: 300%;
  z-index: 100;
  top: -100%;
  left: -100%;
  background: repeating-linear-gradient(
		45deg,
		rgba(255, 255, 255, 0.1),
		rgba(255, 255, 255, 0.1) 60px,
		rgba(255, 255, 255, 0.2) 60px,
		rgba(255, 255, 255, 0.2) 120px
  );

  animation-name: pending-scroll;
  animation-timing-function: linear;
  animation-duration: 2s;
  animation-iteration-count: infinite;
}
</style>