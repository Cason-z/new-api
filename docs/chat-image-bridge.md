# Chat Image Bridge

Some OpenAI-compatible clients send image generation models through `/v1/chat/completions`.
When the requested model is recognized as an image generation model, New API converts the last user message into an internal `/v1/images/generations` request.

Behavior:
- The prompt is read from the latest non-empty `user` message text.
- `gpt-5.4` and `gpt-5.4-mini` chat requests are also routed to `MAI-Image-2.5` when the latest user message clearly asks to generate or edit an image, poster, logo, cover, concept art, meme, wallpaper, avatar, or similar visual output.
- When the chat request already carries an explicit image-generation tool signal, the bridge prefers that signal over plain keyword matching.
- Requests that ask for prompt writing, copywriting, or concept explanation, such as image prompt drafting or poster copy, stay on the normal text path.
- Normal `gpt-5.4` and `gpt-5.4-mini` text requests, including requests to write image prompts or descriptions, stay on the text chat path.
- GPT image-intent routing is applied before channel selection so the request uses a channel that actually serves `MAI-Image-2.5`, instead of selecting a GPT text channel first and failing later during image generation.
- The image request defaults to `n: 1`.
- When the chat request does not provide `size`, the bridge infers a simple size from the prompt: `1920x1080`, `1080p`, `16:9`, `横屏`, `宽屏`, `壁纸`, or `wallpaper` become `1365x768`; `9:16`, `竖屏`, `手机壁纸`, or `mobile wallpaper` become `768x1365`; otherwise it uses `1024x1024`.
- For `MAI-Image-*` models, the bridge also sends matching `width` and `height` fields because the MAI image API uses explicit dimensions.
- The upstream image response is returned to the client as a chat completion.
- If the original chat request used `stream: true`, the response is emitted as chat-compatible SSE chunks.
- The chat response keeps the client-requested GPT model name in the returned `model` field even when the upstream request is internally routed to `MAI-Image-2.5`.
- Returned images are embedded as Markdown image links. Base64 image responses are returned as `data:image/png;base64,...`.

This keeps clients such as Cherry Studio compatible without requiring them to call `/v1/images/generations` directly.
