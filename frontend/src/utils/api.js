const BASE = import.meta.env.VITE_API_BASE_URL || ''

export const apiUrl = (path) => `${BASE}${path}`

export const apiFetch = (path, options) => fetch(apiUrl(path), options)
