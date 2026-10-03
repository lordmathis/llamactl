import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { AuthProvider } from '@/contexts/AuthContext'
import LoginDialog from '@/components/LoginDialog'
import { authApi } from '@/lib/api'

vi.mock('@/lib/api', () => ({
  authApi: {
    whoami: vi.fn(),
  },
}))

const whoamiMock = vi.mocked(authApi.whoami)

function renderLogin() {
  return render(
    <AuthProvider>
      <LoginDialog open />
    </AuthProvider>
  )
}

describe('AuthContext OIDC callback errors', () => {
  beforeEach(() => {
    whoamiMock.mockResolvedValue({ authenticated: false, oidc_enabled: true, user: null })
  })

  it('shows a group-gate denial in the login dialog and strips it from the URL', async () => {
    window.history.pushState({}, '', '/?auth_error=denied')

    renderLogin()

    // The banner must survive the dialog's mount-time clearError, since the
    // OIDC error is separate state from key-login errors
    expect(
      await screen.findByText('Access denied: your account is not a member of an allowed group.')
    ).toBeInTheDocument()

    // A refresh must not replay the error
    await waitFor(() => expect(window.location.search).toBe(''))
    expect(await screen.findByRole('button', { name: /sign in with sso/i })).toBeInTheDocument()
  })

  it('falls back to a generic message for unknown codes', async () => {
    window.history.pushState({}, '', '/?auth_error=something_new')

    renderLogin()

    expect(await screen.findByText('Login failed. Please try again.')).toBeInTheDocument()
  })
})
