// Package seed loads curriculum, rewards, licenses and questions from the
// repository's content/ and assets/ directories. It is idempotent: existing
// rows are matched by code and questions that already exist are never
// overwritten (their reviewed content belongs to the CMS from then on).
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/admin"
	"github.com/nocify/eduplay/api/internal/questions"
)

type curriculumFile struct {
	Curriculum struct{ Code, Name string } `json:"curriculum"`
	Levels     []struct {
		Code, Name string
		Sort       int
		Grades     []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			SessionSize int    `json:"session_size"`
			UXBand      string `json:"ux_band"`
		} `json:"grades"`
	} `json:"levels"`
	Subjects []struct{ Code, Name, Motif string } `json:"subjects"`
	Chapters []struct {
		Code, Name, Grade, Subject string
		Skills                     []struct{ Code, Name, Description string } `json:"skills"`
	} `json:"chapters"`
}

type rewardsFile struct {
	Badges  []struct{ Code, Name, Description, Icon string } `json:"badges"`
	Rewards []struct {
		Code, Kind, Name, Icon string
		XPRequired             int `json:"xp_required"`
	} `json:"rewards"`
}

type licenseFile []struct {
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	CommercialUse         bool   `json:"commercial_use"`
	AttributionRequired   bool   `json:"attribution_required"`
	RedistributionAllowed bool   `json:"redistribution_allowed"`
	Notes                 string `json:"notes"`
}

type questionFile struct {
	Source    string `json:"source"`
	Questions []struct {
		Code              string          `json:"code"`
		Skill             string          `json:"skill"`
		Type              string          `json:"type"`
		Difficulty        int             `json:"difficulty"`
		LearningObjective string          `json:"learning_objective"`
		Content           json.RawMessage `json:"content"`
		Answer            json.RawMessage `json:"answer"`
		Explanation       string          `json:"explanation"`
		Hint              string          `json:"hint"`
	} `json:"questions"`
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Options.Publish marks seeded questions as published (development only). In
// every other environment they land in "review" so humans run the quality gate.
type Options struct {
	ContentDir string
	AssetsDir  string
	Publish    bool
}

func Run(ctx context.Context, pool *pgxpool.Pool, o Options) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := seedCurriculum(ctx, tx, filepath.Join(o.ContentDir, "curriculum", "curriculum.json")); err != nil {
			return err
		}
		if err := seedRewards(ctx, tx, filepath.Join(o.ContentDir, "curriculum", "rewards.json")); err != nil {
			return err
		}
		if err := seedLicenses(ctx, tx, filepath.Join(o.AssetsDir, "licenses", "licenses.json")); err != nil {
			return err
		}
		if err := seedAssets(ctx, tx, filepath.Join(o.AssetsDir, "registry.json"), o.Publish); err != nil {
			return err
		}
		return seedQuestions(ctx, tx, filepath.Join(o.ContentDir, "questions"), o.Publish)
	})
}

func seedCurriculum(ctx context.Context, tx pgx.Tx, path string) error {
	var f curriculumFile
	if err := readJSON(path, &f); err != nil {
		return err
	}
	var cvID int
	if err := tx.QueryRow(ctx, `INSERT INTO curriculum_versions (code, name, is_active) VALUES ($1,$2,true)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name RETURNING id`, f.Curriculum.Code, f.Curriculum.Name).Scan(&cvID); err != nil {
		return err
	}
	for li, l := range f.Levels {
		var levelID int
		if err := tx.QueryRow(ctx, `INSERT INTO education_levels (code, name, sort) VALUES ($1,$2,$3)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, sort = EXCLUDED.sort RETURNING id`, l.Code, l.Name, li+1).Scan(&levelID); err != nil {
			return err
		}
		for gi, g := range l.Grades {
			if _, err := tx.Exec(ctx, `INSERT INTO grades (education_level_id, code, name, sort, session_size, ux_band) VALUES ($1,$2,$3,$4,$5,$6)
				ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, sort = EXCLUDED.sort`, levelID, g.Code, g.Name, (li+1)*100+gi, g.SessionSize, g.UXBand); err != nil {
				return err
			}
		}
	}
	for i, s := range f.Subjects {
		if _, err := tx.Exec(ctx, `INSERT INTO subjects (code, name, motif, sort) VALUES ($1,$2,$3,$4)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, motif = EXCLUDED.motif`, s.Code, s.Name, s.Motif, i+1); err != nil {
			return err
		}
	}
	for ci, c := range f.Chapters {
		var chID int
		if err := tx.QueryRow(ctx, `INSERT INTO chapters (curriculum_version_id, grade_id, subject_id, code, name, sort)
			VALUES ($1, (SELECT id FROM grades WHERE code = $2), (SELECT id FROM subjects WHERE code = $3), $4, $5, $6)
			ON CONFLICT (curriculum_version_id, code) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
			cvID, c.Grade, c.Subject, c.Code, c.Name, ci+1).Scan(&chID); err != nil {
			return fmt.Errorf("chapter %s: %w", c.Code, err)
		}
		for si, s := range c.Skills {
			if _, err := tx.Exec(ctx, `INSERT INTO skills (chapter_id, code, name, description, sort) VALUES ($1,$2,$3,$4,$5)
				ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description`, chID, s.Code, s.Name, s.Description, si+1); err != nil {
				return err
			}
		}
	}
	slog.Info("seed: curriculum", "chapters", len(f.Chapters))
	return nil
}

func seedRewards(ctx context.Context, tx pgx.Tx, path string) error {
	var f rewardsFile
	if err := readJSON(path, &f); err != nil {
		return err
	}
	for _, b := range f.Badges {
		if _, err := tx.Exec(ctx, `INSERT INTO badges (code, name, description, icon) VALUES ($1,$2,$3,$4)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, icon = EXCLUDED.icon`, b.Code, b.Name, b.Description, b.Icon); err != nil {
			return err
		}
	}
	for _, r := range f.Rewards {
		if _, err := tx.Exec(ctx, `INSERT INTO rewards (code, kind, name, icon, xp_required) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, icon = EXCLUDED.icon, xp_required = EXCLUDED.xp_required`, r.Code, r.Kind, r.Name, r.Icon, r.XPRequired); err != nil {
			return err
		}
	}
	return nil
}

func seedLicenses(ctx context.Context, tx pgx.Tx, path string) error {
	var f licenseFile
	if err := readJSON(path, &f); err != nil {
		return err
	}
	for _, l := range f {
		if _, err := tx.Exec(ctx, `INSERT INTO asset_licenses (name, url, commercial_use, attribution_required, redistribution_allowed, notes)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (name) DO UPDATE SET url = EXCLUDED.url, notes = EXCLUDED.notes`,
			l.Name, l.URL, l.CommercialUse, l.AttributionRequired, l.RedistributionAllowed, l.Notes); err != nil {
			return err
		}
	}
	return nil
}

func seedQuestions(ctx context.Context, tx pgx.Tx, dir string, publish bool) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	created := 0
	for _, path := range files {
		var f questionFile
		if err := readJSON(path, &f); err != nil {
			return err
		}
		for _, q := range f.Questions {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM questions WHERE code = $1)`, q.Code).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			if err := questions.Validate(q.Type, q.Content, q.Answer); err != nil {
				return fmt.Errorf("%s %s: %w", filepath.Base(path), q.Code, err)
			}
			var skillID int
			if err := tx.QueryRow(ctx, `SELECT id FROM skills WHERE code = $1`, q.Skill).Scan(&skillID); err != nil {
				return fmt.Errorf("%s %s: unknown skill %q", filepath.Base(path), q.Code, q.Skill)
			}
			in := admin.QuestionInput{Code: q.Code, SkillID: skillID, Type: q.Type, Difficulty: q.Difficulty,
				LearningObjective: q.LearningObjective, Source: f.Source, Content: q.Content, Answer: q.Answer,
				Explanation: q.Explanation, Hint: q.Hint}
			id, err := admin.CreateQuestion(ctx, tx, in, uuid.Nil)
			if err != nil {
				return fmt.Errorf("%s: %w", q.Code, err)
			}
			status := "review"
			if publish {
				status = "published"
			}
			if _, err := tx.Exec(ctx, `UPDATE questions SET status = $2,
				published_version_id = CASE WHEN $2 = 'published' THEN current_version_id END,
				published_at = CASE WHEN $2 = 'published' THEN now() END WHERE id = $1`, id, status); err != nil {
				return err
			}
			created++
		}
	}
	slog.Info("seed: questions", "created", created, "published", publish)
	return nil
}

type registryFile struct {
	Assets []struct {
		AssetID           string   `json:"asset_id"`
		Name              string   `json:"name"`
		Kind              string   `json:"kind"`
		Source            string   `json:"source"`
		Creator           string   `json:"creator"`
		License           string   `json:"license"`
		LicenseURL        string   `json:"license_url"`
		DownloadedAt      string   `json:"downloaded_at"`
		OriginalFile      string   `json:"original_file"`
		Modified          bool     `json:"modified"`
		ModificationNotes string   `json:"modification_notes"`
		AIGenerated       bool     `json:"ai_generated"`
		Usage             []string `json:"usage"`
		SizeBytes         int64    `json:"size_bytes"`
	} `json:"assets"`
}

// seedAssets records bundled UI assets (assets/registry.json) in the asset
// library so their provenance is reviewable in the CMS. Outside development
// they land as license_verified and still need design/performance approval.
func seedAssets(ctx context.Context, tx pgx.Tx, path string, publish bool) error {
	var f registryFile
	if err := readJSON(path, &f); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	status := "license_verified"
	if publish {
		status = "published"
	}
	for _, a := range f.Assets {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assets (code, name, kind, source, creator, license_id, license_url, downloaded_at, original_file,
			                    modified, modification_notes, ai_generated, usage, size_bytes, mime_type, status)
			VALUES ($1,$2,$3,$4,$5,(SELECT id FROM asset_licenses WHERE name = $6),$7,$8::date,$9,$10,$11,$12,$13,$14,'image/svg+xml',$15)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, usage = EXCLUDED.usage, size_bytes = EXCLUDED.size_bytes,
				modification_notes = EXCLUDED.modification_notes, updated_at = now()`,
			a.AssetID, a.Name, a.Kind, a.Source, a.Creator, a.License, a.LicenseURL, a.DownloadedAt, a.OriginalFile,
			a.Modified, a.ModificationNotes, a.AIGenerated, a.Usage, a.SizeBytes, status); err != nil {
			return fmt.Errorf("asset %s: %w", a.AssetID, err)
		}
	}
	slog.Info("seed: assets", "count", len(f.Assets))
	return nil
}
