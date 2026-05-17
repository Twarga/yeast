# GitHub Pages Setup

Yeast has a static landing page in the `docs/` directory.

Main files:

- `docs/index.html`
- `docs/site.css`
- `docs/banner.png`

Expected public URL after Pages is enabled:

```text
https://twarga.github.io/yeast/
```

## Recommended Setup

Use GitHub Pages with GitHub Actions.

The workflow is:

- source folder: `docs/`
- hosted artifact: static HTML/CSS/images
- no build step

In GitHub:

1. Open the repository settings.
2. Go to Pages.
3. Set the source to GitHub Actions.
4. Run the `Deploy GitHub Pages` workflow.

## Local Preview

From the repository root:

```bash
python3 -m http.server 8080 --directory docs
```

Then open:

```text
http://127.0.0.1:8080
```

## Notes

The site intentionally uses plain static files. There is no frontend framework, no package install, and no build tool. That keeps the landing page easy to maintain while Yeast is still in early release work.
