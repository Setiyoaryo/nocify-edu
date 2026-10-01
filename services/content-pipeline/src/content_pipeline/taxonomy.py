"""Read-only view of the canonical curriculum taxonomy (§16, §19).

The pipeline never writes canonical tables. This module reads the active
curriculum version, its skills and their Capaian Pembelajaran so chunks can be
mapped onto it.
"""

from __future__ import annotations

from dataclasses import dataclass

from content_pipeline.db import Database


@dataclass(frozen=True)
class SkillRef:
    id: int
    code: str
    name: str
    description: str
    chapter_name: str
    subject_code: str
    grade_code: str
    cp_id: int | None
    cp_code: str
    phase_code: str


@dataclass(frozen=True)
class Taxonomy:
    curriculum_version_id: int
    skills: tuple[SkillRef, ...]


_SKILLS_SQL = """
    SELECT k.id, k.code, k.name, k.description, c.name,
           s.code, g.code,
           cp.id, COALESCE(cp.code, ''), COALESCE(ph.code, '')
    FROM skills k
    JOIN chapters c ON c.id = k.chapter_id
    JOIN subjects s ON s.id = c.subject_id
    JOIN grades g ON g.id = c.grade_id
    JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
    LEFT JOIN curriculum_cp cp ON cp.id = k.cp_id
    LEFT JOIN phases ph ON ph.id = cp.phase_id
    ORDER BY c.sort, k.sort
"""


def load_taxonomy(db: Database) -> Taxonomy:
    """Load the active curriculum's skills. Returns an empty taxonomy when none is active."""
    with db.transaction() as cur:
        cur.execute("SELECT id FROM curriculum_versions WHERE is_active ORDER BY id LIMIT 1")
        row = cur.fetchone()
        if row is None:
            return Taxonomy(curriculum_version_id=0, skills=())
        version_id = int(row[0])

        cur.execute(_SKILLS_SQL)
        skills = tuple(
            SkillRef(
                id=int(r[0]),
                code=str(r[1]),
                name=str(r[2]),
                description=str(r[3]),
                chapter_name=str(r[4]),
                subject_code=str(r[5]),
                grade_code=str(r[6]),
                cp_id=int(r[7]) if r[7] is not None else None,
                cp_code=str(r[8]),
                phase_code=str(r[9]),
            )
            for r in cur.fetchall()
        )
    return Taxonomy(curriculum_version_id=version_id, skills=skills)
