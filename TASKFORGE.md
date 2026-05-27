# TaskForge Autonomous Coding Board

A repo-centered task management and execution system for agentic software development.

## Core Mission

TaskForge exists to manage software work for an agentic coding team. It combines:

- A human-visible task board
- Repo-native Markdown task specifications
- Isolated agent workspaces using git worktrees
- Task branches and pull requests
- Automatic continuation policies
- Explicit human-intervention gates
- Project status summaries

## Operating Model

Three layers:

1. **Human-visible board** — GitHub Issues/Projects, Plane, Linear, Jira, or repo-native Markdown
2. **Repo-native task specs** — the execution contract (these Markdown files)
3. **Agent execution in isolated worktrees** — the isolation boundary

## Task Workflow

```
Inbox → Needs Spec → Ready → In Progress → Review → Verify → Done
                         ↓
                      Blocked
```

## CLI Commands

| Command | Description |
|---|---|
| `taskforge init` | Initialize TaskForge in this repo |
| `taskforge next` | Return highest-priority safe task |
| `taskforge start TASK-123` | Set up worktree, branch, begin task |
| `taskforge status` | Show project status summary |
| `taskforge summary` | Show full project summary |
| `taskforge block TASK-123 "reason"` | Mark task as blocked |
| `taskforge done TASK-123` | Mark task as done |

See the full specification for agent roles, continuation policy, and integration details.
