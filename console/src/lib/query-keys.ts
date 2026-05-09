export const authKeys = {
  currentUser: ['auth', 'current-user'] as const,
}

export const projectKeys = {
  all: ['projects'] as const,
  activeProjects: ['projects', 'active'] as const,
  runtimeStats: ['projects', 'runtime-stats'] as const,
}
