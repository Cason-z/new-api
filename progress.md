## 2026-06-16 - Task: Bridge chat-completions image model requests
### What was done
- Added backend compatibility for OpenAI-style clients that send image generation models, such as `MAI-Image-2.5`, to `/v1/chat/completions`.
- Converted the latest user chat message into an internal image generation request and returned the image result as chat-compatible Markdown, including SSE output when the client requests streaming.
- Kept normal text model requests on the existing chat path.
- Documented the compatibility behavior for future deployment and client setup.
### Testing
- Added unit coverage for converting chat messages into image generation requests.
- Local Go verification could not be executed because `go` and `gofmt` are not installed on this machine.
- Static checks were performed for referenced error codes, response DTOs, model detection, and quota settlement paths.
### Notes
- `common/model.go`: added `MAI-Image-2.5` family detection through the `mai-image-` image model prefix.
- `common/model_test.go`: added image model recognition coverage.
- `controller/chat_image_bridge.go`: added chat-to-image request detection and prompt extraction.
- `controller/chat_image_bridge_test.go`: added conversion tests for simple text, array text, and empty prompt rejection.
- `controller/relay.go`: routes image-model chat requests into the image relay mode before channel selection and billing continue.
- `relay/chat_image_bridge_handler.go`: sends the converted image request upstream and wraps the image response as chat JSON or chat SSE.
- `docs/chat-image-bridge.md`: documents the compatibility behavior.
- Rollback: revert this working tree change set or remove the listed files and restore `common/model.go` plus `controller/relay.go` to the previous commit.

## 2026-06-16 - Task: Keep playground image relay aligned
### What was done
- Added a playground image generation endpoint so the built-in playground can call image generation directly when needed.
- Extended playground client-side handling for `MAI-Image-2.5` style image models so generated images are displayed as Markdown image results.
- Let playground group selection apply consistently across playground relay endpoints.
### Testing
- Covered indirectly by the controller package compile run during `go test ./common ./controller -run 'TestIsImageGenerationModelRecognizesMAIImage|TestBuildImageRequestFromChatRequest'`.
- Frontend build was not run because frontend dependencies are not installed locally.
### Notes
- `controller/playground.go`: split temporary token setup into a reusable helper and added `PlaygroundImage`.
- `middleware/distributor.go`: allowed playground group parsing for all `/pg/` relay paths.
- `relay/constant/relay_mode.go`: recognized `/pg/images/generations` as image generation relay mode.
- `router/relay-router.go`: registered `/pg/images/generations`.
- `web/default/src/features/playground/api.ts`: added the playground image generation API call.
- `web/default/src/features/playground/hooks/use-chat-handler.ts`: routed `mai-image-` models through image generation and displayed returned images.
- `web/default/src/features/playground/types.ts`: added image generation request/response types.
- Rollback: revert the listed playground files to the previous commit; the backend chat-image bridge can remain independent.

## 2026-06-16 - Task: Verify chat image bridge locally
### What was done
- Downloaded a temporary local Go 1.25.1 toolchain under the Windows temp directory to avoid using server resources.
- Ran `gofmt` on the changed Go files.
- Ran focused tests for the new image model recognition and chat-to-image conversion behavior.
- Ran a full Go test sweep to identify remaining repository-level blockers before deployment.
### Testing
- Passed: `go test ./common ./controller -run 'TestIsImageGenerationModelRecognizesMAIImage|TestBuildImageRequestFromChatRequest'`.
- Passed: `go test ./relay -run TestNonExistent`, which compiled the relay package including the new bridge handler.
- Failed: `go test ./...` due to existing/environmental repository issues unrelated to this bridge, including missing built frontend assets (`web/classic/dist`), controller SQLite database state, Claude file conversion tests, and service channel-affinity usage cache tests.
### Notes
- No server-side build was performed.
- Changed files for this verification: none beyond formatting previously changed Go files.
- Rollback: no separate rollback is required for verification-only work; remove the temporary `%TEMP%\codex-go-1.25.1` directory if local cleanup is desired.

## 2026-06-16 - Task: Add fork-side GHCR build workflow
### What was done
- Added a minimal GitHub Actions workflow for the `codex/chat-image-bridge` branch.
- The workflow builds only the linux/amd64 Docker image and pushes it to GitHub Container Registry.
- Avoided DockerHub secrets and avoided any server-side image build.
### Testing
- Not run locally; this workflow is intended to be verified by GitHub Actions after push.
### Notes
- `.github/workflows/codex-ghcr-build.yml`: added the fork-specific GHCR build workflow.
- Rollback: delete `.github/workflows/codex-ghcr-build.yml` and push the branch again.

## 2026-06-17 - Task: Deploy chat image bridge image
### What was done
- Built the `codex/chat-image-bridge` branch in GitHub Actions under the `Cason-z/new-api` fork.
- Pulled `ghcr.io/cason-z/new-api:chat-image-bridge` on the New API server.
- Replaced the running `new-api` container with the GHCR image while preserving the old container as a rollback point.
- Verified Cherry-style image chat requests and normal text chat requests against the live service.
### Testing
- GitHub Actions run `27630488770` completed successfully for the GHCR image build.
- Live image bridge test passed: `POST /v1/chat/completions` with `MAI-Image-2.5` returned HTTP 200, `text/event-stream`, chat chunks, Markdown image content, a `data:image` URL, and no `Prompt must be provided` error.
- Live text model test passed: `POST /v1/chat/completions` with `gpt-5.4-mini` returned HTTP 200 JSON and did not include image Markdown.
### Notes
- Server container now runs `ghcr.io/cason-z/new-api:chat-image-bridge`.
- Rollback container is `new-api-backup-20260616-213308`, using the previous `calciumion/new-api:latest` image.
- Rollback command: `docker rm -f new-api && docker rename new-api-backup-20260616-213308 new-api && docker start new-api`.

## 2026-06-17 - Task: Infer image bridge dimensions from chat prompts
### What was done
- Updated the chat image bridge so Cherry-style chat requests can infer a widescreen or portrait image size from prompt text when the client does not send an explicit `size`.
- Mapped prompts such as `1920x1080p`, `1080p`, `16:9`, `横屏`, and `壁纸` to a MAI-safe widescreen size of `1365x768` instead of the old square `1024x1024` default.
- Added MAI-only `width` and `height` fields so the MAI image API receives explicit dimensions without sending those extra fields to other image models.
### Testing
- Passed: `go test ./common ./controller -run 'TestIsImageGenerationModelRecognizesMAIImage|TestBuildImageRequestFromChatRequest' -count=1`.
### Notes
- `controller/chat_image_bridge.go`: inferred image sizes from chat prompts and limited `width`/`height` emission to `MAI-Image-*` models.
- `controller/chat_image_bridge_test.go`: added coverage for widescreen prompt inference, square defaults, and non-MAI model protection.
- `dto/openai_image.go`: added optional `width` and `height` fields for MAI image requests.
- `docs/chat-image-bridge.md`: documented size inference and MAI dimension behavior.
- Rollback: revert this task's changes in the four listed files, then rebuild and redeploy the previous `chat-image-bridge` image.

## 2026-06-17 - Task: Route GPT image intents to MAI image generation
### What was done
- Updated the chat image bridge so `gpt-5.4` and `gpt-5.4-mini` chat requests with explicit image generation or image editing intent are internally routed to `MAI-Image-2.5`.
- Kept normal GPT text requests on the existing text chat path, including requests to write or refine image prompts.
- Reused the existing MAI image bridge response path so Cherry Studio receives the generated image as a chat response.
- Moved GPT image-intent model rewriting into the distributor stage so channel selection and model mapping both use `MAI-Image-2.5` instead of falling back to the original GPT text model.
### Testing
- Passed: `go test ./common ./controller -run 'TestIsImageGenerationModelRecognizesMAIImage|TestShouldBridgeGPT54ImageIntentToMAIImage|TestShouldNotBridgeGPT54NormalTextChat|TestShouldNotBridgeGPT54ImagePromptWriting|TestBuildImageRequestFromChatRequest' -count=1`.
- Passed: `go test ./common ./controller ./middleware ./service -run 'TestIsImageGenerationModelRecognizesMAIImage|TestShouldBridgeGPT54ImageIntentToMAIImage|TestShouldNotBridgeGPT54NormalTextChat|TestShouldNotBridgeGPT54ImagePromptWriting|TestBuildImageRequestFromChatRequest|TestRewriteChatImageIntentModel' -count=1`.
### Notes
- `service/chat_image_intent.go`: centralized GPT image-intent detection, latest-user-text extraction, and the `MAI-Image-2.5` target model constant.
- `middleware/distributor.go`: rewrites qualifying GPT chat requests to `MAI-Image-2.5` before channel selection.
- `middleware/distributor_test.go`: added coverage for distributor-stage GPT-to-MAI model rewriting.
- `controller/chat_image_bridge.go`: now reuses the shared GPT image-intent helper while converting chat content into image requests.
- `controller/chat_image_bridge_test.go`: added coverage for GPT image intent routing and non-image text safeguards.
- `docs/chat-image-bridge.md`: documented GPT-to-MAI automatic routing behavior.
- `progress.md`: recorded this task's implementation, verification, and rollback point.
- Rollback: revert this task's changes in the six listed files, then rebuild and redeploy the previous `chat-image-bridge` image.

## 2026-06-17 - Task: Tighten GPT image-intent detection and preserve client-facing model disguise
### What was done
- Tightened GPT-to-MAI image intent detection so the bridge can prefer explicit image-tool signals when present, while still handling plain-language image requests.
- Added boundary protections so prompt-writing, poster copywriting, and concept-explanation requests stay on the text path instead of being misrouted to image generation.
- Preserved the client-requested GPT model name in chat-image bridge responses even when the upstream request is internally rewritten to `MAI-Image-2.5`.
- Documented the stricter intent rules and the client-facing response-model behavior.
### Testing
- Passed: `go test ./common ./controller ./middleware ./relay ./service -run 'TestShouldRouteChatImageIntent|TestShouldBridgeGPT54ImageIntentToMAIImage|TestShouldNotBridgeGPT54NormalTextChat|TestShouldNotBridgeGPT54ImagePromptWriting|TestBuildImageRequestFromChatRequest|TestRewriteChatImageIntentModel|TestWriteChatImage' -count=1`.
### Notes
- `constant/context_key.go`: added a dedicated context key to preserve the client-requested model name separately from the rewritten upstream model.
- `middleware/distributor.go`: stores the original client model before GPT image-intent requests are rewritten to `MAI-Image-2.5`.
- `relay/chat_image_bridge_handler.go`: now returns the preserved client model in both JSON and SSE chat-image bridge responses.
- `relay/chat_image_bridge_handler_test.go`: added coverage for JSON and SSE response `model` disguise behavior.
- `service/chat_image_intent.go`: added tool-signal-aware intent detection, image-edit attachment handling, and tighter false-positive guards.
- `service/chat_image_intent_test.go`: added boundary coverage for logo generation, image editing, tool-driven routing, poster copywriting, and concept explanation.
- `docs/chat-image-bridge.md`: documented the stricter routing signals and preserved response model name.
- Rollback: revert this task's changes in the seven listed files, then rebuild and redeploy the previous `chat-image-bridge` image.
