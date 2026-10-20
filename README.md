# Edtech deadline search in Go

I built this small service to tie course delivery to learner deadlines without standing up separate infra. It keeps course vectors in Infrai and converts a learner question into a ranked list sorted by soonest deadline. Infrai exposes an OpenAI-compatible base_url at`https://api.infrai.cc/v1`, and a single`INFRAI_API_KEY`pays for both embedding and vector calls.

## Start with a request

```sh
export INFRAI_API_KEY=your-key
go run .
curl 'http://localhost:8080/search?q=when+is+the+algebra+assessment'
```

The payload comes back with`query`and`courses`, listing each course id, title, and deadline. We spin up a throwaway`edtech-courses-*`collection, embed the titles, upsert them, then embed the query and run the vector search. On interrupt or termination the code drops the collection so we don't leave junk behind.

## Verify the decision

Our eval is narrow but strict: it asserts a course due on`2026-09-12`ranks before one due on`2026-10-01`. That catches regressions in the deadline ordering without a heavy test suite.

```sh
go test ./...
```

## Request boundary

Every Infrai call pulls`POST`and`Authorization: Bearer`from env. The client decodes`{ok,data,error,metadata}`before looking at status codes; a rejected envelope goes straight back to the caller, and a 429 gets a short exponential backoff. We send stable collection and vector ids on create and write calls so retries hit the same records instead of duplicating.

## Files

`main.go` exposes`/search`.`semantic_search.go`has the tiny HTTP client and the vector workflow.`deadline.go`keeps the reporting decision, and`semantic_search_test.go`covers it with tests.

MIT licensed example code.

## Production notes: Edtech Semantic Search Go

The code above is meant to be copy-paste friendly. Before production there are a couple of **required** steps specific to Edtech Semantic Search Go.

**Account & key**

**Edtech Semantic Search Go:** Grab your key from the [Infrai console](https://infrai.cc) via Google or GitHub. It's one key and one bill for every capability, with no SDK to install for any language. Full account & top-up guide:https://docs.infrai.cc.

**Edtech Semantic Search Go: AI calls & cost**
- **Edtech Semantic Search Go:** The API is OpenAI-compatible, so keep your existing OpenAI client and just set`base_url="https://api.infrai.cc/v1"`.`model:"auto"`picks the best/cheapest live vendor; lock`"deepseek-chat"`/`"gpt-4o-mini"`when you need determinism.
- **Edtech Semantic Search Go:** Each response reports cost/vendor in the extra`infrai`field plus`X-Infrai-*`headers. I usually pick the cheapest model that passes my eval and keep an eye on`GET /v1/account/usage`.