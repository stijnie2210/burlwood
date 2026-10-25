# Burlwood website

The single-page site for Burlwood, built with [Astro](https://astro.build) as static HTML.

```bash
npm install
npm run dev       # local dev server
npm run build     # static output → dist/
npm run preview   # serve dist/
```

Or as a container, serving the static build with nginx on port 8080:

```bash
docker build -t burlwood-website .
docker run --rm -p 8080:8080 burlwood-website

# or
docker compose up --build
```

The build asks the GitHub API for the latest release tag and falls back to the version in `src/lib/release.ts` when it can't reach it. Download buttons point at `releases/latest/download/…`, so they follow new releases without a rebuild.

The screenshots in `src/assets/` are copies and crops of `../assets/screenshots/`; retake those and recrop when the app changes.
