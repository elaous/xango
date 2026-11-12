# Security policy

## Supported versions

Xangô is at v0. Security fixes MUST target the `main` branch and ship in the next patch release.

## Reporting a vulnerability

Reporters MUST NOT open a public issue or pull request for a vulnerability.

Use GitHub private vulnerability reporting: open the repository's **Security** tab, choose
**Report a vulnerability**, and describe the problem. This creates a private security advisory visible only to
the maintainers and you.

A report SHOULD include:

- the affected version or commit;
- steps to reproduce, or a proof of concept;
- the impact you expect (for example: markup that renders unescaped, a `.xg` file that makes `xgc` write outside its
  directory, or a resource import that reads a file outside the module).

The maintainers will try to acknowledge a report within 7 days and to agree on a fix and disclosure date with you.
Reporters SHOULD allow time for a fix before disclosing publicly.

## Scope

In scope: the code in this repository (the runtime, `domrt`, the `xgc` compiler, the `gio` renderer and the examples).

Out of scope: vulnerabilities in [gojiberry](https://github.com/elaous/gojiberry), which SHOULD be reported in that
repository, and vulnerabilities in third-party dependencies, which SHOULD be reported to their own maintainers.
