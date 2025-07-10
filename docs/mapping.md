# Comprehensive Mapping Between Anthropic Messages API and OpenAI Chat Completions API

<!--toc:start-->
- [Comprehensive Mapping Between Anthropic Messages API and OpenAI Chat Completions API](#comprehensive-mapping-between-anthropic-messages-api-and-openai-chat-completions-api)
  - [Introduction](#introduction)
  - [API Endpoints](#api-endpoints)
  - [Count Tokens Endpoint](#count-tokens-endpoint)
  - [Request Mapping (Anthropic -> OpenAI)](#request-mapping-anthropic---openai)
    - [Authentication & Headers](#authentication--headers)
    - [Request Body Parameters](#request-body-parameters)
    - [Messages and Content Blocks](#messages-and-content-blocks)
      - [Anthropic Messages Format](#anthropic-messages-format)
      - [OpenAI Messages Format](#openai-messages-format)
      - [Mapping Messages (User & Assistant Turns)](#mapping-messages-user--assistant-turns)
      - [Mapping Tool Result Messages (User Turn)](#mapping-tool-result-messages-user-turn)
    - [System Prompts](#system-prompts)
    - [Tools and Functions](#tools-and-functions)
      - [Tool Definitions](#tool-definitions)
      - [Mapping Tool Definitions](#mapping-tool-definitions)
      - [Tool Usage Control](#tool-usage-control)
      - [Mapping Tool Choices](#mapping-tool-choices)
  - [Response Mapping (OpenAI -> Anthropic)](#response-mapping-openai---anthropic)
    - [Response Body Structure](#response-body-structure)
      - [Anthropic Response Structure](#anthropic-response-structure)
      - [OpenAI Response Structure](#openai-response-structure)
    - [Mapping Response Fields](#mapping-response-fields)
    - [Mapping Assistant Message Content](#mapping-assistant-message-content)
      - [Text Content](#text-content)
      - [Tool Use Content (Function Call)](#tool-use-content-function-call)
    - [Stop Reasons and Finish Reasons](#stop-reasons-and-finish-reasons)
      - [Mapping Stop/Finish Reasons](#mapping-stopfinish-reasons)
    - [Usage Statistics](#usage-statistics)
      - [Mapping Usage](#mapping-usage)
  - [Streaming Responses](#streaming-responses)
    - [Anthropic Streaming Format](#anthropic-streaming-format)
    - [OpenAI Streaming Format](#openai-streaming-format)
    - [Mapping Streaming Responses (OpenAI -> Anthropic)](#mapping-streaming-responses-openai---anthropic)
  - [Error Handling](#error-handling)
    - [Error Response Structures](#error-response-structures)
      - [Anthropic Error Structure](#anthropic-error-structure)
      - [OpenAI Error Structure](#openai-error-structure)
    - [HTTP Status Code Mapping](#http-status-code-mapping)
    - [API Client Errors](#api-client-errors)
    - [Implementation Considerations](#implementation-considerations)
  - [Important Considerations & Gaps](#important-considerations--gaps)
<!--toc:end-->

## Introduction

This document provides a detailed, field-by-field mapping between Anthropic’s **Claude v3 Messages API** and OpenAI’s **GPT-4 (Chat Completions API)**, based on deep research into both APIs. It covers translating requests (Anthropic -> OpenAI) and responses (OpenAI -> Anthropic), focusing on accuracy for features like message roles, content blocks, system prompts, tool usage (function calling), and streaming. This mapping is crucial for building a proxy server that allows clients using the Anthropic API format to interact seamlessly with OpenAI's backend. Differences in fields, values, and behavior are noted, along with required transformations and potential gaps.

**Scope:** Assumes a stateless translator (full context per request) supporting Claude 3 features via OpenAI's equivalent mechanisms (e.g., function calling). The reference models are Claude 3 and GPT-4/GPT-4-Turbo.

---

## API Endpoints

- **Anthropic Messages API Endpoint:**
  ```
  POST /v1/messages
  ```
- **OpenAI Chat Completions API Endpoint:**
  ```
  POST /v1/chat/completions
  ```

---

## Count Tokens Endpoint

- **Anthropic:** Provides `POST /v1/messages/count_tokens` for calculating input token count.
- **OpenAI:** No direct HTTP endpoint. Token usage is returned in completion responses. For estimation, use libraries like `tiktoken`. The proxy needs its own logic for consistent token counting if pre-computation is required.

---

## Request Mapping (Anthropic -> OpenAI)

### Authentication & Headers

- **Authorization:** Translate between header formats.
  - Anthropic: `x-api-key: YOUR_ANTHROPIC_API_KEY`
  - OpenAI: `Authorization: Bearer YOUR_SSY_API_KEY`
- **API Version:** Include Anthropic's version header if needed by the client.
  - Anthropic: `anthropic-version: VERSION_STRING` (e.g., `2023-06-01`)
  - OpenAI: Version is typically tied to the model or API path, not a specific header.

### Request Body Parameters

Mapping Anthropic request fields to OpenAI:

| Anthropic Parameter  | OpenAI Parameter        | Mapping and Notes                                                                                                                                                                                               |
| -------------------- | ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `model`              | `model`                 | Map the requested Claude model name (e.g., `claude-3-opus-20240229`) to a corresponding OpenAI model name (e.g., `gpt-4-turbo`). The proxy must maintain this mapping.                                            |
| `system` (string)    | `messages`              | If present, prepend the `system` string as the *first* message in the OpenAI `messages` array: `{"role": "system", "content": "<system_string>"}`. If absent, omit this system message.                         |
| `messages`           | `messages`              | Translate the array of message objects. Roles and content structure require careful conversion (see details below).                                                                                              |
| `max_tokens`         | `max_tokens`            | Direct mapping. The maximum number of tokens to generate in the response. Ensure value respects the target OpenAI model's limits.                                                                                |
| `stop_sequences`     | `stop`                  | Direct mapping. Pass the array of stop strings. Note OpenAI returns `finish_reason: "stop"` if triggered.                                                                                                       |
| `stream`             | `stream`                | Direct mapping (`true`/`false`). If `true`, handle streaming response conversion (see Streaming section).                                                                                                      |
| `temperature`        | `temperature`           | Direct mapping (float, 0.0 to ~2.0). Both default around 1.0.                                                                                                                                                   |
| `top_p`              | `top_p`                 | Direct mapping (float, 0.0 to 1.0). OpenAI defaults to 1.0. Anthropic recommends using only one of `temperature` or `top_p`.                                                                                     |
| `top_k`              | Not supported           | Anthropic-specific sampling parameter. OpenAI Chat API does not support `top_k`. **Action:** Ignore/drop this parameter. Behavior cannot be perfectly replicated.                                                 |
| `metadata.user_id`   | `user`                  | Map the optional `metadata.user_id` string from Anthropic to OpenAI's top-level `user` string for tracking/monitoring. Other fields in `metadata` are not mappable.                                               |
| `tools`              | `functions`             | Map the array of Anthropic tool definitions to OpenAI's `functions` array. (See Tool Definitions mapping).                                                                                                      |
| `tool_choice`        | `function_call`         | Map Anthropic's tool choice mechanism to OpenAI's function call control. (See Mapping Tool Choices).                                                                                                            |
| `stream_options`     | Not directly supported  | Anthropic's `stream_options` (e.g., `include_usage`) doesn't map directly. Usage in OpenAI streams is not provided per-chunk. Proxy needs to handle usage reporting at the end of the stream.                  |


### Messages and Content Blocks

#### Anthropic Messages Format

-   `role`: Must be `user` or `assistant` within the `messages` array.
-   `content`: Can be a string OR an array of content blocks (`text`, `image`, `tool_result`).

```json
// Anthropic User Message with Text
{
  "role": "user",
  "content": "Hello, Claude."
}

// Anthropic User Message with Image (Requires special handling)
{
  "role": "user",
  "content": [
    {"type": "text", "text": "Describe this image:"},
    {"type": "image", "source": {...}} // GPT-4 Chat API cannot process this directly
  ]
}

// Anthropic User Message with Tool Result (Follow-up after tool_use)
{
  "role": "user",
  "content": [
    {"type": "tool_result", "tool_use_id": "toolu_123", "content": "<tool_output_string_or_JSON>"}
  ]
}

// Anthropic Assistant Message (Response)
{
  "role": "assistant",
  "content": [{"type": "text", "text": "Hi there!"}] // Or tool_use block
}
```

#### OpenAI Messages Format

-   `role`: Can be `system`, `user`, `assistant`, or `function`.
-   `content`: Typically a string (for `system`, `user`, `assistant`, `function` roles). Can be `null` for `assistant` messages containing only a `function_call`.
-   `function_call`: Optional object in `assistant` messages indicating a function invocation.
-   `name`: Required for `function` role messages, identifying the function whose result is provided.

```json
// OpenAI System Message
{"role": "system", "content": "You are helpful."}

// OpenAI User Message
{"role": "user", "content": "Hello."}

// OpenAI Assistant Message (Text Response)
{"role": "assistant", "content": "Hi there!"}

// OpenAI Assistant Message (Function Call Request)
{"role": "assistant", "content": null, "function_call": {"name": "get_weather", "arguments": "{\"location\": \"Paris\"}"}}

// OpenAI Function Result Message (Provides result back to model)
{"role": "function", "name": "get_weather", "content": "{\"temperature\": 22, \"unit\": \"celsius\"}"}
```

#### Mapping Messages (User & Assistant Turns)

-   **Roles:**
    -   Anthropic `user` -> OpenAI `user`.
    -   Anthropic `assistant` -> OpenAI `assistant`.
    -   Anthropic only allows `user` and `assistant` in the `messages` array. OpenAI also uses `system` (mapped from top-level `system`) and `function` (mapped from Anthropic `tool_result`, see below).
-   **Content Conversion:**
    -   **Text:** If Anthropic `content` is a string or a single `text` block, use the text directly as OpenAI `content`. If multiple `text` blocks, concatenate them into a single string for OpenAI `content`.
    -   **Image Blocks:** Anthropic `image` blocks (`type: "image"`) are **not supported** by the standard OpenAI Chat Completions API. **Action:** The proxy must either:
        1.  Omit the image block entirely.
        2.  Attempt conversion (e.g., use a separate Vision API or tool to generate a text description) and include the description in the OpenAI message `content`. This is a significant gap.
    -   **Partial Assistant Prefill:** Anthropic allows the last message to be `role: "assistant"` to provide a prefix for the model to continue. OpenAI does **not** support this "prefill" mechanism directly. **Action:** This feature cannot be reliably proxied. Best approach is to disallow or ignore such partial assistant messages in the request.

#### Mapping Tool Result Messages (User Turn)

This is critical for multi-turn tool use:

-   **Anthropic Input:** A `user` message containing one or more `tool_result` content blocks.
    ```json
    {
      "role": "user",
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "toolu_abc",
          "content": "{\"temp\": 72}" // Can be string or JSON object/array
        }
      ]
    }
    ```
-   **OpenAI Output:** Map **each** `tool_result` block to a separate OpenAI message with `role: "function"`.
    -   `role`: `"function"`
    -   `name`: The name of the tool/function corresponding to the `tool_use_id`. The proxy needs to track the mapping between the `tool_use_id` generated in the previous assistant response and the tool name.
    -   `content`: The output/result provided in the `content` field of the `tool_result`. OpenAI expects this to be a string (usually JSON stringified).

    ```json
    {
      "role": "function",
      "name": "get_current_weather", // Retrieved via toolu_abc mapping
      "content": "{\"temp\": 72}"
    }
    ```
-   **Placement:** These `function` role messages should be placed in the OpenAI `messages` array immediately after the `assistant` message that contained the corresponding `function_call` (which was mapped from Anthropic's `tool_use`).

### System Prompts

-   **Anthropic:** Uses a top-level `system` parameter (string).
-   **OpenAI:** Uses a message with `role: "system"` at the beginning of the `messages` array.
-   **Mapping:** Convert Anthropic's `system` string into `{"role": "system", "content": "<system_string>"}` and make it the first element (`messages[0]`) in the OpenAI request `messages` list. Ensure this is done for every turn if the conversation is stateful on the client side.

### Tools and Functions

#### Tool Definitions

-   **Anthropic `tools`:** Array of objects, each with `name`, `description`, `input_schema` (JSON Schema).
-   **OpenAI `functions`:** Array of objects, each with `name`, `description`, `parameters` (JSON Schema).

#### Mapping Tool Definitions

-   Directly map each Anthropic `tool` to an OpenAI `function`:
    -   `name` -> `name`
    -   `description` -> `description`
    -   `input_schema` -> `parameters` (both expect JSON Schema format).
-   **Built-in Tools:** Anthropic mentions beta built-in tools (e.g., `bash`). OpenAI has no direct equivalent.Proxy should treat these as custom tools/functions if needed, defining the expected schema, or simply not support them.

#### Tool Usage Control

-   **Anthropic `tool_choice`:** Object controlling how the model uses tools (`type`: `auto`, `any`, `tool`, `none`).
-   **OpenAI `function_call`:** String or object controlling function usage (`auto`, `none`, `{"name": "..."}`).

#### Mapping Tool Choices

| Anthropic `tool_choice`                     | OpenAI `function_call`      | Notes                                                                                                   |
| ------------------------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------- |
| `{"type": "auto"}` (or omitted)             | `"auto"` (or omitted)       | Model decides whether to call a function and which one. (Default behavior for both).                      |
| `{"type": "any"}`                           | `"auto"`                    | Force the model to use *any* available tool. OpenAI has no direct equivalent. Map to `"auto"` and potentially add instructions in the system prompt (e.g., "You must use a tool if appropriate"). |
| `{"type": "tool", "name": "tool_name"}`     | `{"name": "tool_name"}`     | Force the model to call the specified tool/function.                                                    |
| Omitted / Default                           | Omitted / Default (`"auto"`) | If Anthropic `tool_choice` is not provided, use OpenAI's default (`"auto"`).                             |
| *(Note: Anthropic also has a "none" type implied by omitting tools)* | `"none"` | If no tools are provided, or if explicit prevention is needed, OpenAI can use `"none"`. This doesn't seem to directly map from an Anthropic option but might be needed for specific proxy logic. |


---

## Response Mapping (OpenAI -> Anthropic)

### Response Body Structure

#### Anthropic Response Structure

```json
{
  "id": "msg_...", // Message ID
  "type": "message", // Fixed type for successful response
  "role": "assistant", // Fixed role
  "model": "claude-3-opus-...", // Model name requested by client
  "content": [ ... ], // Array of content blocks (text or tool_use)
  "stop_reason": "end_turn", // Reason generation stopped
  "stop_sequence": null, // Sequence that caused stop, if applicable
  "usage": {
    "input_tokens": 10,
    "output_tokens": 25
  }
}
```

#### OpenAI Response Structure

```json
{
  "id": "chatcmpl-...", // Completion ID
  "object": "chat.completion",
  "created": 1677652288, // Timestamp
  "model": "gpt-4-turbo-...", // Model name that generated response
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello!", // Can be null if function_call is present
        "function_call": null // Or {"name": "...", "arguments": "..."}
      },
      "logprobs": null,
      "finish_reason": "stop" // Reason generation stopped
    }
    // Potentially more choices if n > 1, but Anthropic only expects one.
  ],
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 12,
    "total_tokens": 21
  },
  "system_fingerprint": "fp_..."
}
```

### Mapping Response Fields

Translate fields from the **first choice** (`choices[0]`) of the OpenAI response to the Anthropic format:

| OpenAI Field                  | Anthropic Field     | Mapping and Notes                                                                                                                                           |
| ----------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                          | `id`                | Use the OpenAI `id` (e.g., `"chatcmpl-..."`) or generate a new one in Anthropic format (e.g., `"msg_..."`). Using OpenAI's ID is simpler for traceability.     |
| `object` ("chat.completion")  | `type`              | Set Anthropic `type` to `"message"` for successful completions.                                                                                               |
| `model`                       | `model`             | Return the **Anthropic model name** that the client originally requested (e.g., `"claude-3-opus-..."`), not the OpenAI model name used internally.             |
| `choices[0].message.role`     | `role`              | Should always be `"assistant"` from OpenAI. Set Anthropic `role` to `"assistant"`.                                                                            |
| `choices[0].message.content`  | `content`           | Map based on whether it's text or null (see below).                                                                                                         |
| `choices[0].message.function_call` | `content`      | If present, map to a `tool_use` content block (see below).                                                                                                  |
| `choices[0].finish_reason`    | `stop_reason`       | Map the reason code (see Stop/Finish Reasons table).                                                                                                        |
| N/A                           | `stop_sequence`     | Set only if OpenAI `finish_reason` was `"stop"` AND a stop sequence from the request was matched. Echo the matched sequence here. OpenAI doesn't return this. |
| `usage`                       | `usage`             | Map token counts (see Usage Statistics).                                                                                                                    |
| `created`, `system_fingerprint`, `logprobs`, `choices[0].index` | N/A           | These OpenAI fields have no equivalent in the Anthropic response. Omit them.                                                                |

### Mapping Assistant Message Content

Map `choices[0].message` to Anthropic's `content` array:

#### Text Content

-   **If OpenAI `message.content` is a non-null string and `function_call` is null:**
    -   Create an Anthropic `content` array containing a single `text` block:
        ```json
        "content": [{"type": "text", "text": "<OpenAI message.content string>"}]
        ```

#### Tool Use Content (Function Call)

-   **If OpenAI `message.function_call` is present (and `content` might be null):**
    -   Create an Anthropic `content` array containing a single `tool_use` block:
        ```json
        "content": [
          {
            "type": "tool_use",
            "id": "toolu_<generated_unique_id>", // Generate a unique ID for this tool call
            "name": "<OpenAI function_call.name>",
            "input": <parsed_arguments_object> // Parse the arguments JSON string into a JSON object/value
          }
        ]
        ```
    -   **Crucially:**
        -   Generate a unique `id` (e.g., `toolu_...`) for the `tool_use` block. This ID must be tracked by the proxy if the client will send back a `tool_result` referencing it.
        -   Parse the `arguments` string from OpenAI (which is JSON *stringified*) into an actual JSON object or primitive value for Anthropic's `input` field.

### Stop Reasons and Finish Reasons

#### Mapping Stop/Finish Reasons

| OpenAI `finish_reason` | Anthropic `stop_reason` | Notes                                                                                                                                       |
| ---------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `"stop"`               | `"end_turn"`            | Model finished naturally. (Default case)                                                                                                     |
| `"stop"`               | `"stop_sequence"`       | **Condition:** If the stop occurred because a sequence in `stop_sequences` was hit. Proxy needs to detect this. Set `stop_sequence` field too. |
| `"length"`             | `"max_tokens"`          | Model hit the `max_tokens` limit.                                                                                                           |
| `"function_call"`      | `"tool_use"`            | Model is requesting a tool/function call. This corresponds to the `tool_use` content block.                                                 |
| `"content_filter"`     | `"stop_sequence"` ?     | OpenAI flagged content. Anthropic has no direct equivalent. Could map to `stop_sequence` (as an external stop) or handle as an error.        |
| `null` (streaming)     | `null` (streaming)      | Generation is ongoing during streaming.                                                                                                     |

### Usage Statistics

#### Mapping Usage

Map fields from OpenAI's `usage` object to Anthropic's `usage` object:

| OpenAI Usage Field    | Anthropic Usage Field |
| --------------------- | --------------------- |
| `prompt_tokens`       | `input_tokens`        |
| `completion_tokens`   | `output_tokens`       |
| `total_tokens`        | *(Omit)*              |

Resulting Anthropic structure:
```json
"usage": {
  "input_tokens": <OpenAI prompt_tokens>,
  "output_tokens": <OpenAI completion_tokens>
}
```

---

## Streaming Responses

Both APIs use Server-Sent Events (SSE), but formats differ significantly. The proxy must translate OpenAI SSE chunks into Anthropic SSE events.

### Anthropic Streaming Format

-   Event-based, with named events (`message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`, `ping`).
-   Structured JSON payloads for each event type.
-   Sends message metadata (`message_start`), then content blocks incrementally (`content_block_*` events), then closing metadata (`message_delta`, `message_stop`).
-   Text deltas (`content_block_delta` with `delta.type: "text_delta"`) and tool argument deltas (`delta.type: "input_json_delta"`) are possible.

```sse
event: message_start
data: {"type": "message_start", "message": {"id": "msg_123", "type": "message", "role": "assistant", ...}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": " world"}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}

# If tool call occurs...
event: content_block_start
data: {"type": "content_block_start", "index": 1, "content_block": {"type": "tool_use", "id": "toolu_abc", "name": "...", "input": {}}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "input_json_delta", "partial_json": "{\"location\":\""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "input_json_delta", "partial_json": "Paris\"}"}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 1}


event: message_delta # Contains stop_reason, usage updates
data: {"type": "message_delta", "delta": {"stop_reason": "tool_use", ...}, "usage": {"output_tokens": 15}}

event: message_stop
data: {"type": "message_stop"}
```

### OpenAI Streaming Format

-   Single stream of unnamed `data:` events containing `chat.completion.chunk` JSON objects.
-   Each chunk has a `delta` field with incremental changes (`role`, `content`, or `function_call` fragments).
-   First chunk usually contains `delta: {"role": "assistant"}`.
-   Subsequent chunks contain `delta: {"content": "..."}` or `delta: {"function_call": {"name": "...", "arguments": "..."}}` (arguments often streamed as partial JSON string fragments).
-   Final chunk has `finish_reason` set.
-   Stream ends with `data: [DONE]`. **Usage is NOT included in SSE chunks.**

```sse
data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"role":"assistant"}, "finish_reason":null}]}

data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"content":"Hello"}, "finish_reason":null}]}

data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"content":" world"}, "finish_reason":null}]}

# Function call streaming example
data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"function_call": {"name": "get_weather"}}, "finish_reason":null}]}

data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"function_call": {"arguments": "{\"loca"}}, "finish_reason":null}]}

data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"function_call": {"arguments": "tion\":\"P"}}, "finish_reason":null}]}

data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{"function_call": {"arguments": "aris\"}"}}, "finish_reason":null}]}


data: {"id":"...", "object":"chat.completion.chunk", "choices":[{"index":0, "delta":{}, "finish_reason":"function_call"}]}

data: [DONE]
```

### Mapping Streaming Responses (OpenAI -> Anthropic)

The proxy must maintain state during streaming to construct Anthropic events:

1.  **On first OpenAI chunk (`delta.role`):** Send Anthropic `message_start` event containing initial message metadata (generate `message_id`, set `role: 'assistant'`, include model). Send initial `ping`? (Optional, depends on client needs).
2.  **On first OpenAI `delta.content` chunk:** Send Anthropic `content_block_start` for index 0 (`type: "text"`).
3.  **On subsequent `delta.content` chunks:** Send Anthropic `content_block_delta` with `delta.type: "text_delta"` and the content fragment.
4.  **On first OpenAI `delta.function_call.name` chunk:** Send Anthropic `content_block_start` for the next available index (e.g., 1) (`type: "tool_use"`, generate `tool_use_id`, include `name`). Accumulate arguments internally.
5.  **On OpenAI `delta.function_call.arguments` chunks:** Send Anthropic `content_block_delta` for the tool's index with `delta.type: "input_json_delta"` and the `partial_json` fragment. Reconstruct the full arguments JSON internally.
6.  **When OpenAI stream provides `finish_reason`:**
    *   If text content was streaming, send `content_block_stop` for index 0.
    *   If tool call was streaming, send `content_block_stop` for the tool's index.
    *   Map OpenAI `finish_reason` to Anthropic `stop_reason`.
    *   Send `message_delta` containing the final `stop_reason` and potentially calculated `usage` (input tokens known from request, output tokens counted from stream).
    *   Send `message_stop`.
7.  **If OpenAI stream ends (`data: [DONE]`):** Ensure all pending `content_block_stop`, `message_delta`, and `message_stop` events have been sent.
8.  **Handling Multiple Blocks:** If OpenAI hypothetically interleaved text and tool calls (unlikely but possible), manage multiple content blocks with correct indexing for `content_block_*` events.
9.  **Usage:** Since OpenAI doesn't stream usage, the proxy must calculate output tokens by summing streamed content/argument tokens (using a tokenizer like `tiktoken`) and report it in the final `message_delta`. Input tokens are calculated from the original request.

---

## Error Handling

Map error responses between the APIs.

### Error Response Structures

#### Anthropic Error Structure

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error", // Specific error type
    "message": "Error message details."
  }
}
```

#### OpenAI Error Structure

```json
{
  "error": {
    "message": "Error message details.",
    "type": "invalid_request_error", // Specific error type
    "param": null, // Parameter causing error, if applicable
    "code": null // Specific error code, if applicable
  }
}
```

### HTTP Status Code Mapping

Translate HTTP status codes and inherent error types:

| OpenAI HTTP Code | OpenAI Error Type (`error.type`)                | Anthropic HTTP Code | Anthropic Error Type (`error.type`) | Notes                                        |
| ---------------- | ----------------------------------------------- | ------------------- | ----------------------------------- | -------------------------------------------- |
| 400              | `invalid_request_error`                         | 400                 | `invalid_request_error`             | Bad request (syntax, missing fields, etc.)  |
| 401              | `authentication_error`                          | 401                 | `authentication_error`              | Invalid API key                              |
| 403              | `permission_denied_error`                       | 403                 | `permission_error`                  | Insufficient permissions/access              |
| 404              | `not_found_error`                               | 404                 | `not_found_error`                   | Resource/model not found                   |
| 429              | `rate_limit_error`                              | 429                 | `rate_limit_error`                  | Rate limit exceeded                          |
| 500              | `internal_server_error`, `api_error`            | 500                 | `api_error`                         | Internal server error on provider side       |
| 503              | `service_unavailable_error`, `overloaded_error` | 529                 | `overloaded_error`                  | Server overloaded / temporarily unavailable |
| 400              | *e.g., context length exceeded*                 | 400                 | `invalid_request_error`             | Map specific 400s appropriately          |

*Note:* Error type names might vary slightly depending on specific SDK versions. Use the canonical types where possible.

### API Client Errors

Translate common client-side errors (network issues, timeouts) consistently:

| OpenAI Client Exception     | Anthropic Equivalent Context | Notes                                               |
| --------------------------- | ---------------------------- | --------------------------------------------------- |
| `openai.APIConnectionError` | Network connectivity issue   | Could not connect to OpenAI API.                      |
| `openai.APITimeoutError`    | Request timeout issue        | Request to OpenAI timed out.                        |
| `openai.RateLimitError`     | Rate limit exceeded          | Received 429 from OpenAI.                         |
| `openai.BadRequestError`    | Invalid request              | Received 400 from OpenAI (validation, etc.).        |
| `openai.AuthenticationError`| Authentication failed        | Received 401 from OpenAI.                         |
| `openai.PermissionDeniedError`| Permission issue           | Received 403 from OpenAI.                         |
| `openai.NotFoundError`      | Resource not found           | Received 404 from OpenAI.                         |
| `openai.InternalServerError`| Upstream server error        | Received 5xx from OpenAI.                         |

The proxy should catch OpenAI client errors and return corresponding Anthropic-style HTTP errors and JSON bodies.

### Implementation Considerations

-   **Preserve Messages:** Include the original OpenAI error message within the translated Anthropic error structure for debugging.
-   **Request IDs:** Pass through relevant request IDs (`X-Request-ID`, etc.) if available.
-   **Proxy Context:** Add context indicating the error originated from the upstream provider (OpenAI).
-   **Streaming Errors:** Handle errors that occur *after* the stream has started (e.g., network drop). May require terminating the SSE stream with an error signal if possible, or logging.
-   **Retries:** Implement appropriate retry logic (e.g., exponential backoff) for transient errors like rate limits (429) or server issues (5xx).

---

## Important Considerations & Gaps

-   **Model Behavior:** Mapping APIs doesn't guarantee identical model performance, reasoning, alignment, or adherence to instructions. Claude and GPT-4 have inherent differences.
-   **Unsupported Anthropic Features:**
    -   `top_k`: Cannot be mapped.
    -   Partial Assistant Prefill: Cannot be mapped.
    -   Built-in Tools (beta): Require custom mapping or are unsupported.
-   **Unsupported Content Types:**
    -   **Images:** Standard OpenAI Chat API does not accept image inputs. This is a major gap requiring workarounds (omission, OCR/captioning).
-   **Role Mapping for Tool Results:** The conversion between Anthropic's `user` role + `tool_result` content and OpenAI's `function` role is crucial and requires careful state management in the proxy.
-   **Tool Choice `any`:** Anthropic's `{"type": "any"}` cannot be directly enforced in OpenAI; mapping to `"auto"` is the closest functional equivalent.
-   **System Prompt Handling:** Ensure the Anthropic `system` prompt is consistently prepended as the first `system` role message in the OpenAI request history for every turn.
-   **Streaming Usage:** OpenAI does not provide usage stats in stream chunks. Proxy must calculate and append at the end.
-   **Error Granularity:** Error type details might differ slightly. Aim for the closest conceptual match.

This mapping provides a comprehensive guide for translating between the Anthropic Messages API and OpenAI Chat Completions API, leveraging the detailed information from the provided research. Careful implementation considering the nuances and gaps identified is essential for a functional proxy.
