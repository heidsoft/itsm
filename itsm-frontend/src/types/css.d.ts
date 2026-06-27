// Global CSS module type declaration
// Multiple npm packages (reactflow, tailwind layer resets, antd reset.css, etc.)
// ship CSS files imported as side-effects for styling. Next.js 15's built-in CSS
// handling does not always emit TypeScript declarations (observed with
// TypeScript 6.0.3 / next 15.5.12: `Cannot find module or type declarations for
// side-effect import of '.../*.css'`). This wildcard makes every `*.css`
// import type-check without forking the upstream packages.
//
// Local files (e.g. `./globals.css`, `../styles/theme-variables.css`) and
// package CSS (e.g. `reactflow/dist/style.css`) are all covered.
declare module '*.css';