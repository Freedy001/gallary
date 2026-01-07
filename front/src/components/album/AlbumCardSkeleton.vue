<template>
  <div
      class="relative aspect-square rounded-2xl overflow-hidden bg-white/5 ring-1 ring-white/10 animate-pulse"
  >
    <!-- 封面占位 -->
    <div class="w-full h-full flex items-center justify-center bg-gradient-to-br from-gray-800/50 to-gray-900/50">
      <div class="h-12 w-12 rounded-lg bg-white/5"></div>
    </div>

    <!-- 渐变遮罩 -->
    <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>

    <!-- 信息栏占位 -->
    <div class="absolute bottom-0 left-0 right-0 p-4">
      <div class="h-4 w-24 bg-white/10 rounded mb-2"></div>
      <div class="h-3 w-16 bg-white/5 rounded"></div>
    </div>
  </div>
</template>
