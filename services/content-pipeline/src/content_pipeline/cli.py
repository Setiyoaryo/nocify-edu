"""Command-line entry point for the content pipeline.

    content-pipeline migrate
    content-pipeline sources seed-defaults
    content-pipeline sources add --code ... --name ... --base-url ... --tier 1
    content-pipeline sources list
    content-pipeline sources status --code ... --status active --enable
    content-pipeline crawl [--source CODE] [--limit N] [--dry-run]
    content-pipeline extract [--source CODE] [--limit N] [--force]
    content-pipeline map [--source CODE] [--limit N] [--force]
    content-pipeline materials [--source CODE] [--limit N] [--force]
    content-pipeline questions [--source CODE] [--limit N] [--force]
"""

from __future__ import annotations

import argparse
import logging
import sys
from collections.abc import Sequence

from content_pipeline import __version__
from content_pipeline.config import Settings
from content_pipeline.crawler import CrawlRunner, CrawlSummary
from content_pipeline.db import Database
from content_pipeline.extract import Extractor
from content_pipeline.mapping import Mapper
from content_pipeline.materials import MaterialBuilder
from content_pipeline.migrate import migrate
from content_pipeline.questions import QuestionBuilder
from content_pipeline.registry import (
    add_source,
    get_source,
    list_sources,
    seed_defaults,
    set_status,
)
from content_pipeline.storage import RawArchive
from content_pipeline.validation import QuestionSchema

SOURCE_TYPES = ("official_curriculum", "institution", "licensed", "public", "community")
STATUSES = ("discovered", "verified", "approved", "active", "paused", "retired")


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="content-pipeline", description=__doc__)
    parser.add_argument("--version", action="version", version=__version__)
    parser.add_argument("-v", "--verbose", action="store_true", help="debug logging")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("migrate", help="apply content_staging migrations")

    sources = sub.add_parser("sources", help="manage the source registry")
    sources_sub = sources.add_subparsers(dest="sources_command", required=True)
    sources_sub.add_parser("list", help="list registered sources")
    sources_sub.add_parser("seed-defaults", help="insert the official allowlist (disabled)")

    add = sources_sub.add_parser("add", help="add or update a source")
    add.add_argument("--code", required=True)
    add.add_argument("--name", required=True)
    add.add_argument("--base-url", required=True)
    add.add_argument("--source-type", required=True, choices=SOURCE_TYPES)
    add.add_argument("--tier", required=True, type=int, choices=range(1, 6))
    add.add_argument("--publisher", default="")
    add.add_argument("--license-type", default="unknown")
    add.add_argument("--license-url", default="")
    add.add_argument("--terms-url", default="")
    add.add_argument("--max-pages", type=int, default=50)
    add.add_argument("--rate-limit-ms", type=int, default=1000)

    status = sources_sub.add_parser("status", help="change a source's governance status")
    status.add_argument("--code", required=True)
    status.add_argument("--status", required=True, choices=STATUSES)
    toggle = status.add_mutually_exclusive_group()
    toggle.add_argument("--enable", dest="enabled", action="store_true", default=None)
    toggle.add_argument("--disable", dest="enabled", action="store_false", default=None)

    crawl = sub.add_parser("crawl", help="run the crawler for active sources")
    crawl.add_argument("--source", action="append", dest="sources", metavar="CODE")
    crawl.add_argument("--limit", type=int, help="max pages per source")
    crawl.add_argument("--dry-run", action="store_true", help="discover links without fetching or writing")

    extract = sub.add_parser("extract", help="extract text and chunks from archived documents")
    extract.add_argument("--source", action="append", dest="sources", metavar="CODE")
    extract.add_argument("--limit", type=int, help="max documents to process")
    extract.add_argument("--force", action="store_true", help="re-extract documents already processed")

    map_cmd = sub.add_parser("map", help="map chunks onto the curriculum (CP and skills)")
    map_cmd.add_argument("--source", action="append", dest="sources", metavar="CODE")
    map_cmd.add_argument("--limit", type=int, help="max chunks to map")
    map_cmd.add_argument("--force", action="store_true", help="re-map chunks already mapped")

    materials = sub.add_parser("materials", help="build material candidates from mapped chunks")
    materials.add_argument("--source", action="append", dest="sources", metavar="CODE")
    materials.add_argument("--limit", type=int, help="max candidates to build")
    materials.add_argument("--force", action="store_true", help="rebuild existing candidates")

    questions = sub.add_parser("questions", help="build question candidates from mapped chunks")
    questions.add_argument("--source", action="append", dest="sources", metavar="CODE")
    questions.add_argument("--limit", type=int, help="max candidates to build")
    questions.add_argument("--force", action="store_true", help="rebuild existing candidates")

    return parser


def _settings() -> Settings:
    try:
        return Settings.from_env()
    except ValueError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(2) from exc


def _cmd_migrate(db: Database) -> int:
    result = migrate(db)
    for name in result.applied:
        print(f"applied {name}")
    for name in result.skipped:
        print(f"up-to-date {name}")
    if not result.applied:
        print("no migrations to apply")
    return 0


def _cmd_sources(db: Database, args: argparse.Namespace) -> int:
    if args.sources_command == "list":
        for source in list_sources(db):
            state = "enabled" if source.enabled else "disabled"
            print(f"{source.code:28} {source.status:10} {state:8} tier={source.tier} {source.base_url}")
        return 0

    if args.sources_command == "seed-defaults":
        created = seed_defaults(db)
        print(f"seeded {created} new source(s) (disabled until verified)")
        return 0

    if args.sources_command == "add":
        source = add_source(
            db,
            code=args.code,
            name=args.name,
            base_url=args.base_url,
            source_type=args.source_type,
            tier=args.tier,
            publisher=args.publisher,
            license_type=args.license_type,
            license_url=args.license_url,
            terms_url=args.terms_url,
            max_pages=args.max_pages,
            rate_limit_ms=args.rate_limit_ms,
        )
        print(f"{source.code}: {source.status} ({'enabled' if source.enabled else 'disabled'})")
        return 0

    if args.sources_command == "status":
        updated = set_status(db, args.code, args.status, enabled=args.enabled)
        if updated is None:
            print(f"error: unknown source {args.code!r}", file=sys.stderr)
            return 1
        print(f"{updated.code}: {updated.status} ({'enabled' if updated.enabled else 'disabled'})")
        return 0

    return 2


def _print_summary(summary: CrawlSummary) -> None:
    print(
        f"{summary.source_code}: {summary.status} "
        f"discovered={summary.pages_discovered} fetched={summary.pages_fetched} "
        f"changed={summary.pages_changed} errors={summary.errors}"
        + (f" ({summary.message})" if summary.message else "")
    )


def _cmd_crawl(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    runner = CrawlRunner(db, settings)

    if args.dry_run:
        if not args.sources:
            print("error: --dry-run requires --source", file=sys.stderr)
            return 2
        for code in args.sources:
            source = get_source(db, code)
            if source is None:
                print(f"error: unknown source {code!r}", file=sys.stderr)
                return 1
            links = runner.preview(source, limit=args.limit)
            print(f"{code}: {len(links)} link(s) discovered")
            for link in links:
                print(f"  {link}")
        return 0

    summaries = runner.run(codes=args.sources, limit=args.limit)
    if not summaries:
        print("no active sources to crawl (activate one with: sources status --code X --status active --enable)")
        return 0
    for summary in summaries:
        _print_summary(summary)
    return 0


def _cmd_extract(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    archive = RawArchive(settings)
    if not archive.enabled:
        print("error: extraction needs the raw archive; set S3_BUCKET", file=sys.stderr)
        return 2
    summaries = Extractor(db, archive).run(codes=args.sources, limit=args.limit, force=args.force)
    if not summaries:
        print("nothing to extract (crawl first, or use --force)")
        return 0
    for summary in summaries:
        print(
            f"{summary.source_code}: documents={summary.documents} chunks={summary.chunks} "
            f"needs_ocr={summary.needs_ocr} skipped={summary.skipped} failed={summary.failed}"
        )
    return 0


def _cmd_map(db: Database, args: argparse.Namespace) -> int:
    summaries = Mapper(db).run(codes=args.sources, limit=args.limit, force=args.force)
    if not summaries:
        print("nothing to map (extract first, or use --force)")
        return 0
    for summary in summaries:
        print(
            f"{summary.source_code}: mapped={summary.mapped} unmapped={summary.unmapped} "
            f"high={summary.high} review={summary.review} manual={summary.manual}"
        )
    return 0


def _cmd_materials(db: Database, args: argparse.Namespace) -> int:
    summaries = MaterialBuilder(db).run(codes=args.sources, limit=args.limit, force=args.force)
    if not summaries:
        print("nothing to build (map first, or use --force)")
        return 0
    for summary in summaries:
        print(
            f"{summary.source_code}: generated={summary.generated} drafts={summary.drafts} "
            f"needs_review={summary.needs_review} duplicates={summary.duplicates}"
        )
    return 0


def _cmd_questions(db: Database, args: argparse.Namespace) -> int:
    try:
        schema = QuestionSchema.load()
    except FileNotFoundError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    summaries = QuestionBuilder(db, schema).run(codes=args.sources, limit=args.limit, force=args.force)
    if not summaries:
        print("nothing to build (map first, or use --force)")
        return 0
    for summary in summaries:
        print(
            f"{summary.source_code}: generated={summary.generated} drafts={summary.drafts} "
            f"needs_review={summary.needs_review} duplicates={summary.duplicates} "
            f"skipped={summary.skipped} invalid={summary.invalid}"
        )
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    args = _build_parser().parse_args(argv)
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    settings = _settings()
    with Database(settings.database_url) as db:
        if args.command == "migrate":
            return _cmd_migrate(db)
        if args.command == "sources":
            return _cmd_sources(db, args)
        if args.command == "crawl":
            return _cmd_crawl(db, settings, args)
        if args.command == "extract":
            return _cmd_extract(db, settings, args)
        if args.command == "map":
            return _cmd_map(db, args)
        if args.command == "materials":
            return _cmd_materials(db, args)
        if args.command == "questions":
            return _cmd_questions(db, args)
    return 2


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
