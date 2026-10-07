// Command eduplay is the single EduPlay backend binary (§18):
//
//	eduplay serve     HTTP API (runs migrations first)
//	eduplay worker    periodic maintenance jobs
//	eduplay migrate   apply migrations and exit
//	eduplay seed      load content/ (use --publish in development only)
//	eduplay user      create or promote a staff account
//	eduplay recompute rebuild skill_progress from raw attempts
//	eduplay recompute-attribution rebuild question creator attribution from raw attempts
//	eduplay publish-candidates  publish approved pipeline candidates to canonical
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/nocify/eduplay/api/internal/attribution"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/candidates"
	"github.com/nocify/eduplay/api/internal/config"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/media"
	"github.com/nocify/eduplay/api/internal/progress"
	"github.com/nocify/eduplay/api/internal/rewards"
	"github.com/nocify/eduplay/api/internal/seed"
	"github.com/nocify/eduplay/api/internal/server"
	"github.com/nocify/eduplay/api/internal/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: eduplay serve|worker|migrate|seed|user|recompute|recompute-attribution|publish-candidates")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		slog.Error("fatal", "cmd", os.Args[1], "err", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch cmd {
	case "migrate":
		return db.Migrate(ctx, pool)

	case "serve":
		if err := db.Migrate(ctx, pool); err != nil {
			return err
		}
		if err := rewards.LoadFreeAvatars(ctx, pool); err != nil {
			return err
		}
		var storage media.Storage = media.LocalStorage{Dir: cfg.StorageDir}
		if cfg.StorageDriver == "s3" {
			s3, err := media.NewS3(ctx, media.S3Config{Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket,
				AccessKey: cfg.StorageAccessKey, SecretKey: cfg.StorageSecretKey, UseSSL: cfg.S3UseSSL})
			if err != nil {
				return err
			}
			storage = s3
		}
		srv := &http.Server{Addr: cfg.HTTPAddr, Handler: server.New(cfg, pool, storage),
			ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
		go func() {
			<-ctx.Done()
			shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdown)
		}()
		slog.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil

	case "worker":
		slog.Info("worker started")
		worker.Run(ctx, pool, time.Minute)
		return nil

	case "seed":
		fs := flag.NewFlagSet("seed", flag.ExitOnError)
		content := fs.String("content", envOr("CONTENT_DIR", "../../content"), "content directory")
		assets := fs.String("assets", envOr("ASSETS_DIR", "../../assets"), "assets directory")
		publish := fs.Bool("publish", false, "publish seeded questions immediately (development only)")
		_ = fs.Parse(args)
		if *publish && cfg.Env != "development" {
			return errors.New("--publish is only allowed in development; seeded content must pass the review gate")
		}
		if err := db.Migrate(ctx, pool); err != nil {
			return err
		}
		return seed.Run(ctx, pool, seed.Options{ContentDir: *content, AssetsDir: *assets, Publish: *publish})

	case "user":
		fs := flag.NewFlagSet("user", flag.ExitOnError)
		email := fs.String("email", "", "email")
		password := fs.String("password", "", "password (min 8 chars)")
		name := fs.String("name", "Staff", "display name")
		role := fs.String("role", "admin", "parent|content_editor|reviewer|admin")
		_ = fs.Parse(args)
		if *email == "" || len(*password) < 8 {
			return errors.New("--email and --password (min 8 chars) are required")
		}
		hash, err := auth.HashPassword(*password)
		if err != nil {
			return err
		}
		var id uuid.UUID
		err = pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, display_name, role) VALUES (lower($1), $2, $3, $4)
			ON CONFLICT (lower(email)) DO UPDATE SET role = EXCLUDED.role, password_hash = EXCLUDED.password_hash RETURNING id`,
			*email, hash, *name, *role).Scan(&id)
		if err != nil {
			return err
		}
		slog.Info("user saved", "id", id, "role", *role)
		return nil

	case "publish-candidates":
		fs := flag.NewFlagSet("publish-candidates", flag.ExitOnError)
		limit := fs.Int("limit", 50, "rows per batch")
		all := fs.Bool("all", false, "drain the queue until empty")
		dry := fs.Bool("dry-run", false, "validate only, write nothing")
		retry := fs.Bool("retry-failed", false, "re-queue failed publications first")
		maxAttempts := fs.Int("max-attempts", 0, "with --retry-failed: only retry rows below this attempt count (0 = no limit)")
		_ = fs.Parse(args)
		if err := db.Migrate(ctx, pool); err != nil {
			return err
		}
		rep, err := candidates.Publish(ctx, pool, candidates.PublishOptions{
			Limit: *limit, All: *all, DryRun: *dry, RetryFailed: *retry, MaxAttempts: *maxAttempts,
		})
		if err != nil {
			return err
		}
		slog.Info("publish-candidates", "claimed", rep.Claimed, "published", rep.Published,
			"linked", rep.Linked, "failed", rep.Failed, "skipped", rep.Skipped,
			"remaining", rep.Remaining, "dry_run", rep.DryRun)
		for _, e := range rep.Errors {
			slog.Warn("publish-candidates", "error", e)
		}
		return nil

	case "recompute":
		rows, err := pool.Query(ctx, `SELECT id FROM children`)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if err := progress.RecomputeChild(ctx, pool, id); err != nil {
				return err
			}
		}
		slog.Info("recomputed skill progress", "children", len(ids))
		return nil

	case "recompute-attribution":
		fs := flag.NewFlagSet("recompute-attribution", flag.ExitOnError)
		creatorID := fs.String("creator", "", "rebuild only this creator's questions (uuid)")
		questionID := fs.String("question", "", "rebuild a single question (uuid)")
		_ = fs.Parse(args)
		switch {
		case *questionID != "":
			id, err := uuid.Parse(*questionID)
			if err != nil {
				return err
			}
			if err := attribution.RecomputeQuestions(ctx, pool, []uuid.UUID{id}); err != nil {
				return err
			}
			slog.Info("recomputed attribution", "questions", 1)
		case *creatorID != "":
			id, err := uuid.Parse(*creatorID)
			if err != nil {
				return err
			}
			rows, err := pool.Query(ctx, `SELECT id FROM questions WHERE creator_id = $1`, id)
			if err != nil {
				return err
			}
			var qids []uuid.UUID
			for rows.Next() {
				var qid uuid.UUID
				if err := rows.Scan(&qid); err != nil {
					rows.Close()
					return err
				}
				qids = append(qids, qid)
			}
			rows.Close()
			if err := attribution.RecomputeQuestions(ctx, pool, qids); err != nil {
				return err
			}
			slog.Info("recomputed attribution", "creator", *creatorID, "questions", len(qids))
		default:
			if err := attribution.RecomputeAll(ctx, pool); err != nil {
				return err
			}
			slog.Info("recomputed attribution", "scope", "all")
		}
		return nil
	}
	return fmt.Errorf("unknown command %q", cmd)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
