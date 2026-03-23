---
title: Serving This Website Over SSH
date: 2026-03-22
description: Why I built a terminal-based personal website you can access with just an SSH client.
draft: true
---

## The terminal rabbit hole

- Been living in the terminal lately — tmux, nvim, TUIs for everything
- Discovered terminal.shop and eieio.games — SSH as a platform, not just remote access
- terminal.shop: love the simplicity of the SSH app
- eieio.games: love the esoteric, playful use of the medium
- Thought: what if my personal site lived here too?

## Why SSH?

- "Why not just make a normal website?" — fair question
- It's not practical. That's kind of the point.
- But genuinely:
  - No JS, no cookies, no tracking — privacy by architecture, nothing to block
  - Universal client — every dev machine has `ssh`, zero install
  - Keyboard-first nav — natural for terminal users, no mouse
- Honest about tradeoffs: image rendering is... creative

## The stack

- Go + Bubble Tea + Wish (Charm ecosystem)
- Shared `content/` dir — same markdown posts serve both SSH and HTTP (Astro) site
- Brief nod to the dual-site monorepo approach

## Show & tell

- Side-by-side SSH vs HTTP: same post rendered in terminal vs browser
- Chafa image rendering: show how images look in the terminal
  - Honest: it's not great, but it's charming
- Goal was feature parity — read the same content either way

## Try it

- `ssh ssh.paullj.com`
- Link to GitHub repo
- Encourage others to build their own
