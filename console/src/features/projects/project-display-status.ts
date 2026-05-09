import type { ProjectDisplayStatus } from './types'

export const projectDisplayStatusLabels: Record<ProjectDisplayStatus, string> = {
  starting: 'Starting',
  running: 'Running',
  deploying: 'Deploying',
  sleeping: 'Sleeping',
  waking: 'Waking',
  failed: 'Failed',
  stopped: 'Stopped',
}
