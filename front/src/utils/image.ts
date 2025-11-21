/**
 * 创建图片缩略图
 * @param file 原始文件
 * @param maxWidth 最大宽度
 * @param maxHeight 最大高度
 * @returns Promise<string> 缩略图 URL (Blob URL)
 */
export async function createThumbnail(file: File, maxWidth = 200, maxHeight = 200): Promise<string> {
  return new Promise((resolve, reject) => {
    // 如果不是图片，返回空
    if (!file.type.startsWith('image/')) {
      resolve('')
      return
    }

    // 如果图片本身很小，直接使用原图
    if (file.size < 200 * 1024) { // 小于 200KB
      resolve(URL.createObjectURL(file))
      return
    }

    const img = new Image()
    const url = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(url) // 释放原图 URL

      let width = img.width
      let height = img.height

      // 计算缩放比例
      if (width > maxWidth || height > maxHeight) {
        const ratio = Math.min(maxWidth / width, maxHeight / height)
        width = Math.floor(width * ratio)
        height = Math.floor(height * ratio)
      }

      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height

      const ctx = canvas.getContext('2d')
      if (!ctx) {
        reject(new Error('无法创建 Canvas 上下文'))
        return
      }

      // 绘制图片
      ctx.drawImage(img, 0, 0, width, height)

      // 导出为 Blob
      canvas.toBlob((blob) => {
        if (blob) {
          resolve(URL.createObjectURL(blob))
        } else {
          reject(new Error('无法生成缩略图'))
        }
      }, file.type, 0.7) // 0.7 质量
    }

    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片加载失败'))
    }

    img.src = url
  })
}
