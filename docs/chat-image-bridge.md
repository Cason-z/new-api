# Chat Image Bridge

Some OpenAI-compatible clients send image generation models through `/v1/chat/completions`.
When the requested model is recognized as an image generation model, New API converts the last user message into an internal `/v1/images/generations` request.

Behavior:
- The prompt is read from the latest non-empty `user` message text.
- The image request defaults to `n: 1`.
- When the chat request does not provide `size`, the bridge infers a simple size from the prompt: `1920x1080`, `1080p`, `16:9`, `横屏`, `宽屏`, `壁纸`, or `wallpaper` become `1365x768`; `9:16`, `竖屏`, `手机壁纸`, or `mobile wallpaper` become `768x1365`; otherwise it uses `1024x1024`.
- For `MAI-Image-*` models, the bridge also sends matching `width` and `height` fields because the MAI image API uses explicit dimensions.
- The upstream image response is returned to the client as a chat completion.
- If the original chat request used `stream: true`, the response is emitted as chat-compatible SSE chunks.
- Returned images are embedded as Markdown image links. Base64 image responses are returned as `data:image/png;base64,...`.

This keeps clients such as Cherry Studio compatible without requiring them to call `/v1/images/generations` directly.
