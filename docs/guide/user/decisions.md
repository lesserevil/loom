# Decisions & Approvals

Sometimes I need a human. Not because I can't decide -- I handle routine decisions on my own all day -- but because some choices carry enough weight that you should be the one making them.

When one of my agents hits something ambiguous, risky, or politically sensitive, it creates a **decision bead** and I put it in front of you.

## Reviewing Decisions

1. Open the **Decisions** tab
2. Read what the agent is asking, what options it sees, and what context it's providing
3. Make your call and submit

Or through the API:

```bash
# What's waiting on me?
curl http://localhost:8080/api/v1/decisions

# Here's my answer
curl -X PUT http://localhost:8080/api/v1/decisions/<id> \
  -H "Content-Type: application/json" \
  -d '{
    "resolution": "yes",
    "resolved_by": "admin",
    "comment": "Approved — proceed with option A"
  }'
```

## What I Handle Myself

I don't bother you with everything. Low-risk code fixes -- typos, missing imports, formatting, single-file changes -- I can auto-approve those. I assess risk based on:

- **Low risk**: Typos, formatting, single-file changes, missing imports. I handle these.
- **Medium risk**: Multi-file changes, API modifications. I'll usually ask.
- **High risk**: Security changes, database migrations, infrastructure changes. I always ask.

If something self-assesses as "Risk Level: Low" or matches my low-risk heuristics, I let it through. Everything else comes to you.

## Timeouts

I give you 48 hours on a decision by default. I'm patient, but dependent work is blocked while you think it over. If the clock runs out, I'll either escalate to the CEO persona or auto-resolve based on the decision type.

Check your Decisions tab. If I put something there, it's because I genuinely couldn't proceed without you.

## Consensus

For decisions that really matter, I can poll multiple agents before proceeding. This is consensus mode -- I gather opinions, require a quorum, and only advance when enough voices agree. I use this sparingly, but it's there when the stakes are high.
