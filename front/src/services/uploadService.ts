import {imageApi, type PrepareUploadRequest, type PrepareUploadResponse} from '@/api/image'
import {calculateSHA256} from '@/utils/fileHash'
import {type ExifData, extractExifData, getImageDimensions} from '@/utils/exif'
import {createThumbnailBlob, type ThumbnailResult} from '@/utils/image'

/**
 * 文件预处理结果
 */
export interface FilePreprocessResult {
  fileHash: string
  exifData: ExifData | null
  dimensions: { width: number; height: number }
  thumbnail: ThumbnailResult | null
}

/**
 * 上传进度回调
 */
export interface UploadProgressCallback {
  onThumbnail?: (url: string) => void          // 缩略图生成完成（可用于预览）
  onPreprocess?: () => void                    // 开始预处理
  onPrepare?: () => void                       // 开始准备上传
  onUpload?: (progress: number) => void        // 上传进度 (0-100)
  onConfirm?: () => void                       // 开始确认
}

/**
 * 上传结果
 */
export interface UploadResult {
  success: boolean
  isDuplicate: boolean
  imageId?: number
  error?: string
}

/**
 * 上传服务 - 封装完整的图片上传流程
 */
export const uploadService = {
  /**
   * 预处理文件：计算哈希、提取 EXIF、获取尺寸、生成缩略图
   */
  async preprocessFile(file: File): Promise<FilePreprocessResult> {
    const [fileHash, exifData, dimensions, thumbnail] = await Promise.all([
      calculateSHA256(file),
      extractExifData(file),
      getImageDimensions(file),
      createThumbnailBlob(file, 400, 400, 0.85).catch(() => null)
    ])

    return {fileHash, exifData, dimensions, thumbnail}
  },

  /**
   * 准备上传：发送预处理数据到后端，获取上传凭证
   */
  async prepareUpload(
    file: File,
    preprocessResult: FilePreprocessResult,
    albumId?: number
  ): Promise<PrepareUploadResponse> {
    const request: PrepareUploadRequest = {
      file_hash: preprocessResult.fileHash,
      file_size: file.size,
      width: preprocessResult.dimensions.width,
      height: preprocessResult.dimensions.height,
      mime_type: file.type,
      original_name: file.name,
      album_id: albumId,
      exif_data: preprocessResult.exifData || undefined,
      thumbnail_width: preprocessResult.thumbnail?.width,
      thumbnail_height: preprocessResult.thumbnail?.height
    }

    const response = await imageApi.prepareUpload(request)
    return response.data
  },

  /**
   * 执行上传：根据凭证类型选择上传方式
   */
  async executeUpload(
    file: File,
    thumbnail: ThumbnailResult | null,
    prepareData: PrepareUploadResponse,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    const {upload_token, thumbnail_path} = prepareData

    if (!upload_token) {
      throw new Error('无效的上传凭证')
    }

    // 进度计算：原图 80%，缩略图 20%
    let originalProgress = 0
    let thumbnailProgress = thumbnail ? 0 : 100

    const updateProgress = () => {
      const total = Math.round(originalProgress * 0.8 + thumbnailProgress * 0.2)
      onProgress?.(total)
    }

    const uploadPromises: Promise<void>[] = []

    // 上传原图
    uploadPromises.push(
      imageApi.uploadWithCredential(
        upload_token.original,
        file,
        file.type,
        (p) => { originalProgress = p; updateProgress() }
      )
    )

    // 上传缩略图
    if (thumbnail && thumbnail_path && upload_token.thumbnail) {
      uploadPromises.push(
        imageApi.uploadWithCredential(
          upload_token.thumbnail,
          thumbnail.blob,
          'image/jpeg',
          (p) => { thumbnailProgress = p; updateProgress() }
        )
      )
    }

    await Promise.all(uploadPromises)
  },

  /**
   * 确认上传：通知后端文件已上传完成
   */
  async confirmUpload(
    prepareData: PrepareUploadResponse
  ): Promise<number> {
    const {upload_id} = prepareData

    if (!upload_id) {
      throw new Error('无效的上传信息')
    }

    const response = await imageApi.confirmUpload({
      upload_id
    })

    return response.data.id
  },

  /**
   * 完整上传流程
   */
  async upload(
    file: File,
    albumId?: number,
    callbacks?: UploadProgressCallback
  ): Promise<UploadResult> {
    try {
      // 1. 预处理
      callbacks?.onPreprocess?.()
      const preprocessResult = await this.preprocessFile(file)

      // 通知缩略图已生成（可用于 UI 预览）
      if (preprocessResult.thumbnail?.url) {
        callbacks?.onThumbnail?.(preprocessResult.thumbnail.url)
      }

      // 2. 准备上传
      callbacks?.onPrepare?.()
      const prepareData = await this.prepareUpload(file, preprocessResult, albumId)

      // 3. 检查重复
      if (prepareData.is_duplicate && prepareData.existing_image) {
        return {
          success: true,
          isDuplicate: true,
          imageId: prepareData.existing_image.id
        }
      }

      // 4. 执行上传
      await this.executeUpload(
        file,
        preprocessResult.thumbnail,
        prepareData,
        callbacks?.onUpload
      )

      // 5. 确认上传
      callbacks?.onConfirm?.()
      const imageId = await this.confirmUpload(prepareData)

      return {
        success: true,
        isDuplicate: false,
        imageId
      }
    } catch (error) {
      return {
        success: false,
        isDuplicate: false,
        error: error instanceof Error ? error.message : '上传失败'
      }
    }
  }
}
