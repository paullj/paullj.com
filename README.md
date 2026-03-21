# paullj.com

A blog served over ssh, built with go, [bubble tea](https://github.com/charmbracelet/bubbletea), and [wish](https://github.com/charmbracelet/wish).

![Demo](docs/demo.gif)

```bash
ssh ssh.paullj.com
```

## Prerequisites

- [mise](https://mise.jdx.dev) — manages all project tooling (go, linters, vhs, etc.)
- [ffmpeg](https://ffmpeg.org) — required for VHS demo recording (install via system package manager)
- [ttyd](https://github.com/tsl0922/ttyd) — required for VHS demo recording (install via system package manager)

## Setup

```bash
git clone https://github.com/paullj/paullj.com.git
cd paullj.com
mise install
```

## Usage

```bash
mise run dev           # start server with hot reload
ssh -p 2222 localhost  # connect
```

## Development

```bash
mise run lint      # run linter
mise run fmt       # format code
mise run build     # build binary
mise run deploy    # deploy to fly.io
```
