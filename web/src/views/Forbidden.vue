<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50">
    <div class="text-center">
      <h1 class="text-6xl font-bold text-gray-300">403</h1>
      <p class="mt-4 text-xl text-gray-600">Access Denied</p>
      <p class="mt-2 text-gray-500">You don't have permission to access this page.</p>
      <RouterLink to="/dashboard" class="mt-6 inline-block rounded-lg bg-blue-600 px-6 py-2 text-white hover:bg-blue-700">
        Back to Dashboard
      </RouterLink>
    </div>
  </div>
</template>
