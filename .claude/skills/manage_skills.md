---
name: manage-skills
description: A meta-skill to learn and adapt by documenting recurring patterns, API quirks, or project rules in .claude/skills/ or docs/.
---
# Skill: Manage and Evolve Context/Skills

## Description
A meta-skill that allows you to learn and adapt. When a new recurring pattern, a specific KIS API quirk, or a new user preference is discovered, you must document it so you don't forget it in future sessions.

## Trigger
Execute this skill when you resolve a complex or repetitive bug, discover a new project rule, or when the user explicitly asks you to "remember this rule" or "create a new skill."

## Instructions
1. **Identify the Learning:** Determine if the new information is a project rule, an API quirk, or a reusable workflow.
2. **Determine Target:**
   - If it's a new executable workflow or rule for you (the agent), create or update a `.md` file in `.claude/skills/`.
   - If it's a KIS API quirk or known bug pattern, add it to the `## KIS API 알려진 Quirks` section of `.claude/skills/implement_kis_feature.md`.
   - If it's factual reference data (like API specs), update the corresponding file in `docs/kis-api/`.
3. **Document:** Write clear, concise instructions or facts using the existing skill/doc templates.
4. **Report:** Notify the user in Korean: "새로운 규칙/스킬이 `.claude/skills/` (또는 `docs/`)에 성공적으로 업데이트되어 향후 작업에 반영됩니다."