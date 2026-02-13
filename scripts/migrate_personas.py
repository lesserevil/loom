#!/usr/bin/env python3
"""
Migrate Loom personas to Agent Skills format.

Converts PERSONA.md + AI_START_HERE.md to standardized SKILL.md with YAML frontmatter.
"""

import os
import re
import yaml
from pathlib import Path
from typing import Dict, Optional, List


def extract_metadata_from_persona(content: str, persona_name: str) -> Dict:
    """Extract metadata from PERSONA.md content."""
    metadata = {
        'name': persona_name,
        'description': '',
        'metadata': {}
    }

    # Extract first paragraph as description
    lines = content.strip().split('\n')
    desc_lines = []
    in_header = True

    for line in lines:
        line = line.strip()
        if not line:
            if desc_lines:
                break
            continue
        if line.startswith('#'):
            in_header = True
            # Extract role from first header
            if not metadata['metadata'].get('role'):
                role = line.lstrip('#').strip()
                metadata['metadata']['role'] = role
            continue
        if in_header and line:
            desc_lines.append(line)
            in_header = False
        elif not in_header and not line.startswith('#') and not line.startswith('-'):
            desc_lines.append(line)
        else:
            break

    metadata['description'] = ' '.join(desc_lines)[:500]  # Max 500 chars

    # Extract autonomy level
    autonomy_match = re.search(r'Autonomy Level[:\s]+([a-z]+)', content, re.IGNORECASE)
    if autonomy_match:
        metadata['metadata']['autonomy_level'] = autonomy_match.group(1).lower()

    # Extract specialties from last line
    specialties_match = re.search(r'Specialties?:\s*(.+)$', content, re.MULTILINE | re.IGNORECASE)
    if specialties_match:
        specialties_text = specialties_match.group(1)
        specialties = [s.strip() for s in specialties_text.split(',')]
        metadata['metadata']['specialties'] = specialties

    return metadata


def create_skill_md(persona_dir: Path) -> bool:
    """Convert a persona directory to SKILL.md format."""
    persona_name = persona_dir.name

    # Read existing files
    persona_md_path = persona_dir / 'PERSONA.md'
    ai_start_path = persona_dir / 'AI_START_HERE.md'

    if not persona_md_path.exists():
        print(f"  ⚠️  No PERSONA.md found in {persona_dir}")
        return False

    persona_content = persona_md_path.read_text()
    ai_start_content = ai_start_path.read_text() if ai_start_path.exists() else ""

    # Extract metadata
    metadata = extract_metadata_from_persona(persona_content, persona_name)

    # Add standard fields
    metadata['license'] = 'Proprietary'
    metadata['compatibility'] = 'Designed for Loom'
    metadata['metadata']['author'] = 'loom'
    metadata['metadata']['version'] = '1.0'

    # Build SKILL.md content
    frontmatter = yaml.dump(metadata, default_flow_style=False, sort_keys=False)

    # Combine body content
    body_parts = []

    # Add AI_START_HERE.md content first (quick start)
    if ai_start_content:
        # Remove the first header if it exists
        ai_start_content = re.sub(r'^#[^#\n]+\n+', '', ai_start_content.strip())
        if ai_start_content:
            body_parts.append("# Quick Start\n\n" + ai_start_content)

    # Add PERSONA.md content (detailed instructions)
    if persona_content:
        # Remove any front matter or metadata sections
        persona_content = re.sub(r'^---\n.*?\n---\n', '', persona_content, flags=re.DOTALL)
        body_parts.append(persona_content.strip())

    body = '\n\n---\n\n'.join(body_parts)

    # Create SKILL.md
    skill_md_content = f"---\n{frontmatter}---\n\n{body}\n"

    skill_md_path = persona_dir / 'SKILL.md'
    skill_md_path.write_text(skill_md_content)

    print(f"  ✅ Created SKILL.md")

    # Move additional files to references/
    ref_dir = persona_dir / 'references'
    additional_files = [
        f for f in persona_dir.iterdir()
        if f.is_file() and f.name not in ['SKILL.md', 'PERSONA.md', 'AI_START_HERE.md']
    ]

    if additional_files:
        ref_dir.mkdir(exist_ok=True)
        for file in additional_files:
            dest = ref_dir / file.name
            file.rename(dest)
            print(f"  📁 Moved {file.name} → references/")

    # Delete old files
    if persona_md_path.exists():
        persona_md_path.unlink()
        print(f"  🗑️  Deleted PERSONA.md")

    if ai_start_path.exists():
        ai_start_path.unlink()
        print(f"  🗑️  Deleted AI_START_HERE.md")

    return True


def main():
    """Convert all personas to Agent Skills format."""
    root = Path('personas')

    if not root.exists():
        print("❌ personas/ directory not found")
        return 1

    # Find all persona directories
    persona_dirs = []
    for org_dir in root.iterdir():
        if org_dir.is_dir() and org_dir.name not in ['templates']:
            for persona_dir in org_dir.iterdir():
                if persona_dir.is_dir():
                    persona_dirs.append(persona_dir)

    print(f"Found {len(persona_dirs)} personas to convert\n")

    success_count = 0
    for persona_dir in sorted(persona_dirs):
        print(f"Converting {persona_dir.relative_to(root)}...")
        if create_skill_md(persona_dir):
            success_count += 1
        print()

    print(f"\n✅ Successfully converted {success_count}/{len(persona_dirs)} personas")

    return 0


if __name__ == '__main__':
    exit(main())
