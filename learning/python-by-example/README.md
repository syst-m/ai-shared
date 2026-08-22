# Python by Example

A zero-to-100 Python refresher modeled on [Go by Example](https://gobyexample.com/) — written for
experienced programmers in other languages who need to get current on Python's modern idioms,
best practices, and implementation nuances. **Backend/web-API weighted** (FastAPI, Pydantic,
httpx, async I/O, packaging, pytest).

## What it is

A **single self-contained HTML file** — [`python-by-example.html`](python-by-example.html).
Double-click to open in any browser; no server or build step.

- **57 editable playgrounds** — every example is real, runnable code with Run/Reset buttons and
  output shown below. Edit the code and re-run.
- **In-browser CPython 3.14** via [Pyodide](https://pyodide.org/) (loaded from CDN on first Run,
  ~10 MB one-time). Blocks run in one shared persistent interpreter (notebook semantics), and
  top-level `await` just works.
- **Your edits persist** in localStorage per block.
- Blocks that need server-side packages (FastAPI, real HTTP servers, pip) are marked
  *reference* — shown for the pattern, not executed.

## Contents

| Section | Highlights |
|---------|-----------|
| Getting started | f-strings, venv, pip, the REPL |
| Language basics | comprehensions, slicing, `match`/`case`, `__main__` |
| Functions | `*args`/`**kwargs`, closures, decorators, generators |
| Data, files & time | `with`, `pathlib`, `json`/`csv`/`tomllib`, `datetime`/`zoneinfo` |
| OOP | dataclasses, `@property`, protocols / duck typing |
| Type hints | modern PEP 604 / PEP 695 syntax, `Self`, `TypeAlias` |
| Async Python | `async/await`, `asyncio.gather`, task groups |
| Backend patterns | httpx, FastAPI + Pydantic, config & secrets, logging |
| Testing & tooling | pytest fixtures/parametrize, stdlib toolbox |

## Verification

Every runnable example's output was verified against three engines before shipping: local
CPython 3.14, the real Pyodide 314.0.5 engine (fetched from the same CDN the page uses), and an
end-to-end headless-browser run that clicks Run on all 57 playgrounds — **57/57 on all three**.
