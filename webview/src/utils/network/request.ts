// HTTP请求工具
import { API_BASE_URL, API_ENDPOINTS } from '@/config/api'
import cache from '@/plugins/cache'

interface RequestConfig extends RequestInit {
  params?: Record<string, any>
  _retry?: boolean
  skipAuthRefresh?: boolean
}

interface UploadOptions {
  onCancel?: (cancel: () => void) => void
  _retry?: boolean
}

class AuthExpiredError extends Error {}

let refreshTokenPromise: Promise<string> | null = null

const AUTH_REFRESH_URL = API_ENDPOINTS.AUTH.REFRESH
const AUTH_SKIP_REFRESH_URLS = new Set<string>(Object.values(API_ENDPOINTS.AUTH))

const redirectToLogin = () => {
  cache.local.remove('token')
  window.location.href = '/login'
}

const shouldRefreshToken = (url: string, options: RequestConfig = {}) => {
  const path = url.split('?')[0]
  return !options._retry && !options.skipAuthRefresh && !AUTH_SKIP_REFRESH_URLS.has(path)
}

const extractToken = (data: any): string | null => {
  return data?.token || data?.data?.token || null
}

const refreshAccessToken = async (): Promise<string> => {
  if (!refreshTokenPromise) {
    refreshTokenPromise = (async () => {
      const token = cache.local.get('token')
      const response = await fetch(API_BASE_URL + AUTH_REFRESH_URL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        }
      })

      let data: any
      try {
        data = await response.json()
      } catch (e) {
        data = {}
      }

      if (!response.ok || (data.code && data.code !== 200)) {
        throw new Error(data.message || '登录已过期，请重新登录')
      }

      const newToken = extractToken(data)
      if (!newToken) {
        throw new Error(data.message || '刷新 token 失败')
      }

      cache.local.set('token', newToken)
      return newToken
    })().finally(() => {
      refreshTokenPromise = null
    })
  }

  return refreshTokenPromise
}

// 请求拦截器 - 添加token
const requestInterceptor = (config: RequestConfig): RequestConfig => {
  const token = cache.local.get('token')
  if (token) {
    config.headers = {
      ...config.headers,
      Authorization: `Bearer ${token}`
    }
  }
  return config
}

// 响应拦截器 - 处理错误
const responseInterceptor = async <T = any>(response: Response): Promise<T> => {
  let data: any
  try {
    data = await response.json()
  } catch (e) {
    data = {}
  }

  // 先检查 HTTP 状态码
  if (!response.ok) {
    // 处理 HTTP 错误
    if (response.status === 401) {
      // 401: 未授权，需要登录
      throw new AuthExpiredError(data.message || '登录已过期，请重新登录')
    }
    if (response.status === 403) {
      // 403: 禁止访问，已登录但无权限
      throw new Error(data.message || '权限不足')
    }
    throw new Error(data.message || '请求失败')
  }

  // 检查业务状态码（后端统一返回格式：{code, message, data}）
  if (data.code && data.code !== 200) {
    // 业务错误
    if (data.code === 401) {
      // 401: 未授权，需要登录
      throw new AuthExpiredError(data.message || '登录已过期，请重新登录')
    }
    if (data.code === 403) {
      // 403: 禁止访问，已登录但无权限
      throw new Error(data.message || '权限不足')
    }
    return data
  }

  return data
}

// 基础请求方法
const request = async <T = any>(url: string, options: RequestConfig = {}): Promise<T> => {
  const config: RequestConfig = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    }
  }

  // 应用请求拦截器
  const interceptedConfig = requestInterceptor(config)

  try {
    const response = await fetch(API_BASE_URL + url, {
      ...interceptedConfig,
      signal: options.signal
    })

    return await responseInterceptor<T>(response)
  } catch (error: any) {
    if (error.name === 'AbortError') {
      throw new Error('请求已取消')
    }
    if (error instanceof AuthExpiredError && shouldRefreshToken(url, options)) {
      try {
        await refreshAccessToken()
        return request<T>(url, {
          ...options,
          _retry: true
        })
      } catch {
        redirectToLogin()
      }
    }
    if (error instanceof AuthExpiredError) {
      redirectToLogin()
    }
    throw error
  }
}

// GET请求
export const get = <T = any>(
  url: string,
  params: Record<string, any> = {},
  options: RequestConfig = {}
): Promise<T> => {
  const queryString = new URLSearchParams(params).toString()
  const fullUrl = queryString ? `${url}?${queryString}` : url

  return request<T>(fullUrl, {
    method: 'GET',
    ...options
  })
}

// POST请求
export const post = <T = any>(url: string, data: any = {}, options: RequestConfig = {}): Promise<T> => {
  return request<T>(url, {
    method: 'POST',
    body: JSON.stringify(data),
    ...options
  })
}

// PUT请求
export const put = <T = any>(url: string, data: any = {}, options: RequestConfig = {}): Promise<T> => {
  return request<T>(url, {
    method: 'PUT',
    body: JSON.stringify(data),
    ...options
  })
}

// DELETE请求
export const del = <T = any>(url: string, options: RequestConfig = {}): Promise<T> => {
  return request<T>(url, {
    method: 'DELETE',
    ...options
  })
}

// 文件上传
export const upload = <T = any>(
  url: string,
  file: File,
  outerParams: FormData,
  onProgress?: (percent: number, loaded?: number, total?: number) => void,
  options: UploadOptions = {}
): Promise<T> => {
  const formData = new FormData()
  outerParams.forEach((value, key) => {
    formData.append(key, value)
  })
  formData.append('file', file)

  const token = cache.local.get('token')

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    const retryOrRejectAuthError = async (message?: string) => {
      if (options._retry) {
        redirectToLogin()
        reject(new Error(message || '登录已过期，请重新登录'))
        return
      }

      try {
        await refreshAccessToken()
        resolve(
          await upload<T>(url, file, outerParams, onProgress, {
            ...options,
            _retry: true
          })
        )
      } catch {
        redirectToLogin()
        reject(new Error(message || '登录已过期，请重新登录'))
      }
    }

    // 上传进度
    if (onProgress) {
      xhr.upload.addEventListener('progress', e => {
        if (e.lengthComputable) {
          const percentComplete = (e.loaded / e.total) * 100
          onProgress(percentComplete, e.loaded, e.total)
        }
      })
    }

    // 请求完成
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const response = JSON.parse(xhr.responseText)
          // 检查业务状态码
          if (response.code && response.code !== 200) {
            if (response.code === 401) {
              // 401: 未授权，需要登录
              retryOrRejectAuthError(response.message)
              return
            }
            if (response.code === 403) {
              // 403: 禁止访问，已登录但无权限
              reject(new Error(response.message || '权限不足'))
              return
            }
            reject(new Error(response.message || '上传失败'))
            return
          }
          resolve(response)
        } catch (e) {
          resolve(xhr.responseText as any)
        }
      } else {
        // 处理 HTTP 状态码错误
        if (xhr.status === 401) {
          retryOrRejectAuthError()
          return
        }
        if (xhr.status === 403) {
          reject(new Error('权限不足'))
          return
        }
        try {
          const error = JSON.parse(xhr.responseText)
          reject(new Error(error.message || '上传失败'))
        } catch (e) {
          reject(new Error('上传失败'))
        }
      }
    })

    // 请求失败
    xhr.addEventListener('error', () => {
      reject(new Error('网络错误'))
    })

    // 请求中止
    xhr.addEventListener('abort', () => {
      reject(new Error('上传已取消'))
    })

    xhr.open('POST', API_BASE_URL + url)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }

    xhr.send(formData)

    // 返回取消方法
    if (options.onCancel) {
      options.onCancel(() => xhr.abort())
    }
  })
}

// 文件下载
export const download = async (url: string, filename: string): Promise<void> => {
  const runDownload = async (retried = false): Promise<void> => {
    const token = cache.local.get('token')
    const response = await fetch(API_BASE_URL + url, {
      method: 'GET',
      headers: {
        Authorization: token ? `Bearer ${token}` : ''
      }
    })

    if (!response.ok) {
      // 处理 HTTP 状态码错误
      if (response.status === 401) {
        if (!retried) {
          try {
            await refreshAccessToken()
            return runDownload(true)
          } catch {
            redirectToLogin()
          }
        } else {
          redirectToLogin()
        }
        throw new Error('登录已过期，请重新登录')
      }
      if (response.status === 403) {
        throw new Error('权限不足')
      }
      // 尝试解析错误消息
      try {
        const errorData = await response.json()
        throw new Error(errorData.message || '下载失败')
      } catch (e) {
        throw new Error('下载失败')
      }
    }

    const blob = await response.blob()
    const downloadUrl = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = downloadUrl
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(downloadUrl)
  }

  try {
    await runDownload()
  } catch (error: any) {
    throw new Error(error.message || '下载失败')
  }
}

export default {
  get,
  post,
  put,
  del,
  upload,
  download
}
