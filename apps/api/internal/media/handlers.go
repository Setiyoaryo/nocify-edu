package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

type Handler struct {
	DB      *pgxpool.Pool
	Storage Storage
}

type Asset struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	Source            string     `json:"source"`
	Creator           string     `json:"creator"`
	LicenseID         *int       `json:"license_id"`
	LicenseName       *string    `json:"license_name"`
	LicenseURL        string     `json:"license_url"`
	DownloadedAt      *time.Time `json:"downloaded_at"`
	OriginalFile      string     `json:"original_file"`
	Modified          bool       `json:"modified"`
	ModificationNotes string     `json:"modification_notes"`
	AIGenerated       bool       `json:"ai_generated"`
	Usage             []string   `json:"usage"`
	MimeType          *string    `json:"mime_type"`
	SizeBytes         *int64     `json:"size_bytes"`
	Status            string     `json:"status"`
	URL               *string    `json:"url"`
	BudgetBytes       int64      `json:"budget_bytes"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

const selectAsset = `SELECT a.id, a.code, a.name, a.kind, a.source, a.creator, a.license_id, l.name, a.license_url, a.downloaded_at,
	a.original_file, a.modified, a.modification_notes, a.ai_generated, a.usage, a.mime_type, a.size_bytes, a.status,
	a.storage_key IS NOT NULL, a.updated_at
	FROM assets a LEFT JOIN asset_licenses l ON l.id = a.license_id`

func scanAsset(row pgx.Row) (Asset, error) {
	var a Asset
	var hasFile bool
	err := row.Scan(&a.ID, &a.Code, &a.Name, &a.Kind, &a.Source, &a.Creator, &a.LicenseID, &a.LicenseName, &a.LicenseURL, &a.DownloadedAt,
		&a.OriginalFile, &a.Modified, &a.ModificationNotes, &a.AIGenerated, &a.Usage, &a.MimeType, &a.SizeBytes, &a.Status, &hasFile, &a.UpdatedAt)
	if hasFile {
		u := "/api/v1/media/" + a.ID.String()
		a.URL = &u
	}
	a.BudgetBytes = Budget(a.Kind)
	return a, err
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), selectAsset+`
		WHERE ($1 = '' OR a.status = $1) AND ($2 = '' OR a.kind = $2)
		  AND ($3 = '' OR a.name ILIKE '%' || $3 || '%' OR a.code ILIKE '%' || $3 || '%' OR a.source ILIKE '%' || $3 || '%' OR a.creator ILIKE '%' || $3 || '%')
		ORDER BY a.updated_at DESC LIMIT 200`, q.Get("status"), q.Get("kind"), q.Get("q"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Asset, error) { return scanAsset(row) })
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"assets": out})
}

type assetInput struct {
	Name              *string   `json:"name"`
	Kind              *string   `json:"kind"`
	Source            *string   `json:"source"`
	Creator           *string   `json:"creator"`
	LicenseID         *int      `json:"license_id"`
	LicenseURL        *string   `json:"license_url"`
	DownloadedAt      *string   `json:"downloaded_at"`
	OriginalFile      *string   `json:"original_file"`
	Modified          *bool     `json:"modified"`
	ModificationNotes *string   `json:"modification_notes"`
	AIGenerated       *bool     `json:"ai_generated"`
	Usage             *[]string `json:"usage"`
}

var codePrefix = map[string]string{
	"logo": "LOG", "mascot": "MAS", "character": "CHR", "illustration": "ILL", "icon": "ICO", "sticker": "STK",
	"reward": "RWD", "background": "BGD", "shape": "SHP", "animation": "ANI", "sound": "SND", "diagram": "DGM", "photo": "PHO",
}

const maxUpload = 5 << 20

// Create accepts multipart form: "meta" (JSON assetInput) and optional "file".
func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+(64<<10))
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Upload maksimal 5 MB.")
		return
	}
	var in assetInput
	dec := json.NewDecoder(strings.NewReader(r.FormValue("meta")))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "meta: "+err.Error())
		return
	}
	f := map[string]string{}
	if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
		f["name"] = "Nama wajib diisi."
	}
	if in.Kind == nil || codePrefix[*in.Kind] == "" {
		f["kind"] = "Jenis asset tidak valid."
	}
	if in.Source == nil || strings.TrimSpace(*in.Source) == "" {
		f["source"] = "Sumber wajib dicatat."
	}
	if in.Creator == nil || strings.TrimSpace(*in.Creator) == "" {
		f["creator"] = "Kreator wajib dicatat."
	}
	if len(f) > 0 {
		httpx.Invalid(w, f)
		return
	}
	u := auth.MustUser(r.Context())
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	prefix := codePrefix[*in.Kind]
	var seq int
	if err := tx.QueryRow(r.Context(), `SELECT COALESCE(max(substring(code from '[0-9]+$')::int), 0) + 1 FROM assets WHERE code LIKE $1 || '-%'`, prefix).Scan(&seq); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO assets (code, name, kind, source, creator, created_by) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		fmt.Sprintf("%s-%04d", prefix, seq), strings.TrimSpace(*in.Name), *in.Kind, strings.TrimSpace(*in.Source), strings.TrimSpace(*in.Creator), u.ID).Scan(&id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := applyInput(r, tx, id, in); err != nil {
		httpx.Invalid(w, map[string]string{"meta": err.Error()})
		return
	}
	if file, hdr, err := r.FormFile("file"); err == nil {
		defer file.Close()
		if err := h.store(r, tx, id, file, hdr.Header.Get("Content-Type")); err != nil {
			httpx.Invalid(w, map[string]string{"file": err.Error()})
			return
		}
	}
	audit.Log(r.Context(), tx, u.ID, "asset.create", "asset", id.String(), nil)
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.respond(w, r, id, http.StatusCreated)
}

func applyInput(r *http.Request, tx pgx.Tx, id uuid.UUID, in assetInput) error {
	var dl any
	if in.DownloadedAt != nil && *in.DownloadedAt != "" {
		t, err := time.Parse("2006-01-02", *in.DownloadedAt)
		if err != nil {
			return errors.New("downloaded_at harus YYYY-MM-DD")
		}
		dl = t
	}
	if in.Kind != nil && codePrefix[*in.Kind] == "" {
		return errors.New("kind tidak valid")
	}
	_, err := tx.Exec(r.Context(), `UPDATE assets SET
		name = COALESCE($2, name), kind = COALESCE($3, kind), source = COALESCE($4, source), creator = COALESCE($5, creator),
		license_id = COALESCE($6, license_id), license_url = COALESCE($7, license_url), downloaded_at = COALESCE($8, downloaded_at),
		original_file = COALESCE($9, original_file), modified = COALESCE($10, modified), modification_notes = COALESCE($11, modification_notes),
		ai_generated = COALESCE($12, ai_generated), usage = COALESCE($13, usage), updated_at = now()
		WHERE id = $1`, id, in.Name, in.Kind, in.Source, in.Creator, in.LicenseID, in.LicenseURL, dl, in.OriginalFile, in.Modified,
		in.ModificationNotes, in.AIGenerated, in.Usage)
	return err
}

func (h Handler) store(r *http.Request, tx pgx.Tx, id uuid.UUID, file io.Reader, declared string) error {
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	head = head[:n]
	mt, _, _ := mime.ParseMediaType(declared)
	sniffed := http.DetectContentType(head)
	// Trust the declared type only for formats the sniffer cannot identify (SVG, JSON);
	// otherwise the sniffed type must agree.
	switch mt {
	case "image/svg+xml":
		if !strings.Contains(string(head), "<svg") && !strings.HasPrefix(sniffed, "text/xml") {
			return errors.New("file is not an SVG")
		}
	case "application/json":
		if !strings.HasPrefix(sniffed, "text/plain") {
			return errors.New("file is not JSON")
		}
	default:
		if sniffed != mt && !(mt == "audio/ogg" && sniffed == "application/ogg") {
			return fmt.Errorf("file content (%s) does not match type %s", sniffed, mt)
		}
	}
	ext, ok := AllowedMIME[mt]
	if !ok {
		return fmt.Errorf("type %s not allowed", mt)
	}
	key := "assets/" + id.String() + ext
	size, err := h.Storage.Put(r.Context(), key, io.MultiReader(strings.NewReader(string(head)), file))
	if err != nil {
		return err
	}
	_, err = tx.Exec(r.Context(), `UPDATE assets SET storage_key = $2, mime_type = $3, size_bytes = $4, updated_at = now() WHERE id = $1`, id, key, mt, size)
	return err
}

func (h Handler) respond(w http.ResponseWriter, r *http.Request, id uuid.UUID, status int) {
	a, err := scanAsset(h.DB.QueryRow(r.Context(), selectAsset+` WHERE a.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, status, map[string]any{"asset": a})
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assetID")
	if !ok {
		return
	}
	h.respond(w, r, id, http.StatusOK)
}

// Update edits metadata. Changing provenance of a verified asset sends it back to "imported".
func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assetID")
	if !ok {
		return
	}
	var in assetInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := applyInput(r, tx, id, in); err != nil {
		httpx.Invalid(w, map[string]string{"meta": err.Error()})
		return
	}
	if in.Source != nil || in.Creator != nil || in.LicenseID != nil || in.LicenseURL != nil || in.AIGenerated != nil {
		if _, err := tx.Exec(r.Context(), `UPDATE assets SET status = 'imported' WHERE id = $1 AND status <> 'imported'`, id); err != nil {
			httpx.Internal(w, r, err)
			return
		}
	}
	audit.Log(r.Context(), tx, auth.MustUser(r.Context()).ID, "asset.update", "asset", id.String(), nil)
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.respond(w, r, id, http.StatusOK)
}

// Upload replaces the file; the asset must pass performance/design review again.
func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assetID")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+(64<<10))
	file, hdr, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "File wajib (maks. 5 MB).")
		return
	}
	defer file.Close()
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := h.store(r, tx, id, file, hdr.Header.Get("Content-Type")); err != nil {
		httpx.Invalid(w, map[string]string{"file": err.Error()})
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE assets SET status = 'license_verified' WHERE id = $1 AND status IN ('design_approved','performance_checked','published')`, id); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(r.Context(), tx, auth.MustUser(r.Context()).ID, "asset.upload", "asset", id.String(), nil)
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.respond(w, r, id, http.StatusOK)
}

func (h Handler) Transition(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assetID")
	if !ok {
		return
	}
	var in struct {
		To    string `json:"to"`
		Notes string `json:"notes"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	var f AssetFacts
	err = tx.QueryRow(r.Context(), `SELECT status, source, creator, license_id, license_url, ai_generated, storage_key IS NOT NULL, COALESCE(size_bytes, 0), kind
		FROM assets WHERE id = $1 FOR UPDATE`, id).Scan(&f.Status, &f.Source, &f.Creator, &f.LicenseID, &f.LicenseURL, &f.AIGenerated, &f.HasFile, &f.SizeBytes, &f.Kind)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := CheckTransition(f, in.To); err != nil {
		httpx.Error(w, http.StatusConflict, "gate_failed", err.Error())
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE assets SET status = $2, updated_at = now() WHERE id = $1`, id, in.To); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(r.Context(), tx, auth.MustUser(r.Context()).ID, "asset.transition", "asset", id.String(),
		map[string]any{"from": f.Status, "to": in.To, "notes": in.Notes})
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.respond(w, r, id, http.StatusOK)
}

// Serve streams an asset file. Published assets are public; anything else is
// visible to staff only (object storage access control, §39).
func (h Handler) Serve(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assetID")
	if !ok {
		return
	}
	var key, mt, status string
	var updated time.Time
	err := h.DB.QueryRow(r.Context(), `SELECT storage_key, mime_type, status, updated_at FROM assets WHERE id = $1 AND storage_key IS NOT NULL`, id).
		Scan(&key, &mt, &status, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	u, signedIn := auth.UserFrom(r.Context())
	staff := signedIn && u.Is(auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin)
	if status != "published" && !staff {
		httpx.NotFound(w)
		return
	}
	f, err := h.Storage.Open(r.Context(), key)
	if err != nil {
		httpx.NotFound(w)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", mt)
	// Inline SVG must never execute script.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:")
	if status == "published" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	http.ServeContent(w, r, "", updated, f)
}

// ---------------------------------------------------------------- licenses

type License struct {
	ID                    int    `json:"id"`
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	CommercialUse         bool   `json:"commercial_use"`
	AttributionRequired   bool   `json:"attribution_required"`
	RedistributionAllowed bool   `json:"redistribution_allowed"`
	Notes                 string `json:"notes"`
}

func (h Handler) Licenses(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `SELECT id, name, url, commercial_use, attribution_required, redistribution_allowed, notes FROM asset_licenses ORDER BY name`)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[License])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"licenses": out})
}

func (h Handler) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var in License
	if !httpx.Decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || !strings.HasPrefix(in.URL, "https://") {
		httpx.Invalid(w, map[string]string{"name": "Nama dan URL lisensi (https) wajib."})
		return
	}
	err := h.DB.QueryRow(r.Context(), `INSERT INTO asset_licenses (name, url, commercial_use, attribution_required, redistribution_allowed, notes)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, in.Name, in.URL, in.CommercialUse, in.AttributionRequired, in.RedistributionAllowed, in.Notes).Scan(&in.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(r.Context(), h.DB, auth.MustUser(r.Context()).ID, "license.create", "asset_license", fmt.Sprint(in.ID), nil)
	httpx.JSON(w, http.StatusCreated, map[string]any{"license": in})
}
