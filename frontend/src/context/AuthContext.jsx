import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    const stored = localStorage.getItem('user')
    return stored ? JSON.parse(stored) : null
  })
  const [token, setToken] = useState(() => localStorage.getItem('token'))
  const [isLoading, setIsLoading] = useState(() => Boolean(localStorage.getItem('token')))

  const login = useCallback(async (email, password) => {
    const response = await api.post('/auth/login', { email, password })
    const { token: newToken, user: newUser } = response.data
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))
    setToken(newToken)
    setUser(newUser)
    return newUser
  }, [])

  const register = useCallback(async (data) => {
    const response = await api.post('/auth/register', data)
    return response.data
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }, [])

  const fetchProfile = useCallback(async () => {
    const response = await api.get('/auth/profile')
    const profileUser = response.data.user
    localStorage.setItem('user', JSON.stringify(profileUser))
    setUser(profileUser)
    return profileUser
  }, [])

  // Saat aplikasi pertama kali dimuat, validasi token tersimpan dengan mengambil profile terbaru.
  useEffect(() => {
    const storedToken = localStorage.getItem('token')
    if (!storedToken) {
      return
    }

    let isActive = true

    api
      .get('/auth/profile')
      .then((response) => {
        if (!isActive) return
        const profileUser = response.data.user
        localStorage.setItem('user', JSON.stringify(profileUser))
        setUser(profileUser)
      })
      .catch(() => {
        if (!isActive) return
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        setToken(null)
        setUser(null)
      })
      .finally(() => {
        if (isActive) setIsLoading(false)
      })

    return () => {
      isActive = false
    }
  }, [])

  const value = { user, token, isLoading, login, register, logout, fetchProfile }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components -- hook intentionally colocated with its provider
export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth harus dipakai di dalam AuthProvider')
  }
  return context
}
