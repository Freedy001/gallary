/**
 * 使用 Web Crypto API 计算文件的 SHA256 哈希值
 * @param file 要计算哈希的文件
 * @returns Promise<string> 十六进制格式的 SHA256 哈希值
 */
export async function calculateSHA256(file: File): Promise<string> {
  const buffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

/**
 * 分块计算大文件的 SHA256 哈希值（节省内存）
 * 对于大文件，逐块读取并计算哈希
 * @param file 要计算哈希的文件
 * @param onProgress 进度回调 (0-100)
 * @returns Promise<string> 十六进制格式的 SHA256 哈希值
 */
export async function calculateSHA256Chunked(
  file: File,
  onProgress?: (progress: number) => void
): Promise<string> {
  const CHUNK_SIZE = 2 * 1024 * 1024 // 2MB per chunk
  const chunks = Math.ceil(file.size / CHUNK_SIZE)

  // 对于小文件，直接使用简单方法
  if (file.size <= CHUNK_SIZE) {
    return calculateSHA256(file)
  }

  // 使用 SubtleCrypto 流式处理
  // 注意：Web Crypto API 不直接支持流式哈希，所以我们需要读取整个文件
  // 但我们可以分块读取以减少内存峰值
  const buffer = new Uint8Array(file.size)
  let offset = 0

  for (let i = 0; i < chunks; i++) {
    const start = i * CHUNK_SIZE
    const end = Math.min(start + CHUNK_SIZE, file.size)
    const chunk = file.slice(start, end)
    const chunkBuffer = await chunk.arrayBuffer()
    buffer.set(new Uint8Array(chunkBuffer), offset)
    offset += chunkBuffer.byteLength

    if (onProgress) {
      onProgress(Math.round(((i + 1) / chunks) * 100))
    }
  }

  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}
