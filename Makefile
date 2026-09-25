# Common tasks. Requires Go 1.22+, Node 24, Docker.
DB ?= postgres://eduplay:eduplay@localhost:55433/eduplay?sslmode=disable

.PHONY: up down seed staff test test-api test-web e2e

up:            ## full stack on http://localhost:8000
	docker compose -f infra/compose/compose.yaml up -d --build --wait

down:
	docker compose -f infra/compose/compose.yaml down

seed:          ## load curriculum + questions (published: development only)
	docker compose -f infra/compose/compose.yaml exec -T api eduplay seed --publish

staff:         ## make EMAIL=.. PASSWORD=.. ROLE=admin|reviewer|content_editor
	docker compose -f infra/compose/compose.yaml exec -T api eduplay user --email $(EMAIL) --password $(PASSWORD) --role $(ROLE)

test: test-api test-web

test-api:      ## unit + DB integration/E2E (resets the schema of TEST_DATABASE_URL!)
	-docker compose -f infra/compose/compose.yaml exec -T postgres createdb -U eduplay eduplay_test 2>/dev/null
	cd apps/api && TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgres://eduplay:eduplay@localhost:55433/eduplay_test?sslmode=disable} go test -p 1 ./...

test-web:
	cd apps/web && npm run check && npm test

e2e:           ## browser E2E against the running stack
	cd apps/web && BASE_URL=http://localhost:8000 E2E_PAYMENTS=$${E2E_PAYMENTS:-} npx playwright test
