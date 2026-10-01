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


@dataclass(frozen=True)
class SourceDocumentRef:
    """A canonical curriculum source document the watcher can poll (M7)."""

    id: str  # uuid, copied by value
    code: str
    title: str
    source_url: str
    document_hash: str
    curriculum_version_id: int | None


@dataclass(frozen=True)
class CpRef:
    id: int
    code: str
    statement: str


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


def load_active_version_id(db: Database) -> int | None:
    """The active curriculum version id, or None when none is active (M7 fallback)."""
    with db.transaction() as cur:
        cur.execute("SELECT id FROM curriculum_versions WHERE is_active ORDER BY id LIMIT 1")
        row = cur.fetchone()
    return int(row[0]) if row is not None else None


_SOURCE_DOCUMENTS_SQL = """
    SELECT d.id::text, d.code, d.title, d.source_url, d.document_hash,
           (SELECT cv.id FROM curriculum_versions cv
            WHERE cv.source_document_id = d.id
            ORDER BY cv.is_active DESC, cv.id LIMIT 1)
    FROM curriculum_source_documents d
    ORDER BY d.code
"""


def load_source_documents(db: Database) -> tuple[SourceDocumentRef, ...]:
    """Load the canonical curriculum source documents the watcher may poll (M7)."""
    with db.transaction() as cur:
        cur.execute(_SOURCE_DOCUMENTS_SQL)
        rows = cur.fetchall()
    return tuple(
        SourceDocumentRef(
            id=str(r[0]),
            code=str(r[1]),
            title=str(r[2]),
            source_url=str(r[3]),
            document_hash=str(r[4]),
            curriculum_version_id=int(r[5]) if r[5] is not None else None,
        )
        for r in rows
    )


def load_cp_statements(db: Database, curriculum_version_id: int) -> dict[str, CpRef]:
    """Load a curriculum version's CP statements keyed by code (M7 diff baseline)."""
    with db.transaction() as cur:
        cur.execute(
            "SELECT id, code, statement FROM curriculum_cp "
            "WHERE curriculum_version_id = %s ORDER BY sort, id",
            (curriculum_version_id,),
        )
        rows = cur.fetchall()
    return {str(r[1]): CpRef(id=int(r[0]), code=str(r[1]), statement=str(r[2])) for r in rows}
