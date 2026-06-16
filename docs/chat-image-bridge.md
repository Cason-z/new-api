# Chat Image Bridge

Some OpenAI-compatible clients send image generation models through `/v1/chat/completions`.
When the requested model is recognized as an image generation model, New API converts the last user message into an internal `/v1/images/generations` request.

Behavior:
- The prompt is read from the latest non-empty `user` message text.
- The image request defaults to `n: 1` and `size: 1024x1024` when the chat request does not provide them.
- The upstream image response is returned to the client as a chat completion.
- If the original chat request used `stream: true`, the response is emitted as chat-compatible SSE chunks.
- Returned images are embedded as Markdown image links. Base64 image responses are returned as `data:image/png;base64,...`.

This keeps clients such as Cherry Studio compatible without requiring them to call `/v1/images/generations` directly.
