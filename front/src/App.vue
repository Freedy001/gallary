<template>
  <div id="app" class="relative min-h-screen bg-transparent text-white">
    <!-- 动态极光背景 -->
    <div class="fixed inset-0 z-0 overflow-hidden pointer-events-none select-none">
      <!-- 紫色光团 - 左上 -->
<!--      <div class="absolute top-[-10%] left-[-10%] w-[50vw] h-[50vw] rounded-full bg-primary-700/20 mix-blend-screen filter blur-[120px] animate-aurora-1"></div>-->
      <!-- 靛蓝光团 - 右上 -->
<!--      <div class="absolute top-[0%] right-[-10%] w-[45vw] h-[45vw] rounded-full bg-indigo-900/20 mix-blend-screen filter blur-[20x] animate-aurora-2"></div>-->
      <!-- 深紫光团 - 底部 -->
<!--      <div class="absolute bottom-[-20%] left-[10%] w-[70vw] h-[50vw] rounded-full bg-violet-900/20 mix-blend-screen filter blur-[140px] animate-aurora-3"></div>-->

      <!-- 流星特效 Canvas -->
<!--      <canvas ref="meteorCanvas" class="absolute inset-0 w-full h-full z-[1]"></canvas>-->
    </div>

    <!-- 主体内容 -->
    <div class="relative z-10">
      <router-view v-slot="{ Component, route }">
        <keep-alive :include="keepAliveComponents">
          <component :is="Component" :key="route.meta.usePathAsKey ? route.fullPath : undefined" />
        </keep-alive>
      </router-view>
    </div>

    <!-- 全局确认对话框 -->
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import {onMounted, onUnmounted, ref} from 'vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

// 需要缓存的视图组件名称
const keepAliveComponents = ['GalleryView', 'AlbumsView']

// 应用根组件
const meteorCanvas = ref<HTMLCanvasElement | null>(null)
let ctx: CanvasRenderingContext2D | null = null
let animationFrameId: number
let meteors: Meteor[] = []
let width = 0
let height = 0

// 流星类定义
class Meteor {
  x: number
  y: number
  vx: number
  vy: number
  length: number
  maxLength: number
  width: number
  opacity: number
  state: 'appearing' | 'moving' | 'flashing' | 'fading'
  timer: number
  flashDuration: number
  trail: { x: number, y: number, opacity: number }[] // 用于残影

  constructor(canvasWidth: number, canvasHeight: number) {
    // 随机起始位置：主要在右侧和顶部
    if (Math.random() > 0.5) {
      this.x = Math.random() * canvasWidth * 0.5 + canvasWidth * 0.5 // 右半边
      this.y = -100
    } else {
      this.x = canvasWidth + 100
      this.y = Math.random() * canvasHeight * 0.5 // 上半边
    }

    // 角度：35-75度（右上向左下）
    // 转换为弧度：135度是正左下。0是右，90是下，180是左。
    // 右上向左下，角度大约在 125度 到 165度之间 (以向右为0度)
    // 或者简单的计算向量
    const angle = (Math.random() * 40 + 115) * (Math.PI / 180) // 115-155度

    // 速度 800-1200 px/s。假设60fps，每帧 13-20px
    const speed = (Math.random() * 400 + 800) / 60

    this.vx = Math.cos(angle) * speed
    this.vy = Math.sin(angle) * speed

    this.length = 0
    this.maxLength = Math.random() * 200 + 300 // 300-500px 长尾巴
    this.width = Math.random() * 1 + 1 // 初始宽度
    this.opacity = 0
    this.state = 'appearing'
    this.timer = 0
    this.flashDuration = Math.random() * 10 + 5 // 约0.1-0.2秒 (6-12帧)
    this.trail = []
  }

  update() {
    // 状态机
    switch (this.state) {
      case 'appearing':
        this.opacity += 0.1
        this.length += this.maxLength * 0.05
        if (this.opacity >= 1) {
          this.opacity = 1
          this.state = 'moving'
        }
        break

      case 'moving':
        this.length += (this.maxLength - this.length) * 0.1
        // 随机触发爆闪，或者快到底部时
        if (Math.random() < 0.01 && this.timer > 30) {
          this.state = 'flashing'
          this.timer = 0
        }
        break

      case 'flashing':
        this.timer++
        this.width += 0.5
        if (this.timer > this.flashDuration) {
          this.state = 'fading'
        }
        break

      case 'fading':
        this.opacity -= 0.02 // 慢速消失
        this.width *= 0.9
        if (this.opacity <= 0) {
          return false // 死亡
        }
        break
    }

    // 移动
    this.x += this.vx
    this.y += this.vy
    this.timer++

    // 记录残影轨迹点 (每隔几帧记录一次，或者记录所有点但只保留一部分)
    // 这里为了性能，简单计算尾部坐标即可，不需要复杂数组

    // 边界检查
    if (this.state !== 'fading' && (this.x < -this.maxLength || this.y > height + this.maxLength)) {
      this.state = 'fading'
    }

    return true
  }

  draw(ctx: CanvasRenderingContext2D) {
    const tailX = this.x - this.vx * (this.length / Math.sqrt(this.vx*this.vx + this.vy*this.vy))
    const tailY = this.y - this.vy * (this.length / Math.sqrt(this.vx*this.vx + this.vy*this.vy))

    // 创建渐变
    const gradient = ctx.createLinearGradient(this.x, this.y, tailX, tailY)

    // 爆闪时颜色更亮
    if (this.state === 'flashing') {
      gradient.addColorStop(0, `rgba(255, 255, 255, ${this.opacity})`)
      gradient.addColorStop(0.1, `rgba(200, 230, 255, ${this.opacity})`)
      gradient.addColorStop(1, `rgba(173, 216, 230, 0)`)

      // 爆闪光晕
      ctx.shadowBlur = 20
      ctx.shadowColor = 'white'
    } else {
      gradient.addColorStop(0, `rgba(255, 255, 255, ${this.opacity})`)
      gradient.addColorStop(1, `rgba(173, 216, 230, 0)`)
      ctx.shadowBlur = 0
    }

    ctx.beginPath()
    ctx.moveTo(this.x, this.y)
    ctx.lineTo(tailX, tailY)
    ctx.strokeStyle = gradient
    ctx.lineWidth = this.width
    ctx.lineCap = 'round'
    ctx.stroke()

    // 头部亮点
    if (this.state !== 'fading') {
      ctx.beginPath()
      ctx.arc(this.x, this.y, this.state === 'flashing' ? this.width * 2 : 1.5, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(255, 255, 255, ${this.opacity})`
      ctx.fill()
    }
  }
}

const initMeteorSystem = () => {
  if (!meteorCanvas.value) return
  const canvas = meteorCanvas.value
  ctx = canvas.getContext('2d')
  if (!ctx) return

  const resize = () => {
    width = window.innerWidth
    height = window.innerHeight
    canvas.width = width
    canvas.height = height
  }
  window.addEventListener('resize', resize)
  resize()

  const loop = () => {
    if (!ctx) return
    // 清除画布
    ctx.clearRect(0, 0, width, height)

    // 随机生成流星 (频率控制)
    // 随机间隔 2-5秒 -> 120-300帧
    // 这里的概率 0.01 大概是每100帧(1.6秒)一个，稍微调整
    if (meteors.length < 5 && Math.random() < 0.001) {
      meteors.push(new Meteor(width, height))
    }

    // 更新和绘制
    ctx.globalCompositeOperation = 'lighter' // 叠加混合模式
    meteors = meteors.filter(meteor => {
      const alive = meteor.update()
      if (alive) meteor.draw(ctx!)
      return alive
    })

    animationFrameId = requestAnimationFrame(loop)
  }

  loop()

  return () => {
    window.removeEventListener('resize', resize)
    cancelAnimationFrame(animationFrameId)
  }
}

let cleanup: (() => void) | undefined

onMounted(() => {
  cleanup = initMeteorSystem()
})

onUnmounted(() => {
  if (cleanup) cleanup()
})
</script>
