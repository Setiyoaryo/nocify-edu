"""Command-line entry point for the content pipeline.

    content-pipeline migrate
    content-pipeline sources seed-defaults
    content-pipeline sources add --code ... --name ... --base-url ... --tier 1
    content-pipeline sources list
    content-pipeline sources status --code ... --status active --enable
    content-pipeline crawl [--source CODE] [--limit N] [--dry-run]
    content-pipeline extract [--source CODE] [--limit N] [--force]
    content-pipeline embed [--source CODE] [--limit N] [--force] [--kind chunk|skill|all]
    content-pipeline map [--source CODE] [--limit N] [--force] [--semantic]
    content-pipeline dedup [--source CODE] [--limit N] [--kind material|question|all]
    content-pipeline materials [--source CODE] [--limit N] [--force]
    content-pipeline questions [--source CODE] [--limit N] [--force]
    content-pipeline generate [--source CODE] [--limit N] [--template T] [--force]
                              [--dry-run] [--cap N] [--allow-unknown-license]
    content-pipeline watch [--document-code CODE] [--limit N] [--dry-run] [--force] [--due] [--from-file PATH]
    content-pipeline watch register --document-code CODE --url URL [--hash H] [--frequency F]
    content-pipeline changes [--document-code CODE] [--status S] [--limit N] [--json]
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from collections.abc import Sequence

from content_pipeline import __version__
from content_pipeline.assisted import COMPREHENSION, SCENARIO, AssistedGenerator
from content_pipeline.config import Settings
from content_pipeline.crawler import CrawlRunner, CrawlSummary
from content_pipeline.db import Database
from content_pipeline.dedup import MATERIAL, QUESTION, Deduplicator, MissingEmbeddingsError
from content_pipeline.embeddings import OWNER_CHUNK, OWNER_SKILL, EmbeddingBuilder, EmbeddingStore
from content_pipeline.extract import Extractor
from content_pipeline.mapping import Mapper
from content_pipeline.materials import MaterialBuilder
from content_pipeline.migrate import migrate
from content_pipeline.provider import Provider, build_provider
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
from content_pipeline.watch import (
    CurriculumWatcher,
    list_change_events,
    register_document,
)

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

    embed = sub.add_parser("embed", help="cache embeddings for chunks and skills (M6)")
    embed.add_argument("--source", action="append", dest="sources", metavar="CODE")
    embed.add_argument("--limit", type=int, help="max chunks to embed")
    embed.add_argument("--force", action="store_true", help="re-embed even when the text is unchanged")
    embed.add_argument("--kind", choices=("chunk", "skill", "all"), default="all")

    map_cmd = sub.add_parser("map", help="map chunks onto the curriculum (CP and skills)")
    map_cmd.add_argument("--source", action="append", dest="sources", metavar="CODE")
    map_cmd.add_argument("--limit", type=int, help="max chunks to map")
    map_cmd.add_argument("--force", action="store_true", help="re-map chunks already mapped")
    map_cmd.add_argument("--semantic", action="store_true", help="blend embedding similarity (run `embed` first)")

    dedup = sub.add_parser("dedup", help="detect near-duplicate chunks and flag candidates (§29)")
    dedup.add_argument("--source", action="append", dest="sources", metavar="CODE")
    dedup.add_argument("--limit", type=int, help="max chunks to compare")
    dedup.add_argument("--kind", choices=("material", "question", "all"), default="all")
    dedup.add_argument("--model", help="embedding model to compare (defaults to the configured provider)")

    materials = sub.add_parser("materials", help="build material candidates from mapped chunks")
    materials.add_argument("--source", action="append", dest="sources", metavar="CODE")
    materials.add_argument("--limit", type=int, help="max candidates to build")
    materials.add_argument("--force", action="store_true", help="rebuild existing candidates")

    questions = sub.add_parser("questions", help="build question candidates from mapped chunks")
    questions.add_argument("--source", action="append", dest="sources", metavar="CODE")
    questions.add_argument("--limit", type=int, help="max candidates to build")
    questions.add_argument("--force", action="store_true", help="rebuild existing candidates")

    generate = sub.add_parser(
        "generate", help="AI-assisted scenario/comprehension questions (M6, §23)"
    )
    generate.add_argument("--source", action="append", dest="sources", metavar="CODE")
    generate.add_argument("--limit", type=int, help="max chunks to consider per template")
    generate.add_argument(
        "--template", action="append", dest="templates", choices=(SCENARIO, COMPREHENSION),
        help="assisted template (default: scenario and comprehension)",
    )
    generate.add_argument("--force", action="store_true", help="re-generate even when cached (re-bills)")
    generate.add_argument("--dry-run", action="store_true", help="report without provider calls or writes")
    generate.add_argument("--cap", type=int, help="max provider calls this run")
    generate.add_argument(
        "--allow-unknown-license", action="store_true",
        help="allow generation from sources with an unknown licence (§25)",
    )

    watch = sub.add_parser("watch", help="detect curriculum source-document changes (§44)")
    watch.add_argument("--document-code", action="append", dest="documents", metavar="CODE")
    watch.add_argument("--limit", type=int, help="max documents to check")
    watch.add_argument("--dry-run", action="store_true", help="fetch and diff without writing anything")
    watch.add_argument("--force", action="store_true", help="re-analyse even when the hash is unchanged")
    watch.add_argument("--due", action="store_true", help="only check documents whose frequency is due")
    watch.add_argument("--from-file", metavar="PATH", help="read the body from a local file (offline/dev)")
    watch_sub = watch.add_subparsers(dest="watch_command")
    register = watch_sub.add_parser("register", help="register a watched document URL (staging override)")
    register.add_argument("--document-code", required=True, dest="register_document")
    register.add_argument("--url", required=True)
    register.add_argument("--hash", default="", help="seed the baseline hash (defaults to canonical)")
    register.add_argument(
        "--frequency", choices=("daily", "weekly", "monthly", "manual"), default="weekly"
    )

    changes = sub.add_parser("changes", help="list detected curriculum changes (§44)")
    changes.add_argument("--document-code", action="append", dest="documents", metavar="CODE")
    changes.add_argument("--status", choices=("detected", "analyzed", "failed", "superseded"))
    changes.add_argument("--limit", type=int, default=50)
    changes.add_argument("--json", action="store_true", help="print JSON")

    return parser


def _settings() -> Settings:
    try:
        return Settings.from_env()
    except ValueError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(2) from exc


def _resolve_provider(settings: Settings) -> tuple[Provider | None, str]:
    """Return the configured provider, or (None, reason) when AI is unavailable."""
    try:
        provider = build_provider(settings)
    except ValueError as exc:
        return None, str(exc)
    if provider is None:
        return None, "AI provider not configured; set AI_PROVIDER (mock or openai)"
    return provider, ""


def _embed_kinds(value: str) -> tuple[str, ...]:
    return (OWNER_CHUNK, OWNER_SKILL) if value == "all" else (value,)


def _dedup_kinds(value: str) -> tuple[str, ...]:
    return (MATERIAL, QUESTION) if value == "all" else (value,)


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


def _cmd_embed(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    provider, reason = _resolve_provider(settings)
    if provider is None:
        print(f"error: {reason}", file=sys.stderr)
        return 2
    try:
        report = EmbeddingBuilder(db, provider, batch_size=settings.ai_embedding_batch_size).run(
            codes=args.sources, limit=args.limit, force=args.force, kinds=_embed_kinds(args.kind)
        )
    finally:
        provider.close()
    for summary in report.sources:
        print(
            f"{summary.source_code}: chunks_embedded={summary.chunks_embedded} "
            f"chunks_cached={summary.chunks_cached}"
        )
    print(f"skills: embedded={report.skills_embedded} cached={report.skills_cached}")
    if not report.sources and not report.skills_embedded and not report.skills_cached:
        print("nothing to embed (extract first, or use --force)")
    return 0


def _cmd_map(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    if not args.semantic:
        summaries = Mapper(db).run(codes=args.sources, limit=args.limit, force=args.force)
    else:
        provider, reason = _resolve_provider(settings)
        if provider is None:
            print(f"error: {reason}", file=sys.stderr)
            return 2
        try:
            mapper = Mapper(
                db, embeddings=EmbeddingStore(db), semantic_model=provider.embedding_model
            )
            summaries = mapper.run(
                codes=args.sources, limit=args.limit, force=args.force, semantic=True
            )
        finally:
            provider.close()
    if not summaries:
        print("nothing to map (extract first, or use --force)")
        return 0
    for summary in summaries:
        print(
            f"{summary.source_code}: mapped={summary.mapped} unmapped={summary.unmapped} "
            f"high={summary.high} review={summary.review} manual={summary.manual}"
        )
    return 0


def _cmd_dedup(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    model = args.model or settings.ai_embedding_model
    if not model:
        provider, _ = _resolve_provider(settings)
        if provider is not None:
            model = provider.embedding_model
            provider.close()
    if not model:
        print("error: set AI_EMBEDDING_MODEL (or --model) to the model used by `embed`", file=sys.stderr)
        return 2
    try:
        report = Deduplicator(db, model=model).run(
            kinds=_dedup_kinds(args.kind), codes=args.sources, limit=args.limit
        )
    except MissingEmbeddingsError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    print(
        f"chunks={report.chunks_checked} exact={report.exact} "
        f"near_duplicate={report.near_duplicate} related={report.related} unique={report.unique} "
        f"candidates_flagged={report.candidates_flagged}"
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


def _cmd_generate(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    provider, reason = _resolve_provider(settings)
    if provider is None:
        print(f"error: {reason}", file=sys.stderr)
        return 2
    try:
        schema = QuestionSchema.load()
    except FileNotFoundError as exc:
        provider.close()
        print(f"error: {exc}", file=sys.stderr)
        return 2
    templates = tuple(args.templates) if args.templates else (SCENARIO, COMPREHENSION)
    try:
        report = AssistedGenerator(
            db,
            provider,
            schema,
            prompt_version=settings.ai_prompt_version,
            cap=settings.ai_generation_cap,
            max_output_tokens=settings.ai_max_output_tokens,
        ).run(
            codes=args.sources,
            limit=args.limit,
            templates=templates,
            force=args.force,
            dry_run=args.dry_run,
            cap=args.cap,
            allow_unknown_license=args.allow_unknown_license,
        )
    finally:
        provider.close()
    print(
        f"{'dry-run ' if report.dry_run else ''}"
        f"generated={report.generated} cached={report.cached} rejected={report.rejected} "
        f"failed={report.failed} license_skipped={report.license_skipped}"
        + (f" would_generate={report.would_generate}" if report.dry_run else "")
        + (f" aborted={report.aborted}" if report.aborted else "")
        + f" tokens={report.total_tokens}"
    )
    return 0


def _cmd_watch(db: Database, settings: Settings, args: argparse.Namespace) -> int:
    if args.watch_command == "register":
        if args.from_file:
            print("error: --from-file is not valid with register", file=sys.stderr)
            return 2
        code = register_document(
            db, code=args.register_document, url=args.url, baseline_hash=args.hash,
            frequency=args.frequency,
        )
        if code is None:
            print(f"error: unknown curriculum source document {args.register_document!r}", file=sys.stderr)
            return 1
        print(f"{code}: watching {args.url} (frequency={args.frequency})")
        return 0

    if args.from_file and not args.documents:
        print("error: --from-file requires --document-code", file=sys.stderr)
        return 2
    if args.from_file and len(args.documents) != 1:
        print("error: --from-file accepts exactly one --document-code", file=sys.stderr)
        return 2

    report = CurriculumWatcher(db, settings).run(
        codes=args.documents, limit=args.limit, dry_run=args.dry_run, force=args.force,
        due=args.due, from_file=args.from_file,
    )
    if report.locked:
        print("another watch run is in progress; nothing done")
        return 0
    if not report.summaries:
        print(
            "no watched curriculum documents "
            "(register one with: watch register --document-code X --url Y)"
        )
        return 0
    prefix = "dry-run " if report.dry_run else ""
    for summary in report.summaries:
        print(
            f"{prefix}{summary.document_code}: {summary.status} "
            f"added={summary.cp_added} changed={summary.cp_changed} removed={summary.cp_removed} "
            f"affected_questions={summary.affected_questions} ({summary.message})"
        )
    return 0


def _cmd_changes(db: Database, args: argparse.Namespace) -> int:
    events = list_change_events(db, codes=args.documents, status=args.status, limit=args.limit)
    if args.json:
        print(json.dumps([event.__dict__ for event in events], default=str, ensure_ascii=False))
        return 0
    if not events:
        print("no curriculum changes detected")
        return 0
    for event in events:
        print(
            f"#{event.id} {event.document_code} {event.status} "
            f"added={event.cp_added} changed={event.cp_changed} removed={event.cp_removed} "
            f"{event.created_at:%Y-%m-%d %H:%M} {event.url}"
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
        if args.command == "embed":
            return _cmd_embed(db, settings, args)
        if args.command == "map":
            return _cmd_map(db, settings, args)
        if args.command == "dedup":
            return _cmd_dedup(db, settings, args)
        if args.command == "materials":
            return _cmd_materials(db, args)
        if args.command == "questions":
            return _cmd_questions(db, args)
        if args.command == "generate":
            return _cmd_generate(db, settings, args)
        if args.command == "watch":
            return _cmd_watch(db, settings, args)
        if args.command == "changes":
            return _cmd_changes(db, args)
    return 2


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
