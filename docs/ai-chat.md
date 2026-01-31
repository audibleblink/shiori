# AI Chat

Shiori includes an AI-powered chat interface that allows you to ask natural language questions about your bookmarked content. Instead of searching by keywords, you can ask questions like "What did I save about Go error handling?" and receive synthesized answers with links to the source bookmarks.

## Requirements

The AI Chat feature requires:

- **PostgreSQL database** with pgvector extension (SQLite and MySQL are not supported)
- **Ollama** running locally for embedding generation
- **Anthropic API key** for chat responses

See [AI Chat Admin Guide](ai-chat-admin.md) for setup instructions.

## Using the Chat Interface

### Opening the Chat Panel

Click the **Chat** button in the sidebar navigation to open the chat panel. The panel slides out from the right side of the screen.

On desktop, the chat panel appears alongside your bookmarks. On mobile devices, it overlays the main content.

### Asking Questions

Type your question in the input field at the bottom of the panel and press Enter or click Send. Questions can be natural language queries about your bookmarked content:

- "What articles did I save about database optimization?"
- "Summarize the Go programming tutorials I bookmarked"
- "What security best practices have I saved?"
- "Compare the different approaches to error handling in my bookmarks"

### Understanding Responses

Responses are generated based on the content of your bookmarks. Each response includes:

- **Answer**: A synthesized response drawn from your bookmarked content
- **Source Citations**: Links to the bookmarks that informed the answer

Click on any source citation to navigate directly to that bookmark.

### Multi-Turn Conversations

The chat maintains conversation context within your browser session. You can ask follow-up questions that reference previous messages:

- User: "What did I save about React?"
- Assistant: [response about React bookmarks]
- User: "What about performance optimization specifically?"
- Assistant: [response focused on React performance from your bookmarks]

### Starting a New Conversation

Click the **New Conversation** button to clear the chat history and start fresh. This is useful when you want to ask about an unrelated topic.

### Session Persistence

Chat history is stored in your browser session. It persists if you navigate to other pages within Shiori, but clears when you:

- Close the browser tab
- Close the browser
- Click "New Conversation"

Chat history is intentionally not stored on the server for privacy.

## Limitations

### Content Scope

- The chat only has access to bookmarks you own
- It can only answer questions based on content that has been saved and embedded
- Newly saved bookmarks may take a few seconds to become searchable via chat

### Answer Quality

- Answers are only as good as the content in your bookmarks
- If no relevant content is found, the assistant will indicate this rather than guessing
- Very specific or niche questions may not yield results if you haven't saved relevant content

### Database Requirement

AI Chat requires PostgreSQL with the pgvector extension. Users with SQLite or MySQL databases will see the chat panel in a disabled state with instructions for upgrading.

## Troubleshooting

### Chat panel shows "AI features not enabled"

This means the server hasn't been configured for AI features. Contact your administrator or see the [admin guide](ai-chat-admin.md).

### Chat panel shows "PostgreSQL required"

AI Chat requires PostgreSQL with pgvector. If you're using SQLite or MySQL, you'll need to migrate to PostgreSQL to use this feature.

### No relevant results found

If the assistant says it can't find relevant information:

1. Try rephrasing your question
2. Use more specific keywords that might appear in your bookmarks
3. Check if you have bookmarks on the topic (they may not have been embedded yet)
4. Wait a few moments if you just saved new bookmarks

### Slow responses

Chat response time depends on:

- Number of bookmarks being searched
- Ollama server performance (for query embedding)
- Anthropic API response time

First responses may take 2-3 seconds. If consistently slow, check with your administrator.

### Connection errors

If you see connection errors:

1. Refresh the page and try again
2. Check your internet connection
3. The server may be restarting - wait a moment and retry

## Privacy

- Chat messages are processed on the server to search your bookmarks
- Messages are sent to Anthropic's API for response generation
- No chat history is stored on the Shiori server
- Only your own bookmarks are accessible via chat
