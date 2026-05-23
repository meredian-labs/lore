// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    "introduction",
    "installation",
    "quickstart",
    {
      type: "category",
      label: "Concepts",
      collapsed: false,
      items: [
        "concepts/tasks",
        "concepts/blobs",
        "concepts/nodes",
        "concepts/graph",
      ],
    },
    {
      type: "category",
      label: "Integrations",
      collapsed: false,
      items: [
        "integrations/claude-code",
        "integrations/cursor",
        "integrations/openhands",
        "integrations/custom-agent",
      ],
    },
    {
      type: "category",
      label: "CLI Reference",
      collapsed: false,
      items: [
        "cli/init",
        "cli/log",
        "cli/show",
        "cli/why",
        "cli/trace",
        "cli/graph",
        "cli/status",
        "cli/record",
        "cli/doctor",
        "cli/node",
        "cli/assign",
        "cli/glh",
      ],
    },
    {
      type: "category",
      label: "Architecture",
      collapsed: true,
      items: [
        "architecture/storage",
        "architecture/schema",
        "architecture/trust-model",
      ],
    },
  ],
};

export default sidebars;
