# ccx — Claude Code Explorer

A terminal UI for browsing, inspecting, and managing [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

Browse sessions, read conversations, inspect tool calls, view agent hierarchies, explore configs/plugins, and get aggregated stats — all from your terminal.

## Install

```bash
go install github.com/sendbird/ccx@latest
```

Or build from source:

```bash
git clone https://github.com/sendbird/ccx.git
cd ccx
make build      # -> bin/ccx
make install    # -> ~/.local/bin/ccx
```

## Usage

```bash
ccx                        # launch TUI
ccx -view config           # start in config explorer
ccx -view stats            # start in global stats
ccx -view plugins          # start in plugin explorer
ccx -group tree            # start with tree grouping
ccx -preview stats         # start with stats preview open
ccx -search "is:live"      # start filtered to live sessions
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `-version`, `-v` | Print version and exit |
| `-dir PATH` | Claude data directory (default: `~/.claude`) |
| `-view MODE` | Initial view: `sessions`, `config`, `plugins`, `stats` |
| `-group MODE` | Initial grouping: `flat`, `proj`, `tree`, `chain`, `fork` |
| `-preview MODE` | Initial preview: `conv`, `stats`, `mem`, `tasks` |
| `-search QUERY` | Start with session filter applied |
| `-tmux` | Enable tmux integration (auto-detected) |
| `-tmux-auto-live` | Auto-enter live session in same tmux window |
| `-worktree-dir NAME` | Worktree subdirectory name (default: `.worktree`) |

The Claude data directory is resolved in order: `--dir` flag → `CLAUDE_CONFIG_DIR` env → `~/.claude`.

## Views

### Session Browser

Browse all Claude Code sessions across projects, sorted by recency.

- **Live/Busy badges** — see which sessions are actively running
- **Search** (`/`) — filter by project, branch, prompt, window name, or tags
- **Group modes** (`G` or `:group:*`):
  - **Flat** — simple list sorted by time
  - **Project** — clustered by project path
  - **Tree** — team hierarchy with leader/teammate nesting
  - **Chain** — resume-chain grouping (parent → child)
  - **Fork** — agent-fork grouping
- **Directory filter** (`g`) — scope to a single project directory
- **Preview pane** (`Tab` to cycle): conversation, stats, memory, tasks/plan, live
- **Multi-select** (`Space`) — bulk delete, copy paths, send input
- **Actions menu** (`x`) — delete, move, resume, copy path, worktree, kill, input, jump
- **Command mode** (`:`) — vim-style commands with fuzzy suggestions

#### Search Filters

| Filter | Matches |
|--------|---------|
| `is:live` | Running Claude process |
| `is:busy` | Actively responding |
| `is:wt` | In a git worktree |
| `is:team` | Part of a team session |
| `is:fork` | Forked from another session |
| `has:mem` | Has memory file |
| `has:todo` | Has todos |
| `has:task` | Has tasks |
| `has:plan` | Has plan |
| `has:agent` | Has subagents |
| `has:compact` | Uses message compaction |
| `has:skill` | Used skills |
| `has:mcp` | Used MCP tools |
| `team:NAME` | Filter by team name |
| `win:NAME` | Filter by tmux window name |

Plain text terms match against project path, name, branch, session ID, first prompt, and teammate name. Multiple terms are AND-matched.

### Conversation View

Drill into any session to read the full conversation.

- **Split-pane preview** (`Tab`/`→`) — foldable message detail
- **Block navigation** (`↑`/`↓`) — navigate text, tool calls, and results
- **Fold/unfold** (`←`/`→`, `f`/`F`) — collapse/expand content blocks
- **Block filter** (`/`) — filter by `is:tool`, `is:hook`, `is:error`, `tool:Name`
- **Preview modes** (`Tab`) — cycle between text, tool, and hook views
- **Agent drill-down** (`Enter` on agent) — recursive sub-session navigation
- **Full conversation** (`c`) — scrollable concatenated view with copy mode
- **Live tail** (`L`) — auto-follow active sessions in real-time
- **Send input** (`I`) — send text to running Claude via tmux
- **Jump to pane** (`J`) — switch to the tmux pane running the session

### Detail View

Full-screen message viewer with block-level navigation.

- **Block cursor** (`↑`/`↓`) — navigate between blocks
- **Fold/unfold** (`←`/`→`, `f`/`F`) — collapse/expand blocks
- **Message navigation** (`n`/`N`) — step through messages
- **Copy mode** (`v`) — select and copy text ranges
- **Pager** (`o`) — open in external pager

### Global Stats (`v` → `s`)

Aggregated metrics across all sessions with detail drill-down.

- **Overview** — total sessions, messages, tokens, duration
- **Tools** (`p` → `t`) — built-in tool usage with timelines
- **MCP Tools** (`p` → `m`) — MCP tool usage with timelines
- **Agents** (`p` → `a`) — agent type breakdown
- **Skills** (`p` → `s`) — skill usage with error trends
- **Commands** (`p` → `c`) — command usage with error trends
- **Errors** (`p` → `e`) — error breakdown by category
- **Hooks** — hook usage with timestamp analysis

### Config Explorer (`v` → `c`)

Browse and manage all Claude Code configuration files.

- **Category filter** (`Tab`) — global, project, local, skills, agents, commands, MCP, hooks
- **Split preview** — file content with syntax awareness
- **Multi-select** (`Space`) — select configs for testing
- **Test env** (`t`) — launch isolated Claude session with only selected configs
- **Edit** (`e` / `Enter`) — open in `$EDITOR`
- **Actions menu** (`x`) — edit, copy path, open shell at path

The test environment creates an isolated `HOME` with only selected memory/config files symlinked, preserving your editor config and extracting OAuth from keychain for connector MCP access.

### Plugin Explorer (`v` → `p`)

Browse installed Claude Code plugins and their components.

- **Component drill-down** (`Enter`) — view plugin agents, skills, commands, hooks, MCP servers
- **Multi-select** (`Space`) — select components for batch editing
- **Edit** (`e`) — open component files in `$EDITOR`
- **Actions menu** (`x`) — edit, copy path, open shell
- **Component badges** — e.g. `[3a 2s 1c]` = 3 agents, 2 skills, 1 command

## Keybindings

### Sessions

| Key | Action |
|-----|--------|
| `Enter` | Open conversation view |
| `/` | Search/filter sessions |
| `g` | Filter by project directory |
| `G` | Cycle group mode |
| `Tab` | Cycle preview mode |
| `Shift+Tab` | Reverse cycle preview |
| `→` | Open/focus preview |
| `←` | Close/unfocus preview |
| `[` / `]` | Adjust split ratio |
| `Space` | Multi-select toggle |
| `x` | Actions menu |
| `v` | Views menu (stats/config/plugins) |
| `:` | Command mode |
| `L` | Live preview (tmux) |
| `I` | Send input to live session |
| `J` | Jump to tmux pane |
| `R` | Refresh |
| `S` | Global stats |
| `?` | Help |
| `q` | Quit |

### Conversation

| Key | Action |
|-----|--------|
| `Enter` | Open detail / drill into agent |
| `c` | Full conversation view |
| `/` | Filter blocks |
| `Tab` | Cycle preview detail (text/tool/hook) |
| `↑` / `↓` | Navigate messages/blocks |
| `←` / `→` | Fold/unfold blocks |
| `f` / `F` | Fold/unfold all |
| `[` / `]` | Adjust split ratio |
| `L` | Toggle live tail |
| `I` | Send input |
| `J` | Jump to pane |
| `e` | Open in editor |
| `R` | Refresh |

### Detail View

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate blocks |
| `←` / `→` | Fold/unfold block |
| `f` / `F` | Fold/unfold all |
| `n` / `N` | Next/prev message |
| `v` | Copy mode |
| `y` | Copy to clipboard |
| `o` | Open in pager |

### Command Mode (`:`)

| Command | Action |
|---------|--------|
| `group:flat` | Switch to flat grouping |
| `group:proj` | Switch to project grouping |
| `group:tree` | Switch to tree grouping |
| `group:chain` | Switch to chain grouping |
| `group:fork` | Switch to fork grouping |
| `preview:conv` | Conversation preview |
| `preview:stats` | Stats preview |
| `preview:mem` | Memory preview |
| `preview:tasks` | Tasks preview |
| `view:stats` | Open global stats |
| `view:config` | Open config explorer |
| `view:plugins` | Open plugin explorer |
| `refresh` | Refresh sessions |
| `keymap:edit` | Edit keymap config |

Short aliases: `g:flat`, `p:conv`, `v:stats`, `R`, `km:edit`.

### Global

| Key | Action |
|-----|--------|
| `Esc` | Go back / close |
| `q` | Quit |

## Configuration

Keymap config: `~/.config/ccx/config.yaml` (bootstrap with `:keymap:edit`)

## How It Works

ccx reads Claude Code's session files from `~/.claude/projects/`. Each session is a JSONL file containing the full conversation history — user prompts, assistant responses, tool calls, and results.

Session metadata is cached to `~/.claude/sessions.gob` for instant startup (~1ms). A full async scan runs in the background to pick up new sessions.

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Requirements

- Go 1.25+
- Claude Code sessions in `~/.claude/projects/`
- tmux (optional, for live session features)

## License

MIT
