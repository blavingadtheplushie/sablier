// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  // make dev serves pages through the go server on :8080, but vite's live
  // reload socket breaks behind that proxy, so it connects to astro directly
  vite: { server: { ws: { clientPort: 4321 } } },
  integrations: [
    starlight({
      title: "sablier",
      components: {
        ThemeProvider: "./src/components/LightOnly.astro",
        ThemeSelect: "./src/components/Nothing.astro",
      },
      head: [{ tag: "link", attrs: { rel: "stylesheet", href: "/tokens.css" } }],
      customCss: ["./src/styles/custom.css"],
      expressiveCode: {
        themes: ["github-light"],
        defaultProps: { frame: "none" },
        styleOverrides: {
          borderRadius: "var(--radius)",
          borderColor: "var(--color-border)",
          codeBackground: "var(--color-surface)",
          codeFontFamily: "var(--font-mono)",
          frames: { shadowColor: "transparent" },
        },
      },
      description:
        "open source time tracker to track the time you actually spend on making things.",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/espcaa/sablier",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/espcaa/sablier/edit/main/website/",
      },
      sidebar: [
        {
          label: "Start here",
          items: [
            { label: "What is sablier?", slug: "docs" },
            { label: "Getting started", slug: "docs/getting-started" },
            { label: "Supported agents", slug: "docs/supported-agents" },
          ],
        },
        {
          label: "Thingies",
          items: [
            { label: "How sablier counts time", slug: "docs/concepts/time" },
            { label: "Privacy", slug: "docs/concepts/privacy" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Configuration", slug: "docs/reference/configuration" },
            {
              label: "CLI",
              collapsed: true,
              items: [{ autogenerate: { directory: "docs/reference/cli" } }],
            },
          ],
        },
      ],
    }),
  ],
});
