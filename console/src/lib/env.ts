const rivetApiURL = import.meta.env.VITE_RIVET_API_URL

if (typeof rivetApiURL !== 'string' || rivetApiURL.trim() === '') {
  throw new Error('VITE_RIVET_API_URL is required')
}

export const env = {
  rivetApiURL: rivetApiURL.replace(/\/+$/, ''),
}
