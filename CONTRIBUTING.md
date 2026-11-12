# Contributing

Thank you for helping. Before you send a change:

- Run the checks the repository runs: `make lint fmt-check generate-check docs-check test`. `make check` runs everything,
  including the nested modules and the VS Code grammar suite.
- `scripts/targeted-tests.sh` runs only the packages a change touches; `scripts/targeted-tests.sh --list` prints what it
  would run.
- Regenerate and commit the `*.xg.go` files whenever you change a `.xg` file or the compiler (`make generate`), and the
  API dumps under `docs/` whenever you change an exported identifier (`make docs`).
- Update the docs that describe what you changed (`README.md`, the package docs).

## Licence of contributions

The project is under the GNU AGPL, version 3, and also offers a commercial licence ([COMMERCIAL.md](COMMERCIAL.md)). To be
able to do both, a contribution is accepted on these terms (a short contributor agreement; a lawyer should review the final
wording):

1. You wrote the contribution, or you have the right to give it, and nobody else's rights or employer's terms stand in the
   way.
2. You license it to everyone under the GNU AGPL, version 3, like the rest of the project.
3. You also grant the project's owner a perpetual, worldwide, non-exclusive, royalty-free licence to use, change and
   sublicense your contribution under other terms, including the commercial licence, and to relicense it under another free
   software licence if the project moves to one.
4. You keep the copyright in your contribution.

Sending a pull request says you agree. If you cannot agree, say so in the pull request and we will talk about it before
anything is merged.
