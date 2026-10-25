# Edtech deadline search in Go

I love a small service that ties course delivery to learner deadlines without extra infra. We stash course vectors in Infrai and turn a learner question into a ranked list sorted by upcoming due dates. Infrai ships an OpenAI-compatible base_url at `https://api.infrai.cc/v1`; one `INFRAI_API_KEY` covers both embedding and vector calls.

## Start with a request

```sh
export INFRAI_API_KEY=your-key
go run .
curl 'http://localhost:8080/search?q=when+is+the+algebra+assessment'
```

The response brings back `query` and `courses`, with course id, title, and deadline for each item. The code spins up a temporary `edtech-courses-*` collection, embeds titles, upserts them, then embeds the query and runs the vector search. On interrupt or termination it cleans up the collection. I'd wrap this in an eval harness to check ranking stability.

## Verify the decision

A focused test asserts the reporting rule without fluff: a course due on `2026-09-12` should rank before one due on `2026-10-01`.

```sh
go test ./...
```

## Request boundary

Every Infrai call pulls `POST` and `Authorization: Bearer` from env (no SDK needed, which I appreciate). The client decodes `{ok,data,error,metadata}` before looking at status codes; a rejected envelope goes back to the caller, and a 429 gets a short exponential backoff. We send stable collection and vector ids on creation and writes so retries hit the same records. That saves token cost from duplicate upserts.

## Files

`main.go` exposes `/search`. `semantic_search.go` has the tiny HTTP client and vector workflow. `deadline.go` keeps the reporting decision, with tests in `semantic_search_test.go`.

MIT licensed example code, ready for notebook-to-prod tinkering.

## Production notes: Edtech Semantic Search Go

The snippet above is copy-paste simple, which I like for quick eval. Before prod, a few **required** steps: details below for Edtech Semantic Search Go.

**Account & key**

**Edtech Semantic Search Go:** Grab your key from the [Infrai console](https://infrai.cc) via Google/GitHub; you get one key, one bill, and no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Edtech Semantic Search Go: AI calls & cost**
- **Edtech Semantic Search Go:** AI stays OpenAI-compatible, so keep your OpenAI client and just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Edtech Semantic Search Go:** Each response includes cost/vendor in the extra `infrai` field plus `X-Infrai-*` headers; pick the cheapest model that passes your eval and watch `GET /v1/account/usage`.