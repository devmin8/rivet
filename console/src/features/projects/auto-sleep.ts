import { z } from 'zod'

export const autoSleepAfterSecondsSchema = z
  .string()
  .trim()
  .refine(
    (value) => value === '' || /^\d+$/.test(value),
    'Enter whole seconds, or leave empty to disable auto sleep.',
  )
  .refine(
    (value) => value === '' || Number(value) >= 60,
    'Auto sleep must be at least 60 seconds.',
  )

export function autoSleepAfterMsToSeconds(autoSleepAfterMs: number | null): string {
  if (autoSleepAfterMs === null) {
    return ''
  }

  return String(autoSleepAfterMs / 1000)
}

export function autoSleepAfterSecondsToMs(autoSleepAfterSeconds: string): number | null {
  const seconds = autoSleepAfterSeconds.trim()
  return seconds === '' ? null : Number(seconds) * 1000
}
