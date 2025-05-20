# templates/SUMMARY.md

This folder contains Go HTML templates used to render user interfaces, potentially for the `vibrant canvas` server or other web frontends within the project. Templates are defined using Go's `text/template` or `html/template` syntax.

Currently, it includes:
*   `BasePage.html`: A base HTML structure that other pages can extend. It includes setup for Tailwind CSS, HTMX, and a dark mode theme toggle.
*   Other HTML files (like `Header.html`, `ModalContainer.html`, `ToastContainer.html`) seem to be partials intended for inclusion in `BasePage.html` or other main page templates.

The `vibrant canvas` command in `cmd/canvas.go` uses `panyam/templar` to serve these templates, making them accessible for UI development and testing.

Files like `INSTRUCTIONS.md` within the project root suggest a convention for how these templates and associated TypeScript components (from a `./components` folder, though not directly part of this summary's scope) are meant to interact and be built.
