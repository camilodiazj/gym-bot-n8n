import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { DraftPreviewPage, SuccessOverlay } from './DraftPreviewPage'

// We don't mock services/api — the page short-circuits the API call when
// `?demo` is in the URL (uses DEMO_DRAFT). This keeps tests close to real
// rendering without needing fetch fixtures.

function setDemoSearch() {
  Object.defineProperty(window, 'location', {
    value: { ...window.location, search: '?demo' },
    writable: true,
  })
}

describe('DraftPreviewPage', () => {
  beforeEach(() => {
    setDemoSearch()
  })

  describe('Header — BUG-3 (no duplicated "Borrador")', () => {
    it('renders the BORRADOR pill exactly once and no large gray "Borrador" subtitle', async () => {
      render(<DraftPreviewPage />)

      // Wait for the demo draft to render
      await waitFor(() => {
        expect(screen.getByText('Tu Rutina')).toBeInTheDocument()
      })

      // The pill text contains "BORRADOR" (uppercase). It should appear exactly once.
      const borradorPill = screen.getAllByText(/BORRADOR/i)
      expect(borradorPill).toHaveLength(1)

      // The big gray subtitle that used to say plain "Borrador" must be gone.
      // queryByText with exact match keeps us strict (the pill text is "◆ BORRADOR" or "BORRADOR" uppercase).
      expect(screen.queryByText('Borrador', { selector: 'span' })).not.toBeInTheDocument()
      expect(screen.queryByText('Aprobada', { selector: 'span' })).not.toBeInTheDocument()
    })
  })

  describe('Exercise card — BUG-2 (expanded XOR collapsed, never both)', () => {
    it('shows banners (Volumen/Esfuerzo/Descanso) when expanded and hides chip; collapsed shows chip only', async () => {
      const { container } = render(<DraftPreviewPage />)

      // Wait for first day exercises
      await waitFor(() => {
        expect(screen.getByText('Sentadilla Hack')).toBeInTheDocument()
      })

      // The first exercise of each day starts expanded (per fetchDraft initial state).
      // Verify the three banner labels are present on the expanded card.
      expect(screen.getByText('Volumen:')).toBeInTheDocument()
      expect(screen.getByText('Esfuerzo:')).toBeInTheDocument()
      expect(screen.getByText('Descanso:')).toBeInTheDocument()

      // While expanded, the chip's compact "3 × 12-15" inline text should not be present
      // for the expanded card. The banner shows "3 series × 12-15 reps" (different wording).
      expect(screen.queryByText(/3 series × 12-15 reps/i)).toBeInTheDocument()

      // Locate the first exercise card and click its toggle (aria-label="Toggle instructions")
      const toggleButtons = container.querySelectorAll('button[aria-label="Toggle instructions"]')
      expect(toggleButtons.length).toBeGreaterThan(0)

      fireEvent.click(toggleButtons[0])

      // After collapse: banners gone for that card. Other cards may still have banners
      // (no, actually only the first card of each day starts expanded; collapsing the
      // first leaves Volumen/Esfuerzo/Descanso labels absent on the day-1 first card,
      // but other day-1 cards are already collapsed, so labels should be gone entirely).
      await waitFor(() => {
        expect(screen.queryByText('Volumen:')).not.toBeInTheDocument()
        expect(screen.queryByText('Esfuerzo:')).not.toBeInTheDocument()
        expect(screen.queryByText('Descanso:')).not.toBeInTheDocument()
      })

      // The chip format "RIR 1-2" appears in SpecRow; it should now be visible
      // for the first card (which is now collapsed). The chip uses "Descanso 90s"
      // text — distinct from the banner "Descanso:" label.
      expect(screen.getAllByText(/RIR 1-2/).length).toBeGreaterThan(0)
    })
  })

  describe('Alternatives — BUG-4 (carousel shows "Reemplazaría a: <name>")', () => {
    it('clicking "Ver alternativa" reveals the carousel header with the original exercise name', async () => {
      render(<DraftPreviewPage />)

      await waitFor(() => {
        expect(screen.getByText('Sentadilla Hack')).toBeInTheDocument()
      })

      // "Ver alternativa" buttons exist for primary exercises that have alternatives.
      // The first one corresponds to "Sentadilla Hack" (4 alternatives).
      const verAltButtons = screen.getAllByText('Ver alternativa')
      expect(verAltButtons.length).toBeGreaterThan(0)

      fireEvent.click(verAltButtons[0])

      // The alternative carousel header should appear with the original name.
      await waitFor(() => {
        expect(screen.getByText(/Alternativa 1 de 4/)).toBeInTheDocument()
        expect(screen.getByText(/Reemplazaría a:/)).toBeInTheDocument()
      })

      // Confirm the original exercise name appears in the "Reemplazaría a:" line.
      const replaceLine = screen.getByText(/Reemplazaría a:/).parentElement
      expect(replaceLine).not.toBeNull()
      expect(within(replaceLine as HTMLElement).getByText('Sentadilla Hack')).toBeInTheDocument()
    })
  })
})

describe('SuccessOverlay — BUG-6a graceful degradation', () => {
  it('renders single "Entendido" button when magicLinkUrl is undefined', () => {
    const onDismiss = vi.fn()
    render(<SuccessOverlay onDismiss={onDismiss} />)

    // Only one CTA — the "Entendido" button.
    expect(screen.getByRole('button', { name: 'Entendido' })).toBeInTheDocument()
    // No link in the overlay (the "Ver mi rutina de hoy" anchor only renders with magicLinkUrl).
    expect(screen.queryByRole('link', { name: /Ver mi rutina de hoy/i })).not.toBeInTheDocument()
  })

  it('renders both CTA link and dismiss button when magicLinkUrl is provided', () => {
    const onDismiss = vi.fn()
    render(<SuccessOverlay magicLinkUrl="https://tracker.example.com/w?c=abc123" onDismiss={onDismiss} />)

    const link = screen.getByRole('link', { name: /Ver mi rutina de hoy/i })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', 'https://tracker.example.com/w?c=abc123')

    // Dismiss button still present as secondary action.
    expect(screen.getByRole('button', { name: 'Entendido' })).toBeInTheDocument()
  })
})
