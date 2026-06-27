// reactflow package type declarations
// reactflow 11.x ships a CSS file (dist/style.css) that consumers import for styles,
// but the package does not include TypeScript declarations for the CSS module.
// This declaration allows `import 'reactflow/dist/style.css'` to type-check.
declare module 'reactflow/dist/style.css' {
  const cssUrl: string;
  export default cssUrl;
}
