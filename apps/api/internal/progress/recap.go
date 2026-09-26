package progress

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/httpx"
)

type RecapStats struct {
	TotalChallenges int        `json:"total_challenges"`
	TotalQuestions  int        `json:"total_questions"`
	Accuracy        float64    `json:"accuracy"`
	Stars           int        `json:"stars"`
	MasteredSkills  int        `json:"mastered_skills"`
	Streak          int        `json:"streak"`
	FavoriteSubject string     `json:"favorite_subject"`
	TopBadge        *BadgeInfo `json:"top_badge,omitempty"`
}

type BadgeInfo struct {
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

type Archetype struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Praise       string `json:"praise"`
	Highlight    string `json:"highlight"`
	Tag          string `json:"tag"`
	Theme        string `json:"theme"`
	Icon         string `json:"icon"`
	Tier         string `json:"tier"`          // starter | rare | epic | legendary
	TierName     string `json:"tier_name"`     // Pemula | Langka | Epik | Legendaris
	NextTierHint string `json:"next_tier_hint"`
}

type ChildRecap struct {
	Child     children.Child `json:"child"`
	Stats     RecapStats     `json:"stats"`
	Archetype Archetype      `json:"archetype"`
}

// DetermineArchetype calculates card tier, dynamic persona, and psychological reinforcement praise.
func DetermineArchetype(child children.Child, stats RecapStats) Archetype {
	tier := "starter"
	tierName := "Pemula"
	nextTierHint := "Selesaikan 3 tantangan untuk membuka tier Langka!"

	if child.Level >= 5 || stats.Streak >= 7 || stats.TotalChallenges >= 20 {
		tier = "legendary"
		tierName = "Legendaris"
		nextTierHint = "Kamu berada di puncak legenda! Terus pertahankan prestasimu!"
	} else if child.Level >= 3 || stats.Streak >= 3 || stats.TotalChallenges >= 8 {
		tier = "epic"
		tierName = "Epik"
		nextTierHint = "Capai Level 5 atau 7 hari streak untuk membuka gelar Legendaris!"
	} else if child.Level >= 2 || stats.TotalChallenges >= 3 {
		tier = "rare"
		tierName = "Langka"
		nextTierHint = "Capai Level 3 atau 3 hari streak untuk membuka gelar Epik!"
	}

	var arch Archetype
	favSubj := strings.ToLower(stats.FavoriteSubject)

	switch {
	case stats.Streak >= 5:
		arch = Archetype{
			Code:      "ksatria_api",
			Title:     "KSATRIA API PANTANG PADAM",
			Subtitle:  "Streak Membara!",
			Praise:    fmt.Sprintf("Kamu anaknya gigih dan pantang mundur! %d hari berturut-turut belajar tanpa bolong. Api belajarmu membakar semua tantangan!", stats.Streak),
			Highlight: fmt.Sprintf("%d Hari Streak", stats.Streak),
			Tag:       "Konsistensi Juara",
			Theme:     "tangerine",
			Icon:      "fire",
		}
	case stats.Accuracy >= 85 && stats.TotalQuestions >= 10:
		arch = Archetype{
			Code:      "detektif_elang",
			Title:     "DETEKTIF MATA ELANG",
			Subtitle:  "Super Teliti & Tajam!",
			Praise:    fmt.Sprintf("Kamu anaknya super jeli dan teliti deh! Akurasi belajarmu tembus %.0f%%. Jebakan soal tersulit pun berhasil kamu pecahkan!", stats.Accuracy),
			Highlight: fmt.Sprintf("%.0f%% Akurasi", stats.Accuracy),
			Tag:       "Fokus Sempurna",
			Theme:     "sky",
			Icon:      "target",
		}
	case strings.Contains(favSubj, "matematika") || strings.Contains(favSubj, "hitung") || strings.Contains(favSubj, "angka"):
		arch = Archetype{
			Code:      "penyihir_angka",
			Title:     "PENYIHIR ANGKA AJAIB",
			Subtitle:  "Master Berhitung!",
			Praise:    "Kamu jagoan matematika sejati! Soal hitung-hitungan langsung takluk sama mantramu. Angka jadi arena bermain yang seru!",
			Highlight: fmt.Sprintf("%d Soal Selesai", stats.TotalQuestions),
			Tag:       "Logika Tajam",
			Theme:     "grape",
			Icon:      "lightning",
		}
	case strings.Contains(favSubj, "bahasa") || strings.Contains(favSubj, "kata") || strings.Contains(favSubj, "baca"):
		arch = Archetype{
			Code:      "penjelajah_kata",
			Title:     "PENJELAJAH KOSAKATA SAKTI",
			Subtitle:  "Gudang Cerita & Kata!",
			Praise:    "Imajinasi dan kosakatamu luar biasa kaya! Membaca dan memahami cerita jadi petualangan seru yang berhasil kamu tuntaskan.",
			Highlight: fmt.Sprintf("Level %d Petualang", child.Level),
			Tag:       "Imajinasi Luas",
			Theme:     "peach",
			Icon:      "book",
		}
	case stats.Stars >= 20:
		arch = Archetype{
			Code:      "kolektor_bintang",
			Title:     "KOLEKTOR BINTANG GALAKSI",
			Subtitle:  "Kilau Bintang Emas!",
			Praise:    fmt.Sprintf("Bintang-bintang bertaburan karena prestasimu! Berhasil mengumpulkan %d bintang emas. Calon bintang masa depan!", stats.Stars),
			Highlight: fmt.Sprintf("%d Bintang Emas", stats.Stars),
			Tag:       "Penuh Prestasi",
			Theme:     "sun",
			Icon:      "star",
		}
	case stats.TotalChallenges >= 5:
		arch = Archetype{
			Code:      "ksatria_kilat",
			Title:     "KSATRIA KILAT PEMBELAJAR",
			Subtitle:  "Cepat & Berani!",
			Praise:    fmt.Sprintf("%d tantangan berhasil kamu selesaikan dengan ceria. Tiap hari makin tangkas dan bertambah pintar!", stats.TotalChallenges),
			Highlight: fmt.Sprintf("%d Tantangan", stats.TotalChallenges),
			Tag:       "Penuh Semangat",
			Theme:     "mint",
			Icon:      "lightning",
		}
	default:
		arch = Archetype{
			Code:      "pahlawan_baru",
			Title:     "PAHLAWAN BINTANG BARU",
			Subtitle:  "Petualangan Dimulai!",
			Praise:    fmt.Sprintf("Langkah awal yang hebat buat %s! Terus taklukkan tantangan tiap hari untuk membuka gelar legendaris berikutnya!", child.Nickname),
			Highlight: fmt.Sprintf("Level %d", child.Level),
			Tag:       "Calon Legenda",
			Theme:     "lilac",
			Icon:      "sparkle",
		}
	}

	arch.Tier = tier
	arch.TierName = tierName
	arch.NextTierHint = nextTierHint
	return arch
}

// Recap handles GET /children/{childID}/recap
func (h Handler) Recap(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	ctx := r.Context()
	var stats RecapStats

	err := h.DB.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM assessments WHERE child_id = $1 AND status = 'completed')::int,
			(SELECT count(*) FROM question_attempts WHERE child_id = $1 AND attempt_no = 1)::int,
			(SELECT COALESCE(round(100.0 * avg(CASE WHEN is_correct THEN 1 ELSE 0 END), 1), 0)::float8 FROM question_attempts WHERE child_id = $1 AND attempt_no = 1),
			(SELECT COALESCE(sum(stars), 0) FROM assessments WHERE child_id = $1 AND status = 'completed')::int,
			(SELECT count(*) FROM skill_progress WHERE child_id = $1 AND mastery >= 70)::int`,
		child.ID).Scan(&stats.TotalChallenges, &stats.TotalQuestions, &stats.Accuracy, &stats.Stars, &stats.MasteredSkills)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	streak, err := Streak(ctx, h.DB, child.ID, h.TZ)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	stats.Streak = streak

	var fav string
	_ = h.DB.QueryRow(ctx, `
		SELECT s.name FROM assessments a
		JOIN subjects s ON s.id = a.subject_id
		WHERE a.child_id = $1 AND a.status = 'completed'
		GROUP BY s.name ORDER BY count(*) DESC LIMIT 1`, child.ID).Scan(&fav)
	stats.FavoriteSubject = fav

	var b BadgeInfo
	if err := h.DB.QueryRow(ctx, `
		SELECT b.name, b.icon, b.description FROM child_inventory i
		JOIN badges b ON b.code = i.item_code
		WHERE i.child_id = $1 AND i.item_kind = 'badge'
		ORDER BY i.acquired_at DESC LIMIT 1`, child.ID).Scan(&b.Name, &b.Icon, &b.Description); err == nil {
		stats.TopBadge = &b
	}

	archetype := DetermineArchetype(child, stats)
	httpx.JSON(w, http.StatusOK, ChildRecap{
		Child:     child,
		Stats:     stats,
		Archetype: archetype,
	})
}
