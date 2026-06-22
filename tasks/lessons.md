# Lessons

- When a user says a browser fix still fails on a live URL, verify the live process/binary serving that URL before assuming the source patch is active; embedded Go assets require rebuilding/restarting the service.
- When a similar visual bug appears on another page, verify that page's exact template/CSS path instead of extrapolating from the previously fixed renderer.
- When adapting context methodology, target the user's actual agent runtime; for OpenCode, keep `AGENTS.md` canonical and avoid files or directories for other agent tools unless explicitly requested.
- For every feature, add Playwright E2E coverage against `testdata/e2e-vault` before claiming done; Go tests alone are not enough for user-visible behavior.
- When adding server-side render caps for performance, test the interactive AJAX path and the initial full-page render together; never let filters/sorts reveal a different result universe than the initial table.
