# 🕸️ Universal Dependency Visualizer

**Understand your project's architecture before it becomes technical debt.**

Most dependency tools are locked into a single ecosystem. This project provides a **language-agnostic** engine that parses `go.mod`, `package.json`, `requirements.txt`, and more into a unified, interactive visual map.

Whether you are performing a security audit, debugging version conflicts, or onboarding onto a massive monorepo, this tool gives you the "bird's-eye view" you need to stay productive.

### Key Highlights:
- **Language Agnostic:** Support for Go, JS/TS, Python, and Java out of the box.
- **Blast Radius Analysis:** See exactly what breaks when you upgrade a library.
-️ **Security Integrated:** Overlays vulnerability data directly onto your graph.
- **Go-Powered:** Blazing fast analysis, even for projects with thousands of dependencies.
- **Multiple Outputs:** Export to interactive HTML, SVG, Mermaid.js, or DOT.