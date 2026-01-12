/**
 * 缩略图生成结果
 */
export interface ThumbnailResult {
  blob: Blob          // 缩略图 Blob
  width: number       // 缩略图宽度
  height: number      // 缩略图高度
  url: string         // Blob URL（用于预览）
}

/**
 * 创建图片缩略图（返回 Blob 用于上传）
 * @param file 原始文件
 * @param maxWidth 最大宽度
 * @param maxHeight 最大高度
 * @param quality 压缩质量 (0-1)
 * @returns Promise<ThumbnailResult> 缩略图结果
 */
export async function createThumbnailBlob(
  file: File,
  maxWidth = 400,
  maxHeight = 400,
  quality = 0.85
): Promise<ThumbnailResult> {
  return new Promise((resolve, reject) => {
    if (!file.type.startsWith('image/')) {
      reject(new Error('不是图片文件'))
      return
    }

    const img = new Image()
    const url = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(url)

      let width = img.naturalWidth
      let height = img.naturalHeight

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

      ctx.drawImage(img, 0, 0, width, height)

      // 导出为 JPEG（压缩效果更好）
      canvas.toBlob((blob) => {
        if (blob) {
          resolve({
            blob,
            width,
            height,
            url: URL.createObjectURL(blob)
          })
        } else {
          reject(new Error('无法生成缩略图'))
        }
      }, 'image/jpeg', quality)
    }

    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片加载失败'))
    }

    img.src = url
  })
}