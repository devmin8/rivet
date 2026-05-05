package caddy

func notFoundRoute() map[string]any {
	return map[string]any{
		"handle": []any{
			map[string]any{
				"handler":     "static_response",
				"status_code": "404",
				"headers": map[string][]string{
					"Content-Type": {"text/html; charset=utf-8"},
				},
				"body": notFoundHTML,
			},
		},
		"terminal": true,
	}
}

const notFoundHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Rivet - 404</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #f8fafc;
      --text: #111827;
      --muted: #64748b;
      --line: #d7dde7;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #0f141b;
        --text: #f8fafc;
        --muted: #9aa7b8;
        --line: #263241;
      }
    }
    * {
      box-sizing: border-box;
    }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    main {
      width: min(100% - 48px, 520px);
      border-top: 1px solid var(--line);
      padding-top: 28px;
    }
    .brand {
      margin: 0 0 18px;
      font-size: 14px;
      font-weight: 700;
      letter-spacing: 0;
    }
    h1 {
      margin: 0;
      font-size: 44px;
      line-height: 1;
      letter-spacing: 0;
    }
    p {
      margin: 16px 0 0;
      color: var(--muted);
      font-size: 16px;
      line-height: 1.6;
    }
  </style>
</head>
<body>
  <main>
    <p class="brand">Rivet</p>
    <h1>404</h1>
    <p>No route is configured for this address.</p>
  </main>
</body>
</html>`
