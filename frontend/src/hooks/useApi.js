import { useState, useEffect, useCallback, useRef } from 'react'
import { apiUrl } from '../utils/api'

export function useApi(url, options = {}) {
  const { pollInterval, ...fetchOptions } = options
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const intervalRef = useRef(null)
  const isFirstFetch = useRef(true)

  const fetchData = useCallback(async () => {
    if (isFirstFetch.current) {
      setLoading(true)
    }
    setError(null)
    try {
      const res = await fetch(apiUrl(url), fetchOptions)
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || `HTTP ${res.status}`)
      }
      setData(await res.json())
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
      isFirstFetch.current = false
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url])

  useEffect(() => {
    isFirstFetch.current = true
    fetchData()
    if (pollInterval) {
      intervalRef.current = setInterval(fetchData, pollInterval)
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [fetchData, pollInterval])

  return { data, loading, error, refetch: fetchData }
}

export default useApi
