---
layout: home

hero:
  name: mdv
  text: Module Dependency Visualizer
  tagline: Understand your project's architecture before it becomes technical debt.
  image:
    src: /hero.svg
    alt: mdv dependency graph illustration
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/devekkx/module-dependency-visualizer

features:
  - icon: 🔍
    title: Language-agnostic
    details: Parses go.mod, package.json, requirements.txt, pyproject.toml, poetry.lock and more into a single unified graph.
  - icon: 🛡️
    title: Security & License Audit
    details: Queries the OSV database for known vulnerabilities, flags license conflicts, and detects version mismatches across dependencies.
  - icon: 🌐
    title: Interactive Web UI
    details: Run `mdv serve` to launch an embedded D3.js graph with zoom, pan, filtering, and a live audit panel - no external tools required.
  - icon: 📊
    title: Multiple Export Formats
    details: Export to JSON (schema v1.1), Graphviz DOT, or Mermaid. Pipe directly into CI reports, Notion, or GitHub Markdown.
  - icon: 🔀
    title: Dependency Diff
    details: Compare dependency snapshots across git refs and see exactly what changed - additions, removals, and version bumps.
  - icon: ⚙️
    title: GitHub Actions Ready
    details: A drop-in composite action analyses, audits, diffs, and generates DEPENDENCIES.md on every PR.
---
