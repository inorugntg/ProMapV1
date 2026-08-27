import { useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext'
import api from '../services/api'
import './Dashboard.css'

const USER_LIST_ROLES = ['Super Admin', 'Admin Operasional']

export default function Dashboard() {
  const { user, logout } = useAuth()
  const [users, setUsers] = useState([])
  const [usersError, setUsersError] = useState('')
  const canViewUsers = USER_LIST_ROLES.includes(user?.role)

  useEffect(() => {
    if (!canViewUsers) return

    let isMounted = true
    api
      .get('/users')
      .then((response) => {
        if (isMounted) setUsers(response.data.users || [])
      })
      .catch(() => {
        if (isMounted) setUsersError('Gagal memuat daftar user.')
      })

    return () => {
      isMounted = false
    }
  }, [canViewUsers])

  return (
    <div className="dashboard-page">
      <header className="dashboard-header">
        <div>
          <h1>Dashboard</h1>
          <p>
            {user?.nama} — {user?.role}
          </p>
        </div>
        <button type="button" onClick={logout}>
          Logout
        </button>
      </header>

      {canViewUsers && (
        <section className="dashboard-users">
          <h2>Daftar User</h2>
          {usersError && <p className="auth-error">{usersError}</p>}
          <table>
            <thead>
              <tr>
                <th>Nama</th>
                <th>Email</th>
                <th>Role</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.ID}>
                  <td>{u.nama}</td>
                  <td>{u.email}</td>
                  <td>{u.role}</td>
                  <td>{u.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}
    </div>
  )
}
