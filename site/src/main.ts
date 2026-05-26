function toggleTheme() {
  const root = document.documentElement
  const isDark = root.classList.toggle('dark')
  root.style.colorScheme = isDark ? 'dark' : 'light'
  localStorage.setItem('rivet-theme', isDark ? 'dark' : 'light')
}

const ROW_FADE_MS = 360
const TYPE_MS = 950
const ROW_GAP_MS = 450
const PHASE_HOLD_MS = 1800

function sleep(ms: number) {
  return new Promise<void>((resolve) => window.setTimeout(resolve, ms))
}

function rowRevealDuration(row: HTMLElement) {
  const hasTypewriter = row.querySelector('.terminal-type') !== null
  return ROW_FADE_MS + (hasTypewriter ? TYPE_MS : 0) + ROW_GAP_MS
}

function resetPhase(phase: HTMLElement) {
  phase.classList.remove('is-active')
  phase.querySelectorAll('.terminal-row').forEach((row) => {
    row.classList.remove('is-visible')
  })
}

function initTerminalTypeWidths(phases: HTMLElement[]) {
  for (const typeEl of phases.flatMap((phase) =>
    Array.from(phase.querySelectorAll<HTMLElement>('.terminal-type')),
  )) {
    const steps = typeEl.textContent?.length ?? 0
    typeEl.style.setProperty('--terminal-type-steps', String(Math.max(steps, 1)))
    typeEl.style.setProperty('--terminal-type-width', `${steps}ch`)
  }
}

async function playPhase(phase: HTMLElement) {
  resetPhase(phase)
  phase.classList.add('is-active')

  const rows = Array.from(phase.querySelectorAll<HTMLElement>('.terminal-row'))
  for (const row of rows) {
    row.classList.add('is-visible')
    await sleep(rowRevealDuration(row))
  }

  await sleep(PHASE_HOLD_MS)
  resetPhase(phase)
}

async function animateTerminal() {
  const phases = Array.from(document.querySelectorAll<HTMLElement>('.terminal-phase'))
  if (phases.length === 0) return

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    const first = phases[0]
    if (first) {
      first.classList.add('is-active')
      first.querySelectorAll('.terminal-row').forEach((row) => {
        row.classList.add('is-visible')
      })
    }
    return
  }

  phases.forEach(resetPhase)
  initTerminalTypeWidths(phases)

  while (true) {
    for (const phase of phases) {
      await playPhase(phase)
    }
  }
}

document.getElementById('themeToggle')?.addEventListener('click', toggleTheme)

function initDocsToc() {
  const tocLinks = Array.from(document.querySelectorAll<HTMLAnchorElement>('.docs-toc-link'))
  if (tocLinks.length === 0) return

  const sections = tocLinks
    .map((link) => {
      const id = link.hash.slice(1)
      const el = id ? document.getElementById(id) : null
      return el ? { id, el } : null
    })
    .filter((item): item is { id: string; el: HTMLElement } => item !== null)

  if (sections.length === 0) return

  let ticking = false

  const scrollOffset = () => {
    const sample = sections[0]?.el
    if (!sample) return 80
    return parseFloat(getComputedStyle(sample).scrollMarginTop) || 80
  }

  const setActive = (id: string) => {
    for (const link of tocLinks) {
      link.classList.toggle('is-active', link.hash === `#${id}`)
    }
  }

  const update = () => {
    ticking = false
    let current = sections[0].id
    const offset = scrollOffset()

    for (const { id, el } of sections) {
      if (el.getBoundingClientRect().top <= offset) {
        current = id
      }
    }

    setActive(current)
  }

  const onScroll = () => {
    if (ticking) return
    ticking = true
    requestAnimationFrame(update)
  }

  update()
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
}

initDocsToc()

const terminalPhases = document.querySelectorAll<HTMLElement>('.terminal-phase')
if (terminalPhases.length > 0) {
  void animateTerminal()
}
