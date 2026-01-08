<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center py-20">
    <div class="text-center relative z-10" style="user-select: none">
      <div class="relative group mx-auto mb-8">
        <!-- 背景光晕 -->
        <div class="absolute -inset-4 rounded-full bg-primary-500/20 blur-2xl opacity-50 group-hover:opacity-75 transition-all duration-700"></div>

        <!-- 图标容器 -->
        <div class="relative mx-auto flex h-32 w-32 items-center justify-center rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-xl shadow-2xl transition-all duration-500 group-hover:scale-105 group-hover:bg-white/10 group-hover:ring-white/20">
          <SparklesIcon class="h-16 w-16 text-primary-400/90 drop-shadow-[0_0_15px_rgba(139,92,246,0.5)]"/>
        </div>

        <!-- 装饰元素 -->
        <div class="absolute top-0 right-0 h-8 w-8 rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-md flex items-center justify-center animate-bounce" style="animation-duration: 3s;">
          <CloudArrowUpIcon class="h-4 w-4 text-primary-300"/>
        </div>
      </div>

      <h3 class="text-3xl font-bold tracking-tight text-transparent bg-clip-text bg-gradient-to-b from-white to-white/60 sm:text-4xl font-display mb-3">
        暂无图片
      </h3>
      <p class="mt-2 max-w-sm mx-auto text-lg text-white/50 leading-relaxed font-light">
        上传您的第一张图片，开启您的光影画廊
      </p>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {CloudArrowUpIcon, SparklesIcon} from '@heroicons/vue/24/outline'
</script>
