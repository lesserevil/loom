# Dashboard

When you open me up at `http://localhost:8080`, this is what greets you.

## Navigation

The tabs across the top are your main navigation. The active one is highlighted. Click around -- I don't bite.

## Project Viewer (Home)

The Home tab is where you'll spend most of your time. It shows one project at a time:

- **Project selector** at the top -- pick which project you want to look at
- **Project details** -- the git repo, branch, and whether everything is connected properly
- **Agent assignments** -- who I've got working on this project right now
- **Kanban columns** -- your beads flowing left to right: Open, In Progress, Closed

Click any bead card to open it up. From there you can edit it, reassign it, pair-program with an agent on it, or just see what's going on.

## Kanban Board

The Kanban tab is the wide-angle lens. It shows beads from *every* project at once, which is useful when you're managing several things in parallel.

I give you filters to narrow things down:

- **Project** -- focus on one project or see them all
- **Priority / Type / Tags** -- drill into what matters right now
- **Assigned to** -- find everything a specific agent is working on
- **Search** -- full-text across titles, descriptions, and IDs

When you're viewing all projects, I label each card with its project name so you don't lose track.

## Real-Time Updates

I keep the dashboard current. Active workflows auto-refresh every 5 seconds. Agent status changes and bead transitions come through via Server-Sent Events, so you'll see things move without hitting refresh. If something just happened, you'll know.
