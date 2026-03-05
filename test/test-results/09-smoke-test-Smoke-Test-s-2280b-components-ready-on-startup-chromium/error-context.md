# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - navigation [ref=e2]:
    - button "Chat" [ref=e3] [cursor=pointer]
    - button "Settings" [ref=e4] [cursor=pointer]
    - button "Identity Files" [ref=e5] [cursor=pointer]
  - generic [ref=e6]:
    - generic [ref=e7]:
      - generic [ref=e8]:
        - generic [ref=e9]: Model
        - combobox [ref=e10]:
          - option "Anthropic / claude-sonnet-4-5" [selected]
          - option "Anthropic / claude-opus-4"
          - option "OpenAI / gpt-4o"
          - option "OpenRouter / claude-sonnet-4-5"
      - generic [ref=e13]: Welcome to WebClaw! Type a message to start chatting.
      - generic [ref=e15]:
        - textbox "Type a message... (Enter to send, Shift+Enter for newline)" [ref=e16]
        - button "Send" [ref=e17] [cursor=pointer]
    - heading "Tool Activity" [level=2] [ref=e20]
```