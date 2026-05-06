# gRPC Server — Test Cases

## Run

```bash
go test ./face-recognition/grpc/server/... -v
```

## Infrastructure

Tests use `bufconn` (in-process gRPC, no real port) so they run without a live server or database.

A `mockRepo` struct implements the full `FaceRepository` interface. Only `GetFilesBySubject` is configurable per-test; the rest are no-ops.

```
newTestClient(t, repo) → (FaceRecognitionServiceClient, cleanup)
```

Spins up a real gRPC server in-process and returns a connected client. Call `cleanup()` in `defer` to stop the server and close the connection.

---

## Test Cases

### RecognizeImage

| Test | Input | Expected |
|---|---|---|
| `TestRecognizeImage_EmptyPath_ReturnsInvalidArgument` | `file_path: ""` | `codes.InvalidArgument` |

**Coverage:** input validation guard — empty path must be rejected before reaching CompreFace.

---

### GetSubjectFiles

| Test | Input | Expected |
|---|---|---|
| `TestGetSubjectFiles_EmptySubject_ReturnsInvalidArgument` | `subject: ""` | `codes.InvalidArgument` |
| `TestGetSubjectFiles_ReturnsFilesFromRepo` | `subject: "phoebe"`, mock returns `["a.jpg", "b.jpg"]` | `resp.Subject == "phoebe"`, `len(resp.Filenames) == 2` |

**Coverage:** input validation + happy path — verifies mock repo is wired correctly and response is mapped to proto fields.

---

### RecognizeBatch (bidirectional stream)

| Test | Input | Expected |
|---|---|---|
| `TestRecognizeBatch_EmptyPath_ReturnsEmptyPersons` | stream: `file_path: ""` then `CloseSend` | `resp.Persons` is empty, no error |

**Coverage:** stream lifecycle — client can open a stream, send a request, close the send side, and receive a valid (empty) response. `GetFaceFromImage` short-circuits on empty path so no HTTP call is made.

---

## What is NOT covered (next steps)

- `RecognizeImage` with a real image path — requires CompreFace running (integration test)
- `InsertFaceAndImage` failure path — mock repo returning an error should yield `codes.Internal`
- `RecognizeBatch` with multiple messages in one stream
- Concurrent batch requests
