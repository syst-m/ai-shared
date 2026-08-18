# PR: Qwen3.8 Chat Template Fixes for OpenClaw

**Date:** 2026-08-17
**OpenClaw HEAD:** `fdcd353543249e76346a504333c941b22443e770` ("updates to local bin")
**Source:** [froggeric/Qwen-Fixed-Chat-Templates](https://huggingface.co/froggeric/Qwen-Fixed-Chat-Templates) (v22.1, 2026-08-16)

---

## Executive Summary

Qwen3.8 introduces significant changes to the chat template that break local inference in OpenClaw's llama.cpp integration. This PR documents four critical issues and provides ready-to-apply fixes.

### Models Affected
- `Qwen3.8-2.4T-A95B` (2.4T MoE, 95B activated, text-only, thinking required)
- `Qwen3.8-27B` (27B dense, vision-language, thinking on by default)

---

## Issue 1: Tool Call Crashes with Standard OpenAI JSON String Arguments

### Problem
The official Qwen3.8 template crashes with:
```
TypeError: Can only get item pairs from a mapping
```
when clients send tool arguments as JSON strings (standard OpenAI API behavior). The official template uses `tool_call.arguments|items` which only works on Python dict objects, not on JSON-serialized strings from OpenAI-compatible clients.

### Root Cause (Official Template)
```jinja
{%- for args_name, args_value in tool_call.arguments|items %}
```
The `|items` filter crashes when `tool_call.arguments` is a string (e.g., `'{"path": "/tmp/file.txt"}'`).

### Fix
Replace the `|items` loop with a type-aware handler that supports both dict and JSON string arguments:

```jinja
{%- if tool_call.arguments is defined and tool_call.arguments != '' %}
    {%- if tool_call.arguments is mapping %}
        {%- for args_name, args_value in tool_call.arguments|items %}
            {{- '<parameter=' + args_name + '>\\n' }}
            {%- set args_value = args_value | string if args_value is string else args_value | tojson | safe %}
            {{- args_value }}
            {{- '\\n</parameter>\\n' }}
        {%- endfor %}
    {%- elif tool_call.arguments is string and tool_call.arguments %}
        {{- tool_call.arguments }}
    {%- endif %}
{%- endif %}
```

### Froggeric's Full Fix (v22.1)
The froggeric template adds a universal tool argument handler:
```jinja
{%- if tc.arguments is defined and tc.arguments is not none %}
    {%- if tc.arguments is mapping %}
        {%- set _args = tc.arguments | tojson %}
    {%- elif tc.arguments is string and tc.arguments %}
        {%- set _args = tc.arguments %}
    {%- endif %}
{%- endif %}
```
Plus XML-format equivalent:
```jinja
{%- if tc.arguments is defined and tc.arguments is not none %}
    {%- if tc.arguments is mapping %}
        {%- for args_name, args_value in tc.arguments.items() %}
            {{- '<parameter=' + args_name + '>\\n' }}
            {%- if args_value is mapping or (args_value is sequence and args_value is not string) %}
                {%- set _av = args_value | tojson %}
            {%- else %}
                {%- set _av = args_value | string %}
            {%- endif %}
            {%- if max_tool_arg_chars > 0 and _av | length > max_tool_arg_chars %}
                {{- _av[:max_tool_arg_chars] + '\\n[TRUNCATED - original length ' ~ (_av | length | string) ~ ' chars]' }}
            {%- else %}
                {{- _av }}
            {%- endif %}
            {{- '\\n</parameter>\\n' }}
        {%- endfor %}
    {%- elif tc.arguments is string and tc.arguments %}
        {{- tc.arguments }}
    {%- endif %}
{%- endif %}
```

---

## Issue 2: Chat History "Poisoning" from Blank Thinking Tags

### Problem
The official Qwen3.8 template injects blank `<think></think>` blocks before real thoughts in multi-turn chat history. This "empty think poisoning" causes:
- 80%+ premature turn abort rate in agentic loops
- Model associates empty thoughts with immediate tool calls
- Degenerate reasoning spirals on repeated failures

### Root Cause
The official template extracts `reasoning_content` from messages but then unconditionally wraps it in `<think>...</think>` tags during rendering. When a message has no reasoning content (e.g., a plain text response), it still emits `<think>\n\n</think>\n\n` — a blank think block that confuses the model.

### Official Template (Buggy)
```jinja
{%- if preserve_thinking is undefined or preserve_thinking is true or loop.index0 > ns.last_query_index %}
    {{- '[REMOVED_SPECIAL_TOKEN]' + message.role + '\\n<think>\\n' + reasoning_content + '\\n</think>\\n\\n' + content }}
{%- else %}
    {{- '[REMOVED_SPECIAL_TOKEN]' + message.role + '\\n' + content }}
{%- endif %}
```
When `reasoning_content` is empty, this produces: `assistant\n<think>\n\n</think>\n\n<content>`

### Froggeric's Fix (v22.1)
Only emit thinking tags when there's actual reasoning content:
```jinja
{%- if (_preserve_thinking or loop.index0 > ns.last_query_index) and reasoning_content %}
    {{- '[REMOVED_SPECIAL_TOKEN]assistant\\n<think>\\n' + reasoning_content + '\\n</think>\\n\\n' + content }}
{%- else %}
    {{- '[REMOVED_SPECIAL_TOKEN]assistant\\n' + content }}
{%- endif %}
```

Plus robust reasoning extraction from content when `reasoning_content` field is absent:
```jinja
{%- set _think_end = '' %}
{%- if content.startswith('</think>') %}
    {%- set _think_end = '</think>' %}
{%- elif content.startswith('</thinking>') %}
    {%- set _think_end = '</thinking>' %}
{%- elif '\\n</think>' in content %}
    {%- set _think_end = '\\n</think>' %}
{%- elif '\\n</thinking>' in content %}
    {%- set _think_end = '\\n</thinking>' %}
{%- elif '\\n</ think>' in content %}
    {%- set _think_end = '\\n</ think>' %}
{%- elif '\\n</think >' in content %}
    {%- set _think_end = '\\n</think >' %}
{%- endif %}
```

---

## Issue 3: Thinking Mode Bugs

### Problem A: `enable_thinking=false` Causes Fatal Exception
The official Qwen3.8 template throws a runtime exception when `enable_thinking=false` is passed. This breaks fast mode (non-reasoning responses) entirely.

**Official:** Hard-locked — thinking cannot be disabled.

**Froggeric fix:** Removes the lockdown, supports both kwargs and inline tags:
```jinja
{%- set enable_thinking = enable_thinking if enable_thinking is defined else true %}
{%- set _effort_raw = (reasoning_effort | string | lower) if reasoning_effort is defined and reasoning_effort is not none else 'medium' %}
{%- if _effort_raw == 'none' %}
    {%- set _initial_thinking = false %}
{%- endif %}
```

### Problem B: Default `reasoning_effort=xhigh` Burns Token Budgets
The official template defaults to `xhigh` reasoning effort, which injects a long instruction text and pushes the model to explore alternate branches until hitting `max_tokens`, returning zero content.

**Froggeric fix:** Defaults to `medium` (zero injected tokens), preserving 100% prefix cache parity:
```jinja
{%- set _initial_effort = 'medium' %}
{%- if _effort_raw == 'high' or _effort_raw == 'xhigh' or _effort_raw == 'max' %}
    {%- set _initial_effort = 'xhigh' %}
{%- elif _effort_raw == 'minimal' or _effort_raw == 'low' %}
    {%- set _initial_effort = 'low' %}
{%- endif %}
```

### Problem C: No Per-Prompt Thinking Control
The official template has no way to control thinking depth per-message in chat interfaces.

**Froggeric fix:** Inline chat tags for per-prompt steering:
- `<|think_low|>` — concise reasoning
- `<|think_medium|>` — standard reasoning
- `<|think_xhigh|>` — deep reasoning
- `<|think_off|>` — fast mode (no reasoning)
- `<|think_on|>` — enable reasoning
- `<|think_minimal|>` — alias for low

All tags are automatically stripped from the rendered prompt.

---

## Issue 4: Need for `--reasoning-format deepseek` in llama.cpp

### Problem
When connecting coding agents (OpenCode, Claude Code, Pi.dev) to `llama-server`, raw thinking tokens leak into the text stream and can stop tool calls midway. The `reasoning_content` field must be properly separated.

### Fix
Start llama-server with:
```bash
llama-server -m your_model.gguf --jinja --chat-template-file chat_template.jinja --reasoning-format deepseek
```

The `--reasoning-format deepseek` flag extracts `<think>` blocks into the dedicated `reasoning_content` API response field.

### Native CLI Flag
Recent llama.cpp builds support `--reasoning-preserve` directly for 100% prefix KV cache retention.

---

## OpenClaw-Specific Changes Required

### 4a. llama.cpp Preset Config

**Current** (`dist/extensions/llama-cpp/index.js`, line ~406):
```javascript
return [
    "version = 1",
    "",
    `[${chatId}]`,
    `model = ${chatModelPath}`,
    `ctx-size = ${contextSize}`,
    `n-predict = ${maxTokens}`,
    "jinja = true",
    "",
    ...
].join("\n");
```

**Required changes:**
1. Add `--reasoning-format deepseek` to llama-server args
2. Add `--chat-template-file` pointing to the fixed template
3. Optionally add `--reasoning-preserve`

**New preset format:**
```ini
version = 1

[qwen3.8-27b]
model = /path/to/Qwen3.8-27B-Q4_K_M.gguf
ctx-size = 65536
n-predict = 2048
jinja = true
```

**New llama-server args:**
```javascript
args: [
    "--host", "127.0.0.1",
    "--port", String(port),
    "--models-preset", presetPath,
    "--models-max", "2",
    "--metrics",
    "--no-ui",
    "--reasoning-format", "deepseek",
    "--chat-template-file", "/path/to/chat_template.jinja",  // optional, preset handles it
],
```

### 4b. Model Catalog Entry

Add Qwen3.8 models to the `qwen` provider plugin catalog:

```json
{
    "id": "qwen3.8-27b",
    "name": "Qwen3.8-27B",
    "reasoning": true,
    "input": ["text", "image"],
    "contextWindow": 262144,
    "maxTokens": 65536,
    "compat": {
        "thinkingFormat": "qwen-chat-template"
    }
}
```

For the 2.4T-A95B (API-only):
```json
{
    "id": "qwen3.8-2.4t-a95b",
    "name": "Qwen3.8-2.4T-A95B",
    "reasoning": true,
    "input": ["text"],
    "contextWindow": 1000000,
    "maxTokens": 65536,
    "compat": {
        "thinkingFormat": "qwen-chat-template"
    }
}
```

### 4c. Chat Template Kwargs

OpenClaw already supports `chat_template_kwargs` via `compat.thinkingFormat: "qwen-chat-template"`. The thinking layer in `packages/ai/src/transports/openai-completions-params.ts` already handles this:

```typescript
// Currently sends:
{ "chat_template_kwargs": { "enable_thinking": true/false } }
```

**Enhancement needed:** Also pass `reasoning_effort` and `preserve_thinking`:
```typescript
// Should also send:
{
    "chat_template_kwargs": {
        "enable_thinking": true,
        "reasoning_effort": "medium",  // or "xhigh", "low"
        "preserve_thinking": true
    }
}
```

---

## Fixed Chat Template: `chat_template.jinja`

The complete fixed template from froggeric (v22.1) is provided below. This is a drop-in replacement for the official template.

```jinja
{%- set template_version = "qwen3.8-froggeric-v22.1" %}
{%- set _tool_format = tool_call_format if tool_call_format is defined else 'xml' %}
{%- set image_count = namespace(value=0) %}
{%- set video_count = namespace(value=0) %}
{%- set add_vision_id = add_vision_id if add_vision_id is defined else false %}
{%- set enable_thinking = enable_thinking if enable_thinking is defined else true %}
{%- set auto_disable_thinking_with_tools = auto_disable_thinking_with_tools if auto_disable_thinking_with_tools is defined else false %}
{%- if preserve_reasoning is defined and preserve_reasoning is not none %}
    {%- set _preserve_thinking = preserve_reasoning %}
{%- elif preserve_thinking is defined and preserve_thinking is not none %}
    {%- set _preserve_thinking = preserve_thinking %}
{%- else %}
    {%- set _preserve_thinking = true %}
{%- endif %}
{%- set max_tool_arg_chars = max_tool_arg_chars if max_tool_arg_chars is defined else 0 %}
{%- set max_tool_response_chars = max_tool_response_chars if max_tool_response_chars is defined else 0 %}
{%- set _has_tools = (tools is defined and tools and tools is iterable and tools is not mapping) %}
{%- set _effort_raw = (reasoning_effort | string | lower) if reasoning_effort is defined and reasoning_effort is not none else 'medium' %}
{%- set _initial_thinking = enable_thinking %}
{%- set _initial_effort = 'medium' %}
{%- if _effort_raw == 'none' %}
    {%- set _initial_thinking = false %}
    {%- set _initial_effort = 'medium' %}
{%- elif _effort_raw == 'minimal' or _effort_raw == 'low' %}
    {%- set _initial_effort = 'low' %}
{%- elif _effort_raw == 'high' or _effort_raw == 'xhigh' or _effort_raw == 'max' %}
    {%- set _initial_effort = 'xhigh' %}
{%- else %}
    {%- set _initial_effort = 'medium' %}
{%- endif %}
{%- set ns_state = namespace(thinking=_initial_thinking, effort=_initial_effort) %}
{%- if auto_disable_thinking_with_tools and _has_tools %}
    {%- set ns_state.thinking = false %}
{%- endif %}
{%- for msg in messages %}
    {%- if msg.role == 'system' or msg.role == 'developer' or msg.role == 'user' %}
        {%- if msg.content is string %}
            {%- if '<|think_off|>' in msg.content %}
                {%- set ns_state.thinking = false %}
            {%- elif '<|think_on|>' in msg.content %}
                {%- set ns_state.thinking = true %}
            {%- elif '<|think_xhigh|>' in msg.content or '<|think_high|>' in msg.content %}
                {%- set ns_state.thinking = true %}
                {%- set ns_state.effort = 'xhigh' %}
            {%- elif '<|think_low|>' in msg.content or '<|think_minimal|>' in msg.content %}
                {%- set ns_state.thinking = true %}
                {%- set ns_state.effort = 'low' %}
            {%- elif '<|think_medium|>' in msg.content %}
                {%- set ns_state.thinking = true %}
                {%- set ns_state.effort = 'medium' %}
            {%- endif %}
        {%- elif msg.content is iterable and msg.content is not mapping %}
            {%- for item in msg.content %}
                {%- if item is string %}
                    {%- set _item_text = item %}
                {%- elif item is mapping and 'text' in item and item.text is string %}
                    {%- set _item_text = item.text %}
                {%- else %}
                    {%- set _item_text = '' %}
                {%- endif %}
                {%- if _item_text %}
                    {%- if '<|think_off|>' in _item_text %}
                        {%- set ns_state.thinking = false %}
                    {%- elif '<|think_on|>' in _item_text %}
                        {%- set ns_state.thinking = true %}
                    {%- elif '<|think_xhigh|>' in _item_text or '<|think_high|>' in _item_text %}
                        {%- set ns_state.thinking = true %}
                        {%- set ns_state.effort = 'xhigh' %}
                    {%- elif '<|think_low|>' in _item_text or '<|think_minimal|>' in _item_text %}
                        {%- set ns_state.thinking = true %}
                        {%- set ns_state.effort = 'low' %}
                    {%- elif '<|think_medium|>' in _item_text %}
                        {%- set ns_state.thinking = true %}
                        {%- set ns_state.effort = 'medium' %}
                    {%- endif %}
                {%- endif %}
            {%- endfor %}
        {%- endif %}
    {%- endif %}
{%- endfor %}
{%- set reasoning_instructions = '' %}
{%- if ns_state.thinking %}
    {%- if ns_state.effort == 'xhigh' %}
        {%- set reasoning_instructions = 'Reasoning effort is set to xhigh. Please think carefully through the task, validate key assumptions, consider plausible alternatives, and prioritize correctness, consistency, and clarity in the final answer.' %}
    {%- elif ns_state.effort == 'low' %}
        {%- set reasoning_instructions = 'Reasoning effort is set to low. Keep your thinking brief and focused, moving directly to the conclusion without unnecessary elaboration.' %}
    {%- endif %}
{%- endif %}
{%- macro render_content(content, do_vision_count, is_system_content=false) %}
    {%- if content is string %}
        {{- content }}
    {%- elif content is iterable and content is not mapping %}
        {%- for item in content %}
            {%- if item is mapping %}
                {%- if item.type == 'image' or 'image' in item or 'image_url' in item %}
                    {%- if is_system_content %}
                        {{- raise_exception('System message cannot contain images.') }}
                    {%- endif %}
                    {%- if do_vision_count %}
                        {%- set image_count.value = image_count.value + 1 %}
                    {%- endif %}
                    {%- if add_vision_id %}
                        {{- 'Picture ' ~ image_count.value ~ ': ' }}
                    {%- endif %}
                    {{- '' }}
                {%- elif item.type == 'video' or 'video' in item %}
                    {%- if is_system_content %}
                        {{- raise_exception('System message cannot contain videos.') }}
                    {%- endif %}
                    {%- if do_vision_count %}
                        {%- set video_count.value = video_count.value + 1 %}
                    {%- endif %}
                    {%- if add_vision_id %}
                        {{- 'Video ' ~ video_count.value ~ ': ' }}
                    {%- endif %}
                    {{- 'ปะกิ' }}
                {%- elif 'text' in item %}
                    {{- item.text }}
                {%- else %}
                    {{- raise_exception('Unexpected item type in content.') }}
                {%- endif %}
            {%- else %}
                {{- item | string }}
            {%- endif %}
        {%- endfor %}
    {%- elif content is none or content is undefined %}
        {{- '' }}
    {%- else %}
        {{- raise_exception('Unexpected content type.') }}
    {%- endif %}
{%- endmacro %}
{%- if not messages %}
    {{- raise_exception('No messages provided.') }}
{%- endif %}
{%- set _first_role = messages[0].role %}
{%- if _first_role == 'system' or _first_role == 'developer' %}
    {%- set _sys_msg = messages[0] %}
    {%- set _msgs = messages[1:] %}
{%- else %}
    {%- set _sys_msg = none %}
    {%- set _msgs = messages %}
{%- endif %}
{%- set _sc = '' %}
{%- if _sys_msg is not none %}
    {%- set _sc = render_content(_sys_msg.content, false, true) | trim %}
    {%- if '<|think_off|>' in _sc %}{%- set _sc = _sc.split('<|think_off|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_on|>' in _sc %}{%- set _sc = _sc.split('<|think_on|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_xhigh|>' in _sc %}{%- set _sc = _sc.split('<|think_xhigh|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_high|>' in _sc %}{%- set _sc = _sc.split('<|think_high|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_medium|>' in _sc %}{%- set _sc = _sc.split('<|think_medium|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_low|>' in _sc %}{%- set _sc = _sc.split('<|think_low|>') | join('') | trim %}{%- endif %}
    {%- if '<|think_minimal|>' in _sc %}{%- set _sc = _sc.split('<|think_minimal|>') | join('') | trim %}{%- endif %}
{%- endif %}
{%- if _has_tools %}
    {{- '[REMOVED_SPECIAL_TOKEN]system\n' }}
    {%- if reasoning_instructions %}
        {{- reasoning_instructions + '\n\n' }}
    {%- endif %}
    {{- '# Tools\n\nYou have access to the following functions:\n\n<tools>' }}
    {%- for tool in tools %}
        {{- '\n' }}
        {{- tool | tojson }}
    {%- endfor %}
    {{- '\n</tools>' }}
    {%- if _tool_format == 'json' %}
        {{- '\n\nIf you choose to call a function ONLY reply in the following format with NO suffix:\n\n<think>\nBrief explanation of tool call\n</think>\n<tool_call>\n{"name": "example_function_name", "arguments": {"example_parameter_1": "value_1", "example_parameter_2": "This is the value for the second parameter"}}\n</tool_call>\n\n<IMPORTANT>\nReminder:\n- You can use the <think></think> block to plan your next tool call OR to synthesize data and formulate your final response to the user.\n- ALL explanation and reasoning MUST be placed strictly inside the <think></think> block.\n- Function calls MUST follow the specified format: a single JSON object with "name" and "arguments" keys inside <tool_call></tool_call> XML tags.\n- If you choose to call a tool, you MUST output the <tool_call> block IMMEDIATELY after thinking, with NO conversational text before it.\n- The <tool_call> tag MUST be at the very beginning of a new line, with NO spaces or indentation before it.\n- To call multiple functions, output a separate, completely closed <tool_call></tool_call> block for EACH function. Do NOT nest <tool_call> blocks.\n- If you have all necessary data, provide your final answer directly to the user without any tool call.\n</IMPORTANT>' }}
    {%- else %}
        {{- '\n\nIf you choose to call a function ONLY reply in the following format with NO suffix:\n\n<think>\nBrief explanation of tool call\n</think>\n<tool_call>\n<function=example_function_name>\n<parameter=example_parameter_1>\nvalue_1\n</parameter>\n<parameter=example_parameter_2>\nThis is the value for the second parameter\nthat can span\nmultiple lines\n</parameter>\n</function>\n</tool_call>\n\n<IMPORTANT>\nReminder:\n- You can use the <think></think> block to plan your next tool call OR to synthesize data and formulate your final response to the user.\n- ALL explanation and reasoning MUST be placed strictly inside the <think></think> block.\n- Function calls MUST follow the specified format: an inner <function=...></function> block must be nested within <tool_call></tool_call> XML tags.\n- If you choose to call a tool, you MUST output the <tool_call> block IMMEDIATELY after thinking, with NO conversational text before it.\n- The <tool_call> and <function> tags MUST be at the very beginning of a new line, with NO spaces or indentation before them.\n- To call multiple functions, output a separate, completely closed <tool_call></tool_call> block for EACH function. Do NOT nest <tool_call> blocks.\n- If you have all necessary data, provide your final answer directly to the user without any tool call.\n</IMPORTANT>' }}
    {%- endif %}
    {%- if _sc %}
        {{- '\n\n' + _sc }}
    {%- endif %}
    {{- '[REMOVED_SPECIAL_TOKEN]\n' }}
{%- else %}
    {%- if _sc %}
        {{- '[REMOVED_SPECIAL_TOKEN]system\n' + (reasoning_instructions + '\n\n' if reasoning_instructions else '') + _sc + '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- elif reasoning_instructions %}
        {{- '[REMOVED_SPECIAL_TOKEN]system\n' + reasoning_instructions + '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- endif %}
{%- endif %}
{%- set _last_idx = _msgs | length - 1 %}
{%- set ns = namespace(multi_step_tool=true, last_query_index=_last_idx) %}
{%- for message in _msgs[::-1] %}
    {%- set index = (_msgs | length - 1) - loop.index0 %}
    {%- if ns.multi_step_tool and message.role == 'user' %}
        {%- set _rc = render_content(message.content, false) | trim %}
        {%- if not (_rc.startswith('<tool_response>') and _rc.endswith('</tool_response>')) %}
            {%- set ns.multi_step_tool = false %}
            {%- set ns.last_query_index = index %}
        {%- endif %}
    {%- endif %}
{%- endfor %}
{%- if ns.multi_step_tool %}
    {%- if _last_idx > 50 %}
        {%- set ns.last_query_index = _last_idx %}
    {%- else %}
        {%- set ns.last_query_index = 0 %}
    {%- endif %}
{%- endif %}
{%- set ns2 = namespace(prev_role='', consecutive_failures=0) %}
{%- for message in _msgs %}
    {%- set is_system = (message.role == "system" or message.role == "developer") %}
    {%- set content = render_content(message.content, true, is_system) | trim %}
    {%- if is_system or message.role == 'user' %}
        {%- if '<|think_off|>' in content %}{%- set content = content.split('<|think_off|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_on|>' in content %}{%- set content = content.split('<|think_on|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_xhigh|>' in content %}{%- set content = content.split('<|think_xhigh|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_high|>' in content %}{%- set content = content.split('<|think_high|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_medium|>' in content %}{%- set content = content.split('<|think_medium|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_low|>' in content %}{%- set content = content.split('<|think_low|>') | join('') | trim %}{%- endif %}
        {%- if '<|think_minimal|>' in content %}{%- set content = content.split('<|think_minimal|>') | join('') | trim %}{%- endif %}
    {%- endif %}
    {%- if is_system %}
        {{- '[REMOVED_SPECIAL_TOKEN]system\n' + content + '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- elif message.role == 'user' %}
        {%- set ns2.consecutive_failures = 0 %}
        {{- '[REMOVED_SPECIAL_TOKEN]user\n' + content + '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- elif message.role == 'assistant' %}
        {%- set reasoning_content = '' %}
        {%- if message.reasoning_content is defined and message.reasoning_content is not none %}
            {%- if message.reasoning_content is string %}
                {%- set reasoning_content = message.reasoning_content %}
            {%- else %}
                {%- set reasoning_content = message.reasoning_content | string %}
            {%- endif %}
        {%- elif message.thinking is defined and message.thinking is not none %}
            {%- if message.thinking is string %}
                {%- set reasoning_content = message.thinking %}
            {%- else %}
                {%- set reasoning_content = message.thinking | string %}
            {%- endif %}
        {%- else %}
            {%- set _think_end = '' %}
            {%- if content.startswith('</think>') %}
                {%- set _think_end = '</think>' %}
            {%- elif content.startswith('</thinking>') %}
                {%- set _think_end = '</thinking>' %}
            {%- elif '\n</think>' in content %}
                {%- set _think_end = '\n</think>' %}
            {%- elif '\n</thinking>' in content %}
                {%- set _think_end = '\n</thinking>' %}
            {%- elif '\n</ think>' in content %}
                {%- set _think_end = '\n</ think>' %}
            {%- elif '\n</think >' in content %}
                {%- set _think_end = '\n</think >' %}
            {%- endif %}
            {%- if _think_end %}
                {%- if 'thinking' in _think_end %}
                    {%- set _think_start = '<thinking>' %}
                {%- else %}
                    {%- set _think_start = '<think>' %}
                {%- endif %}
                {%- set reasoning_content = content.split(_think_end)[0].rstrip('\n') %}
                {%- if _think_start in reasoning_content %}
                    {%- set reasoning_content = reasoning_content.split(_think_start)[-1].lstrip('\n') %}
                {%- endif %}
                {%- set content = content.split(_think_end)[-1].lstrip('\n') %}
            {%- endif %}
        {%- endif %}
        {%- set reasoning_content = reasoning_content | trim %}
        {%- if (_preserve_thinking or loop.index0 > ns.last_query_index) and reasoning_content %}
            {{- '[REMOVED_SPECIAL_TOKEN]assistant\n<think>\n' + reasoning_content + '\n</think>\n\n' + content }}
        {%- else %}
            {{- '[REMOVED_SPECIAL_TOKEN]assistant\n' + content }}
        {%- endif %}
        {%- if message.tool_calls is defined and message.tool_calls and message.tool_calls is iterable and message.tool_calls is not mapping %}
            {%- for tool_call in message.tool_calls %}
                {%- if tool_call.function is defined and tool_call.function is not none %}
                    {%- set tc = tool_call.function %}
                {%- else %}
                    {%- set tc = tool_call %}
                {%- endif %}
                {%- set tc_name = tc.name if (tc.name is defined and tc.name is not none) else '' %}
                {%- if _tool_format == 'json' %}
                    {%- if not loop.first or content | trim %}
                        {{- '\n\n' }}
                    {%- endif %}
                    {%- set _args = '{}' %}
                    {%- if tc.arguments is defined and tc.arguments is not none %}
                        {%- if tc.arguments is mapping %}
                            {%- set _args = tc.arguments | tojson %}
                        {%- elif tc.arguments is string and tc.arguments %}
                            {%- set _args = tc.arguments %}
                        {%- endif %}
                    {%- endif %}
                    {{- '<tool_call>\n{"name": ' }}{{- tc_name | tojson }}{{- ', "arguments": ' }}{{- _args }}{{- '}\n</tool_call>' }}
                {%- else %}
                    {%- if loop.first %}
                        {%- if content | trim %}
                            {{- '\n\n<tool_call>\n<function=' + tc_name + '>\n' }}
                        {%- else %}
                            {{- '<tool_call>\n<function=' + tc_name + '>\n' }}
                        {%- endif %}
                    {%- else %}
                        {{- '\n\n<tool_call>\n<function=' + tc_name + '>\n' }}
                    {%- endif %}
                    {%- if tc.arguments is defined and tc.arguments is not none %}
                        {%- if tc.arguments is mapping %}
                            {%- for args_name, args_value in tc.arguments.items() %}
                                {{- '<parameter=' + args_name + '>\n' }}
                                {%- if args_value is mapping or (args_value is sequence and args_value is not string) %}
                                    {%- set _av = args_value | tojson %}
                                {%- else %}
                                    {%- set _av = args_value | string %}
                                {%- endif %}
                                {%- if max_tool_arg_chars > 0 and _av | length > max_tool_arg_chars %}
                                    {{- _av[:max_tool_arg_chars] + '\n[TRUNCATED - original length ' ~ (_av | length | string) ~ ' chars]' }}
                                {%- else %}
                                    {{- _av }}
                                {%- endif %}
                                {{- '\n</parameter>\n' }}
                            {%- endfor %}
                        {%- elif tc.arguments is string and tc.arguments %}
                            {{- tc.arguments }}
                        {%- endif %}
                    {%- endif %}
                    {{- '</function>\n</tool_call>' }}
                {%- endif %}
            {%- endfor %}
        {%- endif %}
        {{- '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- elif message.role == 'tool' %}
        {%- set _content_lower = content | lower %}
        {%- set _content_head = _content_lower[:80] %}
        {%- if content | length < 500 and '$ ' not in content and 'took ' not in _content_lower and ('"error":' in _content_head or 'error:' in _content_head or 'err!' in _content_head or 'fatal:' in _content_head or 'exception:' in _content_head or 'traceback' in _content_head or 'command not found' in _content_head or 'invalid syntax' in _content_head or 'failed to' in _content_head) %}
            {%- set ns2.consecutive_failures = ns2.consecutive_failures + 1 %}
        {%- else %}
            {%- set ns2.consecutive_failures = 0 %}
        {%- endif %}
        {%- if ns2.prev_role != 'tool' %}
            {{- '[REMOVED_SPECIAL_TOKEN]user' }}
        {%- endif %}
        {%- if max_tool_response_chars > 0 and content | length > max_tool_response_chars %}
            {%- set content = content[:max_tool_response_chars] + '\n[TRUNCATED - original length ' ~ (content | length | string) ~ ' chars]' %}
        {%- endif %}
        {{- '\n<tool_response>\n' + content }}
        {%- if ns2.consecutive_failures >= 2 %}
            {{- '\n\n⚠️ SYSTEM WARNING: ' ~ ns2.consecutive_failures ~ ' consecutive tool errors detected. Your previous approach is incorrect. You MUST use a fundamentally different approach or corrected arguments.' }}
        {%- elif ns2.consecutive_failures == 1 %}
            {{- '\n\n⚠️ SYSTEM WARNING: The previous tool call returned an error. Diagnose the failure and retry with completely corrected arguments.' }}
        {%- endif %}
        {{- '\n\rho' }}
        {%- if loop.last %}
            {{- '[REMOVED_SPECIAL_TOKEN]\n' }}
        {%- else %}
            {%- set _next_role = _msgs[loop.index0 + 1].role %}
            {%- if _next_role != 'tool' %}
                {{- '[REMOVED_SPECIAL_TOKEN]\n' }}
            {%- endif %}
        {%- endif %}
    {%- else %}
        {{- '[REMOVED_SPECIAL_TOKEN]user\n[' + message.role + ']: ' + content + '[REMOVED_SPECIAL_TOKEN]\n' }}
    {%- endif %}
    {%- set ns2.prev_role = message.role %}
{%- endfor %}
{%- if add_generation_prompt %}
    {{- '[REMOVED_SPECIAL_TOKEN]assistant\n' }}
    {%- if not ns_state.thinking %}
        {{- '<think>\n\n</think>\n\n' }}
    {%- else %}
        {{- '<think>\n' }}
    {%- endif %}
{%- endif %}
```

---

## Installation Guide

### Option A: Drop-in llama.cpp replacement

1. Save `chat_template.jinja` to your model directory
2. Start llama-server with:
   ```bash
   llama-server -m Qwen3.8-27B-Q4_K_M.gguf \
       --jinja \
       --chat-template-file chat_template.jinja \
       --reasoning-format deepseek
   ```

### Option B: vLLM

1. Replace the `chat_template` string in `tokenizer_config.json` with the contents of `chat_template_oneline.txt`
2. Serve with:
   ```bash
   vllm serve Qwen/Qwen3.8-27B --tool-call-parser qwen3_xml
   ```

### Option C: LM Studio

1. Open your Qwen model in the right side panel
2. Scroll to **Prompt Template**
3. Replace with the contents of `chat_template.jinja`
4. Click **Save**

---

## OpenClaw Configuration Changes

### For llama.cpp local models

The llama.cpp extension in `dist/extensions/llama-cpp/index.js` needs two changes:

1. **Add `--reasoning-format deepseek` to the server args** — ensures proper separation of thinking content from response text
2. **Add `--chat-template-file` to the preset** — points to the fixed template

### For Qwen Cloud API (remote)

The existing `compat.thinkingFormat: "qwen-chat-template"` configuration already works. Add the following to your model config:

```json5
{
  models: {
    providers: {
      qwen: {
        models: [
          {
            id: "qwen3.8-27b",
            name: "Qwen3.8-27B",
            reasoning: true,
            input: ["text", "image"],
            contextWindow: 262144,
            maxTokens: 65536,
            compat: { thinkingFormat: "qwen-chat-template" }
          }
        ]
      }
    }
  }
}
```

### For vLLM local models

```json5
{
  models: {
    providers: {
      vllm: {
        models: [
          {
            id: "Qwen/Qwen3.8-27B",
            name: "Qwen3.8-27B",
            reasoning: true,
            input: ["text", "image"],
            contextWindow: 262144,
            maxTokens: 65536,
            compat: { thinkingFormat: "qwen-chat-template" }
          }
        ]
      }
    }
  }
}
```

---

## Summary of All Fixes

| Issue | Official Behavior | Fixed Behavior |
|-------|------------------|----------------|
| Tool call crashes | `|items` crashes on JSON string args | Type-aware handler for both dicts and strings |
| Empty think poisoning | Blank `<think>\n\n</think>\n\n` injected | Only emits thinking tags when content exists |
| Thinking lockdown | `enable_thinking=false` throws exception | Full freedom to disable reasoning |
| Token budget burn | Default `xhigh` with injected instructions | Default `medium` (zero injected tokens) |
| No per-prompt control | No inline thinking tags | `<|think_low|>`, `<|think_medium|>`, `<|think_xhigh|>`, `<|think_off|>` |
| Reasoning format | Raw thinking in text stream | `--reasoning-format deepseek` extracts to `reasoning_content` |
| KV cache safety | Mutated past turns break cache | Chronological history, 100% KV cache hit rate |
| minijinja compat | Python-only filters crash C++ engines | All filters rewritten for minijinja |
| AST performance | Deep nesting drops llama.cpp speed 80% | Flattened AST architecture |
| False error detection | Broad matching triggers retry loops | Strict structural guards |
| Dynamic truncation | No payload limits | `max_tool_arg_chars`, `max_tool_response_chars` |

---

## References

- [froggeric/Qwen-Fixed-Chat-Templates](https://huggingface.co/froggeric/Qwen-Fixed-Chat-Templates) — v22.1 (2026-08-16)
- [Qwen/Qwen3.8-2.4T-A95B](https://huggingface.co/Qwen/Qwen3.8-2.4T-A95B) — Official model card
- [Qwen/Qwen3.8-27B](https://huggingface.co/Qwen/Qwen3.8-27B) — Official model card
- [Qwen3.8 Blog Post](https://qwen.ai/blog?id=qwen3.8)
- OpenClaw llama.cpp extension: `dist/extensions/llama-cpp/index.js`
- OpenClaw thinking format: `packages/ai/src/transports/openai-completions-params.ts`
- OpenClaw types: `packages/llm-core/src/types.ts` (lines 496-504)
