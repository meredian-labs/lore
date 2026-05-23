// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking.

import { themes as prismThemes } from "prism-react-renderer";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Lore",
  tagline: "Git stores what changed. Lore stores why.",
  favicon: "img/favicon.ico",

  url: "https://nishchay.github.io",
  baseUrl: "/lore/",

  organizationName: "nishchay",
  projectName: "lore",
  trailingSlash: false,

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          // Docs-only mode: serve docs at the root path
          routeBasePath: "/",
          path: "../docs",
          sidebarPath: "./sidebars.js",
          editUrl: "https://github.com/nishchay/lore/edit/main/",
          showLastUpdateTime: true,
          showLastUpdateAuthor: false,
          // Exclude internal dev docs — these live in docs/ for CLAUDE.md references
          // but are not part of the public site.
          exclude: [
            "ARCHITECTURE.md",
            "SCHEMA.md",
            "ROADMAP.md",
            "rules/**",
          ],
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      colorMode: {
        defaultMode: "dark",
        disableSwitch: false,
        respectPrefersColorScheme: true,
      },

      navbar: {
        title: "Lore",
        logo: {
          alt: "Lore Logo",
          src: "img/logo.svg",
          srcDark: "img/logo-dark.svg",
        },
        items: [
          {
            type: "docSidebar",
            sidebarId: "docs",
            position: "left",
            label: "Introduction",
          },
          {
            href: "https://github.com/nishchay/lore",
            label: "GitHub",
            position: "right",
          },
          {
            href: "https://github.com/nishchay/lore#installation",
            label: "lore init",
            position: "right",
            className: "navbar__cta",
          },
        ],
      },

      footer: {
        style: "dark",
        links: [
          {
            title: "Docs",
            items: [
              {
                label: "Introduction",
                to: "/",
              },
              {
                label: "Installation",
                to: "/installation",
              },
              {
                label: "Quick Start",
                to: "/quickstart",
              },
            ],
          },
          {
            title: "Concepts",
            items: [
              {
                label: "Tasks",
                to: "/concepts/tasks",
              },
              {
                label: "Blobs",
                to: "/concepts/blobs",
              },
              {
                label: "Nodes",
                to: "/concepts/nodes",
              },
              {
                label: "Knowledge Graph",
                to: "/concepts/graph",
              },
            ],
          },
          {
            title: "More",
            items: [
              {
                label: "GitHub",
                href: "https://github.com/nishchay/lore",
              },
              {
                label: "Issues",
                href: "https://github.com/nishchay/lore/issues",
              },
              {
                label: "Releases",
                href: "https://github.com/nishchay/lore/releases",
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Lore contributors. Built with Docusaurus.`,
      },

      prism: {
        theme: prismThemes.oneLight,
        darkTheme: prismThemes.oneDark,
        additionalLanguages: ["go", "bash", "json", "toml"],
      },

      docs: {
        sidebar: {
          hideable: true,
          autoCollapseCategories: false,
        },
      },
    }),
};

export default config;
