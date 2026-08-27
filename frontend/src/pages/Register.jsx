import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Auth.css'

const ROLES = ['Super Admin', 'Admin Operasional', 'Manager', 'PIC', 'Magang']

const initialForm = {
  nama: '',
  email: '',
  password: '',
  password_confirmation: '',
  role: ROLES[0],
  perusahaan_id: '',
}

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState(initialForm)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleChange = (e) => {
    const { name, value } = e.target
    setForm((prev) => ({ ...prev, [name]: value }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await register({ ...form, perusahaan_id: Number(form.perusahaan_id) })
      navigate('/login')
    } catch (err) {
      setError(err.response?.data?.error || 'Gagal mendaftar. Silakan coba lagi.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-form" onSubmit={handleSubmit}>
        <h1>Register</h1>
        {error && <p className="auth-error">{error}</p>}
        <label>
          Nama
          <input name="nama" value={form.nama} onChange={handleChange} required />
        </label>
        <label>
          Email
          <input type="email" name="email" value={form.email} onChange={handleChange} required />
        </label>
        <label>
          Password
          <input
            type="password"
            name="password"
            value={form.password}
            onChange={handleChange}
            minLength={6}
            required
          />
        </label>
        <label>
          Konfirmasi Password
          <input
            type="password"
            name="password_confirmation"
            value={form.password_confirmation}
            onChange={handleChange}
            minLength={6}
            required
          />
        </label>
        <label>
          Role
          <select name="role" value={form.role} onChange={handleChange}>
            {ROLES.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        </label>
        <label>
          Perusahaan ID
          <input
            type="number"
            name="perusahaan_id"
            value={form.perusahaan_id}
            onChange={handleChange}
            min={1}
            required
          />
        </label>
        <button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Memproses...' : 'Daftar'}
        </button>
        <p>
          Sudah punya akun? <Link to="/login">Login</Link>
        </p>
      </form>
    </div>
  )
}
