import exifr from 'exifr'

/**
 * EXIF 数据结构（与后端 ExifDataRequest 对应）
 */
export interface ExifData {
  taken_at?: string        // ISO 8601 格式的拍摄时间
  latitude?: number        // 纬度
  longitude?: number       // 经度
  camera_make?: string     // 相机制造商
  camera_model?: string    // 相机型号
  aperture?: string        // 光圈
  shutter_speed?: string   // 快门速度
  iso?: number             // ISO
  focal_length?: string    // 焦距
}

/**
 * 从图片文件中提取 EXIF 数据
 * @param file 图片文件
 * @returns Promise<ExifData | null> 提取的 EXIF 数据，如果提取失败返回 null
 */
export async function extractExifData(file: File): Promise<ExifData | null> {
  try {
    // 只处理图片文件
    if (!file.type.startsWith('image/')) {
      return null
    }

    // 使用 exifr 提取 EXIF 数据
    const exif = await exifr.parse(file, {
      // 指定要提取的标签
      pick: [
        'DateTimeOriginal',
        'CreateDate',
        'ModifyDate',
        'GPSLatitude',
        'GPSLongitude',
        'GPSLatitudeRef',
        'GPSLongitudeRef',
        'Make',
        'Model',
        'FNumber',
        'ExposureTime',
        'ISO',
        'ISOSpeedRatings',
        'FocalLength',
        'FocalLengthIn35mmFormat',
      ],
      // 自动转换 GPS 坐标
      translateKeys: true,
      translateValues: true,
    })

    if (!exif) {
      return null
    }

    const result: ExifData = {}

    // 解析拍摄时间
    const dateTime = exif.DateTimeOriginal || exif.CreateDate || exif.ModifyDate
    if (dateTime) {
      // exifr 已经将日期转换为 Date 对象
      if (dateTime instanceof Date) {
        result.taken_at = dateTime.toISOString()
      } else if (typeof dateTime === 'string') {
        // 尝试解析字符串格式
        const parsed = new Date(dateTime.replace(/(\d{4}):(\d{2}):(\d{2})/, '$1-$2-$3'))
        if (!isNaN(parsed.getTime())) {
          result.taken_at = parsed.toISOString()
        }
      }
    }

    // 解析 GPS 坐标 (exifr 已经自动转换为十进制度数)
    if (typeof exif.latitude === 'number' && typeof exif.longitude === 'number') {
      result.latitude = exif.latitude
      result.longitude = exif.longitude
    }

    // 解析相机信息
    if (exif.Make) {
      result.camera_make = String(exif.Make).trim()
    }
    if (exif.Model) {
      result.camera_model = String(exif.Model).trim()
    }

    // 解析光圈
    if (exif.FNumber) {
      result.aperture = `f/${exif.FNumber}`
    }

    // 解析快门速度
    if (exif.ExposureTime) {
      const exposure = exif.ExposureTime
      if (exposure >= 1) {
        result.shutter_speed = `${exposure}s`
      } else {
        // 转换为分数形式
        const denominator = Math.round(1 / exposure)
        result.shutter_speed = `1/${denominator}`
      }
    }

    // 解析 ISO
    const iso = exif.ISO || exif.ISOSpeedRatings
    if (iso) {
      result.iso = Array.isArray(iso) ? iso[0] : iso
    }

    // 解析焦距
    const focalLength = exif.FocalLengthIn35mmFormat || exif.FocalLength
    if (focalLength) {
      result.focal_length = `${Math.round(focalLength)}mm`
    }

    return result
  } catch (error) {
    console.warn('EXIF 提取失败:', error)
    return null
  }
}

/**
 * 获取图片尺寸
 * @param file 图片文件
 * @returns Promise<{width: number, height: number}> 图片尺寸
 */
export function getImageDimensions(file: File): Promise<{width: number, height: number}> {
  return new Promise((resolve, reject) => {
    if (!file.type.startsWith('image/')) {
      reject(new Error('不是图片文件'))
      return
    }

    const img = new Image()
    const url = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve({
        width: img.naturalWidth,
        height: img.naturalHeight
      })
    }

    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片加载失败'))
    }

    img.src = url
  })
}
