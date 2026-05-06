<template>
  <router-view v-slot="{ Component }">
    <transition name="fade" mode="out-in" :duration="{ enter: 160, leave: 130 }">
      <component :is="Component" />
    </transition>
  </router-view>
</template>

<script setup></script>

<style scoped></style>
