import '@testing-library/jest-dom'
import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

// Mock environment variables for testing
vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
vi.stubEnv('VITE_DEV_USER_ID', '0a220ce8-00e8-4eda-bbf4-112a7fd1e57d')

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Mock window.location
const mockLocation = {
  search: '',
  href: 'http://localhost:3000',
  origin: 'http://localhost:3000',
  pathname: '/',
  hash: '',
  host: 'localhost:3000',
  hostname: 'localhost',
  port: '3000',
  protocol: 'http:',
  assign: vi.fn(),
  reload: vi.fn(),
  replace: vi.fn(),
}

Object.defineProperty(window, 'location', {
  value: mockLocation,
  writable: true,
})

// Mock fetch globally
global.fetch = vi.fn()
