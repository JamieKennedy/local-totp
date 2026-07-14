import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "astro/config";

export default defineConfig({
  site: "https://jamiekennedy.github.io",
  base: "/local-totp",
  output: "static",
  trailingSlash: "always",
  integrations: [mdx(), react()],
  markdown: {
    shikiConfig: {
      themes: { light: "github-light", dark: "github-dark" },
      defaultColor: false,
    },
  },
  vite: {
    plugins: [tailwindcss()],
    resolve: { alias: { "@": new URL("./src", import.meta.url).pathname } },
  },
});
