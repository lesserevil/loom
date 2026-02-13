# Agent Skills Migration Plan

## Current State vs Agent Skills Specification

### Loom's Current Persona System

**Structure:**
```
personas/
├── default/
│   ├── cto/
│   │   ├── PERSONA.md         # Role description, responsibilities
│   │   └── AI_START_HERE.md   # Quick start instructions
│   ├── engineering-manager/
│   │   ├── PERSONA.md
│   │   └── AI_START_HERE.md
│   └── ...
```

**Characteristics:**
- **Rigid schema**: Go struct with 24+ fields (Character, Tone, FocusAreas, AutonomyLevel, etc.)
- **Role-based**: Personas represent org chart positions (CTO, Engineering Manager, QA Engineer)
- **Multiple files**: PERSONA.md + AI_START_HERE.md + optional support docs
- **Org-specific**: Tied to Loom's organizational structure
- **Non-portable**: Can't easily share with other agent systems

**Current Persona struct:**
```go
type Persona struct {
    Name                 string
    Character            string
    Tone                 string
    FocusAreas           []string
    AutonomyLevel        string   // "full", "semi", "supervised"
    Capabilities         []string
    DecisionMaking       string
    Housekeeping         string
    Collaboration        string
    Standards            []string
    Mission              string
    Personality          string
    AutonomyInstructions string
    DecisionInstructions string
    PersistentTasks      string
    PersonaFile          string
    InstructionsFile     string
    // ... 24+ fields total
}
```

### Agent Skills Specification (agentskills.io)

**Structure:**
```
skills/
├── triage-authority/
│   ├── SKILL.md           # Required: YAML frontmatter + instructions
│   ├── scripts/           # Optional: executable code
│   ├── references/        # Optional: additional docs
│   └── assets/            # Optional: templates, data files
```

**Characteristics:**
- **Minimal schema**: Only 2 required fields (name, description)
- **Capability-based**: Skills represent abilities, not roles
- **Single entry point**: SKILL.md with YAML frontmatter
- **Portable**: Standard format works across different agent systems
- **Progressive disclosure**: Load metadata → instructions → resources on-demand

**Standard SKILL.md format:**
```yaml
---
name: skill-name
description: What the skill does and when to use it.
license: Apache-2.0        # Optional
compatibility: Claude Code # Optional
metadata:                  # Optional
  author: example-org
  version: "1.0"
---

# Skill Instructions

Step-by-step instructions for performing the task...
```

## Gap Analysis

| Aspect | Current (Loom) | Standard (Agent Skills) | Gap |
|--------|----------------|-------------------------|-----|
| **Purpose** | Define organizational roles | Define reusable capabilities | ❌ Conceptual mismatch |
| **Structure** | Multiple files (PERSONA.md + AI_START_HERE.md) | Single SKILL.md with frontmatter | ❌ Format incompatible |
| **Schema** | 24+ rigid fields | 2 required + flexible metadata | ❌ Overly rigid |
| **Portability** | Loom-specific | Cross-system standard | ❌ Not portable |
| **Discovery** | Load all persona files on startup | Progressive disclosure (metadata → instructions → resources) | ❌ Inefficient |
| **Composability** | Personas are monolithic roles | Skills are composable capabilities | ❌ Not composable |

## Migration Strategy

### Phase 1: Convert Personas to Skills Format

**Goal**: Make personas compatible with Agent Skills spec while maintaining backward compatibility

**Changes:**
1. **Rename files**: `PERSONA.md` → `SKILL.md`
2. **Add YAML frontmatter** to SKILL.md:
   ```yaml
   ---
   name: cto
   description: Technical decision-maker and triage authority. Routes unassigned beads, makes architecture decisions, and unblocks stuck work.
   license: Proprietary
   compatibility: Designed for Loom
   metadata:
     role: Chief Technology Officer
     autonomy_level: semi
     specialties: [triage, architecture, delegation, risk-assessment]
     author: loom
     version: "1.0"
   ---
   ```
3. **Consolidate AI_START_HERE.md**: Merge into SKILL.md body
4. **Move support docs**: Relocate to `references/` subdirectory
5. **Update loader**: Teach persona manager to read SKILL.md format

**Example conversion (CTO):**

**Before:**
```
personas/default/cto/
├── PERSONA.md          # Role description
└── AI_START_HERE.md    # Quick start
```

**After:**
```
personas/default/cto/
├── SKILL.md            # Frontmatter + combined instructions
└── references/
    └── DELEGATION_RULES.md
```

**SKILL.md content:**
```yaml
---
name: cto
description: Technical decision-maker and triage authority for engineering organization. Routes unassigned beads to specialists, makes architecture decisions, unblocks stuck work, and escalates to CEO for business-critical decisions.
license: Proprietary
compatibility: Designed for Loom (or similar multi-agent orchestration systems)
metadata:
  role: Chief Technology Officer
  autonomy_level: semi
  focus_areas: [triage, architecture, delegation, risk-assessment, cross-team-coordination]
  author: loom
  version: "1.0"
---

# Chief Technology Officer (CTO)

You are the ultimate technical decision-maker and default triage authority. Every bead must have an owner, and you are the fallback when no one else is assigned.

## Priority Actions

1. **Check for unassigned beads** — If `assigned_to` is empty, assess and delegate immediately
2. **Check for blocked beads** — If Ralph blocked a bead, read `ralph_blocked_reason` and re-scope or reassign
3. **Check for denied decisions** — If CEO denied work, read `ceo_comment` and coordinate response

## Triage Process

When you receive a bead:
1. Read title, description, and context (dispatch history, loop detection reasons)
2. Determine domain (frontend, backend, infra, docs, etc.)
3. Delegate to appropriate specialist using `delegate_task` action
4. If bead is too vague, break into sub-tasks with `create_bead`

## Delegation Rules

- Frontend/UI → Web Designer or Web Designer Engineer
- Backend/API → Engineering Manager
- DevOps/infrastructure → DevOps Engineer
- Testing/QA → QA Engineer
- Documentation → Documentation Manager
- Code review → Code Reviewer
- Financial/cost → CFO
- Product direction → Product Manager
- Public comms → Public Relations Manager

See [detailed delegation rules](references/DELEGATION_RULES.md) for edge cases.

## Autonomy Level

**semi** — Can triage and delegate autonomously, but escalates to CEO for:
- Budget decisions
- External commitments
- Breaking changes to public APIs
- Cross-org impact

## Architecture Review

For beads touching core infrastructure:
1. Review the approach before approving
2. Assess technical risk
3. Ensure alignment with technical strategy
4. Check for cross-team dependencies

## Git Workflow

Follow standard git workflow:
- Branch naming: `agent/{bead-id}/{description-slug}`
- Create PR with detailed description
- Request review if uncertain

See Engineering Manager's references for complete git workflow.
```

### Phase 2: Extract Composable Skills

**Goal**: Break monolithic personas into reusable skill components

**Approach**: Extract common capabilities that could be reused across different agent systems.

**Example skill decomposition:**

**CTO persona** becomes:
- `triage-authority` skill (routing beads to specialists)
- `architecture-review` skill (evaluating technical decisions)
- `technical-delegation` skill (assigning work to domain experts)
- `risk-assessment` skill (evaluating technical risk)

**Engineering Manager persona** becomes:
- `code-implementation` skill (writing production code)
- `git-workflow` skill (branch, commit, PR creation)
- `technical-documentation` skill (writing technical docs)
- `api-design` skill (designing RESTful APIs)

**Benefits:**
- **Reusability**: Skills can be mixed/matched across different agents
- **Portability**: Skills work with any agent system supporting the spec
- **Composability**: Build complex agents from simple skill building blocks
- **Maintainability**: Update skill once, affects all agents using it
- **Shareability**: Publish skills to public registries for community use

**Example composable skill:**

```yaml
---
name: triage-authority
description: Routes incoming work items to appropriate specialists based on domain expertise. Reviews unassigned tasks, assesses complexity and domain, and delegates to domain experts. Use when managing incoming work queues or operating as a tech lead role.
license: Apache-2.0
metadata:
  author: loom
  version: "1.0"
  category: coordination
---

# Triage Authority

Route incoming work items to the right specialist based on domain, complexity, and current workload.

## When to Use

- You receive an unassigned task/bead/ticket
- A task is returned from another agent (blocked, escalated, rejected)
- Multiple specialists could handle a task and you need to choose

## Triage Process

1. **Read the work item**
   - Title, description, acceptance criteria
   - Any prior attempt history (who tried, why it failed)
   - Tags indicating domain or urgency

2. **Assess the domain**
   - Frontend (UI/UX, styling, React/Vue/etc.)
   - Backend (API, database, business logic)
   - Infrastructure (deployment, scaling, monitoring)
   - Quality (testing, code review, security)
   - Documentation (user guides, API docs, architecture)

3. **Select the specialist**
   - Match domain to specialist capabilities
   - Consider current workload if visible
   - Check if specialist has handled similar items before

4. **Delegate with context**
   - Assign to specialist
   - Add notes about complexity, urgency, or constraints
   - Mention any prior attempts or blockers

5. **Break down if needed**
   - If task spans multiple domains, split into sub-tasks
   - Assign each sub-task to appropriate specialist
   - Track dependencies between sub-tasks

## Delegation Patterns

### Standard Domains

| Domain | Specialist Role |
|--------|----------------|
| Frontend/UI | Web Designer, Frontend Engineer |
| Backend/API | Backend Engineer, Engineering Manager |
| DevOps/Infra | DevOps Engineer, SRE |
| Testing/QA | QA Engineer, Test Engineer |
| Documentation | Technical Writer, Documentation Manager |
| Security | Security Engineer, Code Reviewer |
| Database | Database Engineer, Data Engineer |

### Edge Cases

- **Unclear domain**: Ask for clarification or break into discovery task
- **Cross-domain work**: Create sub-tasks for each domain
- **Novel work**: Assign to most experienced generalist
- **Blocked work**: Review blocker, reassign or re-scope

## Example Interactions

### Simple delegation
```
Input: "Bug in login form - validation not working"
Analysis: Frontend bug, UI-focused
Action: Assign to Frontend Engineer with note "Check form validation logic"
```

### Complex delegation
```
Input: "Add authentication to API"
Analysis: Multi-domain (backend API + frontend integration + security review)
Actions:
  1. Create sub-task "Design auth flow" → Backend Engineer
  2. Create sub-task "Implement API endpoints" → Backend Engineer
  3. Create sub-task "Add auth UI" → Frontend Engineer
  4. Create sub-task "Security review" → Security Engineer
  Mark dependencies: 2 blocks 3, all block 4
```

### Escalation
```
Input: "Should we migrate to microservices?"
Analysis: Architecture decision, high impact
Action: Escalate to CTO or Architecture team with context
```

## Configuration

If your system has custom domain mappings, document them in `references/DOMAIN_MAP.md`.

## See Also

- [Delegation best practices](references/DELEGATION.md)
- [Domain taxonomy](references/DOMAINS.md)
```

### Phase 3: Modernize Persona System

**Goal**: Redesign Loom to use composable skills while maintaining persona-based routing

**Architecture:**

```
skills/
├── coordination/
│   ├── triage-authority/SKILL.md
│   └── technical-delegation/SKILL.md
├── engineering/
│   ├── code-implementation/SKILL.md
│   └── api-design/SKILL.md
├── architecture/
│   └── architecture-review/SKILL.md
└── git/
    └── git-workflow/SKILL.md

personas/
├── cto/
│   ├── PERSONA.md         # References skills
│   └── config.yaml        # Skill composition
└── engineering-manager/
    ├── PERSONA.md
    └── config.yaml
```

**config.yaml (skill composition):**
```yaml
persona: cto
role: Chief Technology Officer
skills:
  - coordination/triage-authority
  - coordination/technical-delegation
  - architecture/architecture-review
  - risk/risk-assessment
autonomy: semi
escalation_rules:
  - condition: budget_impact > 10000
    action: escalate_to_ceo
  - condition: external_commitment
    action: escalate_to_ceo
```

**Updated Persona struct:**
```go
type Persona struct {
    Name          string   // "cto"
    Role          string   // "Chief Technology Officer"
    Skills        []string // References to skill directories
    AutonomyLevel string
    EscalationRules []EscalationRule

    // Optional: backward compatibility
    LegacyFields map[string]interface{}
}

type Skill struct {
    Name          string
    Description   string
    License       string
    Compatibility string
    Metadata      map[string]string
    Instructions  string   // Markdown body
    ScriptsDir    string   // Optional
    ReferencesDir string   // Optional
    AssetsDir     string   // Optional
}
```

## Benefits of Migration

### For Loom Users
- ✅ **Better reusability**: Share personas/skills across projects
- ✅ **Faster onboarding**: Standard format reduces learning curve
- ✅ **Composability**: Mix and match skills to create custom agents
- ✅ **Community sharing**: Publish skills to public registries

### For Ecosystem
- ✅ **Interoperability**: Loom agents work with other agent systems
- ✅ **Standard tooling**: Use `skills-ref` validator and other standard tools
- ✅ **Discovery**: Skills can be indexed and searched in skill registries
- ✅ **Best practices**: Learn from skills created by broader community

### For Development
- ✅ **Simpler schema**: Less rigid, easier to extend
- ✅ **Progressive loading**: Load only what's needed (metadata → instructions → resources)
- ✅ **Better testing**: Validate with standard tools
- ✅ **Version control**: Track skill versions independently

## Implementation Checklist

- [ ] Phase 1: Convert existing personas to SKILL.md format
  - [ ] Add YAML frontmatter to all PERSONA.md files
  - [ ] Merge AI_START_HERE.md into SKILL.md body
  - [ ] Move support docs to references/ subdirectories
  - [ ] Update persona loader to parse SKILL.md frontmatter
  - [ ] Add backward compatibility for old format

- [ ] Phase 2: Extract composable skills
  - [ ] Identify common capabilities across personas
  - [ ] Create skill directories (coordination/, engineering/, etc.)
  - [ ] Write SKILL.md for each extracted skill
  - [ ] Add scripts/ and references/ as needed
  - [ ] Validate with `skills-ref validate`

- [ ] Phase 3: Enable skill composition
  - [ ] Add config.yaml for persona skill composition
  - [ ] Update Persona struct to reference skills
  - [ ] Implement skill loading and composition logic
  - [ ] Update dispatcher to use composed skills
  - [ ] Add skill registry/discovery mechanism

- [ ] Testing & Validation
  - [ ] Test persona backward compatibility
  - [ ] Validate all skills with `skills-ref`
  - [ ] Test skill composition with sample personas
  - [ ] Performance test progressive loading
  - [ ] Integration test with full agent lifecycle

- [ ] Documentation
  - [ ] Update docs/USER_GUIDE.md with skill info
  - [ ] Document skill creation guidelines
  - [ ] Add examples of composable agents
  - [ ] Migration guide for custom personas
  - [ ] API documentation for skill system

## Timeline Estimate

- **Phase 1** (Format conversion): 1-2 weeks
  - Straightforward mechanical transformation
  - Most work is regex/templating + validation

- **Phase 2** (Skill extraction): 2-3 weeks
  - Requires careful analysis of persona overlap
  - Writing clear, reusable skill docs
  - Testing extracted skills

- **Phase 3** (Composition system): 3-4 weeks
  - New architecture for loading/composing skills
  - Integration with existing agent system
  - Comprehensive testing

**Total: 6-9 weeks** for complete migration

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking changes to existing personas | High | Maintain backward compatibility layer |
| Skill composition complexity | Medium | Start with simple composition, add features incrementally |
| Performance degradation | Low | Implement caching for loaded skills |
| Community adoption lag | Low | Document benefits clearly, provide examples |

## Next Steps

1. **Validate approach**: Review this plan with team
2. **Prototype Phase 1**: Convert 1-2 personas to SKILL.md format
3. **Test compatibility**: Ensure converted personas work with existing system
4. **Plan rollout**: Decide on migration schedule and communication plan
5. **Execute phases**: Implement incrementally with testing at each phase

## References

- Agent Skills Specification: https://agentskills.io/specification
- Agent Skills Registry: https://agentskills.io/registry
- skills-ref validator: https://github.com/agentskills/agentskills/tree/main/skills-ref
- Progressive Disclosure Pattern: https://agentskills.io/best-practices#progressive-disclosure
