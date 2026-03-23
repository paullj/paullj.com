---
title: Serving This Website Over SSH
date: 2026-03-22
description: Why I built a terminal-based personal website you can access with just an SSH client.
draft: true
---

## The terminal rabbit hole

- moved away from vscode
- Been enjoying using the terminal lately tmux, nvim, TUIs for everything 
- SSH as a platform/app not just remote access
- terminal.shop: love the simplicity of the SSH app
- eieio.games: love the esoteric, playful use of the medium
- im sure its been done before, but why not serve this site over ssh too

## Why SSH?

- "Why not just make a normal website?" - yeah fair 
- It's not practical, its just for fun
- But real reasons why u might want to read via ssh
  - No JS, no cookies, no tracking — privacy by architecture, nothing to block
  - Universal client - every dev machine has `ssh`, zero install
  - Keyboard-first nav - natural for terminal users, no mouse
  - can stay in one tool
- Honest about tradeoffs: image rendering is... fun/cool but not isnt actually useable for most images

## The stack

- monorepo both http and ssh sites driven by gfm markdown content and yaml files
- Shared `content/` dir - same markdown posts serve both SSH and HTTP (Astro) site

- Go + Bubble Tea + Wish. charm
- astro for markdown

## Show & tell

- Side-by-side SSH vs HTTP: same post rendered in terminal vs browser
- Chafa image rendering: show how images look in the terminal
  - it's not great, but it's charming
- adding in the gfm features like footnotes, alerts and ascii mermaid diagrams. mermaid diagrams are fun but the produced outputs don't always work well in the terminal
- Goal was as close to feature parity as makes sense - read the same content either way

## Try it

- `ssh ssh.paullj.com`
- Link to GitHub repo
- Encourage others to build their own
