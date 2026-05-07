package api

// cookie name for the session
const SessionCookieName = "__Host-rivet_session"

// cookie/header pair for CSRF protection on authenticated unsafe requests.
const CSRFCookieName = "__Host-rivet_csrf"
const CSRFHeaderName = "X-CSRF-Token"

const ImageTagHeader = "X-Rivet-Image-Tag"
