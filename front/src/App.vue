<template>
  <div id="app" class="relative min-h-screen bg-transparent text-white">
    <!-- 动态极光背景 -->
    <div class="fixed inset-0 z-0 overflow-hidden pointer-events-none select-none">
      <!-- 紫色光团 - 左上 -->
      <div class="absolute top-[-10%] left-[-10%] w-[50vw] h-[50vw] rounded-full bg-primary-700/20 mix-blend-screen filter blur-[120px] animate-aurora-1"></div>
      <!-- 靛蓝光团 - 右上 -->
      <div class="absolute top-[0%] right-[-10%] w-[45vw] h-[45vw] rounded-full bg-indigo-900/20 mix-blend-screen filter blur-[120px] animate-aurora-2"></div>
      <!-- 深紫光团 - 底部 -->
      <div class="absolute bottom-[-20%] left-[10%] w-[70vw] h-[50vw] rounded-full bg-violet-900/20 mix-blend-screen filter blur-[140px] animate-aurora-3"></div>
    </div>

    <!-- 主体内容 -->
    <div class="relative z-10">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
// 应用根组件
</script>
