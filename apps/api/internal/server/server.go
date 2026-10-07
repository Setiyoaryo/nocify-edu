// Package server wires every module into one HTTP router (modular monolith, §17).
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/google/uuid"
	"github.com/nocify/eduplay/api/internal/admin"
	"github.com/nocify/eduplay/api/internal/analytics"
	"github.com/nocify/eduplay/api/internal/assessment"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/candidates"
	"github.com/nocify/eduplay/api/internal/changes"
	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/config"
	"github.com/nocify/eduplay/api/internal/creators"
	"github.com/nocify/eduplay/api/internal/curriculum"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/media"
	"github.com/nocify/eduplay/api/internal/missions"
	"github.com/nocify/eduplay/api/internal/pipeline"
	"github.com/nocify/eduplay/api/internal/progress"
	"github.com/nocify/eduplay/api/internal/rewards"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

func New(cfg config.Config, pool *pgxpool.Pool, storage media.Storage) http.Handler {
	authSvc := &auth.Service{DB: pool, Secret: []byte(cfg.SessionSecret), TTL: cfg.SessionTTL, CookieSecure: cfg.CookieSecure}
	policy := subscriptions.Policy{
		FreeDailyChallenges: cfg.FreeDailyChallenges, FreeMaxChildren: cfg.FreeMaxChildren,
		PaidMaxChildren: cfg.PaidMaxChildren, MonthlyPriceIDR: cfg.MonthlyPriceIDR,
	}
	var provider subscriptions.Provider = subscriptions.MidtransProvider{ServerKey: cfg.MidtransServerKey, Production: cfg.MidtransProduction}
	if cfg.PaymentProvider == "mock" {
		provider = subscriptions.MockProvider{Secret: []byte(cfg.PaymentSecret), PublicURL: cfg.PublicURL}
	}

	authH := auth.Handler{S: authSvc, OnRegister: func(ctx context.Context, u auth.User) {
		analytics.Track(ctx, pool, "register", u.ID, uuid.Nil, nil)
	}}
	childH := children.Handler{DB: pool, Policy: policy}
	curH := curriculum.Handler{DB: pool}
	asmH := assessment.Handler{DB: pool, Policy: policy, TZ: cfg.Timezone}
	progH := progress.Handler{DB: pool, Policy: policy, TZ: cfg.Timezone}
	rewH := rewards.Handler{DB: pool}
	subH := subscriptions.Handler{DB: pool, Policy: policy, Provider: provider, PublicURL: cfg.PublicURL}
	mediaH := media.Handler{DB: pool, Storage: storage}
	adminH := admin.Handler{DB: pool}
	candH := candidates.Handler{DB: pool}
	changeH := changes.Handler{DB: pool}
	creatorH := creators.Handler{DB: pool}
	anaH := analytics.Handler{DB: pool}
	misH := missions.Handler{DB: pool}
	pipeH := pipeline.Handler{DB: pool}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, httpx.Observe)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	// /metrics is not routed by Traefik (only /api is), so it stays internal.
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(httpx.SecurityHeaders)

		// Webhooks authenticate by signature: no session, no Origin check.
		r.With(httpx.RateLimit(120, time.Minute)).Post("/webhooks/payment", subH.Webhook)

		r.Group(func(r chi.Router) {
			r.Use(httpx.SameOrigin(append([]string{cfg.PublicURL}, cfg.AllowedOrigins...)))
			r.Use(authSvc.Middleware)

			r.With(httpx.RateLimit(10, time.Minute)).Post("/auth/register", authH.Register)
			r.With(httpx.RateLimit(10, time.Minute)).Post("/auth/login", authH.Login)
			r.Post("/auth/logout", authH.Logout)

			r.Get("/grades", curH.Grades)
			r.Get("/subjects", curH.Subjects)
			r.Get("/skills", curH.Skills)
			r.Get("/media/{assetID}", mediaH.Serve)
			r.With(httpx.RateLimit(60, time.Minute)).Post("/events", anaH.Collect)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireUser)
				r.Get("/me", authH.Me)
				r.Patch("/me", authH.UpdateMe)
				r.With(httpx.RateLimit(10, time.Minute)).Post("/me/password", authH.ChangePassword)
				r.With(httpx.RateLimit(5, time.Minute)).Delete("/me", authH.DeleteAccount)
				r.Get("/me/subscription", subH.Get)

				r.Get("/children", childH.List)
				r.Post("/children", childH.Create)
				r.Route("/children/{childID}", func(r chi.Router) {
					r.Get("/", childH.Get)
					r.Patch("/", childH.Update)
					r.Delete("/", childH.Delete)
					r.With(httpx.RateLimit(10, time.Minute)).Post("/verify-pin", childH.VerifyPIN)
					r.Get("/home", progH.ChildHome)
					r.Get("/rewards", rewH.List)
					r.Get("/progress", progH.Progress)
					r.Get("/history", progH.History)
					r.Get("/recommendations", progH.Recommendations)
					r.Get("/recap", progH.Recap)
					r.Get("/missions", misH.List)
					r.Post("/missions", misH.Create)
					r.Delete("/missions/{missionID}", misH.Cancel)
				})

				r.Post("/assessments", asmH.Start)
				r.Get("/assessments/{assessmentID}", asmH.Get)
				r.Post("/assessments/{assessmentID}/answers", asmH.Answer)
				r.Post("/assessments/{assessmentID}/items/{itemID}/hint", asmH.Hint)
				r.Post("/assessments/{assessmentID}/complete", asmH.Complete)

				r.Get("/subscription", subH.Get)
				r.Post("/subscription/checkout", subH.Checkout)
				r.Get("/subscription/checkout/{ref}", subH.CheckoutStatus)
				r.Post("/subscription/cancel", subH.Cancel)
				r.Post("/subscription/resume", subH.Resume)
				if cfg.PaymentProvider == "mock" && !cfg.IsProduction() {
					r.Post("/dev/mock-checkout/{ref}/pay", subH.MockPay)
				}

				r.Route("/admin", func(r chi.Router) {
					r.Use(auth.RequireRole(auth.RoleEditor, auth.RoleReviewer))
					r.Get("/stats", adminH.Stats)
					r.Get("/question-types", adminH.QuestionTypes)
					r.Get("/questions", adminH.ListQuestions)
					r.Post("/questions", adminH.CreateQuestion)
					r.Get("/questions/{questionID}", adminH.GetQuestion)
					r.Put("/questions/{questionID}", adminH.UpdateQuestion)
					r.Post("/questions/{questionID}/transition", adminH.TransitionQuestion)
					r.Post("/questions/{questionID}/preview", adminH.Preview)

					r.Get("/curriculum", adminH.Curriculum)
					r.Get("/candidates", candH.ListCandidates)
					r.Get("/candidates/{kind}/{id}", candH.GetCandidate)
					r.Get("/content-studio/tree", candH.ContentStudioTree)
					r.Get("/content-studio/lessons/{skillID}", candH.ContentStudioLesson)
					r.Get("/curriculum-changes", changeH.ListChanges)
					r.Get("/curriculum-changes/{id}", changeH.GetChange)
					r.Get("/creators", creatorH.ListCreators)
					r.Get("/creators/{creatorID}", creatorH.GetCreator)
					r.Get("/assets", mediaH.List)
					r.Post("/assets", mediaH.Create)
					r.Get("/assets/{assetID}", mediaH.Get)
					r.Patch("/assets/{assetID}", mediaH.Update)
					r.Post("/assets/{assetID}/file", mediaH.Upload)
					r.Get("/licenses", mediaH.Licenses)
					r.Get("/audit-logs", adminH.AuditLogs)

					r.Get("/pipeline/sources", pipeH.ListSources)
					r.Post("/pipeline/sources/{code}/status", pipeH.SetSourceStatus)
					r.Get("/pipeline/jobs", pipeH.ListJobs)
					r.Post("/pipeline/jobs", pipeH.EnqueueJob)
					r.Get("/pipeline/jobs/{id}", pipeH.GetJob)
					r.Post("/pipeline/jobs/{id}/cancel", pipeH.CancelJob)
					r.Post("/pipeline/jobs/{id}/retry", pipeH.RetryJob)
					r.Get("/pipeline/worker-status", pipeH.WorkerStatus)
					r.Get("/pipeline/status", pipeH.WorkerStatus)

					r.Group(func(r chi.Router) {
						r.Use(auth.RequireRole(auth.RoleEditor))
						r.Post("/subjects", adminH.CreateSubject)
						r.Patch("/subjects/{id}", adminH.UpdateSubject)
						r.Patch("/grades/{id}", adminH.UpdateGrade)
						r.Post("/chapters", adminH.CreateChapter)
						r.Patch("/chapters/{id}", adminH.UpdateChapter)
						r.Post("/skills", adminH.CreateSkill)
						r.Patch("/skills/{id}", adminH.UpdateSkill)
						r.Post("/creators", creatorH.CreateCreator)
						r.Patch("/creators/{creatorID}", creatorH.UpdateCreator)
						r.Put("/questions/{questionID}/creator", creatorH.AssignCreator)
						r.Delete("/questions/{questionID}/creator", creatorH.UnassignCreator)
					})
					r.Group(func(r chi.Router) {
						r.Use(auth.RequireRole(auth.RoleReviewer))
						r.Post("/candidates/{kind}/{id}/decision", candH.DecideCandidate)
						r.Post("/curriculum-changes/{id}/decision", changeH.DecideChange)
						r.Post("/assets/{assetID}/transition", mediaH.Transition)
						r.Post("/licenses", mediaH.CreateLicense)
						r.Post("/pipeline/publish", pipeH.Publish)
					})
				})
			})
		})
	})
	return r
}
