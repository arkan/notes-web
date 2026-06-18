import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  // Global ignores — exclude everything except our TypeScript sources.
  {
    ignores: [
      "**/node_modules/**",
      "**/playwright-report/**",
      "**/test-results/**",
      "internal/**",
      "cmd/**",
      "nix/**",
      "bin/**",
      "scripts/**",
      "*.go",
      "*.md",
      "*.css",
      "*.json",
      "*.yml",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
);
