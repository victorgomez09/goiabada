<script setup lang="ts">
import { auth } from "../constants/login.constant";
</script>

<template>
  <div>
    <button
      @click="auth.loginWithPopup()"
      class="btn btn-primary"
      title="Login with Popup"
    >
      Login with Popup
    </button>
    <button @click="auth.login()" class="btn btn-primary" title="Login">
      Login
    </button>
  </div>
</template>
