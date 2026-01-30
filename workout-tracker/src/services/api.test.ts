import { describe, it, expect, vi, beforeEach } from 'vitest'
import { updateSetWeight, updateSetReps, markSetComplete } from './api'

const mockFetch = vi.fn()
global.fetch = mockFetch

// The test setup stubs VITE_DEV_USER_ID, so auth params will be included
const DEV_USER_ID = '0a220ce8-00e8-4eda-bbf4-112a7fd1e57d'
const AUTH_PARAMS = `?user_id=${DEV_USER_ID}`

describe('API Service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('updateSetWeight', () => {
    it('should call API with correct parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      })

      await updateSetWeight('set-123', '50')

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/api/v1/sets/set-123${AUTH_PARAMS}`,
        {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ kg: '50' }),
        }
      )
    })

    it('should throw error on non-ok response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Bad Request',
        json: () => Promise.resolve({ message: 'Invalid weight' }),
      })

      await expect(updateSetWeight('set-123', 'invalid')).rejects.toThrow('Invalid weight')
    })

    it('should throw generic error when response has no message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
        json: () => Promise.resolve({}),
      })

      await expect(updateSetWeight('set-123', '50')).rejects.toThrow('Failed to update set: Internal Server Error')
    })

    it('should handle json parse error gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Bad Gateway',
        json: () => Promise.reject(new Error('Parse error')),
      })

      await expect(updateSetWeight('set-123', '50')).rejects.toThrow('Failed to update set: Bad Gateway')
    })
  })

  describe('updateSetReps', () => {
    it('should call API with correct parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      })

      await updateSetReps('set-456', 15)

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/api/v1/sets/set-456${AUTH_PARAMS}`,
        {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ reps: 15 }),
        }
      )
    })

    it('should throw error on non-ok response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Not Found',
        json: () => Promise.resolve({ message: 'Set not found' }),
      })

      await expect(updateSetReps('invalid-id', 10)).rejects.toThrow('Set not found')
    })

    it('should throw generic error when response has no message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Server Error',
        json: () => Promise.resolve({}),
      })

      await expect(updateSetReps('set-123', 10)).rejects.toThrow('Failed to update set: Server Error')
    })

    it('should handle json parse error gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Gateway Timeout',
        json: () => Promise.reject(new Error('Parse error')),
      })

      await expect(updateSetReps('set-123', 10)).rejects.toThrow('Failed to update set: Gateway Timeout')
    })
  })

  describe('markSetComplete', () => {
    it('should call API with correct parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      })

      await markSetComplete('set-789')

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/api/v1/sets/set-789/complete${AUTH_PARAMS}`,
        {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
        }
      )
    })

    it('should throw error on non-ok response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Forbidden',
        json: () => Promise.resolve({ message: 'Not authorized' }),
      })

      await expect(markSetComplete('set-789')).rejects.toThrow('Not authorized')
    })

    it('should throw generic error when response has no message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Service Unavailable',
        json: () => Promise.resolve({}),
      })

      await expect(markSetComplete('set-123')).rejects.toThrow('Failed to mark set complete: Service Unavailable')
    })

    it('should handle json parse error gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: 'Bad Request',
        json: () => Promise.reject(new Error('Parse error')),
      })

      await expect(markSetComplete('set-123')).rejects.toThrow('Failed to mark set complete: Bad Request')
    })
  })
})
